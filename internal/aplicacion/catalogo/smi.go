package catalogo

import (
	"context"
	"fmt"
	"time"

	auditoria "github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	registro "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/catalogo"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

var ErrPermisoSMI = dominio.ErrPermiso

type ServicioSMI struct {
	repo      salida.SMI
	tx        plataforma.Transacciones
	auditoria registro.Registro
	ahora     func() time.Time
}

func NuevoSMI(repo salida.SMI, tx plataforma.Transacciones, audit registro.Registro, ahora func() time.Time) *ServicioSMI {
	return &ServicioSMI{repo: repo, tx: tx, auditoria: audit, ahora: ahora}
}

func autorizarSMI(actor dominio.Actor) error {
	if actor.UsuarioID == "" {
		return ErrPermisoSMI
	}
	for _, perfil := range actor.Perfiles {
		if perfil == "coordinacion" {
			return nil
		}
	}
	return ErrPermisoSMI
}

func (s *ServicioSMI) Consultar(ctx context.Context, actor dominio.Actor) ([]dominio.SMI, error) {
	if err := autorizarSMI(actor); err != nil {
		return nil, err
	}
	var filas []dominio.SMI
	err := s.tx.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: actor.Perfiles, EquipoID: actor.EquipoID}, func(ctx context.Context) error {
		var err error
		filas, err = s.repo.SMI(ctx)
		if err != nil {
			return err
		}
		_, err = s.auditoria.Confirmar(ctx, auditoria.Datos{OcurridoEn: s.ahora().UTC().Truncate(time.Microsecond), UsuarioID: actor.UsuarioID, Perfiles: actor.Perfiles, Accion: "catalogo.consultar_smi", Entidad: "catalogo.smi", Resultado: "ok"})
		return err
	})
	return filas, err
}

func (s *ServicioSMI) Guardar(ctx context.Context, actor dominio.Actor, cambio dominio.CambioSMI) (dominio.SMI, error) {
	if err := autorizarSMI(actor); err != nil {
		return dominio.SMI{}, err
	}
	if err := cambio.Validar(); err != nil {
		return dominio.SMI{}, err
	}
	var resultado dominio.SMI
	err := s.tx.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: actor.Perfiles, EquipoID: actor.EquipoID}, func(ctx context.Context) error {
		version, err := s.repo.GuardarSMI(ctx, cambio)
		if err != nil {
			return err
		}
		_, err = s.auditoria.Confirmar(ctx, auditoria.Datos{OcurridoEn: s.ahora().UTC().Truncate(time.Microsecond), UsuarioID: actor.UsuarioID, Perfiles: actor.Perfiles, Accion: fmt.Sprintf("catalogo.actualizar_smi_%d", cambio.Anio), Entidad: "catalogo.smi", Resultado: "ok"})
		if err != nil {
			return err
		}
		filas, err := s.repo.SMI(ctx)
		if err != nil {
			return err
		}
		for _, fila := range filas {
			if fila.Anio == cambio.Anio {
				fila.Version = version
				resultado = fila
				return nil
			}
		}
		return dominio.ErrPersistencia
	})
	return resultado, err
}
