package documento

import (
	"context"
	"errors"
	"fmt"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/documento/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	registro "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/documento"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"maps"
	"time"
)

type Dependencias struct {
	Repositorio           salida.Repositorio
	RepositorioRegistro   salida.RepositorioRegistroSalida
	RegistroSalida        salida.RegistroSalida
	Datos                 salida.FuenteDatos
	Firmantes             salida.Firmantes
	Motor                 salida.GeneradorDocumentos
	Idiomas               salida.Idiomas
	Ficheros              salida.FicherosPlantilla
	Transacciones         plataforma.Transacciones
	Registro              registro.Registro
	RegistroTransaccional salida.RegistroTransaccional
	Reloj                 salida.Reloj
	Identificadores       salida.Identificadores
}
type Servicio struct{ d Dependencias }

var _ entrada.Emisiones = (*Servicio)(nil)

func Nuevo(d Dependencias) *Servicio { return &Servicio{d} }

func (s *Servicio) evento(a d.Actor, accion string, e d.Emision, resultado string) auditoria.Datos {
	if a.UsuarioID == "" {
		a.UsuarioID = "sistema"
		a.Perfiles = nil
	}
	return auditoria.Datos{OcurridoEn: s.d.Reloj.Ahora().UTC().Truncate(time.Microsecond), UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, Accion: accion, Entidad: "documento", EntidadID: e.ID, ExpedienteID: e.ExpedienteID, Resultado: resultado}
}
func (s *Servicio) Rechazar(ctx context.Context, a d.Actor, _ string) error {
	if s.d.Registro == nil || s.d.Reloj == nil || a.UsuarioID != "" && !d.IDValido(a.UsuarioID) {
		return d.ErrNoDisponible
	}
	_, err := s.d.Registro.Confirmar(ctx, s.evento(a, "documento.emision_rechazada", d.Emision{}, "denegado"))
	if err != nil {
		return d.ErrNoDisponible
	}
	return nil
}
func (s *Servicio) rechazar(ctx context.Context, a d.Actor, err error) (d.Resultado, error) {
	if fallo := s.Rechazar(ctx, a, err.Error()); fallo != nil {
		return d.Resultado{}, fallo
	}
	return d.Resultado{}, err
}

