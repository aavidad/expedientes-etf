package identidad

import (
	"context"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/identidad"
)

// La revisión comparte los límites transaccionales y la auditoría de identidad.
type RevisionRestauracion struct {
	usuarios    *AdministracionUsuarios
	repositorio salida.Restauracion
}

func NuevaRevisionRestauracion(u *AdministracionUsuarios, r salida.Restauracion) *RevisionRestauracion {
	return &RevisionRestauracion{u, r}
}
func (s *RevisionRestauracion) ejecutar(ctx context.Context, a entrada.Acceso, accion string, f func(context.Context) error) (err error) {
	defer func() { err = s.usuarios.registrarFallo(ctx, a, accion, "", err) }()
	if !a.Permite(string(dominio.Administracion)) {
		return dominio.ErrAccesoDenegado
	}
	err = s.usuarios.transacciones.Ejecutar(ctx, contextoActor(a), f)
	return errorRestauracion(err)
}
func (s *RevisionRestauracion) Leer(ctx context.Context, a entrada.Acceso) (r dominio.RevisionRestauracion, err error) {
	err = s.ejecutar(ctx, a, "identidad.restauracion_leer", func(ctx context.Context) error {
		var e error
		r, e = s.repositorio.Leer(ctx)
		if e != nil {
			return e
		}
		_, e = s.usuarios.registro.RegistrarEnTransaccion(ctx, s.usuarios.datos(a, "identidad.restauracion_leer", r.ID, "ok"))
		return e
	})
	if err != nil {
		return dominio.RevisionRestauracion{}, err
	}
	return
}
func (s *RevisionRestauracion) Resolver(ctx context.Context, a entrada.Acceso, id, cambio string, version int64, decision string) error {
	return s.ejecutar(ctx, a, "identidad.restauracion_resolver", func(ctx context.Context) error {
		if version < 1 || (decision != "aplicar" && decision != "descartar") {
			return dominio.ErrPerfilesInvalidos
		}
		return s.repositorio.Resolver(ctx, id, cambio, version, decision)
	})
}
func (s *RevisionRestauracion) Confirmar(ctx context.Context, a entrada.Acceso, id string, version int64) error {
	return s.ejecutar(ctx, a, "identidad.restauracion_confirmar", func(ctx context.Context) error {
		if version < 1 {
			return dominio.ErrPerfilesInvalidos
		}
		return s.repositorio.Confirmar(ctx, id, version)
	})
}
func errorRestauracion(err error) error {
	if err == nil {
		return nil
	}
	if err == dominio.ErrSinRevision || err == dominio.ErrEquipoRestauracion || err == dominio.ErrCambiosPendientes || err == dominio.ErrRevisionPropia || err == dominio.ErrRevisionRestauracion {
		return err
	}
	return errorPerfiles(err)
}
