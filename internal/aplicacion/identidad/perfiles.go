package identidad

import (
	"context"
	"errors"
	"time"

	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	auditoria "github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	salidaAuditoria "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/identidad"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

type AdministracionUsuarios struct {
	repositorio   salida.Perfiles
	transacciones plataforma.Transacciones
	registro      salidaAuditoria.RegistroTransaccional
	reloj         salida.Reloj
}

func NuevaAdministracionUsuarios(r salida.Perfiles, t plataforma.Transacciones, a salidaAuditoria.RegistroTransaccional, reloj salida.Reloj) *AdministracionUsuarios {
	return &AdministracionUsuarios{r, t, a, reloj}
}
func contextoActor(a entrada.Acceso) plataforma.Identidad {
	i := plataforma.Identidad{UsuarioID: a.Usuario.ID, EquipoID: a.Usuario.EquipoID}
	for _, p := range a.Usuario.Perfiles {
		i.Perfiles = append(i.Perfiles, string(p))
	}
	return i
}
func (s *AdministracionUsuarios) Listar(ctx context.Context, a entrada.Acceso) (usuarios []entrada.UsuarioAdministrado, err error) {
	defer func() {
		err = s.registrarFallo(ctx, a, "identidad.usuarios_listar", "", err)
		if err != nil {
			usuarios = nil
		}
	}()
	if !a.Permite(string(dominio.Administracion)) {
		return nil, dominio.ErrAccesoDenegado
	}
	err = s.transacciones.Ejecutar(ctx, contextoActor(a), func(ctx context.Context) error {
		var e error
		usuarios, e = s.repositorio.Listar(ctx)
		if e != nil {
			return e
		}
		_, e = s.registro.RegistrarEnTransaccion(ctx, s.datos(a, "identidad.usuarios_listar", "", "ok"))
		return e
	})
	if err != nil {
		return nil, errorPerfiles(err)
	}
	return usuarios, nil
}
func (s *AdministracionUsuarios) Cambiar(ctx context.Context, a entrada.Acceso, id string, c entrada.CambioPerfiles) (usuario entrada.UsuarioAdministrado, err error) {
	defer func() {
		err = s.registrarFallo(ctx, a, "identidad.perfiles_cambiar", id, err)
		if err != nil {
			usuario = entrada.UsuarioAdministrado{}
		}
	}()
	if !a.Permite(string(dominio.Administracion)) {
		return usuario, dominio.ErrAccesoDenegado
	}
	if err = c.Validar(); err != nil {
		return usuario, err
	}
	err = s.transacciones.Ejecutar(ctx, contextoActor(a), func(ctx context.Context) error { var e error; usuario, e = s.repositorio.Cambiar(ctx, id, c); return e })
	if err != nil {
		return entrada.UsuarioAdministrado{}, errorPerfiles(err)
	}
	return usuario, nil
}
func (s *AdministracionUsuarios) datos(a entrada.Acceso, accion, id, resultado string) auditoria.Datos {
	i := contextoActor(a)
	return auditoria.Datos{OcurridoEn: s.reloj.Ahora().UTC().Truncate(time.Microsecond), UsuarioID: i.UsuarioID, Perfiles: i.Perfiles, Accion: accion, Entidad: "identidad.usuario", EntidadID: id, Resultado: resultado}
}
func (s *AdministracionUsuarios) registrarFallo(ctx context.Context, a entrada.Acceso, accion, id string, err error) error {
	if err == nil {
		return nil
	}
	resultado := "error"
	if errors.Is(err, dominio.ErrAccesoDenegado) {
		resultado = "denegado"
	}
	registro, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancelar()
	if _, fallo := s.registro.Confirmar(registro, s.datos(a, accion, id, resultado)); fallo != nil {
		return dominio.ErrPersistencia
	}
	return err
}
func errorPerfiles(err error) error {
	for _, e := range []error{dominio.ErrAccesoDenegado, dominio.ErrPerfilesInvalidos, dominio.ErrConflicto, dominio.ErrUsuarioNoDisponible, context.Canceled, context.DeadlineExceeded} {
		if errors.Is(err, e) {
			return e
		}
	}
	return dominio.ErrPersistencia
}
