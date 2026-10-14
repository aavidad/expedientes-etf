package hoja

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/xml"
	"errors"
	"fmt"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"reflect"
	"strings"
	"testing"
)

func xlsx(t *testing.T, filas [][]string, cambio func(map[string]string)) []byte {
	t.Helper()
	var hoja strings.Builder
	hoja.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	for i, f := range filas {
		fmt.Fprintf(&hoja, `<row r="%d">`, i+1)
		for j, v := range f {
			fmt.Fprintf(&hoja, `<c r="%c%d" t="inlineStr"><is><t>`, 'A'+j, i+1)
			_ = xml.EscapeText(&hoja, []byte(v))
			hoja.WriteString(`</t></is></c>`)
		}
		hoja.WriteString(`</row>`)
	}
	hoja.WriteString(`</sheetData></worksheet>`)
	partes := map[string]string{
		"[Content_Types].xml":        `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/></Types>`,
		"_rels/.rels":                `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`,
		"xl/workbook.xml":            `<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Casos" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`,
		"xl/worksheets/sheet1.xml":   hoja.String(),
	}
	if cambio != nil {
		cambio(partes)
	}
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for n, v := range partes {
		w, err := z.Create(n)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(v))
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestCSVYXLSXConservanFilasYDatosSinFormulas(t *testing.T) {
	rs := [][]string{{"fecha_derivacion", "estado", "tipo_anexo", "subprograma", "fecha_no_aceptacion", "causa_no_aceptacion", "campos_derivacion"}, {"2026-02-01", "no_aceptada", "anexo3", "RIESGO_SOCIAL", "2026-02-04", "La familia recibe atención de otro recurso.", `[{"codigo":"familia.contacto.apellidos","valor":"Serrano Ruiz"}]`}}
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	w.Comma = ';'
	_ = w.WriteAll(rs)
	lector := Lector{}
	csvFilas, err := lector.LeerHoja(context.Background(), "csv", append([]byte{0xef, 0xbb, 0xbf}, b.Bytes()...))
	if err != nil {
		t.Fatal(err)
	}
	xlsxFilas, err := lector.LeerHoja(context.Background(), "xlsx", xlsx(t, rs, nil))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(csvFilas, xlsxFilas) || len(csvFilas) != 1 || csvFilas[0].Numero != 2 || csvFilas[0].Caso.NoAceptacion.Motivo != rs[1][5] {
		t.Fatal("CSV y XLSX deben conservar el mismo contrato")
	}
	t.Log("CSV UTF-8 y XLSX: mismo caso histórico y número de fila")
}

func TestXLSXRechazaContenidoActivoYLimites(t *testing.T) {
	rs := [][]string{{"fecha_derivacion", "estado", "tipo_anexo"}, {"2026-02-01", "no_aceptada", "anexo3"}}
	pruebas := []struct {
		nombre   string
		cambiar  func(map[string]string)
		esperado error
	}{
		{"formula", func(p map[string]string) {
			p["xl/worksheets/sheet1.xml"] = strings.Replace(p["xl/worksheets/sheet1.xml"], "<is>", "<f>1+1</f><is>", 1)
		}, d.ErrHojaFormato},
		{"hoja_sin_referencia", func(p map[string]string) {
			p["xl/worksheets/sin_referencia.xml"] = `<worksheet><sheetData/></worksheet>`
		}, d.ErrHojaFormato},
		{"formula_fuera_de_hoja", func(p map[string]string) { p["xl/styles.xml"] = `<style><f>1+1</f></style>` }, d.ErrHojaFormato},
		{"macro", func(p map[string]string) { p["xl/vbaProject.bin"] = "binario" }, d.ErrHojaFormato},
		{"enlace_externo", func(p map[string]string) {
			p["xl/_rels/workbook.xml.rels"] = strings.Replace(p["xl/_rels/workbook.xml.rels"], `Target="worksheets/sheet1.xml"`, `Target="https://fuera.invalid/a" TargetMode="External"`, 1)
		}, d.ErrHojaFormato},
		{"ruta_traversal", func(p map[string]string) { p["../fuera.xml"] = "<a/>" }, d.ErrHojaFormato},
		{"hoja_oculta", func(p map[string]string) {
			p["xl/workbook.xml"] = strings.Replace(p["xl/workbook.xml"], `sheetId="1"`, `sheetId="1" state="hidden"`, 1)
		}, d.ErrHojaFormato},
		{"fila_oculta", func(p map[string]string) {
			p["xl/worksheets/sheet1.xml"] = strings.Replace(p["xl/worksheets/sheet1.xml"], `r="2"`, `r="2" hidden="1"`, 1)
		}, d.ErrHojaFormato},
		{"muchas_entradas", func(p map[string]string) {
			for i := 0; i < 65; i++ {
				p[fmt.Sprintf("xl/worksheets/extra%d.xml", i)] = "<worksheet/>"
			}
		}, d.ErrHojaLimites},
		{"bomba_zip", func(p map[string]string) {
			p["xl/styles.xml"] = "<style>" + strings.Repeat(" ", maxDescomprimido) + "</style>"
		}, d.ErrHojaLimites},
	}
	for _, pr := range pruebas {
		t.Run(pr.nombre, func(t *testing.T) {
			_, err := (Lector{}).LeerHoja(context.Background(), "xlsx", xlsx(t, rs, pr.cambiar))
			if !errors.Is(err, pr.esperado) {
				t.Fatalf("rechazo esperado: %v", err)
			}
		})
	}
}

func TestHojaInformeNoIncluyeValoresPersonales(t *testing.T) {
	b := []byte("fecha_derivacion,estado,tipo_anexo,familia\n2026-02-01,aceptada,anexo3,Carmen Serrano Ruiz\n")
	filas, err := (Lector{}).LeerHoja(context.Background(), "csv", b)
	if err != nil || len(filas) != 1 || len(filas[0].Incidencias) != 1 {
		t.Fatal(err)
	}
	if filas[0].Incidencias[0].Columna != "familia" || strings.Contains(fmt.Sprint(filas[0].Incidencias), "Carmen") {
		t.Fatal("informe expone valores")
	}
	for _, b := range []string{"fecha_derivacion,estado,estado\n2026-02-01,a,b\n", "fecha_derivacion,estado,tipo_anexo,dni\n2026-02-01,a,b,c\n"} {
		if _, err := (Lector{}).LeerHoja(context.Background(), "csv", []byte(b)); !errors.Is(err, d.ErrHojaFormato) {
			t.Fatal("cabecera inválida aceptada")
		}
	}
}

func TestXLSXSharedStringsYNumerosDeFila(t *testing.T) {
	rs := [][]string{{"fecha_derivacion", "estado", "tipo_anexo"}, {"2026-02-01", "no_aceptada", "anexo3"}}
	b := xlsx(t, rs, func(p map[string]string) {
		p["xl/sharedStrings.xml"] = `<sst><si><t>fecha_derivacion</t></si></sst>`
		p["xl/worksheets/sheet1.xml"] = strings.Replace(p["xl/worksheets/sheet1.xml"], `t="inlineStr"><is><t>fecha_derivacion</t></is>`, `t="s"><v>0</v>`, 1)
		p["xl/worksheets/sheet1.xml"] = strings.NewReplacer(`r="2"`, `r="7"`, `A2`, `A7`, `B2`, `B7`, `C2`, `C7`).Replace(p["xl/worksheets/sheet1.xml"])
	})
	filas, err := (Lector{}).LeerHoja(context.Background(), "xlsx", b)
	if err != nil || len(filas) != 1 || filas[0].Numero != 7 {
		t.Fatal(err)
	}
}

func TestCSVLimiteDe100CasosYMultilinea(t *testing.T) {
	b := "fecha_derivacion,estado,tipo_anexo,causa_no_aceptacion\n2026-02-01,no_aceptada,anexo3,\"Atendida\nen otro recurso.\"\n2026-02-01,no_aceptada,anexo3,Atendida\n"
	filas, err := (Lector{}).LeerHoja(context.Background(), "csv", []byte(b))
	if err != nil || filas[1].Numero != 4 {
		t.Fatal(err)
	}
	b = "fecha_derivacion,estado,tipo_anexo\n" + strings.Repeat("2026-02-01,no_aceptada,anexo3\n", 101)
	if _, err := (Lector{}).LeerHoja(context.Background(), "csv", []byte(b)); !errors.Is(err, d.ErrHojaLimites) {
		t.Fatal("más de 100 casos aceptados")
	}
}

func TestXLSXLimitaEstructurasAntesDeMaterializarlas(t *testing.T) {
	rs := [][]string{{"fecha_derivacion", "estado", "tipo_anexo"}, {"2026-02-01", "no_aceptada", "anexo3"}}
	casos := []struct {
		nombre  string
		cambiar func(map[string]string)
	}{
		{"cadenas_compartidas", func(p map[string]string) {
			p["xl/sharedStrings.xml"] = "<sst>" + strings.Repeat("<si><t>a</t></si>", 1819) + "</sst>"
		}},
		{"celdas_por_fila", func(p map[string]string) {
			p["xl/worksheets/sheet1.xml"] = "<worksheet><sheetData><row r=\"1\">" + strings.Repeat("<c/>", 19) + "</row></sheetData></worksheet>"
		}},
		{"texto_rico", func(p map[string]string) {
			p["xl/sharedStrings.xml"] = "<sst><si>" + strings.Repeat("<r><t>a</t></r>", 65) + "</si></sst>"
		}},
		{"muchos_nodos", func(p map[string]string) { p["xl/styles.xml"] = "<style>" + strings.Repeat("<x/>", 30000) + "</style>" }},
		{"atributos", func(p map[string]string) {
			v := "<style "
			for i := 0; i < 33; i++ {
				v += fmt.Sprintf("a%d=\"x\" ", i)
			}
			p["xl/styles.xml"] = v + "/>"
		}},
		{"etiqueta_larga", func(p map[string]string) { p["xl/styles.xml"] = "<style a=\"" + strings.Repeat(">", 16385) + "\"/>" }},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			_, err := (Lector{}).LeerHoja(context.Background(), "xlsx", xlsx(t, rs, c.cambiar))
			if !errors.Is(err, d.ErrHojaLimites) {
				t.Fatal(err)
			}
		})
	}
	t.Log("Límites XML comprobados antes de crear listas de cadenas o celdas")
}

func TestHojaRechazaListasNullYElementosNull(t *testing.T) {
	for _, valor := range []string{"null", "[null]", "[{},null]"} {
		var b bytes.Buffer
		w := csv.NewWriter(&b)
		_ = w.WriteAll([][]string{{"fecha_derivacion", "estado", "tipo_anexo", "campos_derivacion"}, {"2026-02-01", "no_aceptada", "anexo3", valor}})
		filas, err := (Lector{}).LeerHoja(context.Background(), "csv", b.Bytes())
		if err != nil || len(filas[0].Incidencias) != 1 || filas[0].Incidencias[0].Clave != "expediente.hoja_json" {
			t.Fatal(valor, err)
		}
	}
}
