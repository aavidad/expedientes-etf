package docx

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func fixture(t *testing.T, nombre string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + nombre + ".docx")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestAnexosCompletos(t *testing.T) {
	b, err := os.ReadFile("testdata/esperado.json")
	if err != nil {
		t.Fatal(err)
	}
	var esperados map[string]map[string][]string
	if err = json.Unmarshal(b, &esperados); err != nil {
		t.Fatal(err)
	}
	for _, nombre := range []string{"anexo3", "anexo4"} {
		t.Run(nombre, func(t *testing.T) {
			l, e := Leer(fixture(t, nombre))
			if e != nil {
				t.Fatal(e)
			}
			if !l.PlantillaReconocida || l.Tipo != nombre || l.Version != "cliente-2026-v1" {
				t.Fatal("Identificación incorrecta", l.Tipo, l.Version)
			}
			valores := map[string][]string{}
			for _, c := range l.Campos {
				valores[c.Codigo] = append(valores[c.Codigo], normalizar(c.Valor))
				if c.Pagina != 0 {
					t.Fatal("Paginación inventada")
				}
				if c.Valor == "" && (!c.Dudoso || !slices.Contains(c.Motivos, "anexo.campo_ausente")) {
					t.Fatal("Vacío sin señalar", c.Codigo)
				}
				if c.Valor != "" && c.Dudoso {
					t.Fatal("Valor cumplimentado dudoso", c.Codigo, c.Motivos)
				}
			}
			if len(valores) != len(esperados[nombre]) {
				t.Fatal("Conjunto de campos distinto", len(valores), len(esperados[nombre]))
			}
			for codigo, esperado := range esperados[nombre] {
				if !slices.Equal(valores[codigo], esperado) {
					t.Errorf("%s: recibido %q; esperado %q", codigo, valores[codigo], esperado)
				}
			}
			if len(valores["menores.persona.nombre"]) != 6 {
				t.Fatal("Faltan filas añadidas")
			}
			for codigo := range valores {
				if strings.Contains(codigo, "sexo") {
					t.Fatal("Sexo inferido")
				}
			}
		})
	}
}

// Este contraste usa valores del corpus y equivalencias del PRD escritos aparte
// del generador, para detectar errores compartidos por perfiles y esperado.json.
func TestCorrespondenciaIndependiente(t *testing.T) {
	l, err := Leer(fixture(t, "anexo4"))
	if err != nil {
		t.Fatal(err)
	}
	esperados := map[string][]string{
		"servicio.etf":            {"ETF Alhama"},
		"servicio.subprograma":    {"Reintegracion"},
		"servicio.equipo_menores": {"Equipo Sierra Sur"},
		"menores.persona.nombre":  {"Nerea Pérez Ortega", "Álvaro Pérez Ortega", "Celia Pérez Ortega", "Mateo Pérez Ortega", "Irene Pérez Ortega", "Hugo Pérez Ortega"},
		"sociodemografia.adulto.estudios": {
			"Estudios obligatorios (ESO, EGB)",
			"Estudios secundarios (Bachiller, FP, módulos profesionales)",
			"Sabe leer y escribir, no finaliza estudios obligatorios",
			"Estudios universitarios", "Analfabetismo",
		},
		"valoracion.crisis":                {"SI"},
		"valoracion.actitud":               {"Colaborativa"},
		"valoracion.historia":              {"La familia reside en Alhama desde 2010."},
		"valorame.descripcion":             {"Necesita apoyo educativo y seguimiento familiar."},
		"intervenciones.resultados":        {"Mejora inicial de la coordinación con el centro educativo."},
		"resoluciones.organismo":           {"Delegación Territorial de Granada"},
		"resoluciones.fecha":               {"15/05/2025"},
		"observaciones.otra_documentacion": {"Informe educativo e informe social."},
	}
	for codigo, esperado := range esperados {
		var valores []string
		for _, c := range l.Campos {
			if c.Codigo == codigo {
				valores = append(valores, normalizar(c.Valor))
			}
		}
		if !slices.Equal(valores, esperado) {
			t.Errorf("%s: %q; esperado %q", codigo, valores, esperado)
		}
	}
}

