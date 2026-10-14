package hoja

// Regresión F5.02: el JSON de las celdas se acota antes de materializarlo.

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"runtime"
	"strings"
	"testing"

	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

func TestHojaRechazaJSONMasivoEnCeldasSinMaterializarlo(t *testing.T) {
	masivo := "[" + strings.Repeat("{},", 100000) + "{}]"
	filas := [][]string{{"fecha_derivacion", "estado", "tipo_anexo", "campos_derivacion", "episodios", "familia", "menores"}}
	for i := 0; i < 10; i++ {
		filas = append(filas, []string{"2026-02-01", "aceptada", "anexo3", masivo, masivo, masivo, masivo})
	}
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	_ = w.WriteAll(filas[:2])
	for formato, contenido := range map[string][]byte{"xlsx": xlsx(t, filas, nil), "csv": b.Bytes()} {
		var antes, despues runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&antes)
		_, err := (Lector{}).LeerHoja(context.Background(), formato, contenido)
		runtime.ReadMemStats(&despues)
		if asignado := despues.TotalAlloc - antes.TotalAlloc; !errors.Is(err, d.ErrHojaLimites) || asignado > 160<<20 {
			t.Fatalf("%s: error %v, asignados %d bytes", formato, err, asignado)
		}
	}
}

func TestHojaAdmiteCeldasJSONRazonables(t *testing.T) {
	campos := "[" + strings.Repeat(`{"codigo":"familia.contacto.apellidos","valor":"Serrano Ruiz"},`, 199) + `{"codigo":"familia.contacto.nombre","valor":"Lucía"}]`
	filas := [][]string{{"fecha_derivacion", "estado", "tipo_anexo", "campos_derivacion"}}
	for i := 0; i < 100; i++ {
		filas = append(filas, []string{"2026-02-01", "aceptada", "anexo3", campos})
	}
	res, err := (Lector{}).LeerHoja(context.Background(), "xlsx", xlsx(t, filas, nil))
	if err != nil || len(res) != 100 || len(res[0].Caso.Datos.Extraccion.Campos) != 200 {
		t.Fatalf("hoja razonable rechazada: %v", err)
	}
}

func TestHojaCuentaArraysAnidados(t *testing.T) {
	adultos := `[{"adultos":[` + strings.Repeat("{},", 10000) + `{}]}]`
	filas := [][]string{{"fecha_derivacion", "estado", "tipo_anexo", "familia", "menores"}}
	for i := 0; i < 76; i++ {
		filas = append(filas, []string{"2026-02-01", "aceptada", "anexo3", adultos, `[{"menores":[` + strings.Repeat("{},", 10000) + `{}]}]`})
	}
	var antes, despues runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&antes)
	_, err := (Lector{}).LeerHoja(context.Background(), "xlsx", xlsx(t, filas, nil))
	runtime.ReadMemStats(&despues)
	if asignado := despues.TotalAlloc - antes.TotalAlloc; !errors.Is(err, d.ErrHojaLimites) || asignado > 160<<20 {
		t.Fatalf("arrays anidados: error %v, asignados %d bytes", err, asignado)
	}
}
