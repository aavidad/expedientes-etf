package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/identidad"
	"github.com/jackc/pgx/v5/pgconn"
)

type Perfiles struct{}

var _ salida.Perfiles = (*Perfiles)(nil)

const consultaAdministrados = `SELECT u.id::text,coalesce(u.cuenta_ad,u.cuenta_ad_anterior) AS cuenta_ad,u.nombre_mostrado,
 CASE WHEN u.estado='activo' AND NOT EXISTS(SELECT FROM identidad.usuario_perfil v WHERE v.usuario_id=u.id AND v.desde<=statement_timestamp() AND (v.hasta IS NULL OR v.hasta>statement_timestamp())) THEN 'pendiente_alta' ELSE u.estado END AS estado,u.version,
 coalesce((SELECT jsonb_agg(jsonb_build_object('perfil',p.perfil,'desde',p.desde,'hasta',p.hasta) ORDER BY p.perfil)
 FROM identidad.usuario_perfil p WHERE p.usuario_id=u.id AND (p.hasta IS NULL OR p.hasta>greatest(p.desde,statement_timestamp()))),'[]'::jsonb),
 (SELECT jsonb_build_object('equipo_id',e.equipo_id,'puesto',e.puesto,'desde',e.desde,'hasta',e.hasta)
 FROM identidad.usuario_equipo e WHERE e.usuario_id=u.id AND (e.hasta IS NULL OR e.hasta>greatest(e.desde,statement_timestamp())))
 FROM identidad.usuario u`

//etf:tecnico
func listarAdministrados(ctx context.Context, e Ejecutor, id string) ([]dominio.UsuarioAdministrado, error) {
	q := consultaAdministrados
	args := []any{}
	if id != "" {
		q += ` WHERE u.id=$1`
		args = append(args, id)
	}
	q = `SELECT * FROM (` + q + `) AS resultado ORDER BY (estado='pendiente_alta') DESC,cuenta_ad NULLS LAST,id`
	rows, err := e.Query(ctx, q, args...)
	if err != nil {
		return nil, errorSQLPerfiles(err)
	}
	defer rows.Close()
	usuarios := []dominio.UsuarioAdministrado{}
	for rows.Next() {
		var u dominio.UsuarioAdministrado
		var perfiles, equipo []byte
		if err = rows.Scan(&u.ID, &u.Cuenta, &u.Nombre, &u.Estado, &u.Version, &perfiles, &equipo); err != nil {
			return nil, err
		}
		for _, parte := range strings.Fields(u.Nombre) {
			letras := []rune(parte)
			u.Iniciales += string(letras[0]) + ". "
		}
		u.Iniciales = strings.TrimSpace(u.Iniciales)
		if err = json.Unmarshal(perfiles, &u.Perfiles); err != nil {
			return nil, err
		}
		if equipo != nil {
			if err = json.Unmarshal(equipo, &u.Equipo); err != nil {
				return nil, err
			}
		}
		usuarios = append(usuarios, u)
	}
	return usuarios, rows.Err()
}

//etf:tecnico
func (*Perfiles) Listar(ctx context.Context) ([]dominio.UsuarioAdministrado, error) {
	e, err := DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	// Serializa la lectura con retiradas; el permiso y la auditoría se confirman
	// antes de entregar la lista, sin una ventana entre revalidación y lectura.
	_, err = e.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('identidad.perfiles',0))`)
	if err != nil {
		return nil, err
	}
	var permitido bool
	if err = e.QueryRow(ctx, `SELECT identidad.es_administrador()`).Scan(&permitido); err != nil {
		return nil, err
	}
	if !permitido {
		return nil, dominio.ErrAccesoDenegado
	}
	usuarios, err := listarAdministrados(ctx, e, "")
	if err != nil {
		return nil, err
	}
	return usuarios, nil
}

//etf:tecnico
func (*Perfiles) Cambiar(ctx context.Context, id string, c dominio.CambioPerfiles) (dominio.UsuarioAdministrado, error) {
	e, err := DesdeContexto(ctx)
	if err != nil {
		return dominio.UsuarioAdministrado{}, err
	}
	perfiles := c.Perfiles
	if perfiles == nil {
		perfiles = []dominio.Concesion{}
	}
	p, _ := json.Marshal(perfiles)
	equipo, _ := json.Marshal(c.Equipo)
	var permitido bool
	err = e.QueryRow(ctx, `SELECT identidad.cambiar_perfiles($1,$2,$3,$4,$5)`, id, c.Version, c.Estado, p, equipo).Scan(&permitido)
	if err != nil {
		return dominio.UsuarioAdministrado{}, errorSQLPerfiles(err)
	}
	if !permitido {
		return dominio.UsuarioAdministrado{}, dominio.ErrCambioPropioDenegado
	}
	usuarios, err := listarAdministrados(ctx, e, id)
	if err != nil {
		return dominio.UsuarioAdministrado{}, err
	}
	if len(usuarios) != 1 {
		return dominio.UsuarioAdministrado{}, dominio.ErrUsuarioNoDisponible
	}
	return usuarios[0], nil
}

func errorSQLPerfiles(err error) error {
	var p *pgconn.PgError
	if errors.As(err, &p) {
		for _, clave := range []dominio.Error{dominio.ErrAccesoDenegado, dominio.ErrConflicto, dominio.ErrPerfilesInvalidos, dominio.ErrUsuarioNoDisponible} {
			if p.Message == string(clave) {
				return clave
			}
		}
		if p.Code == "23503" || p.Code == "23514" || p.Code == "23P01" || p.Code == "22P02" {
			return dominio.ErrPerfilesInvalidos
		}
	}
	return dominio.ErrPersistencia
}