func cambiar(t *testing.T, b []byte, modificar func(string, []byte) []byte) []byte {
	t.Helper()
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	w := zip.NewWriter(&out)
	for _, f := range z.File {
		r, e := f.Open()
		if e != nil {
			t.Fatal(e)
		}
		var datos bytes.Buffer
		if _, e = datos.ReadFrom(r); e != nil {
			t.Fatal(e)
		}
		r.Close()
		d := modificar(f.Name, datos.Bytes())
		if d == nil {
			continue
		}
		h := f.FileHeader
		h.Method = zip.Deflate
		dest, e := w.CreateHeader(&h)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = dest.Write(d); e != nil {
			t.Fatal(e)
		}
	}
	if e := w.Close(); e != nil {
		t.Fatal(e)
	}
	return out.Bytes()
}

func modificarDocumento(t *testing.T, b []byte, f func(string) string) []byte {
	return cambiar(t, b, func(nombre string, d []byte) []byte {
		if nombre == "word/document.xml" {
			return []byte(f(string(d)))
		}
		return d
	})
}

func TestDudososYListasLocales(t *testing.T) {
	base := fixture(t, "anexo3")
	for _, tc := range []struct{ valor, motivo string }{{"Elija un elemento.", "anexo.texto_relleno"}, {"Fuera del catálogo", "anexo.valor_fuera_lista"}} {
		b := modificarDocumento(t, base, func(s string) string { return strings.Replace(s, ">ed primaria<", ">"+tc.valor+"<", 1) })
		l, e := Leer(b)
		if e != nil {
			t.Fatal(e)
		}
		for _, c := range l.Campos {
			if c.Codigo == "menores.persona.escolarizacion" {
				if !c.Dudoso || !slices.Contains(c.Motivos, tc.motivo) {
					t.Fatal(c)
				}
				break
			}
		}
	}
	// Una opción añadida por el remitente no amplía el catálogo conocido.
	b := modificarDocumento(t, base, func(s string) string {
		s = strings.Replace(s, ">ed primaria<", ">Universidad remota<", 1)
		return strings.Replace(s, "</w:dropDownList>", `<w:listItem w:displayText="Universidad remota" w:value="Universidad remota" /></w:dropDownList>`, 1)
	})
	l, e := Leer(b)
	if e != nil {
		t.Fatal(e)
	}
	for _, c := range l.Campos {
		if c.Codigo == "menores.persona.escolarizacion" {
			if !c.Dudoso {
				t.Fatal("Lista del remitente aceptada")
			}
			break
		}
	}
	for _, nombre := range []string{"anexo3", "anexo4"} {
		b, e := os.ReadFile("testdata/" + nombre + ".docx")
		if e != nil {
			t.Fatal(e)
		}
		l, e := Leer(b)
		if e != nil {
			t.Fatal(e)
		}
		for _, c := range l.Campos {
			if strings.Contains(c.Valor, "Haga clic") || strings.Contains(c.Valor, "Elija un elemento") {
				if !c.Dudoso || !slices.Contains(c.Motivos, "anexo.texto_relleno") {
					t.Fatal(c.Codigo, c.Motivos)
				}
			}
		}
	}
}

func TestVersionDesconocida(t *testing.T) {
	for _, b := range [][]byte{fixture(t, "version_desconocida"), modificarDocumento(t, fixture(t, "anexo3"), func(s string) string { return strings.Replace(s, "Estructura familiar", "Estructura alterada", 1) })} {
		l, e := Leer(b)
		if e == nil || e.Error() != "anexo.plantilla_desconocida" || len(l.Campos) != 0 {
			t.Fatal(l, e)
		}
	}
}

func TestTamanoDecimal(t *testing.T) {
	_, err := Leer(make([]byte, 10_000_001))
	if err == nil || err.Error() != "anexo.docx_tamano_excedido" {
		t.Fatal("No se respeta el máximo de 10 MB", err)
	}
}

