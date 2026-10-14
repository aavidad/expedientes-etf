package postgres

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

func datosAuditoria() auditoria.Datos {
	return auditoria.Datos{OcurridoEn: time.Date(2026, 10, 13, 8, 42, 11, 123456000, time.UTC),
		UsuarioID: "a66cfd72-7f02-4b1e-b934-5da9d245bc33", Perfiles: []string{"administracion"},
		Accion: "expediente.leer_ficha", Entidad: "expediente",
		EntidadID: "f82d655e-a690-4a02-b794-5463d8880644", Resultado: "ok"}
}

func administracionAuditoria() plataforma.Identidad {
	d := datosAuditoria()
	return plataforma.Identidad{UsuarioID: d.UsuarioID, Perfiles: d.Perfiles}
}

func TestAuditoriaRechazaEntradaSinAbrirConexion(t *testing.T) {
	r := NuevoRegistroAuditoria(&Base{})
	d := datosAuditoria()
	d.OcurridoEn = time.Time{}
	if _, err := r.Confirmar(context.Background(), d); err == nil || err.Error() != "auditoria.fecha_invalida" {
		t.Fatal(err)
	}
	d = datosAuditoria()
	d.UsuarioID = "sistema"
	if _, err := r.Confirmar(context.Background(), d); err == nil || err.Error() != "auditoria.perfiles_invalidos" {
		t.Fatal(err)
	}
	if _, err := r.RegistrarEnTransaccion(context.Background(), datosAuditoria()); !errors.Is(err, ErrTransaccion) {
		t.Fatal(err)
	}
	for _, tramo := range [][2]uint64{{0, 1}, {2, 1}, {1, uint64(math.MaxInt64) + 1}} {
		if _, err := r.Exportar(context.Background(), administracionAuditoria(), tramo[0], tramo[1]); err == nil || err.Error() != "auditoria.tramo_invalido" {
			t.Fatal(err)
		}
	}
}

func TestAuditoriaNoExponeDiagnosticosPostgres(t *testing.T) {
	err := errorAuditoria(&pgconn.PgError{Message: "diagnostico con datos de una persona", Detail: "contenido reservado"})
	if err.Error() != "auditoria.persistencia_fallida" {
		t.Fatal(err)
	}
	err = errorAuditoria(&pgconn.PgError{Message: "auditoria.permiso_denegado", Detail: "contenido reservado"})
	if err.Error() != "auditoria.permiso_denegado" {
		t.Fatal(err)
	}
}
