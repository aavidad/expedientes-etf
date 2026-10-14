package documento

import (
	"context"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/documento/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	registro "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/documento"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"time"
)

type DependenciasPlantillas struct {
	Repositorio           salida.PublicacionPlantillas
	Gestion               salida.GestionPlantillas
	Almacen               salida.AlmacenPlantillas
	Validador             salida.ValidadorPlantillas
	Antivirus             plataforma.Antivirus
	Idiomas               salida.Idiomas
	Ficheros              salida.FicherosPlantilla
	Transacciones         plataforma.Transacciones
	Registro              registro.Registro
	RegistroTransaccional salida.RegistroTransaccional
	Reloj                 salida.Reloj
	Identificadores       salida.Identificadores
}
type AdministracionPlantillas struct{ d DependenciasPlantillas }

var _ entrada.Plantillas = (*AdministracionPlantillas)(nil)

func NuevaAdministracionPlantillas(dep DependenciasPlantillas) *AdministracionPlantillas {
	return &AdministracionPlantillas{dep}
}

func (s *AdministracionPlantillas) evento(a d.Actor, accion, resultado string) auditoria.Datos {
	return auditoria.Datos{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, OcurridoEn: s.d.Reloj.Ahora().UTC().Truncate(time.Microsecond), Accion: accion, Entidad: "documento", Resultado: resultado}
}
func (s *AdministracionPlantillas) Rechazar(ctx context.Context, a d.Actor) error {
	if s.d.Registro == nil || s.d.Reloj == nil {
		return d.ErrNoDisponible
	}
	if a.UsuarioID == "" {
		a.UsuarioID = "sistema"
		a.Perfiles = nil
	}
	ctx, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancelar()
	if _, err := s.d.Registro.Confirmar(ctx, s.evento(a, "documento.plantilla_rechazada", "denegado")); err != nil {
		return d.ErrNoDisponible
	}
	return nil
}
func (s *AdministracionPlantillas) fallo(ctx context.Context, a d.Actor, err error) error {
	if e := s.Rechazar(ctx, a); e != nil {
		return e
	}
	return err
}
func (s *AdministracionPlantillas) Actual(ctx context.Context, a d.Actor, tipo, idioma string) (d.ReferenciaPlantilla, error) {
	var r d.ReferenciaPlantilla
	if err := a.AutorizarPlantillas(); err != nil {
		return r, s.fallo(ctx, a, err)
	}
	if err := (d.PublicarPlantilla{Tipo: tipo, Idioma: idioma, Version: "1.0.0", VersionPrevia: 1}).Validar(); err != nil {
		return r, s.fallo(ctx, a, err)
	}
	err := s.d.Transacciones.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		var err error
		r, err = s.d.Repositorio.Actual(ctx, tipo, idioma)
		if err != nil {
			return err
		}
		_, err = s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "documento.plantilla_consultar", "ok"))
		return err
	})
	if err != nil {
		return d.ReferenciaPlantilla{}, s.fallo(ctx, a, err)
	}
	return r, nil
}
func (s *AdministracionPlantillas) Publicar(ctx context.Context, a d.Actor, p d.PublicarPlantilla, b []byte) (d.ReferenciaPlantilla, error) {
	return s.subir(ctx, a, p, b, true)
}
func (s *AdministracionPlantillas) Subir(ctx context.Context, a d.Actor, p d.PublicarPlantilla, b []byte) (d.ReferenciaPlantilla, error) {
	return s.subir(ctx, a, p, b, false)
}
func (s *AdministracionPlantillas) subir(ctx context.Context, a d.Actor, p d.PublicarPlantilla, b []byte, publicar bool) (d.ReferenciaPlantilla, error) {
	var r d.ReferenciaPlantilla
	rechazar := func(err error) (d.ReferenciaPlantilla, error) { return r, s.fallo(ctx, a, err) }
	if err := a.AutorizarPlantillas(); err != nil {
		return rechazar(err)
	}
	if len(b) > d.MaximoPlantilla {
		return rechazar(d.ErrTamanoPlantilla)
	}
	if len(b) == 0 {
		return rechazar(d.ErrPeticion)
	}
	if err := p.Validar(); err != nil {
		return rechazar(err)
	}
	if s.d.Antivirus == nil {
		return rechazar(d.ErrAntivirus)
	}
	estado, err := s.d.Antivirus.Revisar(ctx, b)
	if estado == plataforma.AntivirusInfectado {
		return rechazar(d.ErrInfectada)
	}
	if err != nil || estado != plataforma.AntivirusLimpio {
		return rechazar(d.ErrAntivirus)
	}
	idioma, err := s.d.Idiomas.Resolver(ctx, p.Idioma)
	if err != nil {
		return rechazar(err)
	}
	var nueva d.Plantilla
	guardada := false
	err = s.d.Transacciones.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		actual, err := s.d.Repositorio.Actual(ctx, p.Tipo, p.Idioma)
		if err != nil {
			return err
		}
		if actual.Revision != p.VersionPrevia {
			return &d.ConflictoPlantilla{VersionActual: actual.Revision}
		}
		previa, err := s.d.Ficheros.Leer(ctx, idioma, actual.Plantilla)
		if err != nil {
			return err
		}
		defer clear(previa)
		if d.Huella(previa) != actual.Plantilla.Huella {
			return d.ErrHuella
		}
		if err := s.d.Validador.Validar(ctx, b, previa); err != nil {
			return err
		}
		id, err := s.d.Identificadores.Nuevo()
		if err != nil || !d.IDValido(id) {
			return d.ErrNoDisponible
		}
		nueva = d.Plantilla{ID: id, Tipo: p.Tipo, Idioma: p.Idioma, Version: p.Version, Huella: d.Huella(b), RolFirmante: actual.Plantilla.RolFirmante}
		nueva.Ruta = d.RutaPublicada(nueva)
		if err := s.d.Almacen.Guardar(ctx, nueva, b); err != nil {
			return err
		}
		guardada = true
		if publicar {
			r, err = s.d.Repositorio.Publicar(ctx, nueva, p.VersionPrevia)
		} else {
			if s.d.Gestion == nil {
				return d.ErrNoDisponible
			}
			r, err = s.d.Gestion.GuardarVersion(ctx, nueva, p.VersionPrevia)
		}
		return err
	})
	if err == nil {
		return r, nil
	}
	if guardada {
		// COMMIT puede haber llegado a PostgreSQL aunque se pierda la respuesta.
		// Consultamos bajo el bloqueo de publicación antes de quitar un huérfano.
		limpieza, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 8*time.Second)
		defer cancelar()
		var registrada bool
		e := s.d.Transacciones.Ejecutar(limpieza, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
			var err error
			registrada, err = s.d.Repositorio.Registrada(ctx, nueva)
			return err
		})
		if e == nil && !registrada {
			if e := s.d.Almacen.Eliminar(limpieza, nueva); e != nil {
				err = d.ErrNoDisponible
			}
		}
	}
	return d.ReferenciaPlantilla{}, s.fallo(ctx, a, err)
}