func TestLimiteMenores(t *testing.T) {
	base := fixture(t, "anexo3")
	patron := regexp.MustCompile(`(?s)<w:tr(?:\s[^>]*)?>.*?</w:tr>`)
	for _, cantidad := range []int{15, 16} {
		b := modificarDocumento(t, base, func(s string) string {
			inicio := strings.Index(s, "<w:tbl>")
			for i := 0; i < 3; i++ {
				inicio += strings.Index(s[inicio:], "</w:tbl>") + len("</w:tbl>")
				inicio += strings.Index(s[inicio:], "<w:tbl>")
			}
			fin := inicio + strings.Index(s[inicio:], "</w:tbl>")
			filas := patron.FindAllString(s[inicio:fin], -1)
			return s[:fin] + strings.Repeat(filas[1], cantidad-6) + s[fin:]
		})
		l, e := Leer(b)
		if cantidad == 16 {
			if e == nil || e.Error() != "anexo.docx_demasiados_menores" || len(l.Campos) != 0 {
				t.Fatal(e)
			}
			continue
		}
		if e != nil {
			t.Fatal(e)
		}
		cuenta := 0
		for _, c := range l.Campos {
			if c.Codigo == "menores.persona.nombre" {
				cuenta++
			}
		}
		if cuenta != 15 {
			t.Fatal(cuenta)
		}
	}
}

func TestCasillasAmbiguas(t *testing.T) {
	b := modificarDocumento(t, fixture(t, "anexo4"), func(s string) string {
		return strings.Replace(s, `w:val="0" /><w14:checkedState`, `w:val="1" /><w14:checkedState`, 1)
	})
	l, e := Leer(b)
	if e != nil {
		t.Fatal(e)
	}
	for _, c := range l.Campos {
		if c.Codigo == "valoracion.crisis" {
			if !c.Dudoso || !slices.Contains(c.Motivos, "anexo.valor_ambiguo") {
				t.Fatal(c)
			}
			return
		}
	}
	t.Fatal("Falta crisis")
}

func TestEntradasHostiles(t *testing.T) {
	base := fixture(t, "anexo3")
	casos := map[string][]byte{
		"vacio": nil, "no_zip": []byte("PKno es un ZIP"), "grande": make([]byte, MaxBytes+1), "truncado": base[:len(base)-30],
		"dtd":     modificarDocumento(t, base, func(s string) string { return `<!DOCTYPE x [<!ENTITY fuga SYSTEM "file:///etc/passwd">]>` + s }),
		"entidad": modificarDocumento(t, base, func(s string) string { return strings.Replace(s, "Lucía", "&fuga;", 1) }),
		"profundo": modificarDocumento(t, base, func(s string) string {
			return strings.Replace(s, "Lucía", strings.Repeat("<w:x>", 65)+"Lucía"+strings.Repeat("</w:x>", 65), 1)
		}),
		"sin_tipos": cambiar(t, base, func(n string, d []byte) []byte {
			if n == "[Content_Types].xml" {
				return nil
			}
			return d
		}),
		"macros": cambiar(t, base, func(n string, d []byte) []byte {
			if n == "[Content_Types].xml" {
				return bytes.ReplaceAll(d, []byte("application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"), []byte("application/vnd.ms-word.document.macroEnabled.main+xml"))
			}
			return d
		}),
	}
	for _, nombre := range []string{"duplicado", "ruta", "bomba", "muchas_partes"} {
		var buf bytes.Buffer
		w := zip.NewWriter(&buf)
		switch nombre {
		case "duplicado":
			for i := 0; i < 2; i++ {
				r, _ := w.Create("word/document.xml")
				r.Write([]byte("x"))
			}
		case "ruta":
			r, _ := w.Create("../document.xml")
			r.Write([]byte("x"))
		case "bomba":
			r, _ := w.Create("word/document.xml")
			r.Write(bytes.Repeat([]byte("x"), MaxDescomprimido+1))
		case "muchas_partes":
			for i := 0; i < 257; i++ {
				r, _ := w.Create(string(rune(0x400+i)) + ".xml")
				r.Write([]byte("x"))
			}
		}
		w.Close()
		casos[nombre] = buf.Bytes()
	}
	for nombre, b := range casos {
		t.Run(nombre, func(t *testing.T) {
			l, e := Leer(b)
			if e == nil || len(l.Campos) != 0 || !strings.HasPrefix(e.Error(), "anexo.") {
				t.Fatal(l, e)
			}
		})
	}
}

func FuzzLeer(f *testing.F) {
	for _, nombre := range []string{"anexo3", "anexo4", "version_desconocida"} {
		b, e := os.ReadFile("testdata/" + nombre + ".docx")
		if e != nil {
			f.Fatal(e)
		}
		f.Add(b)
	}
	f.Add([]byte("PK"))
	f.Fuzz(func(t *testing.T, b []byte) {
		l, e := Leer(b)
		if e != nil && len(l.Campos) > 0 {
			t.Fatal("Extracción parcial")
		}
	})
}
