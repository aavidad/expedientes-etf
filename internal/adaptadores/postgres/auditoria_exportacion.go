package postgres

import (
	"context"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	"github.com/jackc/pgx/v5/pgconn"
)

type LecturaExportacionAuditoria struct{}

func (l LecturaExportacionAuditoria) Delimitar(ctx context.Context, f auditoria.FiltroExportacion) (auditoria.TramoExportacion, error) {
	e, err := DesdeContexto(ctx)
	if err != nil {
		return auditoria.TramoExportacion{}, err
	}
	// Límite adicional del servidor SQL; no depende de la cancelación del cliente.
	if _, err = e.Exec(ctx, `SET LOCAL statement_timeout='25s'`); err != nil { //etf:tecnico
		return auditoria.TramoExportacion{}, falloExportacionSQL(err)
	}
	var t auditoria.TramoExportacion
	err = e.QueryRow(ctx, `SELECT anterior_id,anterior_huella,final_id,final_huella FROM auditoria.delimitar_exportacion($1,$2)`, f.Desde, f.Hasta).Scan(&t.Anterior.ID, &t.Anterior.Huella, &t.Final.ID, &t.Final.Huella) //etf:tecnico
	return t, falloExportacionSQL(err)
}
func (l LecturaExportacionAuditoria) Pagina(ctx context.Context, t auditoria.TramoExportacion, despues uint64) ([]auditoria.Evento, error) {
	e, err := DesdeContexto(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := e.Query(ctx, `SELECT id,ocurrido_en,usuario_id,perfiles,accion,entidad,entidad_id,expediente_id,resultado,huella_anterior,huella FROM auditoria.pagina_exportacion($1,$2)`, despues, t.Final.ID) //etf:tecnico
	if err != nil {
		return nil, falloExportacionSQL(err)
	}
	defer rows.Close()
	eventos := make([]auditoria.Evento, 0, 256)
	for rows.Next() {
		var v auditoria.Evento
		if err = rows.Scan(&v.ID, &v.OcurridoEn, &v.UsuarioID, &v.Perfiles, &v.Accion, &v.Entidad, &v.EntidadID, &v.ExpedienteID, &v.Resultado, &v.HuellaAnterior, &v.Huella); err != nil {
			return nil, falloExportacionSQL(err)
		}
		eventos = append(eventos, v)
	}
	return eventos, falloExportacionSQL(rows.Err())
}
func falloExportacionSQL(err error) error {
	if err == nil {
		return nil
	}
	var p *pgconn.PgError
	if errors.As(err, &p) {
		switch p.Message {
		case "auditoria.permiso_denegado", "auditoria.fechas_invalidas", "auditoria.exportacion_demasiado_grande", "auditoria.tramo_invalido":
			return errors.New(p.Message)
		}
	}
	return errors.New("auditoria.no_disponible")
}