func (s *AdministracionPlantillas) Listar(ctx context.Context, a d.Actor) ([]d.VersionPlantilla, error) {
	if err := a.AutorizarPlantillas(); err != nil {
		return nil, s.fallo(ctx, a, err)
	}
	if s.d.Gestion == nil {
		return nil, s.fallo(ctx, a, d.ErrNoDisponible)
	}
	var versiones []d.VersionPlantilla
	err := s.d.Transacciones.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		var err error
		versiones, err = s.d.Gestion.Listar(ctx)
		if err != nil {
			return err
		}
		_, err = s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "documento.plantilla_historial", "ok"))
		return err
	})
	if err != nil {
		return nil, s.fallo(ctx, a, err)
	}
	return versiones, nil
}
func (s *AdministracionPlantillas) Activar(ctx context.Context, a d.Actor, p d.ActivarPlantilla) (d.ReferenciaPlantilla, error) {
	var resultado d.ReferenciaPlantilla
	if err := a.AutorizarPlantillas(); err != nil {
		return resultado, s.fallo(ctx, a, err)
	}
	if err := p.Validar(); err != nil {
		return resultado, s.fallo(ctx, a, err)
	}
	if s.d.Gestion == nil {
		return resultado, s.fallo(ctx, a, d.ErrNoDisponible)
	}
	idioma, err := s.d.Idiomas.Resolver(ctx, p.Idioma)
	if err != nil {
		return resultado, s.fallo(ctx, a, err)
	}
	err = s.d.Transacciones.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		version, err := s.d.Gestion.Version(ctx, p.ID)
		if err != nil {
			return err
		}
		if version.Idioma != p.Idioma {
			return d.ErrPeticion
		}
		contenido, err := s.d.Ficheros.Leer(ctx, idioma, version)
		if err != nil {
			return err
		}
		defer clear(contenido)
		if d.Huella(contenido) != version.Huella {
			return d.ErrHuella
		}
		actual, err := s.d.Repositorio.Actual(ctx, version.Tipo, version.Idioma)
		if err != nil {
			return err
		}
		if actual.Revision != p.VersionPrevia {
			return &d.ConflictoPlantilla{VersionActual: actual.Revision}
		}
		previa, err := s.d.Ficheros.Leer(ctx, idioma, actual.Plantilla)
		if err != nil {
			return err
		}
		defer clear(previa)
		if d.Huella(previa) != actual.Plantilla.Huella {
			return d.ErrHuella
		}
		if err := s.d.Validador.Validar(ctx, contenido, previa); err != nil {
			return err
		}
		resultado, err = s.d.Gestion.Activar(ctx, version, p.VersionPrevia)
		if err == nil && actual.Plantilla.ID == version.ID {
			// La repetición no cambia la referencia, pero su lectura se audita.
			_, err = s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "documento.plantilla_consultar", "ok"))
		}
		return err
	})
	if err != nil {
		return d.ReferenciaPlantilla{}, s.fallo(ctx, a, err)
	}
	return resultado, nil
}
