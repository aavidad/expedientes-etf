package documento

import (
	"context"
	"encoding/json"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/documento/puertos/entrada"
	estadisticaEntrada "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	est "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	paquete "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	registro "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
	catalogo "github.com/aavidad/expedientes-etf/internal/puertos/salida/catalogo"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/documento"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"maps"
	"strings"
	"time"
)

type DependenciasEmisionSPAF struct {
	Calculo         estadisticaEntrada.CalculoSPAF
	Repositorio     salida.RepositorioSPAF
	Catalogos       catalogo.Repositorio
	Referencias     salida.ReferenciasPresentacion
	Presentacion    salida.Presentacion
	Idiomas         salida.Idiomas
	Ficheros        salida.FicherosPlantilla
	Motor           salida.GeneradorDocumentos
	Transacciones   plataforma.Transacciones
	Auditoria       registro.RegistroTransaccional
	Reloj           salida.Reloj
	Identificadores salida.Identificadores
}
type ServicioEmisionSPAF struct{ d DependenciasEmisionSPAF }

var _ entrada.EmisionesSPAF = (*ServicioEmisionSPAF)(nil)

func NuevoEmisionSPAF(d DependenciasEmisionSPAF) *ServicioEmisionSPAF { return &ServicioEmisionSPAF{d} }
func (s *ServicioEmisionSPAF) evento(a d.Actor, e d.Emision, resultado string) auditoria.Datos {
	if !d.IDValido(a.UsuarioID) {
		a.UsuarioID = "sistema"
		a.Perfiles = nil
	}
	return auditoria.Datos{OcurridoEn: s.d.Reloj.Ahora().UTC().Truncate(time.Microsecond), UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, Accion: "documento.emitir_spaf", Entidad: "documento", EntidadID: e.ID, Resultado: resultado}
}
func (s *ServicioEmisionSPAF) RechazarSPAF(ctx context.Context, a d.Actor) error {
	if s.d.Auditoria == nil || s.d.Reloj == nil {
		return d.ErrNoDisponible
	}
	if _, err := s.d.Auditoria.Confirmar(ctx, s.evento(a, d.Emision{}, "denegado")); err != nil {
		return d.ErrNoDisponible
	}
	return nil
}
func (s *ServicioEmisionSPAF) EmitirSPAF(ctx context.Context, a d.Actor, p d.SolicitudSPAF) (d.Resultado, error) {
	resultado, err := s.emitir(ctx, a, p)
	if err != nil {
		clear(resultado.Contenido)
		if fallo := s.RechazarSPAF(ctx, a); fallo != nil {
			return d.Resultado{}, fallo
		}
		return d.Resultado{}, err
	}
	return resultado, nil
}
func (s *ServicioEmisionSPAF) emitir(ctx context.Context, a d.Actor, p d.SolicitudSPAF) (d.Resultado, error) {
	actor := est.ActorSPAF{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}
	filtro, err := actor.AutorizarSPAF(est.FiltroSPAF{EquipoID: p.EquipoID, Anio: p.Anio, Mes: p.Mes})
	if err != nil {
		if errors.Is(err, est.ErrPermisoSPAF) {
			return d.Resultado{}, d.ErrAcceso
		}
		return d.Resultado{}, d.ErrPeticion
	}
	if err := p.Validar(); err != nil {
		return d.Resultado{}, err
	}
	if s.d.Calculo == nil || s.d.Repositorio == nil || s.d.Catalogos == nil || s.d.Referencias == nil || s.d.Presentacion == nil || s.d.Idiomas == nil || s.d.Ficheros == nil || s.d.Motor == nil || s.d.Transacciones == nil || s.d.Auditoria == nil || s.d.Reloj == nil || s.d.Identificadores == nil {
		return d.Resultado{}, d.ErrNoDisponible
	}
	idioma, err := s.d.Idiomas.Resolver(ctx, p.Idioma)
	if err != nil {
		return d.Resultado{}, err
	}
	ref, err := s.d.Referencias.ResolverCatalogo(ctx, p.Idioma)
	if err != nil {
		return d.Resultado{}, d.ErrNoDisponible
	}
	var r d.Resultado
	var emisionPrevia d.Emision
	peticionIgual := false
	err = s.d.Transacciones.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
		if err := s.d.Repositorio.BloquearSPAF(ctx, a, p); err != nil {
			return err
		}
		previa, peticion, err := s.d.Repositorio.PeticionSPAF(ctx, a, p.Clave)
		if err != nil {
			return err
		}
		if previa.ID != "" {
			versionesPrevias := d.VersionesSPAF{Regla: previa.VersionesDatos["regla"], Fichas: map[string]string{}}
			for k, v := range previa.VersionesDatos {
				if strings.HasPrefix(k, "ficha.") {
					versionesPrevias.Fichas[strings.TrimPrefix(k, "ficha.")] = v
				}
			}
			if previa.EquipoID != p.EquipoID || previa.FechaDocumento.Year() != p.Anio || int(previa.FechaDocumento.Month()) != p.Mes || previa.Plantilla.Idioma != p.Idioma || !p.Versiones.Igual(versionesPrevias) {
				return d.ErrIdempotencia
			}
			emisionPrevia = previa
			peticionIgual = true
		}
		tabla, err := s.d.Calculo.CalcularEnTransaccion(ctx, actor, filtro, s.d.Reloj.Ahora())
		if errors.Is(err, est.ErrEntrada) {
			return d.ErrPeticion
		}
		if err != nil {
			return d.ErrNoDisponible
		}
		if tabla.FiltroSPAF != filtro {
			return d.ErrNoDisponible
		}
		actual := d.VersionesSPAF{Regla: tabla.VersionRegla, Fichas: map[string]string{}}
		for _, fila := range tabla.Filas {
			actual.Fichas[fila.ExpedienteID] = fila.VersionFicha
		}
		if !p.Versiones.Igual(actual) {
			return &d.ConflictoSPAF{Actual: actual}
		}
		plantilla, err := s.d.Repositorio.Activa(ctx, d.TablaSPAF, p.Idioma)
		if previa.ID != "" {
			plantilla = previa.Plantilla
			idioma = previa.IdiomaPaquete
			err = nil
		}
		if err != nil {
			return err
		}
		if previa.ID != "" {
			contenido := previa.VersionesDatos["textos_catalogo"]
			ref = paquete.ReferenciaCatalogo{Codigo: p.Idioma, Contenido: contenido}
			if contenido != "" {
				ref.Ruta = p.Idioma + "/catalogos/.versiones/" + contenido + ".json"
				ref.Huella = contenido
			}
		}

		textos, nombres, err := s.d.Presentacion.Cargar(ctx, idioma, ref)
		if err != nil {
			return d.ErrNoDisponible
		}
		version, err := s.d.Catalogos.Vigente(ctx, "asignacion_municipios", tabla.Corte)
		if err != nil {
			return d.ErrNoDisponible
		}
		cat, err := s.d.Catalogos.Datos(ctx, version)
		if err != nil {
			return d.ErrNoDisponible
		}
		equipo := ""
		for _, e := range cat.Equipos {
			if e.ID == filtro.EquipoID {
				equipo = e.Codigo
			}
		}
		if equipo == "" {
			return d.ErrNoDisponible
		}
		relleno, err := rellenarSPAF(tabla, equipo, textos, nombres)
		if err != nil {
			return err
		}
		versiones := map[string]string{"regla": actual.Regla, "catalogo": version.ID, "textos_catalogo": ref.Contenido}
		for id, v := range actual.Fichas {
			versiones["ficha."+id] = v
		}
		datos, _ := json.Marshal(struct {
			Relleno   d.Relleno
			Versiones map[string]string
		}{relleno, versiones})
		huellaDatos := d.Huella(datos)
		clear(datos)
		pet, _ := json.Marshal(struct {
			Equipo       string
			Anio, Mes    int
			IdiomaCodigo string
			Versiones    d.VersionesSPAF
			Plantilla    d.Plantilla
			Idioma       d.IdiomaPlantilla
			Datos, Motor string
		}{p.EquipoID, p.Anio, p.Mes, p.Idioma, p.Versiones, plantilla, idioma, huellaDatos, s.d.Motor.Version()})
		huellaPeticion := d.Huella(pet)
		clear(pet)
		if previa.ID != "" && peticion != huellaPeticion {
			return &d.ConflictoReproduccion{Emision: previa}
		}
		b, err := s.d.Ficheros.Leer(ctx, idioma, plantilla)
		if err != nil {
			if previa.ID != "" {
				return &d.ConflictoReproduccion{Emision: previa}
			}
			return err
		}
		defer clear(b)
		contenido, err := s.d.Motor.Generar(ctx, b, relleno)
		if err != nil {
			return err
		}
		r.Contenido = contenido
		huella := d.Huella(contenido)
		if previa.ID != "" {
			if previa.HuellaResultado != huella {
				return &d.ConflictoReproduccion{Emision: previa}
			}
			r.Emision = previa
			r.Repetida = true
		} else {
			ultima, err := s.d.Repositorio.UltimaSPAF(ctx, p)
			if err != nil {
				return err
			}
			id, err := s.d.Identificadores.Nuevo()
			if err != nil {
				return d.ErrNoDisponible
			}
			e := d.Emision{ID: id, Tipo: d.TablaSPAF, EquipoID: filtro.EquipoID, Plantilla: plantilla, IdiomaPaquete: idioma, FechaDocumento: tabla.Corte, GeneradaPor: a.UsuarioID, GeneradaEn: s.d.Reloj.Ahora().UTC().Truncate(time.Microsecond), HuellaDatos: huellaDatos, HuellaResultado: huella, VersionesDatos: maps.Clone(versiones), VersionMotor: s.d.Motor.Version()}
			if ultima.ID != "" && (ultima.HuellaDatos != huellaDatos || ultima.Plantilla != plantilla || ultima.IdiomaPaquete != idioma || ultima.VersionMotor != e.VersionMotor) {
				e.Aviso = "documento.regenerado"
			}
			if err := s.d.Repositorio.GuardarSPAF(ctx, a, p, huellaPeticion, e); err != nil {
				return err
			}
			r.Emision = e
		}
		_, err = s.d.Auditoria.RegistrarEnTransaccion(ctx, s.evento(a, r.Emision, "ok"))
		return err
	})
	if err != nil && peticionIgual {
		err = &d.ConflictoReproduccion{Emision: emisionPrevia}
	}
	return r, err
}
