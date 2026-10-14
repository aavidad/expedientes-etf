package docx

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

const especiales = ` < & " ' áéíóú ñ 😀 > `

func acta(t *testing.T) ([]byte, Datos) {
	t.Helper()
	b, err := os.ReadFile("../../../idiomas/es/plantillas/entrada/aceptacion/v1.0.0.docx")
	if err != nil {
		t.Fatal(err)
	}
	datos := Datos{Campos: map[string]string{
		"numero": "04-01-26", "equipo": "ETF - 4 Valle Lecrín-Ogíjares / Sierra Nevada", "centro": "Valle Lecrín-Ogíjares", "municipio": "Dúrcal", "descripcion": especiales, "familia": "Ortega Salas", "madre": "Lucía Ortega Salas", "madre_nacimiento": "14/02/1987", "padre": "", "padre_nacimiento": "", "profesional": "Pablo Ruiz Vílchez", "servicio_derivante": "Servicios Sociales Comunitarios", "firmante": "Inés Moreno Castillo",
	}, Filas: map[string][]map[string]string{"menores": {}}}
	for i := 1; i <= 5; i++ {
		for _, sufijo := range []string{"", "_parentesco", "_nacimiento"} {
			datos.Campos[fmt.Sprintf("adulto%d%s", i, sufijo)] = ""
		}
	}
	nombres := []string{"Nerea", "Hugo", "Vera", "Leo", "Alma", "Gael", "Celia", "Bruno", "Inés", "Darío", "Mara", "Iván", "Aitana", "Nicolás", "Irene"}
	for i, nombre := range nombres {
		datos.Filas["menores"] = append(datos.Filas["menores"], map[string]string{"nombre": nombre + " Ortega Salas", "sexo": "Femenino", "nacimiento": fmt.Sprintf("%02d/05/2016", i+1)})
	}
	return b, datos
}

func partes(t *testing.T, b []byte) map[string][]byte {
	t.Helper()
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	resultado := make(map[string][]byte)
	for _, f := range z.File {
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		v, err := io.ReadAll(r)
		cerrar := r.Close()
		if err != nil || cerrar != nil {
			t.Fatalf("%v; %v", err, cerrar)
		}
		resultado[f.Name] = v
	}
	return resultado
}

