package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/identidad"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Usuarios struct {
	base                *Base
	primerAdministrador string
}
type Sesiones struct{ base *Base }

var _ salida.Usuarios = (*Usuarios)(nil)
var _ salida.Sesiones = (*Sesiones)(nil)

func NuevosUsuarios(base *Base) *Usuarios { return &Usuarios{base: base} }

// NuevosUsuariosConAdministrador recibe sólo configuración del servidor.
func NuevosUsuariosConAdministrador(base *Base, cuenta string) (*Usuarios, error) {
	if cuenta != "" {
		normalizada, err := dominio.NormalizarCuenta(cuenta)
		if err != nil || normalizada != cuenta {
			return nil, dominio.ErrConfiguracion
		}
	}
	return &Usuarios{base: base, primerAdministrador: cuenta}, nil
}
func NuevasSesiones(base *Base) *Sesiones { return &Sesiones{base} }

// Reutiliza la transacción de Modificar al leer al usuario de esa sesión.
// Sin contexto, inicia una operación anónima; RLS deniega la lectura de filas.
func (b *Base) identidad(ctx context.Context, f func(context.Context, Ejecutor) error) error {
	if e, err := DesdeContexto(ctx); err == nil {
		return f(ctx, e)
	}
	return b.Ejecutar(ctx, plataforma.Identidad{}, func(ctx context.Context) error {
		e, err := DesdeContexto(ctx)
		if err != nil {
			return err
		}
		return f(ctx, e)
	})
}

func leerUsuario(row pgx.Row) (dominio.Usuario, error) {
	var u dominio.Usuario
	var guid pgtype.UUID
	err := row.Scan(&u.ID, &guid, &u.CuentaAD, &u.NombreMostrado, &u.Estado)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, dominio.ErrUsuarioNoDisponible
	}
	if err != nil {
		return u, dominio.ErrPersistencia
	}
	u.GUID = dominio.GUID(guid.Bytes)
	return u, nil
}

