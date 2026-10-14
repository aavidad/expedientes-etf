package expediente

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	documentos "github.com/aavidad/expedientes-etf/internal/puertos/salida/documento"
	"slices"
	"strings"

	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/dominio/expediente/ubicacion"
	registro "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/catalogo"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
	usuarios "github.com/aavidad/expedientes-etf/internal/puertos/salida/identidad"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

type Dependencias struct {
	Conservacion          *ServicioConservacion
	Traslados             salida.Traslados
	Seguimientos          salida.Seguimientos
	Avisos                salida.ConsultaAvisos
	CargasHistoricas      salida.CargasHistoricas
	Intervencion          salida.Intervenciones
	CatalogosIntervencion salida.CatalogosIntervencion
	Menores               salida.Menores
	CatalogosMenores      salida.CatalogosMenores
	Familias              salida.Familias
	CatalogosFamilia      salida.CatalogosFamilia
	SMI                   catalogo.SMI
	SalidasServicio       salida.SalidasServicio
	EmisionesFinalizacion documentos.ComprobacionFinalizacion
	Cierres               salida.Cierres
	ConsultaCierreSPM     entrada.ConsultaCierreSPM
	SalidasSPM            salida.SalidasSPM
	Bandeja               salida.Bandeja
	Hitos                 salida.Hitos

	Propuestas            salida.PropuestasCierre
	MotivosCierre         catalogo.MotivosCierre
	Etiquetas             salida.EtiquetasUbicacion
	Borradores            salida.Borradores
	Confirmaciones        salida.Confirmaciones
	Decisiones            salida.Decisiones
	Aceptaciones          salida.Aceptaciones
	Numeracion            catalogo.Numeracion
	Usuarios              usuarios.Usuarios
	Repositorio           salida.Repositorio
	Transacciones         plataforma.Transacciones
	Registro              registro.Registro
	RegistroTransaccional salida.RegistroTransaccional
	Validador             salida.ValidadorFichero
	Antivirus             plataforma.Antivirus
	Lector                salida.LectorAnexo
	Catalogo              catalogo.Repositorio
	NombresMunicipios     salida.NombresMunicipios
	Reloj                 salida.Reloj
	Identificadores       salida.Identificadores
}

type Servicio struct{ d Dependencias }

var _ entrada.Derivaciones = (*Servicio)(nil)

func Nuevo(d Dependencias) *Servicio { return &Servicio{d} }

func identidad(a dominio.Actor) plataforma.Identidad {
	return plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}
}

func (s *Servicio) evento(a dominio.Actor, accion, id, resultado string) auditoria.Datos {
	if a.UsuarioID == "" {
		a.UsuarioID, a.Perfiles = "sistema", nil
	}
	return auditoria.Datos{OcurridoEn: s.d.Reloj.Ahora().UTC().Truncate(1000), UsuarioID: a.UsuarioID, Perfiles: a.Perfiles,
		Accion: accion, Entidad: "expediente", EntidadID: id, ExpedienteID: id, Resultado: resultado}
}

func (s *Servicio) Rechazar(ctx context.Context, a dominio.Actor, clave string) error {
	if (a.UsuarioID != "" && !dominio.IDValido(a.UsuarioID)) || s.d.Registro == nil || s.d.Reloj == nil {
		return dominio.ErrNoDisponible
	}
	accion := "expediente.subida_rechazada"
	if clave == dominio.ErrInfectado.Error() {
		accion = "expediente.anexo_infectado"
	}
	if clave == dominio.ErrAcceso.Error() {
		accion = "expediente.acceso_denegado"
	}
	_, err := s.d.Registro.Confirmar(ctx, s.evento(a, accion, "", "denegado"))
	if err != nil {
		return dominio.ErrNoDisponible
	}
	return nil
}

func (s *Servicio) rechazar(ctx context.Context, a dominio.Actor, err error) (dominio.Derivacion, error) {
	if fallo := s.Rechazar(ctx, a, err.Error()); fallo != nil {
		return dominio.Derivacion{}, fallo
	}
	return dominio.Derivacion{}, err
}

