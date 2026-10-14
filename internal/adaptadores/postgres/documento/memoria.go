package documento

import (
	"context"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/documento"
	"github.com/jackc/pgx/v5"
)

var _ salida.RepositorioMemoria = Repositorio{}

//etf:tecnico
func (Repositorio) BloquearMemoria(ctx context.Context, usuario, clave string) error {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,735421))`, usuario+":"+clave)
	return err
}

//etf:tecnico
func (Repositorio) PeticionMemoria(ctx context.Context, usuario, clave string) (d.EmisionMemoria, error) {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return d.EmisionMemoria{}, err
	}
	var r d.EmisionMemoria
	err = tx.QueryRow(ctx, `SELECT e.id::text,e.anio,coalesce(e.equipo_id::text,''),e.idioma,e.clave,e.generada_por::text,e.generada_en,
 encode(e.huella_peticion,'hex'),encode(e.huella_datos,'hex'),encode(e.huella_resultado,'hex'),e.version_motor,e.ruta_idioma,e.version_idioma,
 p.id::text,p.tipo_documento,p.idioma,p.version,p.ruta_relativa,encode(p.huella_sha256,'hex'),p.rol_firmante
 FROM documento.emision_memoria e JOIN documento.plantilla p ON p.id=e.plantilla_id WHERE e.generada_por=$1 AND e.clave=$2`, usuario, clave).Scan(&r.ID, &r.Solicitud.Anio, &r.Solicitud.EquipoID, &r.Solicitud.Idioma, &r.Solicitud.Clave, &r.GeneradaPor, &r.GeneradaEn, &r.HuellaPeticion, &r.HuellaDatos, &r.HuellaResultado, &r.VersionMotor, &r.IdiomaPaquete.Ruta, &r.IdiomaPaquete.Version, &r.Plantilla.ID, &r.Plantilla.Tipo, &r.Plantilla.Idioma, &r.Plantilla.Version, &r.Plantilla.Ruta, &r.Plantilla.Huella, &r.Plantilla.RolFirmante)
	if errors.Is(err, pgx.ErrNoRows) {
		return d.EmisionMemoria{}, nil
	}
	return r, err
}
func (r Repositorio) PlantillaMemoria(ctx context.Context, idioma string) (d.Plantilla, error) {
	return r.Activa(ctx, "memoria", idioma)
}

//etf:tecnico
func (Repositorio) GuardarMemoria(ctx context.Context, e d.EmisionMemoria) error {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO documento.emision_memoria(id,anio,equipo_id,idioma,clave,plantilla_id,generada_por,generada_en,huella_peticion,huella_datos,huella_resultado,version_motor,ruta_idioma,version_idioma)
 VALUES($1,$2,nullif($3,'')::uuid,$4,$5,$6,$7,$8,decode($9,'hex'),decode($10,'hex'),decode($11,'hex'),$12,$13,$14)`, e.ID, e.Solicitud.Anio, e.Solicitud.EquipoID, e.Solicitud.Idioma, e.Solicitud.Clave, e.Plantilla.ID, e.GeneradaPor, e.GeneradaEn, e.HuellaPeticion, e.HuellaDatos, e.HuellaResultado, e.VersionMotor, e.IdiomaPaquete.Ruta, e.IdiomaPaquete.Version)
	return err
}
