package auditoria_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
)

func exportar(t *testing.T) (auditoria.Exportacion, []byte) {
	t.Helper()
	eventos := cadena(t, 3)
	e, err := auditoria.NuevaExportacion(eventos, auditoria.Inicio(), eventos[2].Punto())
	if err != nil {
		t.Fatal(err)
	}
	b, err := e.Codificar()
	if err != nil {
		t.Fatal(err)
	}
	return e, b
}

func TestExportacionJSONCompletaParcialYVacia(t *testing.T) {
	eventos := cadena(t, 4)
	for _, tramo := range []struct {
		e    []auditoria.Evento
		a, f auditoria.Punto
	}{
		{eventos, auditoria.Inicio(), eventos[3].Punto()},
		{eventos[1:3], eventos[0].Punto(), eventos[2].Punto()},
		{nil, auditoria.Inicio(), auditoria.Inicio()},
		{nil, eventos[1].Punto(), eventos[1].Punto()},
	} {
		e, err := auditoria.NuevaExportacion(tramo.e, tramo.a, tramo.f)
		if err != nil {
			t.Fatal(err)
		}
		b, err := e.Codificar()
		if err != nil {
			t.Fatal(err)
		}
		v, err := auditoria.DecodificarExportacion(b)
		if err != nil || v.Final != tramo.f || len(v.Eventos) != len(tramo.e) {
			t.Fatalf("ida y vuelta: %v", err)
		}
		var indentado bytes.Buffer
		if err := json.Indent(&indentado, b, "", "  "); err != nil {
			t.Fatal(err)
		}
		if _, err := auditoria.DecodificarExportacion(indentado.Bytes()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestJSONRechazaAmbiguedadYContenidoAjeno(t *testing.T) {
	_, b := exportar(t)
	base := string(b)
	casos := map[string]string{
		"vacio":              "",
		"null":               "null",
		"array":              "[]",
		"truncado":           base[:len(base)-1],
		"dos_documentos":     base + base,
		"basura_final":       base + "x",
		"desconocido":        strings.Replace(base, `"version":`, `"nombre":"Inés Moreno","version":`, 1),
		"detalle_libre":      strings.Replace(base, `"accion":`, `"detalle":{"nombre":"Familia Ortega Salas"},"accion":`, 1),
		"version_repetida":   strings.Replace(base, `"version":`, `"version":"otra","version":`, 1),
		"id_repetido":        strings.Replace(base, `"id":1`, `"id":99,"id":1`, 1),
		"alias_mayusculas":   strings.Replace(base, `"id":1`, `"ID":1`, 1),
		"duplicado_escapado": strings.Replace(base, `"id":1`, `"i\u0064":99,"id":1`, 1),
		"eventos_null":       strings.Replace(base, `"eventos":[`, `"eventos":null,"ignorar":[`, 1),
		"orden":              strings.Replace(base, "id_ascendente", "fecha_ascendente", 1),
		"algoritmo":          strings.Replace(base, "sha256", "sha1", 1),
		"canonizacion":       strings.Replace(base, "etf.auditoria.binaria.v1", "otra", 1),
		"version":            strings.Replace(base, "etf.auditoria.v1", "etf.auditoria.v2", 1),
		"id_decimal":         strings.Replace(base, `"id":1`, `"id":1.0`, 1),
		"id_negativo":        strings.Replace(base, `"id":1`, `"id":-1`, 1),
		"profundidad":        strings.Repeat("[", 18) + "0" + strings.Repeat("]", 18),
	}
	for nombre, contenido := range casos {
		t.Run(nombre, func(t *testing.T) {
			if _, err := auditoria.DecodificarExportacion([]byte(contenido)); err == nil {
				t.Fatal("JSON inválido admitido")
			}
		})
	}
	// Límite antes de intentar parsear, sin contenido personal.
	if _, err := auditoria.DecodificarExportacion(make([]byte, auditoria.MaxBytesExportacion+1)); err == nil {
		t.Fatal("límite omitido")
	}
}

func TestExportacionCopiaYNoRepara(t *testing.T) {
	eventos := cadena(t, 3)
	e, err := auditoria.NuevaExportacion(eventos, auditoria.Inicio(), eventos[2].Punto())
	if err != nil {
		t.Fatal(err)
	}
	eventos[0].Perfiles[0] = "consulta"
	eventos[1].Accion = "expediente.listar"
	if err := e.Verificar(); err != nil {
		t.Fatal("alias de eventos", err)
	}
	e.Eventos[0], e.Eventos[1] = e.Eventos[1], e.Eventos[0]
	if _, err := e.Codificar(); err == nil {
		t.Fatal("exportó una cadena rota")
	}
	if _, err := auditoria.NuevaExportacion(e.Eventos, e.Anterior, e.Final); err == nil {
		t.Fatal("reordenó la entrada")
	}
	if _, err := (auditoria.Exportacion{}).Codificar(); err == nil {
		t.Fatal("exportación sin metadatos")
	}
}

func TestExportacionDetectaBorradoDeExtremos(t *testing.T) {
	e, _ := exportar(t)
	for _, eventos := range [][]auditoria.Evento{e.Eventos[1:], e.Eventos[:2], {}, {e.Eventos[1]}} {
		copia := e
		copia.Eventos = eventos
		b, err := json.Marshal(copia)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := auditoria.DecodificarExportacion(b); err == nil {
			t.Fatal("truncamiento admitido")
		}
	}
}

func TestExportacionExigeListaDePerfiles(t *testing.T) {
	eventos := cadena(t, 1)
	d := eventos[0].Datos
	d.UsuarioID = "sistema"
	d.Perfiles = nil
	eventos[0], _ = auditoria.Encadenar(1, d, auditoria.Inicio())
	e, err := auditoria.NuevaExportacion(eventos, auditoria.Inicio(), eventos[0].Punto())
	if err != nil {
		t.Fatal(err)
	}
	b, err := e.Codificar()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auditoria.DecodificarExportacion(b); err != nil {
		t.Fatal(err)
	}
	sinPerfiles := strings.Replace(string(b), `"perfiles":[],`, "", 1)
	if _, err := auditoria.DecodificarExportacion([]byte(sinPerfiles)); err == nil {
		t.Fatal("perfiles omitidos")
	}
	e.Eventos[0].Perfiles = nil
	if _, err := e.Codificar(); err == nil {
		t.Fatal("perfil null exportado")
	}
}

func FuzzDecodificarExportacion(f *testing.F) {
	d := datos()
	d.Perfiles = []string{}
	evento, err := auditoria.Encadenar(1, d, auditoria.Inicio())
	if err != nil {
		f.Fatal(err)
	}
	e, err := auditoria.NuevaExportacion([]auditoria.Evento{evento}, auditoria.Inicio(), evento.Punto())
	if err != nil {
		f.Fatal(err)
	}
	contenido, err := e.Codificar()
	if err != nil {
		f.Fatal(err)
	}
	f.Add(contenido)
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"version":"etf.auditoria.v1","eventos":[]}`))
	f.Add([]byte(`{"id":1,"id":2}`))
	f.Fuzz(func(t *testing.T, b []byte) {
		e, err := auditoria.DecodificarExportacion(b)
		if err != nil {
			return
		}
		if err := e.Verificar(); err != nil {
			t.Fatal(err)
		}
		contenido, err := e.Codificar()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := auditoria.DecodificarExportacion(contenido); err != nil {
			t.Fatal(err)
		}
	})
}
