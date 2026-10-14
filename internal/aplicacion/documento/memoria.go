package documento

import (
	"context"
	"encoding/json"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/documento/puertos/entrada"
	estadistica "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	e "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	audit "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/documento"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"time"
)

type DependenciasMemoria struct {
	Repositorio     salida.RepositorioMemoria
	Fuente          estadistica.FuenteMemoria
	Idiomas         salida.Idiomas
	Ficheros        salida.FicherosPlantilla
	Textos          salida.TextosMemoria
	Motor           salida.GeneradorMemoria
	Transacciones   plataforma.Transacciones
	Registro        audit.RegistroTransaccional
	Reloj           salida.Reloj
	Identificadores salida.Identificadores
}
type ServicioMemoria struct{ d DependenciasMemoria }

var _ entrada.EmisionesMemoria = (*ServicioMemoria)(nil)

func NuevoMemoria(d DependenciasMemoria) *ServicioMemoria { return &ServicioMemoria{d} }
func (s *ServicioMemoria) evento(a d.Actor, accion, id, resultado string) auditoria.Datos {
	if !d.IDValido(a.UsuarioID) {
		a = d.Actor{UsuarioID: "sistema"}
	}
	return auditoria.Datos{OcurridoEn: s.d.Reloj.Ahora().UTC().Truncate(time.Microsecond), UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, Accion: accion, Entidad: "documento", EntidadID: id, Resultado: resultado}
}
func (s *ServicioMemoria) RechazarMemoria(ctx context.Context, a d.Actor) error {
	if s.d.Registro == nil || s.d.Reloj == nil {
		return d.ErrNoDisponible
	}
	_, err := s.d.Registro.Confirmar(ctx, s.evento(a, "documento.memoria_rechazada", "", "denegado"))
	if err != nil {
		return d.ErrNoDisponible
	}
	return nil
}
func (s *ServicioMemoria) EmitirMemoria(ctx context.Context, a d.Actor, p d.SolicitudMemoria) (d.ResultadoMemoria, error) {
	fallo := p.Validar()
	if fallo == nil && s.d.Reloj != nil && p.Anio > s.d.Reloj.Ahora().Year() {
		fallo = d.ErrPeticion
	}
	if fallo == nil {
		_, fallo = e.AutorizarMemoria(e.ActorMemoria{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, e.FiltroMemoria{Anio: p.Anio, EquipoID: p.EquipoID})
	}
	if fallo != nil {
		if err := s.RechazarMemoria(ctx, a); err != nil {
			return d.ResultadoMemoria{}, err
		}
		return d.ResultadoMemoria{}, fallo
	}
	filtro, err := e.AutorizarMemoria(e.ActorMemoria{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, e.FiltroMemoria{Anio: p.Anio, EquipoID: p.EquipoID})
	if err != nil {
		return d.ResultadoMemoria{}, err
	}
	p.EquipoID = filtro.EquipoID
	if s.d.Repositorio == nil || s.d.Fuente == nil || s.d.Idiomas == nil || s.d.Ficheros == nil || s.d.Textos == nil || s.d.Motor == nil || s.d.Transacciones == nil || s.d.Registro == nil || s.d.Reloj == nil || s.d.Identificadores == nil {
		return d.ResultadoMemoria{}, d.ErrNoDisponible
	}
	idioma, err := s.d.Idiomas.Resolver(ctx, p.Idioma)
	if err != nil {
		return d.ResultadoMemoria{}, err
	}
	b, err := json.Marshal(p)
	if err != nil {
		return d.ResultadoMemoria{}, d.ErrPeticion
	}
	peticion := d.Huella(b)
	var r d.ResultadoMemoria
	err = s.d.Transacciones.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
		if err := s.d.Repositorio.BloquearMemoria(ctx, a.UsuarioID, p.Clave); err != nil {
			return err
		}
		previa, err := s.d.Repositorio.PeticionMemoria(ctx, a.UsuarioID, p.Clave)
		if err != nil {
			return err
		}
		if previa.ID != "" && previa.HuellaPeticion != peticion {
			return d.ErrIdempotencia
		}
		ahora := s.d.Reloj.Ahora().UTC().Truncate(time.Microsecond)
		plantilla, err := s.d.Repositorio.PlantillaMemoria(ctx, p.Idioma)
		if err != nil {
			return err
		}
		if previa.ID != "" {
			ahora = previa.GeneradaEn
			if previa.Plantilla.ID != plantilla.ID || previa.IdiomaPaquete != idioma || previa.VersionMotor != s.d.Motor.Version() {
				return d.ErrIdempotencia
			}
		}
		datos, err := s.d.Fuente.MemoriaEnTransaccion(ctx, e.ActorMemoria{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, filtro, ahora)
		if err != nil {
			return err
		}
		textos, err := s.d.Textos.LeerMemoria(ctx, idioma, plantilla)
		if err != nil {
			return err
		}
		campos, err := s.d.Motor.CamposMemoria(ctx, datos, textos)
		if err != nil {
			return err
		}
		serializado, err := json.Marshal(struct {
			Informe e.InformeMemoria
			Campos  map[string]string
		}{datos, campos})
		if err != nil {
			return err
		}
		huellaDatos := d.Huella(serializado)
		clear(serializado)
		if previa.ID != "" && previa.HuellaDatos != huellaDatos {
			return d.ErrIdempotencia
		}
		plantillaBytes, err := s.d.Ficheros.Leer(ctx, idioma, plantilla)
		if err != nil {
			return err
		}
		defer clear(plantillaBytes)
		contenido, err := s.d.Motor.Generar(ctx, plantillaBytes, d.Relleno{Campos: campos})
		if err != nil {
			return err
		}
		huella := d.Huella(contenido)
		if previa.ID != "" {
			if previa.HuellaResultado != huella {
				clear(contenido)
				return d.ErrIdempotencia
			}
			r = d.ResultadoMemoria{Emision: previa, Contenido: contenido, Repetida: true}
			_, err = s.d.Registro.RegistrarEnTransaccion(ctx, s.evento(a, "documento.memoria_repetir", previa.ID, "ok"))
			return err
		}
		id, err := s.d.Identificadores.Nuevo()
		if err != nil {
			clear(contenido)
			return err
		}
		emision := d.EmisionMemoria{ID: id, Solicitud: p, Plantilla: plantilla, IdiomaPaquete: idioma, GeneradaEn: ahora, GeneradaPor: a.UsuarioID, HuellaPeticion: peticion, HuellaDatos: huellaDatos, HuellaResultado: huella, VersionMotor: s.d.Motor.Version()}
		if err := s.d.Repositorio.GuardarMemoria(ctx, emision); err != nil {
			clear(contenido)
			return err
		}
		r = d.ResultadoMemoria{Emision: emision, Contenido: contenido}
		_, err = s.d.Registro.RegistrarEnTransaccion(ctx, s.evento(a, "documento.memoria_emitir", id, "ok"))
		return err
	})
	if err != nil {
		clear(r.Contenido)
		ctxAuditoria, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancelar()
		if rechazo := s.RechazarMemoria(ctxAuditoria, a); rechazo != nil {
			return d.ResultadoMemoria{}, rechazo
		}
		if errors.Is(err, d.ErrIdempotencia) || errors.Is(err, d.ErrPlantilla) || errors.Is(err, d.ErrHuella) || errors.Is(err, e.ErrPermisoMemoria) {
			return d.ResultadoMemoria{}, err
		}
		return d.ResultadoMemoria{}, d.ErrNoDisponible
	}
	return r, nil
}
