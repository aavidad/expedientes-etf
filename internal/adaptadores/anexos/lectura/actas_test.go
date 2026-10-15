package lectura

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"os"
	"testing"
)

func TestActasWordODTPDFFirmadoYSinTexto(t *testing.T) {
	for _, tt := range []struct{ nombre, tipo, numero, fecha string }{
		{"aceptacion-2022.docx", "aceptacion", "02-71-22", "2022-05-17"},
		{"aceptacion-2022.odt", "aceptacion", "02-71-22", "2022-05-17"},
		{"aceptacion-2022.pdf", "aceptacion", "02-71-22", "2022-05-17"},
		{"aceptacion-2022-firmada.pdf", "aceptacion", "02-71-22", "2022-05-17"},
		{"aceptacion-2023.docx", "aceptacion", "02-72-23", "2023-03-09"},
		{"aceptacion-2023.odt", "aceptacion", "02-72-23", "2023-03-09"},
		{"aceptacion-2023.pdf", "aceptacion", "02-72-23", "2023-03-09"},
		{"cierre-2026.docx", "cierre", "02-72-23", "2026-06-20"},
		{"cierre-2026.odt", "cierre", "02-72-23", "2026-06-20"},
		{"cierre-2026.pdf", "cierre", "02-72-23", "2026-06-20"},
	} {
		t.Run(tt.nombre, func(t *testing.T) {
			b, err := os.ReadFile("../actas/testdata/" + tt.nombre)
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			formato, err := (Validador{}).Validar(ctx, tt.nombre, b)
			if err != nil {
				t.Fatal(err)
			}
			lector := NuevoLector()
			r, err := lector.LeerActa(ctx, formato, b)
			if err != nil {
				t.Fatal(err)
			}
			if r.Tipo != tt.tipo || !r.PlantillaReconocida {
				t.Fatalf("tipo %q", r.Tipo)
			}
			campos := map[string]string{}
			for _, c := range r.Campos {
				if !c.Dudoso {
					t.Fatal("lectura sin revisión")
				}
				campos[c.Codigo] = c.Valor
			}
			for codigo, valor := range map[string]string{"numero": tt.numero, "fecha": tt.fecha, "familia": "Familia Villena Robles", "menor.nombre": "Clara Villena Robles", "menor.nacimiento": "2014-06-14", "municipio": "Santa Fe", "centro": "Santa Fe"} {
				if campos[codigo] != valor {
					t.Errorf("%s: %q, esperado %q", codigo, campos[codigo], valor)
				}
			}
			if tt.tipo == "cierre" && campos["motivo"] != "Se alcanzan los objetivos previstos en el proyecto de intervención familiar." {
				t.Errorf("motivo incompleto: %q", campos["motivo"])
			}
			anexo, err := lector.Leer(ctx, formato, b)
			if err != nil && formato == "pdf" {
				t.Fatal(err)
			}
			if anexo.Tipo == "aceptacion" || anexo.Tipo == "cierre" {
				t.Fatal("modo actas contamina derivaciones")
			}
		})
	}
	b, err := os.ReadFile("../pdf/testdata/sin_texto.pdf")
	if err != nil {
		t.Fatal(err)
	}
	r, err := NuevoLector().LeerActa(context.Background(), "pdf", b)
	if err != nil || len(r.Campos) != 0 || r.PlantillaReconocida {
		t.Fatal("escaneado exige carga manual", r, err)
	}
}

func TestMetadatosActasODTLimpios(t *testing.T) {
	for _, nombre := range []string{"aceptacion-2022", "aceptacion-2023", "cierre-2026"} {
		b, err := os.ReadFile("../actas/testdata/" + nombre + ".odt")
		if err != nil {
			t.Fatal(err)
		}
		z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
		if err != nil {
			t.Fatal(err)
		}
		meta := false
		for _, f := range z.File {
			if f.Comment != "" || f.Modified.Year() != 2026 {
				t.Fatal(nombre, "metadatos ZIP sin limpiar")
			}
			if f.Name != "meta.xml" {
				continue
			}
			meta = true
			r, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			datos, err := io.ReadAll(r)
			r.Close()
			if err != nil || !bytes.Contains(datos, []byte("<office:meta/>")) || bytes.Contains(datos, []byte("creator")) || bytes.Contains(datos, []byte("date")) {
				t.Fatal(nombre, "metadatos de autor o fechas sin limpiar", err)
			}
		}
		if !meta {
			t.Fatal(nombre, "faltan metadatos sanitizados")
		}
	}
}
