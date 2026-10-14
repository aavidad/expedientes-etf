package documento

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	"github.com/jackc/pgx/v5"
	"time"
)

func (Repositorio) BloquearSPAF(ctx context.Context, a d.Actor, p d.SolicitudSPAF) error {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='20s'`)
	if err == nil {
		_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "documento.spaf.clave:"+a.UsuarioID+":"+p.Clave) //etf:tecnico
	} //etf:tecnico
	if err == nil {
		_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, fmt.Sprintf("documento.spaf:%s:%d:%d:%s", p.EquipoID, p.Anio, p.Mes, p.Idioma)) //etf:tecnico
	} //etf:tecnico
	if err != nil {
		return d.ErrNoDisponible
	}
	return nil
}

// columnasSPAF permite leer la emisión estadística sin inventar expediente.
const columnasSPAF = `e.id::text,e.tipo_documento,coalesce(e.expediente_id::text,''),coalesce(e.equipo_id::text,''),` +
	`p.id::text,p.tipo_documento,p.idioma,p.version,p.ruta_relativa,encode(p.huella_sha256,'hex'),p.rol_firmante,` +
	`e.ruta_idioma,e.version_idioma,e.fecha_documento,e.generada_por::text,e.generada_en,` +
	`encode(e.huella_datos,'hex'),encode(e.huella_sha256,'hex'),e.versiones_datos,e.version_motor,e.aviso_clave`

func (Repositorio) PeticionSPAF(ctx context.Context, a d.Actor, clave string) (d.Emision, string, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.Emision{}, "", err
	}
	var id, huella string
	err = tx.QueryRow(ctx, `SELECT emision_id::text,encode(huella_peticion,'hex') FROM documento.peticion_spaf WHERE usuario_id=$1 AND clave=$2 AND creada_en>clock_timestamp()-interval '24 hours'`, a.UsuarioID, clave).Scan(&id, &huella) //etf:tecnico
	if errors.Is(err, pgx.ErrNoRows) {
		return d.Emision{}, "", nil
	}
	if err != nil {
		return d.Emision{}, "", d.ErrNoDisponible
	}
	e, err := leer(tx.QueryRow(ctx, `SELECT `+columnasSPAF+` FROM documento.emision e JOIN documento.plantilla p ON p.id=e.plantilla_id WHERE e.id=$1 AND e.tipo_documento='spaf'`, id)) //etf:tecnico
	if err == nil && e.ID == "" {
		err = d.ErrAcceso
	}
	return e, huella, err
}
func (Repositorio) UltimaSPAF(ctx context.Context, p d.SolicitudSPAF) (d.Emision, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.Emision{}, err
	}
	corte := time.Date(p.Anio, time.Month(p.Mes)+1, 0, 0, 0, 0, 0, time.UTC)
	return leer(tx.QueryRow(ctx, `SELECT `+columnasSPAF+` FROM documento.emision e JOIN documento.plantilla p ON p.id=e.plantilla_id WHERE e.tipo_documento='spaf' AND e.equipo_id=$1 AND e.fecha_documento=$2 AND e.idioma=$3 ORDER BY e.generada_en DESC,e.id DESC LIMIT 1`, p.EquipoID, corte, p.Idioma)) //etf:tecnico
}
func (Repositorio) GuardarSPAF(ctx context.Context, a d.Actor, p d.SolicitudSPAF, huella string, e d.Emision) error {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	versiones, err := json.Marshal(e.VersionesDatos)
	if err != nil {
		return d.ErrPeticion
	}
	_, err = tx.Exec(ctx, `INSERT INTO documento.emision(id,tipo_documento,expediente_id,equipo_id,plantilla_id,idioma,fecha_documento,generada_por,generada_en,huella_datos,huella_sha256,versiones_datos,version_motor,ruta_idioma,version_idioma,aviso_clave)
 VALUES($1,'spaf',NULL,$2,$3,$4,$5,$6,$7,decode($8,'hex'),decode($9,'hex'),$10,$11,$12,$13,$14)`, e.ID, e.EquipoID, e.Plantilla.ID, e.Plantilla.Idioma, e.FechaDocumento, e.GeneradaPor, e.GeneradaEn, e.HuellaDatos, e.HuellaResultado, versiones, e.VersionMotor, e.IdiomaPaquete.Ruta, e.IdiomaPaquete.Version, e.Aviso)
	if err != nil {
		return d.ErrNoDisponible
	}
	tag, err := tx.Exec(ctx, `INSERT INTO documento.peticion_spaf(usuario_id,clave,huella_peticion,emision_id,creada_en) VALUES($1,$2,decode($3,'hex'),$4,clock_timestamp())
 ON CONFLICT(usuario_id,clave) DO UPDATE SET huella_peticion=EXCLUDED.huella_peticion,emision_id=EXCLUDED.emision_id,creada_en=EXCLUDED.creada_en WHERE documento.peticion_spaf.creada_en<=clock_timestamp()-interval '24 hours'`, a.UsuarioID, p.Clave, huella, e.ID)
	if err != nil || tag.RowsAffected() != 1 {
		return d.ErrNoDisponible
	}
	return nil
}
