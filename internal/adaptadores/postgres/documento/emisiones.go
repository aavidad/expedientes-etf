package documento

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	"github.com/jackc/pgx/v5"
)

type Repositorio struct{}

func (Repositorio) Bloquear(ctx context.Context, a d.Actor, p d.Solicitud) error {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='20s'`)
	if err == nil {
		_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "documento.clave:"+a.UsuarioID+":"+p.Clave) //etf:tecnico
	}
	if err == nil {
		_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "documento.emision:"+p.ExpedienteID+":"+p.Tipo+":"+p.Idioma) //etf:tecnico
	}
	if err != nil {
		return d.ErrNoDisponible
	}
	return nil
}

const columnas = `e.id::text,e.tipo_documento,e.expediente_id::text,coalesce(e.equipo_id::text,''),
 p.id::text,p.tipo_documento,p.idioma,p.version,p.ruta_relativa,encode(p.huella_sha256,'hex'),p.rol_firmante,
 e.ruta_idioma,e.version_idioma,e.fecha_documento,e.generada_por::text,e.generada_en,
 encode(e.huella_datos,'hex'),encode(e.huella_sha256,'hex'),e.versiones_datos,e.version_motor,e.aviso_clave`

func leer(row pgx.Row) (d.Emision, error) {
	var e d.Emision
	err := row.Scan(&e.ID, &e.Tipo, &e.ExpedienteID, &e.EquipoID, &e.Plantilla.ID, &e.Plantilla.Tipo, &e.Plantilla.Idioma, &e.Plantilla.Version, &e.Plantilla.Ruta, &e.Plantilla.Huella, &e.Plantilla.RolFirmante, &e.IdiomaPaquete.Ruta, &e.IdiomaPaquete.Version, &e.FechaDocumento, &e.GeneradaPor, &e.GeneradaEn, &e.HuellaDatos, &e.HuellaResultado, &e.VersionesDatos, &e.VersionMotor, &e.Aviso)
	if errors.Is(err, pgx.ErrNoRows) {
		return d.Emision{}, nil
	}
	if err != nil {
		return d.Emision{}, d.ErrNoDisponible
	}
	return e, nil
}
func (Repositorio) Peticion(ctx context.Context, a d.Actor, clave string) (d.Emision, string, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.Emision{}, "", err
	}
	var huella string
	err = tx.QueryRow(ctx, `SELECT encode(huella_peticion,'hex') FROM documento.peticion_idempotente WHERE usuario_id=$1 AND clave=$2 AND creada_en>clock_timestamp()-interval '24 hours'`, a.UsuarioID, clave).Scan(&huella) //etf:tecnico
	if errors.Is(err, pgx.ErrNoRows) {
		return d.Emision{}, "", nil
	}
	if err != nil {
		return d.Emision{}, "", d.ErrNoDisponible
	}
	e, err := leer(tx.QueryRow(ctx, `SELECT `+columnas+` FROM documento.peticion_idempotente i JOIN documento.emision e ON e.id=i.emision_id JOIN documento.plantilla p ON p.id=e.plantilla_id WHERE i.usuario_id=$1 AND i.clave=$2`, a.UsuarioID, clave)) //etf:tecnico
	if err == nil && e.ID == "" {
		err = d.ErrAcceso
	}
	return e, huella, err
}
func (Repositorio) Activa(ctx context.Context, tipo, idioma string) (d.Plantilla, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.Plantilla{}, err
	}
	var p d.Plantilla
	err = tx.QueryRow(ctx, `SELECT p.id::text,p.tipo_documento,p.idioma,p.version,p.ruta_relativa,encode(p.huella_sha256,'hex'),p.rol_firmante FROM documento.plantilla p JOIN documento.referencia_activa r ON r.plantilla_id=p.id WHERE r.tipo_documento=$1 AND r.idioma=$2`, tipo, idioma).Scan(&p.ID, &p.Tipo, &p.Idioma, &p.Version, &p.Ruta, &p.Huella, &p.RolFirmante) //etf:tecnico
	if errors.Is(err, pgx.ErrNoRows) {
		return d.Plantilla{}, d.ErrPlantilla
	}
	if err != nil {
		return d.Plantilla{}, d.ErrNoDisponible
	}
	return p, nil
}
func (Repositorio) Ultima(ctx context.Context, p d.Solicitud) (d.Emision, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.Emision{}, err
	}
	return leer(tx.QueryRow(ctx, `SELECT `+columnas+` FROM documento.emision e JOIN documento.plantilla p ON p.id=e.plantilla_id WHERE e.expediente_id=$1 AND e.tipo_documento=$2 AND e.idioma=$3 ORDER BY e.generada_en DESC,e.id DESC LIMIT 1`, p.ExpedienteID, p.Tipo, p.Idioma)) //etf:tecnico
}
func (Repositorio) Guardar(ctx context.Context, a d.Actor, p d.Solicitud, huella string, e d.Emision) error {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	versiones, err := json.Marshal(e.VersionesDatos)
	if err != nil {
		return d.ErrPeticion
	}
	_, err = tx.Exec(ctx, `INSERT INTO documento.emision(id,tipo_documento,expediente_id,equipo_id,plantilla_id,idioma,fecha_documento,generada_por,generada_en,huella_datos,huella_sha256,versiones_datos,version_motor,ruta_idioma,version_idioma,aviso_clave)
 VALUES($1,$2,$3,nullif($4,'')::uuid,$5,$6,$7,$8,$9,decode($10,'hex'),decode($11,'hex'),$12,$13,$14,$15,$16)`, e.ID, e.Tipo, e.ExpedienteID, e.EquipoID, e.Plantilla.ID, e.Plantilla.Idioma, e.FechaDocumento, e.GeneradaPor, e.GeneradaEn, e.HuellaDatos, e.HuellaResultado, versiones, e.VersionMotor, e.IdiomaPaquete.Ruta, e.IdiomaPaquete.Version, e.Aviso)
	if err != nil {
		return d.ErrNoDisponible
	}
	if d.EsOficio(e.Tipo) {
		if _, err := tx.Exec(ctx, `INSERT INTO documento.registro_salida(emision_id,adaptador,estado,version) VALUES($1,'manual','pendiente',1)`, e.ID); err != nil { //etf:tecnico
			return d.ErrNoDisponible
		} //etf:tecnico
	}
	_, err = tx.Exec(ctx, `INSERT INTO documento.peticion_idempotente(usuario_id,clave,ruta,huella_peticion,emision_id,respuesta,creada_en)
 VALUES($1,$2,'/api/documentos/emisiones',decode($3,'hex'),$4::uuid,jsonb_build_object('emision_id',$4::uuid::text),clock_timestamp())
 ON CONFLICT(usuario_id,ruta,clave) DO UPDATE SET huella_peticion=EXCLUDED.huella_peticion,emision_id=EXCLUDED.emision_id,respuesta=EXCLUDED.respuesta,creada_en=EXCLUDED.creada_en
 WHERE documento.peticion_idempotente.creada_en<=clock_timestamp()-interval '24 hours'`, a.UsuarioID, p.Clave, huella, e.ID)
	if err != nil {
		return d.ErrNoDisponible
	}
	return nil
}
