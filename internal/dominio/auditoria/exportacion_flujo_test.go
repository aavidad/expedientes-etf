package auditoria

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestExportacionEnFlujoDetectaHuecosYAmbiguedad(t *testing.T) {
	datos := Datos{OcurridoEn: time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC), UsuarioID: "sistema", Perfiles: []string{}, Accion: "auditoria.consultar", Entidad: "auditoria", Resultado: "ok"}
	e1, _ := Encadenar(2, datos, Inicio())
	e2, _ := Encadenar(5, datos, e1.Punto())
	e, _ := NuevaExportacion([]Evento{e1, e2}, Inicio(), e2.Punto())
	b, _ := e.Codificar()
	tramo, n, err := RecorrerExportacion(bytes.NewReader(b), nil)
	if err != nil || n != 2 || tramo.Final != e2.Punto() {
		t.Fatal(n, err)
	}
	for _, alterado := range []string{
		strings.Replace(string(b), `"id":2`, `"id":3`, 1),
		strings.Replace(string(b), `"version":`, `"version":"etf.auditoria.v1","version":`, 1),
		strings.Replace(string(b), `"anterior":{"id":0`, `"anterior":{"desconocido":1,"id":0`, 1),
		strings.Replace(string(b), `"perfiles":[]`, `"perfiles":null`, 1),
		string(b) + `{}`,
	} {
		if _, _, err := RecorrerExportacion(strings.NewReader(alterado), nil); err == nil {
			t.Fatal("acepta formato ambiguo o cadena rota")
		}
	}
}
func TestFiltrosExportacionFechasYIdentificadores(t *testing.T) {
	desde := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	f := FiltroExportacion{Desde: desde, Hasta: desde.AddDate(0, 0, 31)}
	if f.Validar() != nil {
		t.Fatal("rechaza 31 días")
	}
	f.Hasta = desde.AddDate(0, 0, 32)
	if f.Validar() == nil {
		t.Fatal("acepta 32 días")
	}
	f.Hasta = desde.AddDate(0, 0, 1)
	f.ExpedienteID = "02-14-26"
	if f.Validar() == nil {
		t.Fatal("acepta número en lugar de id")
	}
}
