package expediente

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
	"github.com/jackc/pgx/v5"
)

type Repositorio struct{}

var _ salida.Repositorio = (*Repositorio)(nil)

func (*Repositorio) Guardar(ctx context.Context, d dominio.Derivacion) error {
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	contenido, err := json.Marshal(d.Datos)
	if err != nil {
		return dominio.ErrNoDisponible
	}
	if _, err := e.Exec(ctx, `INSERT INTO expediente.expediente(id, estado, fecha_recepcion) VALUES ($1, $2, $3)`, d.ID, d.Estado, d.Lectura.LeidaEn); err != nil { //etf:tecnico
		return dominio.ErrNoDisponible
	}
	if _, err := e.Exec(ctx, `INSERT INTO expediente.lectura_anexo
		(id, expediente_id, formato, tamano_bytes, huella_sha256, antivirus, antivirus_detalle,
		plantilla_detectada, campos_dudosos, leida_por, leida_en)
		VALUES ($1,$2,$3,$4,decode($5,'hex'),$6,$7,$8,$9,$10,$11)`,
		d.Lectura.ID, d.ID, d.Lectura.Formato, d.Lectura.TamanoBytes, d.Lectura.HuellaSHA256,
		d.Lectura.Antivirus, d.Lectura.AntivirusDetalle, d.Lectura.PlantillaDetectada, d.Lectura.CamposDudosos, d.Lectura.LeidaPor, d.Lectura.LeidaEn); err != nil {
		return dominio.ErrNoDisponible
	}
	if _, err := e.Exec(ctx, `INSERT INTO expediente.derivacion_datos(expediente_id, esquema_version, contenido) VALUES ($1,$2,$3)`, d.ID, d.Datos.EsquemaVersion, contenido); err != nil { //etf:tecnico
		return dominio.ErrNoDisponible
	}
	return nil
}

func (*Repositorio) Obtener(ctx context.Context, id string) (dominio.Derivacion, error) {
	e, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return dominio.Derivacion{}, err
	}
	var d dominio.Derivacion
	var ampliado bool
	var contenido []byte
	var fechaNo, fechaEspera, fechaLectura *time.Time
	var causa, subprograma *string
	err = e.QueryRow(ctx, `SELECT e.id::text, e.estado, e.numero, e.equipo_id::text, e.municipio_id::text, e.version,
		coalesce(l.id::text,''), coalesce(l.formato,''), coalesce(l.tamano_bytes,0), coalesce(encode(l.huella_sha256,'hex'),''), coalesce(l.antivirus,''),
		coalesce(l.antivirus_detalle,''), coalesce(l.plantilla_detectada,''), coalesce(l.campos_dudosos,'{}'), coalesce(l.leida_por::text,''), l.leida_en, dd.contenido,e.fecha_no_aceptacion,e.causa_no_aceptacion,e.subprograma,e.fecha_lista_espera,coalesce(to_jsonb(e)->>'servicio_derivante',''),to_jsonb(e) ? 'copia_traslado',e.origen='carga_inicial',CASE WHEN e.origen='carga_inicial' THEN to_char(e.fecha_recepcion AT TIME ZONE 'UTC','YYYY-MM-DD') ELSE '' END
		FROM expediente.expediente e LEFT JOIN expediente.lectura_anexo l ON l.expediente_id = e.id
		JOIN expediente.derivacion_datos dd ON dd.expediente_id = e.id WHERE e.id = $1`, id).Scan(
		&d.ID, &d.Estado, &d.Numero, &d.EquipoID, &d.MunicipioID, &d.Version,
		&d.Lectura.ID, &d.Lectura.Formato, &d.Lectura.TamanoBytes, &d.Lectura.HuellaSHA256, &d.Lectura.Antivirus,
		&d.Lectura.AntivirusDetalle, &d.Lectura.PlantillaDetectada, &d.Lectura.CamposDudosos, &d.Lectura.LeidaPor, &fechaLectura, &contenido, &fechaNo, &causa, &subprograma, &fechaEspera, &d.ServicioDerivante, &ampliado, &d.CargadoParaRevision, &d.FechaDerivacion)
	if errors.Is(err, pgx.ErrNoRows) {
		return dominio.Derivacion{}, dominio.ErrNoEncontrada
	}
	if err != nil || json.Unmarshal(contenido, &d.Datos) != nil {
		return dominio.Derivacion{}, dominio.ErrNoDisponible
	}
	if fechaNo != nil && causa != nil && subprograma != nil {
		d.NoAceptacion = &dominio.NoAceptacion{Version: d.Version, Fecha: fechaNo.Format("2006-01-02"), Motivo: *causa, Subprograma: *subprograma}
	}
	if fechaEspera != nil {
		fecha := fechaEspera.Format("2006-01-02")
		d.FechaEspera = &fecha
	}
	if fechaLectura != nil {
		d.Lectura.LeidaEn = fechaLectura.UTC()
	}
	if err := cargarAceptacion(ctx, &d); err != nil {
		return dominio.Derivacion{}, dominio.ErrNoDisponible
	}
	if ampliado {
		if err := cargarTrasladoSeguimiento(ctx, &d); err != nil {
			return dominio.Derivacion{}, err
		}
	}
	return d, nil
}
