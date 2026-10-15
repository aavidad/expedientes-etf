package ficheros

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
)

func cadenaExportacionPrueba(t *testing.T) []auditoria.Evento {
	t.Helper()
	var eventos []auditoria.Evento
	previo := auditoria.Inicio()
	for i := 1; i <= 4; i++ {
		d := auditoria.Datos{OcurridoEn: time.Date(2026, 10, 14, 8, i, 0, 0, time.UTC), UsuarioID: "sistema", Perfiles: []string{}, Accion: "expediente.consultar", Entidad: "expediente", Resultado: "ok"}
		if i%2 == 0 {
			d.Accion = "catalogo.consultar"
			d.Entidad = "catalogo"
		}
		e, err := auditoria.Encadenar(uint64(i*2), d, previo)
		if err != nil {
			t.Fatal(err)
		}
		eventos = append(eventos, e)
		previo = e.Punto()
	}
	return eventos
}
func prepararPrueba(t *testing.T, formato string) ([]byte, auditoria.ManifiestoExportacion) {
	t.Helper()
	eventos := cadenaExportacionPrueba(t)
	f := auditoria.FiltroExportacion{Desde: time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC), Hasta: time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC), TipoEvento: "expediente"}
	p := NuevasExportacionesAuditoria()
	a, err := p.Preparar(context.Background(), auditoria.TramoExportacion{Anterior: auditoria.Inicio(), Final: eventos[3].Punto()}, f, formato, "auditoria.instrucciones_verificacion", func(c func(auditoria.Evento) error) error {
		for _, e := range eventos {
			if err := c(e); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := io.ReadAll(a)
	if err != nil {
		t.Fatal(err)
	}
	m, err := VerificarPaqueteAuditoria(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	if m.Filas != 2 || m.EventosCadena != 4 {
		t.Fatalf("selección: %+v", m)
	}
	return b, m
}
func cambiarArchivo(t *testing.T, b []byte, nombre string, cambiar func([]byte) []byte) []byte {
	t.Helper()
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	w := zip.NewWriter(&out)
	for _, f := range z.File {
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		v, err := io.ReadAll(r)
		_ = r.Close()
		if err != nil {
			t.Fatal(err)
		}
		if f.Name == nombre {
			v = cambiar(v)
		}
		e, err := w.Create(f.Name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = e.Write(v); err != nil {
			t.Fatal(err)
		}
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
func TestPaqueteAuditoriaCSVYJSON(t *testing.T) {
	for _, formato := range []string{"csv", "json"} {
		t.Run(formato, func(t *testing.T) { prepararPrueba(t, formato) })
	}
}
func TestPaqueteAuditoriaDetectaAlteracionYSeleccionIncompleta(t *testing.T) {
	b, _ := prepararPrueba(t, "json")
	for _, archivo := range []string{"registro.json", "cadena.json", "manifiesto.json"} {
		t.Run(archivo, func(t *testing.T) {
			alterado := cambiarArchivo(t, b, archivo, func(v []byte) []byte { return bytes.Replace(v, []byte("expediente"), []byte("documento"), 1) })
			if _, err := VerificarPaqueteAuditoria(bytes.NewReader(alterado), int64(len(alterado))); err == nil {
				t.Fatal("acepta alteración")
			}
		})
	}
	// Recalcular sólo la huella del registro no permite ocultar una fila.
	alterado := cambiarArchivo(t, b, "registro.json", func(v []byte) []byte { return []byte("[]\n") })
	alterado = cambiarArchivo(t, alterado, "manifiesto.json", func(v []byte) []byte {
		var m auditoria.ManifiestoExportacion
		_ = json.Unmarshal(v, &m)
		m.Filas = 0
		h := sha256.Sum256([]byte("[]\n"))
		m.HuellaRegistro = hex.EncodeToString(h[:])
		o, _ := json.Marshal(m)
		return o
	})
	if _, err := VerificarPaqueteAuditoria(bytes.NewReader(alterado), int64(len(alterado))); err == nil {
		t.Fatal("acepta selección incompleta")
	}
}
func TestPaqueteAuditoriaLimitaYCancelaTemporales(t *testing.T) {
	p := NuevasExportacionesAuditoria()
	f := auditoria.FiltroExportacion{Desde: time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC), Hasta: time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)}
	tramo := auditoria.TramoExportacion{Anterior: auditoria.Inicio(), Final: auditoria.Inicio()}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := p.Preparar(ctx, tramo, f, "csv", "", func(func(auditoria.Evento) error) error { return nil }); err == nil {
		t.Fatal("acepta cancelación")
	}
	var presupuesto int64 = 3
	if _, err := (limiteAuditoria{io.Discard, &presupuesto, context.Background()}).Write([]byte("1234")); err == nil {
		t.Fatal("sin límite de bytes")
	}
	for i := 0; i < 2; i++ {
		p.plazas <- struct{}{}
	}
	if _, err := p.Preparar(context.Background(), tramo, f, "json", "", nil); err == nil || !strings.Contains(err.Error(), "ocupada") {
		t.Fatal("sin límite de concurrencia")
	}
}