func cambiarXML(t *testing.T, plantilla []byte, transformar func([]byte) []byte) []byte {
	t.Helper()
	z, err := zip.NewReader(bytes.NewReader(plantilla), int64(len(plantilla)))
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for _, f := range z.File {
		if f.Name != "word/document.xml" {
			if err := w.Copy(f); err != nil {
				t.Fatal(err)
			}
			continue
		}
		h := f.FileHeader
		r, err := w.CreateHeader(&h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := r.Write(transformar(partes(t, plantilla)[f.Name])); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func arbol(t *testing.T, b []byte) *nodo {
	t.Helper()
	r, err := leerXML(partes(t, b)["word/document.xml"], nil)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func comprobarPartes(t *testing.T, antes, despues []byte) {
	t.Helper()
	a, b := partes(t, antes), partes(t, despues)
	if len(a) != len(b) {
		t.Fatal("cambió el número de partes")
	}
	for nombre, valor := range a {
		if nombre != "word/document.xml" && !bytes.Equal(valor, b[nombre]) {
			t.Fatalf("se alteró %s", nombre)
		}
	}
}

func TestActaEspecialesQuinceMenoresYVacio(t *testing.T) {
	plantilla, datos := acta(t)
	antes := bytes.Clone(plantilla)
	salida, err := Generar(plantilla, datos)
	if err != nil {
		t.Fatal(err)
	}
	r := arbol(t, salida)
	texto := textoNodo(r)
	for _, esperado := range []string{especiales, "Nº. Expt. E.T.F.:                               04-01-26", "Madre. Lucía Ortega Salas", "Padre. ", "Fdo. Inés Moreno Castillo", "ACTA DE ACEPTACIÓN"} {
		if !strings.Contains(texto, esperado) {
			t.Fatalf("no aparece %q", esperado)
		}
	}
	if strings.Contains(texto, "{{") {
		t.Fatal("quedan marcadores")
	}
	filas := r.buscar("tr")
	if len(filas) != 15 {
		t.Fatalf("filas: %d", len(filas))
	}
	for i, menor := range datos.Filas["menores"] {
		if !strings.Contains(textoNodo(filas[i]), menor["nombre"]) || !strings.Contains(textoNodo(filas[i]), menor["nacimiento"]) {
			t.Fatalf("menor %d perdido o desordenado", i+1)
		}
	}
	if !bytes.Equal(antes, plantilla) {
		t.Fatal("se modificó la entrada")
	}
	comprobarPartes(t, plantilla, salida)
	segunda, err := Generar(plantilla, datos)
	if err != nil || !bytes.Equal(salida, segunda) {
		t.Fatalf("generación no determinista: %v", err)
	}
}

func TestMarcadoresPartidosEntreRuns(t *testing.T) {
	plantilla, datos := acta(t)
	plantilla = cambiarXML(t, plantilla, func(b []byte) []byte {
		return bytes.Replace(b, []byte("{{numero}}"), []byte(`{{nu</w:t></w:r><w:r><w:rPr><w:b/></w:rPr><w:t>me</w:t></w:r><w:r><w:t>ro}}`), 1)
	})
	datos.Campos["numero"] = especiales + " {{no_reinterpretar}} "
	salida, err := Generar(plantilla, datos)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(textoNodo(arbol(t, salida)), datos.Campos["numero"]) {
		t.Fatal("marcador partido perdido")
	}
	if !bytes.Contains(partes(t, salida)["word/document.xml"], []byte("<w:b")) {
		t.Fatal("se perdieron estilos")
	}
}

func TestErroresClarosSinResultadoParcial(t *testing.T) {
	base, datos := acta(t)
	casos := []struct {
		nombre, clave, campo string
		preparar             func([]byte, Datos) ([]byte, Datos)
	}{
		{"desconocido", "docx.marcador_desconocido", "desconocido", func(b []byte, d Datos) ([]byte, Datos) {
			return cambiarXML(t, b, func(x []byte) []byte { return bytes.Replace(x, []byte("{{numero}}"), []byte("{{desconocido}}"), 1) }), d
		}},
		{"bloque", "docx.bloque_invalido", "menores", func(b []byte, d Datos) ([]byte, Datos) {
			return cambiarXML(t, b, func(x []byte) []byte { return bytes.Replace(x, []byte("{{/menores}}"), nil, 1) }), d
		}},
		{"dieciseis", "docx.limite_filas", "", func(b []byte, d Datos) ([]byte, Datos) {
			d.Filas["menores"] = append(d.Filas["menores"], d.Filas["menores"][0])
			return b, d
		}},
		{"caracter_prohibido", "docx.valor_xml_invalido", "", func(b []byte, d Datos) ([]byte, Datos) { d.Campos["descripcion"] = "Lucía\x00"; return b, d }},
		{"utf8", "docx.valor_xml_invalido", "", func(b []byte, d Datos) ([]byte, Datos) { d.Campos["descripcion"] = string([]byte{0xff}); return b, d }},
		{"zip", "docx.zip_invalido", "", func(_ []byte, d Datos) ([]byte, Datos) { return []byte("PK"), d }},
		{"xml", "docx.xml_invalido", "", func(b []byte, d Datos) ([]byte, Datos) {
			return cambiarXML(t, b, func(_ []byte) []byte { return []byte(`<w:document xmlns:w="` + espacioWord + `"><w:p></w:document>`) }), d
		}},
		{"dtd", "docx.xml_directiva_no_admitida", "", func(b []byte, d Datos) ([]byte, Datos) {
			return cambiarXML(t, b, func(x []byte) []byte { return append([]byte("<!DOCTYPE x>"), x...) }), d
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			_, datos = acta(t)
			b, d := caso.preparar(base, datos)
			salida, err := Generar(b, d)
			var fallo *Error
			if !errors.As(err, &fallo) || fallo.Clave != caso.clave || fallo.Campo != caso.campo || salida != nil {
				t.Fatalf("salida %d, error %#v", len(salida), err)
			}
			if strings.Contains(err.Error(), "Lucía") {
				t.Fatal("el error contiene un dato personal")
			}
		})
	}
}

func anexo(t *testing.T) ([]byte, Datos) {
	t.Helper()
	b, err := os.ReadFile("../anexos/docx/testdata/anexo3.docx")
	if err != nil {
		t.Fatal(err)
	}
	datos := Datos{Controles: map[string]string{}}
	for _, c := range arbol(t, b).buscar("sdt") {
		pr := c.hijo("sdtPr")
		if pr.hijo("picture") != nil {
			continue
		}
		id := pr.hijo("id").atributo("val")
		valor := "Lucía Ortega Salas" + especiales
		if pr.hijo("date") != nil {
			valor = "13/10/2026"
		}
		if lista := pr.hijo("dropDownList"); lista != nil {
			items := lista.buscar("listItem")
			valor = items[1].atributo("displayText")
		}
		datos.Controles[id] = valor
	}
	datos.Controles["100000"] = "04-01-26"
	datos.Controles["100003"] = ""
	datos.Controles["100004"] = especiales
	return b, datos
}

func TestAnexoControlesEditables(t *testing.T) {
	b, datos := anexo(t)
	salida, err := Generar(b, datos)
	if err != nil {
		t.Fatal(err)
	}
	antes, despues := arbol(t, b).buscar("sdt"), arbol(t, salida).buscar("sdt")
	if len(antes) != 131 || len(despues) != len(antes) {
		t.Fatalf("controles: %d -> %d", len(antes), len(despues))
	}
	for i, c := range despues {
		pr := c.hijo("sdtPr")
		id := pr.hijo("id").atributo("val")
		if pr.hijo("picture") != nil {
			if textoNodo(c) != textoNodo(antes[i]) {
				t.Fatal("se alteró el control de imagen")
			}
			continue
		}
		if textoNodo(c.hijo("sdtContent")) != datos.Controles[id] {
			t.Fatalf("contenido incorrecto del control %s", id)
		}
		if len(pr.buscar("lock")) != 0 || len(pr.buscar("showingPlcHdr")) != 0 {
			t.Fatalf("control %s bloqueado o marcado como vacío", id)
		}
		for _, tipo := range []string{"id", "text", "date", "dropDownList", "alias"} {
			a, d := antes[i].hijo("sdtPr").hijo(tipo), pr.hijo(tipo)
			if (a == nil) != (d == nil) || a != nil && !reflect.DeepEqual(a.atributos, d.atributos) {
				t.Fatalf("propiedades %s alteradas en %s", tipo, id)
			}
		}
		if len(pr.buscar("listItem")) != len(antes[i].hijo("sdtPr").buscar("listItem")) {
			t.Fatalf("opciones perdidas: %s", id)
		}
	}
	comprobarPartes(t, b, salida)
	// Una segunda edición reemplaza el contenido y conserva el contenedor.
	datos.Controles["100000"] = "04-02-26"
	edicion, err := Generar(salida, datos)
	if err != nil || !strings.Contains(textoNodo(arbol(t, edicion)), "04-02-26") {
		t.Fatalf("segunda edición: %v", err)
	}
	igual, err := Generar(b, datos)
	igual2, err2 := Generar(b, datos)
	if err != nil || err2 != nil || !bytes.Equal(igual, igual2) {
		t.Fatal("anexo no determinista")
	}
}

func TestControlDesconocidoYVinculado(t *testing.T) {
	b, datos := anexo(t)
	datos.Controles["99999999"] = "dato privado"
	if salida, err := Generar(b, datos); salida != nil || err == nil || err.Error() != "docx.control_desconocido" {
		t.Fatalf("%d, %v", len(salida), err)
	}
	delete(datos.Controles, "99999999")
	b = cambiarXML(t, b, func(x []byte) []byte {
		return bytes.Replace(x, []byte("<w:sdtPr>"), []byte(`<w:sdtPr><w:dataBinding w:xpath="/dato"/>`), 1)
	})
	if salida, err := Generar(b, datos); salida != nil || err == nil || err.Error() != "docx.control_vinculado_no_admitido" {
		t.Fatalf("%d, %v", len(salida), err)
	}
}

func TestLimitesDelPaqueteYExpansion(t *testing.T) {
	crear := func(nombres []string, declarado uint64) []byte {
		var b bytes.Buffer
		w := zip.NewWriter(&b)
		for _, nombre := range nombres {
			_, err := w.CreateRaw(&zip.FileHeader{Name: nombre, Method: zip.Store, UncompressedSize64: declarado})
			if err != nil {
				t.Fatal(err)
			}
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		return b.Bytes()
	}
	entradas := make([]string, maxEntradas+1)
	for i := range entradas {
		entradas[i] = fmt.Sprintf("parte%d", i)
	}
	for _, caso := range []struct {
		nombre, clave string
		b             []byte
	}{
		{"descompresion", "docx.limite_descomprimido", crear([]string{"word/document.xml"}, maxDescomprimido+1)},
		{"entradas", "docx.limite_entradas", crear(entradas, 0)},
		{"duplicada", "docx.parte_duplicada", crear([]string{"parte", "parte"}, 0)},
		{"macros", "docx.macros_no_admitidas", crear([]string{"word/vbaProject.bin"}, 0)},
		{"estructura", "docx.estructura_invalida", crear([]string{"parte"}, 0)},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			salida, err := Generar(caso.b, Datos{})
			if salida != nil || err == nil || err.Error() != caso.clave {
				t.Fatalf("%d; %v", len(salida), err)
			}
		})
	}
	b, datos := acta(t)
	b = cambiarXML(t, b, func(x []byte) []byte {
		return bytes.Replace(x, []byte("{{numero}}"), []byte("{{numero}}{{numero}}"), 1)
	})
	datos.Campos["numero"] = strings.Repeat("&", 9<<20)
	if salida, err := Generar(b, datos); salida != nil || err == nil || err.Error() != "docx.limite_resultado" {
		t.Fatalf("expansión: %d; %v", len(salida), err)
	}
}

func TestControlDesbloqueadoYMarcadorMalFormado(t *testing.T) {
	b, datos := anexo(t)
	b = cambiarXML(t, b, func(x []byte) []byte {
		return bytes.Replace(x, []byte("<w:sdtPr>"), []byte(`<w:sdtPr><w:lock w:val="sdtContentLocked"/><w:showingPlcHdr/>`), 1)
	})
	salida, err := Generar(b, datos)
	if err != nil {
		t.Fatal(err)
	}
	pr := arbol(t, salida).buscar("sdt")[0].hijo("sdtPr")
	if pr.hijo("lock") != nil || pr.hijo("showingPlcHdr") != nil {
		t.Fatal("el control sigue bloqueado")
	}
	b, datos = acta(t)
	b = cambiarXML(t, b, func(x []byte) []byte { return bytes.Replace(x, []byte("{{numero}}"), []byte("{{numero"), 1) })
	if salida, err := Generar(b, datos); salida != nil || err == nil || err.Error() != "docx.marcador_invalido" {
		t.Fatalf("%d; %v", len(salida), err)
	}
}

func TestLibreOfficeLocal(t *testing.T) {
	if os.Getenv("ETF_DOCX_PROBAR_LIBREOFFICE") != "1" {
		t.Skip("activar ETF_DOCX_PROBAR_LIBREOFFICE=1 para la comprobación local")
	}
	actaBytes, actaDatos := acta(t)
	anexoBytes, anexoDatos := anexo(t)
	dir := t.TempDir()
	for _, caso := range []struct {
		nombre string
		b      []byte
		datos  Datos
	}{{"acta-15-menores", actaBytes, actaDatos}, {"anexo-3-editable", anexoBytes, anexoDatos}} {
		salida, err := Generar(caso.b, caso.datos)
		if err != nil {
			t.Fatal(err)
		}
		ruta := filepath.Join(dir, caso.nombre+".docx")
		if err = os.WriteFile(ruta, salida, 0600); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		cmd := exec.CommandContext(ctx, "env", "-u", "LD_LIBRARY_PATH", "soffice", "-env:UserInstallation=file://"+filepath.Join(dir, "perfil"), "--headless", "--convert-to", "pdf", "--outdir", dir, ruta)
		log, err := cmd.CombinedOutput()
		cancel()
		t.Logf("%s\n%s", strings.Join(cmd.Args, " "), log)
		if err != nil {
			t.Fatal(err)
		}
		pdf, err := os.ReadFile(filepath.Join(dir, caso.nombre+".pdf"))
		if err != nil || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
			t.Fatalf("PDF inválido: %v", err)
		}
		if _, err := exec.LookPath("pdftotext"); err == nil {
			texto, err := exec.Command("pdftotext", "-layout", filepath.Join(dir, caso.nombre+".pdf"), "-").CombinedOutput()
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(texto, []byte("04-01-26")) {
				t.Fatal("el PDF no contiene el número")
			}
			if caso.nombre == "acta-15-menores" && !bytes.Contains(texto, []byte("Irene Ortega Salas")) {
				t.Fatal("no se renderizó el menor 15")
			}
			t.Logf("PDF %s: %d bytes; número y contenido comprobados", caso.nombre, len(pdf))
		}
	}
}