func (s *Servicio) Subir(ctx context.Context, a dominio.Actor, subida entrada.Subida) (dominio.Derivacion, error) {
	if err := a.Autorizar(); err != nil {
		return s.rechazar(ctx, a, err)
	}
	if len(subida.Contenido) > dominio.MaxBytes {
		return s.rechazar(ctx, a, dominio.ErrTamano)
	}
	if s.d.Validador == nil || s.d.Lector == nil || s.d.Antivirus == nil || s.d.Identificadores == nil {
		return dominio.Derivacion{}, dominio.ErrNoDisponible
	}
	formato, err := s.d.Validador.Validar(ctx, subida.Nombre, subida.Contenido)
	if err != nil {
		return s.rechazar(ctx, a, err)
	}
	estado, falloAV := s.d.Antivirus.Revisar(ctx, subida.Contenido)
	if err := ctx.Err(); err != nil {
		return dominio.Derivacion{}, err
	}
	if estado == plataforma.AntivirusInfectado {
		return s.rechazar(ctx, a, dominio.ErrInfectado)
	}
	antivirus := "no_revisado"
	if estado == plataforma.AntivirusLimpio && falloAV == nil {
		antivirus = "limpio"
	}
	extraida, err := s.d.Lector.Leer(ctx, formato, subida.Contenido)
	if err != nil {
		return s.rechazar(ctx, a, err)
	}
	if err := dominio.ValidarExtraccion(extraida); err != nil {
		return s.rechazar(ctx, a, err)
	}
	huella := sha256.Sum256(subida.Contenido)
	if err := ctx.Err(); err != nil {
		return dominio.Derivacion{}, err
	}
	lecturaID, err := s.d.Identificadores.Nuevo()
	if err != nil {
		return dominio.Derivacion{}, dominio.ErrNoDisponible
	}
	extraida = dominio.DatosPermitidos(extraida)
	d := dominio.Derivacion{
		Lectura: dominio.LecturaAnexo{ID: lecturaID, Formato: formato, TamanoBytes: len(subida.Contenido), HuellaSHA256: hex.EncodeToString(huella[:]),
			Antivirus: antivirus, PlantillaDetectada: extraida.Version, LeidaPor: a.UsuarioID, LeidaEn: s.d.Reloj.Ahora().UTC().Truncate(1000), CamposDudosos: []string{}},
		Datos: dominio.DatosDerivacion{EsquemaVersion: "1", Extraccion: extraida}}
	for _, c := range extraida.Campos {
		if c.Dudoso && !slices.Contains(d.Lectura.CamposDudosos, c.Codigo) {
			d.Lectura.CamposDudosos = append(d.Lectura.CamposDudosos, c.Codigo)
		}
	}
	if antivirus == "no_revisado" {
		d.Lectura.AntivirusDetalle = "expediente.antivirus_no_revisado"
		d.Datos.Extraccion.Avisos = append(d.Datos.Extraccion.Avisos, "expediente.antivirus_no_revisado")
	}
	err = s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		v, err := s.d.Catalogo.Vigente(ctx, "asignacion_municipios", d.Lectura.LeidaEn)
		if err != nil {
			return err
		}
		datosCatalogo, err := s.d.Catalogo.Datos(ctx, v)
		if err != nil {
			return err
		}
		textos, err := s.d.NombresMunicipios.Nombres(ctx)
		if err != nil {
			return err
		}
		municipios := make([]ubicacion.Municipio, 0, len(datosCatalogo.Municipios))
		for _, m := range datosCatalogo.Municipios {
			nombre, ok := textos[m.ClaveTexto]
			if !ok || strings.TrimSpace(nombre) == "" {
				return dominio.ErrNoDisponible
			}
			var variantes []string
			for clave, grafia := range textos {
				if strings.HasPrefix(clave, m.ClaveTexto+".variante.") {
					variantes = append(variantes, grafia)
				}
			}
			slices.Sort(variantes)
			municipios = append(municipios, ubicacion.Municipio{Codigo: m.INE, ClaveTexto: m.ClaveTexto, Nombre: nombre, Variantes: variantes, AmbitoDiputacion: m.AmbitoDiputacion})
		}
		direccion, valoracion := dominio.TextosUbicacion(extraida)
		d.Datos.PropuestaMunicipio = ubicacion.ProponerMunicipio(direccion, valoracion, municipios)
		d.Datos.OpcionesUbicacion = opcionesUbicacion(datosCatalogo, d.Lectura.LeidaEn)
		for i, candidato := range d.Datos.PropuestaMunicipio.Candidatos {
			for _, m := range datosCatalogo.Municipios {
				if m.INE == candidato.Codigo {
					d.Datos.PropuestaMunicipio.Candidatos[i].MunicipioID = m.ID
					d.Datos.PropuestaMunicipio.Candidatos[i].CentroID = m.CentroID
					d.Datos.PropuestaMunicipio.Candidatos[i].EquipoID = m.EquipoID
				}
			}
		}
		if _, err := s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "expediente.subir_anexo", "", "ok")); err != nil {
			return err
		}
		if antivirus == "no_revisado" {
			_, err := s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "expediente.antivirus_no_revisado", "", "ok"))
			return err
		}
		return nil
	})
	if err != nil {
		return dominio.Derivacion{}, dominio.ErrNoDisponible
	}
	if s.d.Borradores == nil || s.d.Borradores.Poner(ctx, d) != nil {
		return dominio.Derivacion{}, dominio.ErrNoDisponible
	}
	return d, nil
}

func (s *Servicio) Consultar(ctx context.Context, a dominio.Actor, id string) (dominio.Derivacion, error) {
	if err := a.AutorizarConsulta(); err != nil {
		return s.rechazar(ctx, a, err)
	}
	if !dominio.IDValido(id) {
		return s.rechazar(ctx, a, dominio.ErrPeticion)
	}
	var d dominio.Derivacion
	err := s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		var err error
		d, err = s.d.Repositorio.Obtener(ctx, id)
		if err != nil {
			return err
		}
		_, err = s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "expediente.leer_derivacion", id, "ok"))
		return err
	})
	if errors.Is(err, dominio.ErrNoEncontrada) {
		return dominio.Derivacion{}, dominio.ErrNoEncontrada
	}
	if err != nil {
		return dominio.Derivacion{}, dominio.ErrNoDisponible
	}
	if d.Seguimiento != nil && d.Seguimiento.Estado == "activo" && d.Seguimiento.FechaFin > dominio.FechaCivil(s.d.Reloj.Ahora()) {
		d.Estado = dominio.SeguimientoTrasCese
	}
	return d, nil
}