//etf:tecnico
func (u *Usuarios) RegistrarAcceso(ctx context.Context, i dominio.IdentidadDirectorio, ahora time.Time) (persona dominio.Usuario, err error) {
	cuenta, fallo := dominio.NormalizarCuenta(i.CuentaAD)
	if fallo != nil || cuenta != i.CuentaAD || i.GUID == (dominio.GUID{}) || ahora.IsZero() {
		return persona, dominio.ErrCredencialesInvalidas
	}
	err = u.base.identidad(ctx, func(ctx context.Context, e Ejecutor) error {
		// Orden único de bloqueo con los cambios de permisos y el alta inicial.
		if _, fallo := e.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('identidad.perfiles',0))`); fallo != nil {
			return fallo
		}
		var fallo error
		persona, fallo = leerUsuario(e.QueryRow(ctx, `SELECT id::text, object_guid_ad, coalesce(cuenta_ad, cuenta_ad_anterior), nombre_mostrado, estado
    FROM identidad.registrar_acceso($1,$2,$3,$4)`, fmt.Sprintf("%x-%x-%x-%x-%x", i.GUID[:4], i.GUID[4:6], i.GUID[6:8], i.GUID[8:10], i.GUID[10:]), cuenta, i.NombreMostrado, ahora))
		if fallo != nil {
			return fallo
		}
		if cuenta == u.primerAdministrador {
			_, fallo = e.Exec(ctx, `SELECT identidad.conceder_administrador_inicial($1,$2,$3,$4)`, persona.ID, fmt.Sprintf("%x-%x-%x-%x-%x", i.GUID[:4], i.GUID[4:6], i.GUID[6:8], i.GUID[8:10], i.GUID[10:]), cuenta, ahora)
			if fallo != nil {
				return fallo
			}
		}
		persona, fallo = u.Obtener(ctx, persona.ID)
		return fallo
	})
	return
}

//etf:tecnico
func (u *Usuarios) Obtener(ctx context.Context, id string) (persona dominio.Usuario, err error) {
	err = u.base.identidad(ctx, func(ctx context.Context, e Ejecutor) error {
		var fallo error
		persona, fallo = leerUsuario(e.QueryRow(ctx, `SELECT id::text, object_guid_ad, coalesce(cuenta_ad, cuenta_ad_anterior), nombre_mostrado, estado
 FROM identidad.usuario WHERE id=$1`, id))
		if fallo != nil {
			return fallo
		}
		rows, fallo := e.Query(ctx, `SELECT perfil FROM identidad.usuario_perfil WHERE usuario_id=$1 AND desde<=statement_timestamp() AND (hasta IS NULL OR hasta>statement_timestamp()) ORDER BY perfil`, id)
		if fallo != nil {
			return fallo
		}
		defer rows.Close()
		for rows.Next() {
			var p dominio.Perfil
			if fallo = rows.Scan(&p); fallo != nil {
				return fallo
			}
			persona.Perfiles = append(persona.Perfiles, p)
		}
		if fallo = rows.Err(); fallo != nil {
			return fallo
		}
		fallo = e.QueryRow(ctx, `SELECT equipo_id::text FROM identidad.usuario_equipo WHERE usuario_id=$1 AND desde<=statement_timestamp() AND (hasta IS NULL OR hasta>statement_timestamp())`, id).Scan(&persona.EquipoID)
		if fallo != nil && !errors.Is(fallo, pgx.ErrNoRows) {
			return fallo
		}
		perfiles := []string{}
		for _, p := range persona.Perfiles {
			perfiles = append(perfiles, string(p))
		}
		equipos := "{}"
		if persona.EquipoID != "" {
			equipos = "{" + persona.EquipoID + "}"
		}
		_, fallo = e.Exec(ctx, `SELECT set_config('app.perfiles',$1,true),set_config('app.equipos',$2,true)`, strings.Join(perfiles, ","), equipos)
		return fallo
	})
	return
}
func (s *Sesiones) Crear(ctx context.Context, ses dominio.Sesion) error {
	if ses.Huella == (dominio.Huella{}) || ses.UsuarioID == "" {
		return dominio.ErrSesionNoValida
	}
	return s.base.identidad(ctx, func(ctx context.Context, e Ejecutor) error {
		_, err := e.Exec(ctx, `SELECT
 identidad.crear_sesion($1,$2,$3,$4,$5)`, ses.Huella[:], ses.UsuarioID, ses.CreadaEn, ses.UltimaActividadEn, ses.CaducaEn)
		return err
	})
}
func (s *Sesiones) Modificar(ctx context.Context, huella dominio.Huella, modificar func(context.Context, *dominio.Sesion) error) error {
	if modificar == nil || huella == (dominio.Huella{}) {
		return dominio.ErrSesionNoValida
	}
	return s.base.identidad(ctx, func(ctx context.Context, e Ejecutor) error {
		var ses dominio.Sesion
		var bytes []byte
		err := e.QueryRow(ctx, `SELECT id_huella, usuario_id::text, creada_en, ultima_actividad_en, caduca_en
 FROM identidad.bloquear_sesion($1)`, huella[:]).Scan(&bytes, &ses.UsuarioID, &ses.CreadaEn, &ses.UltimaActividadEn, &ses.CaducaEn)
		if errors.Is(err, pgx.ErrNoRows) {
			return dominio.ErrSesionNoValida
		}
		if err != nil {
			return err
		}
		copy(ses.Huella[:], bytes)
		anterior := ses
		if err := modificar(ctx, &ses); err != nil {
			return err
		}
		// La modificación sólo puede tocar actividad; no puede transferir ni revivir.
		if ses.Huella != anterior.Huella || ses.UsuarioID != anterior.UsuarioID || !ses.CreadaEn.Equal(anterior.CreadaEn) || !ses.CaducaEn.Equal(anterior.CaducaEn) || ses.UltimaActividadEn.Before(anterior.UltimaActividadEn) || !ses.UltimaActividadEn.Before(ses.CaducaEn) {
			return dominio.ErrSesionNoValida
		}
		_, err = e.Exec(ctx, `SELECT
 identidad.tocar_sesion($1,$2)`, huella[:], ses.UltimaActividadEn)
		return err
	})
}
func (s *Sesiones) Eliminar(ctx context.Context, huella dominio.Huella) error {
	return s.base.identidad(ctx, func(ctx context.Context, e Ejecutor) error {
		_, err := e.Exec(ctx, `SELECT
 identidad.eliminar_sesion($1)`, huella[:])
		return err
	})
}

// EjecutarIdentidad es el límite transaccional del inicio de sesión, después de AD.
func (b *Base) EjecutarIdentidad(ctx context.Context, f func(context.Context) error) error {
	return b.Ejecutar(ctx, plataforma.Identidad{}, f)
}

var _ salida.Operaciones = (*Base)(nil)

// RegistrarFallo usa la función pública de F0.08b, sin leer ni escribir sus
// tablas. Un intento sin identidad comprobada se registra como actor anónimo.
func (b *Base) RegistrarFallo(ctx context.Context, accion, resultado string) error {
	if resultado != "error" && resultado != "denegado" {
		return dominio.ErrPersistencia
	}
	actor := salida.ActorAuditoria(ctx)
	i := plataforma.Identidad{UsuarioID: actor.ID, EquipoID: actor.EquipoID}
	for _, perfil := range actor.Perfiles {
		i.Perfiles = append(i.Perfiles, string(perfil))
	}
	return b.Ejecutar(ctx, i, func(ctx context.Context) error {
		e, err := DesdeContexto(ctx)
		if err != nil {
			return err
		}
		_, err = e.Exec(ctx, `SELECT
 auditoria.registrar(clock_timestamp(),$1,'identidad.sesion',NULL,NULL,$2)`, accion, resultado)
		return err
	})
}