func (s *Servicio) Emitir(ctx context.Context, a d.Actor, p d.Solicitud) (d.Resultado, error) {
	if err := a.AutorizarTipo(p.Tipo); err != nil {
		return s.rechazar(ctx, a, err)
	}
	if err := p.Validar(); err != nil {
		return s.rechazar(ctx, a, err)
	}
	if s.d.Repositorio == nil || s.d.Datos == nil || s.d.Firmantes == nil || s.d.Motor == nil || s.d.Idiomas == nil || s.d.Ficheros == nil || s.d.Transacciones == nil || s.d.RegistroTransaccional == nil || s.d.Reloj == nil || s.d.Identificadores == nil {
		return s.rechazar(ctx, a, d.ErrNoDisponible)
	}
	// Las referencias públicas de paquete no contienen datos personales y su
	// puerto abre su propia lectura. Se fijan antes de la transacción de negocio.
	idioma, err := s.d.Idiomas.Resolver(ctx, p.Idioma)
	if err != nil {
		return s.rechazar(ctx, a, err)
	}
	p.IdiomaPaquete = idioma
	var resultado d.Resultado
	err = s.d.Transacciones.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) (err error) {
		if err := s.d.Repositorio.Bloquear(ctx, a, p); err != nil {
			return err
		}
		previa, peticion, err := s.d.Repositorio.Peticion(ctx, a, p.Clave)
		if err != nil {
			return err
		}
		defer func() {
			if previa.ID != "" && (errors.Is(err, d.ErrPlantilla) || errors.Is(err, d.ErrHuella) || errors.Is(err, d.ErrFirmante) || errors.Is(err, d.ErrNoDisponible)) {
				err = &d.ConflictoReproduccion{Emision: previa}
			}
		}()
		if previa.ID != "" {
			if p.Tipo == d.Anexo6 {
				p.FechaRepeticion = previa.FechaDocumento
			}
			if p.Tipo == d.Anexo5 || p.Tipo == d.ActaCierre {
				p.FechaReproduccion = previa.FechaDocumento
			}
		}
		datos, err := s.d.Datos.Obtener(ctx, a, p)
		if err != nil {
			return err
		}
		if err = datos.Validar(p); err != nil {
			return err
		}
		if err = a.AutorizarEquipo(datos.EquipoID); err != nil {
			return err
		}
		plantilla, err := s.d.Repositorio.Activa(ctx, p.Tipo, p.Idioma)
		if err != nil {
			return err
		}
		var firmante d.Firmante
		if !d.EsConsentimiento(p.Tipo) {
			firmante, err = s.d.Firmantes.Vigente(ctx, plantilla.RolFirmante, datos.FechaDocumento)
			if err != nil {
				return err
			}
			if !d.FirmanteValido(firmante) {
				return d.ErrFirmante
			}
		}
		relleno := datos.Aceptacion.Rellenar(firmante)
		if p.Tipo == d.Anexo6 {
			relleno = d.Relleno{Campos: datos.Salida.Campos, Filas: datos.Salida.Filas}
		}
		if p.Tipo == d.OficioSPM {
			jefe, err := s.d.Firmantes.Vigente(ctx, "jefe_spm", datos.FechaDocumento)
			if err != nil {
				return err
			}
			if !d.FirmanteValido(jefe) {
				return d.ErrFirmante
			}
			relleno.Campos["jefe_spm"] = jefe.Nombre
			datos.Versiones = maps.Clone(datos.Versiones)
			datos.Versiones["destinatario_spm"] = fmt.Sprintf("%s:%d", jefe.ID, jefe.Version)
		}
		huellaDatos, err := d.HuellaDatos(datos, relleno, firmante)
		if err != nil {
			return err
		}
		huellaPeticion, err := d.HuellaPeticion(p, plantilla, idioma, huellaDatos, s.d.Motor.Version())
		if err != nil {
			return err
		}
		if previa.ID != "" && huellaPeticion != peticion {
			return d.ErrIdempotencia
		}
		b, err := s.d.Ficheros.Leer(ctx, idioma, plantilla)
		if err != nil {
			return err
		}
		defer clear(b)
		if d.Huella(b) != plantilla.Huella {
			return d.ErrHuella
		}
		contenido, err := s.d.Motor.Generar(ctx, b, relleno)
		if err != nil {
			return err
		}
		resultado.Contenido = contenido
		huellaResultado := d.Huella(contenido)
		if previa.ID != "" {
			if previa.HuellaResultado != huellaResultado {
				return &d.ConflictoReproduccion{Emision: previa}
			}
			resultado.Emision = previa
			resultado.Repetida = true
			_, err = s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "documento.repetir_descarga", previa, "ok"))
			return err
		}
		ultima, err := s.d.Repositorio.Ultima(ctx, p)
		if err != nil {
			return err
		}
		id, err := s.d.Identificadores.Nuevo()
		if err != nil {
			return d.ErrNoDisponible
		}
		e := d.Emision{ID: id, Tipo: p.Tipo, ExpedienteID: datos.ExpedienteID, EquipoID: datos.EquipoID, Plantilla: plantilla, IdiomaPaquete: idioma, FechaDocumento: datos.FechaDocumento, GeneradaPor: a.UsuarioID, GeneradaEn: s.d.Reloj.Ahora().UTC().Truncate(time.Microsecond), HuellaDatos: huellaDatos, HuellaResultado: huellaResultado, VersionesDatos: datos.Versiones, VersionMotor: s.d.Motor.Version()}
		if ultima.ID != "" && (ultima.HuellaDatos != e.HuellaDatos || ultima.Plantilla.ID != e.Plantilla.ID || ultima.IdiomaPaquete != e.IdiomaPaquete || ultima.VersionMotor != e.VersionMotor) {
			e.Aviso = "documento.regenerado"
		}
		if err := s.d.Repositorio.Guardar(ctx, a, p, huellaPeticion, e); err != nil {
			return err
		}
		if _, err := s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "documento.emitir", e, "ok")); err != nil {
			return err
		}
		resultado.Emision = e
		return nil
	})
	if err != nil {
		clear(resultado.Contenido)
		var conflicto *d.ConflictoReproduccion
		if errors.As(err, &conflicto) {
			return s.rechazar(ctx, a, conflicto)
		}
		return s.rechazar(ctx, a, err)
	}
	return resultado, nil
}
