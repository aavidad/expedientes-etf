package ficheros

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"

	dominio "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
)

type parteZIP struct {
	nombre string
	datos  []byte
	modo   os.FileMode
}

func zipPartes(t *testing.T, partes []parteZIP) []byte {
	t.Helper()
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for _, parte := range partes {
		cabecera := zip.FileHeader{Name: parte.nombre, Method: zip.Deflate}
		if parte.modo != 0 {
			cabecera.SetMode(parte.modo)
		}
		f, err := w.CreateHeader(&cabecera)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(parte.datos); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func partesTema(t *testing.T) []parteZIP {
	t.Helper()
	var partes []parteZIP
	for _, nombre := range []string{"tema.json", "tokens.json", "fuentes/public-sans.woff2", "fuentes/source-serif-4.woff2"} {
		b, err := os.ReadFile("../../../temas/expedientes-etf/" + nombre)
		if err != nil {
			t.Fatal(err)
		}
		partes = append(partes, parteZIP{nombre: nombre, datos: b})
	}
	return partes
}

func validador() *Paquetes { return NuevosPaquetes("../../../idiomas", "../../../temas") }

func TestZIPTemaValido(t *testing.T) {
	contenido, err := validador().Validar(context.Background(), dominio.TipoTema, zipPartes(t, partesTema(t)))
	if err != nil || contenido.Codigo != "expedientes-etf" || !dominio.VersionValida(contenido.Version) || len(contenido.Huella) != 64 {
		t.Fatalf("contenido=%+v error=%v", contenido, err)
	}
}

func TestZIPRutasTiposEnlacesDuplicados(t *testing.T) {
	for _, parte := range []parteZIP{
		{nombre: "../tokens.json", datos: []byte(`{}`)},
		{nombre: "/tokens.json", datos: []byte(`{}`)},
		{nombre: `C:\tokens.json`, datos: []byte(`{}`)},
		{nombre: "iconos/../tokens.json", datos: []byte(`{}`)},
		{nombre: "icono.svg", datos: []byte(`<svg/>`)},
		{nombre: "estilo.css", datos: []byte(`body{}`)},
		{nombre: "tokens.json", datos: []byte(`{}`)},
		{nombre: "icono.png", datos: []byte(`<svg/>`)},
		{nombre: "iconos/flecha.png", datos: []byte("tokens.json"), modo: os.ModeSymlink | 0600},
		{nombre: "anidado.zip", datos: []byte("PK")},
	} {
		t.Run(parte.nombre, func(t *testing.T) {
			partes := append(partesTema(t), parte)
			if _, err := validador().Validar(context.Background(), dominio.TipoTema, zipPartes(t, partes)); !errors.Is(err, dominio.ErrPaqueteInvalido) {
				t.Fatalf("entrada aceptada: %v", err)
			}
		})
	}
}

func TestZIPLimites(t *testing.T) {
	t.Run("entrada", func(t *testing.T) {
		if _, err := validador().Validar(context.Background(), dominio.TipoTema, make([]byte, (20<<20)+1)); !errors.Is(err, dominio.ErrPaqueteInvalido) {
			t.Fatal(err)
		}
	})
	t.Run("1001_entradas", func(t *testing.T) {
		partes := partesTema(t)
		for i := len(partes); i < 1001; i++ {
			partes = append(partes, parteZIP{nombre: "iconos/" + string(rune(0x100+i)) + ".png"})
		}
		if _, err := validador().Validar(context.Background(), dominio.TipoTema, zipPartes(t, partes)); !errors.Is(err, dominio.ErrPaqueteInvalido) {
			t.Fatal(err)
		}
	})
	t.Run("bomba_descompresion", func(t *testing.T) {
		var b bytes.Buffer
		w := zip.NewWriter(&b)
		f, err := w.Create("tokens.json")
		if err != nil {
			t.Fatal(err)
		}
		bloque := bytes.Repeat([]byte("x"), 1<<20)
		for i := 0; i < 101; i++ {
			if _, err := f.Write(bloque); err != nil {
				t.Fatal(err)
			}
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		if b.Len() >= 20<<20 {
			t.Fatal("la prueba no está comprimida")
		}
		if _, err := validador().Validar(context.Background(), dominio.TipoTema, b.Bytes()); !errors.Is(err, dominio.ErrPaqueteInvalido) {
			t.Fatal(err)
		}
	})
}

func TestZIPIdiomaYPlantilla(t *testing.T) {
	docx, err := os.ReadFile("../../../idiomas/es/plantillas/entrada/aceptacion/v1.0.0.docx")
	if err != nil {
		t.Fatal(err)
	}
	partes := []parteZIP{
		{nombre: "idioma.json", datos: []byte(`{"codigo":"fr","nombre":"Français","version":"1.0.0"}`)},
		{nombre: "textos/comun.json", datos: []byte(`{"comun.titulo":"Dossiers familiaux"}`)},
		{nombre: "catalogos/listas.json", datos: []byte(`{"estados":[]}`)},
		{nombre: "plantillas/acta.docx", datos: docx},
	}
	contenido, err := validador().Validar(context.Background(), dominio.TipoIdioma, zipPartes(t, partes))
	if err != nil || contenido.Codigo != "fr" {
		t.Fatalf("%v %v", contenido.Codigo, err)
	}
	for _, datos := range [][]byte{[]byte("no es DOCX"), zipPartes(t, []parteZIP{{nombre: "word/vbaProject.bin", datos: []byte("macros")}})} {
		partes[len(partes)-1].datos = datos
		if _, err := validador().Validar(context.Background(), dominio.TipoIdioma, zipPartes(t, partes)); !errors.Is(err, dominio.ErrPaqueteInvalido) {
			t.Fatal(err)
		}
	}
}

func TestZIPNoEscribeEnDiscoYCancelacion(t *testing.T) {
	base := t.TempDir()
	p := NuevosPaquetes(base, "../../../temas")
	if _, err := p.Validar(context.Background(), dominio.TipoTema, zipPartes(t, partesTema(t))); err != nil {
		t.Fatal(err)
	}
	d, err := os.Open(base)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	nombres, err := d.Readdirnames(1)
	if err != io.EOF || len(nombres) != 0 {
		t.Fatalf("escribió archivos: %v %v", nombres, err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := p.Validar(ctx, dominio.TipoTema, zipPartes(t, partesTema(t))); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestDOCXRechazaMacrosDTDYEstructura(t *testing.T) {
	base := []parteZIP{
		{nombre: "[Content_Types].xml", datos: []byte(`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`)},
		{nombre: "_rels/.rels", datos: []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`)},
		{nombre: "word/document.xml", datos: []byte(`<document xmlns="http://schemas.openxmlformats.org/wordprocessingml/2006/main"/>`)},
	}
	if err := validarDOCX(context.Background(), zipPartes(t, base), &presupuestoZIP{}); err != nil {
		t.Fatal("DOCX mínimo válido", err)
	}
	falso := []parteZIP{{nombre: "[Content_Types].xml", datos: []byte(`<Types/>`)}, {nombre: "_rels/.rels", datos: []byte(`<Relationships/>`)}, {nombre: "word/document.xml", datos: []byte(`<document/>`)}}
	if err := validarDOCX(context.Background(), zipPartes(t, falso), &presupuestoZIP{}); !errors.Is(err, dominio.ErrPaqueteInvalido) {
		t.Fatal("DOCX falso aceptado", err)
	}
	nombresFalsos := append([]parteZIP{}, base...)
	nombresFalsos[0].datos = []byte(`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types" xmlns:x="urn:falso"><Override x:PartName="/word/document.xml" x:ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`)
	nombresFalsos[1].datos = []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships" xmlns:x="urn:falso"><Relationship x:Id="rId1" x:Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" x:Target="word/document.xml"/></Relationships>`)
	if err := validarDOCX(context.Background(), zipPartes(t, nombresFalsos), &presupuestoZIP{}); !errors.Is(err, dominio.ErrPaqueteInvalido) {
		t.Fatal("atributos falsos aceptados", err)
	}
	for _, caso := range []struct {
		nombre string
		extra  parteZIP
	}{
		{"macros", parteZIP{nombre: "word/vbaProject.bin", datos: []byte("macros")}},
		{"DTD", parteZIP{nombre: "word/styles.xml", datos: []byte(`<!DOCTYPE styles><styles/>`)}},
		{"relacion_externa", parteZIP{nombre: "word/_rels/document.xml.rels", datos: []byte(`<Relationships><Relationship TargetMode="External" Target="https://invalid.test"/></Relationships>`)}},
		{"SVG", parteZIP{nombre: "word/media/imagen.svg", datos: []byte(`<svg/>`)}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			datos := zipPartes(t, append(append([]parteZIP{}, base...), caso.extra))
			if err := validarDOCX(context.Background(), datos, &presupuestoZIP{}); !errors.Is(err, dominio.ErrPaqueteInvalido) {
				t.Fatal(err)
			}
		})
	}
	for _, presupuesto := range []*presupuestoZIP{{bytes: 100 << 20}, {entradas: 999}} {
		if err := validarDOCX(context.Background(), zipPartes(t, base), presupuesto); !errors.Is(err, dominio.ErrPaqueteInvalido) {
			t.Fatal("reinició el presupuesto ZIP", err)
		}
	}
}
