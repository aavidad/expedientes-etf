package documento

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"os"
	"strings"
	"testing"

	word "github.com/aavidad/expedientes-etf/internal/adaptadores/docx/documento"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	ev "github.com/aavidad/expedientes-etf/internal/dominio/evaluacion"
)

func TestAnexosEscalasOpcionalesVacioParcialYVariosSujetos(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		for _, anexo := range []string{"anexo11", "anexo12"} {
			for _, parcial := range []bool{false, true} {
				t.Run(idioma+"/"+anexo+"/"+map[bool]string{false: "vacio", true: "parcial"}[parcial], func(t *testing.T) {
					ap := &d.Aceptacion{Finalizacion: map[string]string{}}
					datos := ev.DatosDocumento{}
					if parcial {
						valor1, valor2, valor0 := 7.5, 4.0, 0.0
						datos = ev.DatosDocumento{Iniciales: map[string]string{"adulto1": "L. O. S.", "adulto2": "D. P. M."}, Registro: ev.Registro{
							Definiciones: &ev.Definiciones{Escalas: []ev.Escala{{ID: "version", Instrumento: "SCORE15", Version: 1, Items: []ev.Item{{Codigo: "fortalezas", CampoDocumento: "escala_score15_1"}}}}},
							Escalas: []ev.Aplicacion{
								{ID: "a", Instrumento: "SCORE15", VersionEscalaID: "version", Version: 1, Momento: "inicial", Sujeto: ev.Sujeto{AdultoID: "adulto1"}, Puntuaciones: []ev.Puntuacion{{Item: "fortalezas", ValorNumerico: &valor1}}},
								{ID: "b", Instrumento: "SCORE15", VersionEscalaID: "version", Version: 1, Momento: "inicial", Sujeto: ev.Sujeto{AdultoID: "adulto2"}, Puntuaciones: []ev.Puntuacion{{Item: "fortalezas", ValorNumerico: &valor2}}},
								{ID: "c", Instrumento: "SCORE15", VersionEscalaID: "version", Version: 2, Momento: "final", Sujeto: ev.Sujeto{AdultoID: "adulto1"}, Puntuaciones: []ev.Puntuacion{{Item: "fortalezas", ValorNumerico: &valor0}}},
							},
						}}
					}
					decimal := map[string]string{"es": ",", "en": "."}[idioma]
					versiones := map[string]string{}
					rellenarEscalas(ap, datos, nil, map[string]string{"documento.escala.separador_decimal": decimal}, versiones)
					if parcial {
						if ap.Finalizacion["escala_score15_1_inicial"] != "L. O. S.: 7"+decimal+"5; D. P. M.: 4" || ap.Finalizacion["escala_score15_1_final"] != "L. O. S.: 0" || versiones["evaluacion_c"] != "2" {
							t.Fatal("resultados, cero o versiones perdidos", ap.Finalizacion, versiones)
						}
					}
					plantilla, err := os.ReadFile("../../../idiomas/" + idioma + "/plantillas/finalizacion/" + anexo + "/v1.0.2.docx")
					if err != nil {
						t.Fatal(err)
					}
					// Los campos ajenos a escalas se preparan a partir del contrato de revisión.
					campos := camposPlantilla(t, plantilla)
					for clave, valor := range ap.Finalizacion {
						campos[clave] = valor
					}
					generado, err := (word.Motor{}).Generar(t.Context(), plantilla, d.Relleno{Campos: campos, Filas: map[string][]map[string]string{"menores": {{"nombre": "Nerea Ortega Salas", "nacimiento": "16/05/2015", "escolarizacion": "", "centro": "", "sexo": "", "marca_h": "", "marca_m": ""}}}})
					if err != nil {
						t.Fatal(err)
					}
					contenido := textoVisibleXML(t, parteDocumento(t, generado))
					z, err := zip.NewReader(bytes.NewReader(generado), int64(len(generado)))
					if err != nil {
						t.Fatal(err)
					}
					for _, entrada := range z.File {
						if entrada.Name == "word/settings.xml" {
							r, err := entrada.Open()
							if err != nil {
								t.Fatal(err)
							}
							ajustes, err := io.ReadAll(r)
							r.Close()
							if err != nil {
								t.Fatal(err)
							}
							if bytes.Contains(ajustes, []byte("documentProtection")) {
								t.Fatal("Word protegido: casillas no editables")
							}
						}
					}
					if strings.Contains(contenido, "{{") || strings.Contains(contenido, "Sin dato") || strings.Contains(contenido, "No data") || strings.Contains(contenido, "Puntuación.") || strings.Contains(contenido, "Score.") || strings.Contains(contenido, "Elija un elemento.") {
						t.Fatal("valor de muestra o marcador en el Word")
					}
					if parcial && !strings.Contains(contenido, "L. O. S.: 7"+decimal+"5; D. P. M.: 4") {
						t.Fatal("resultado ausente del Word")
					}
				})
			}
		}
	}
}

func TestEscalaCodigoTraducidoYDescripcionSinInventarPuntuaciones(t *testing.T) {
	valor := "LEVE"
	datos := ev.DatosDocumento{Iniciales: map[string]string{"menor": "N. O. S."}, Registro: ev.Registro{
		Definiciones: &ev.Definiciones{Escalas: []ev.Escala{{ID: "version", Version: 1, Items: []ev.Item{{Codigo: "global", CampoDocumento: "escala_valorame_1"}}}}},
		Escalas: []ev.Aplicacion{{ID: "a", VersionEscalaID: "version", Instrumento: "VALORAME", Version: 1,
			Momento: "final", Sujeto: ev.Sujeto{Tipo: "menor", MenorID: "menor"}, DescripcionCualitativa: "Se afianzan las rutinas.",
			Puntuaciones: []ev.Puntuacion{{Item: "global", ValorCodigo: &valor}}}},
	}}
	a := &d.Aceptacion{Finalizacion: map[string]string{}}
	rellenarEscalas(a, datos, map[string]string{"evaluacion.respuesta.valorame.LEVE": "Leve"}, map[string]string{"fi2.momento.final": "Final"}, map[string]string{})
	if a.Finalizacion["escala_valorame_1_final"] != "N. O. S.: Leve" || a.Finalizacion["escala_valorame_descripcion"] != "N. O. S.: Final: Se afianzan las rutinas." || len(a.Finalizacion) != 2 {
		t.Fatal("traducción, descripción o campos inventados", a.Finalizacion)
	}
}

func textoVisibleXML(t *testing.T, contenido string) string {
	t.Helper()
	decoder := xml.NewDecoder(strings.NewReader(contenido))
	var resultado strings.Builder
	enTexto := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		switch token := token.(type) {
		case xml.StartElement:
			if token.Name.Local == "t" {
				enTexto = true
			}
		case xml.EndElement:
			if token.Name.Local == "t" {
				enTexto = false
			}
		case xml.CharData:
			if enTexto {
				resultado.Write(token)
			}
		}
	}
	return resultado.String()
}

func parteDocumento(t *testing.T, datos []byte) string {
	t.Helper()
	z, err := zip.NewReader(bytes.NewReader(datos), int64(len(datos)))
	if err != nil {
		t.Fatal(err)
	}
	for _, entrada := range z.File {
		if entrada.Name == "word/document.xml" {
			r, err := entrada.Open()
			if err != nil {
				t.Fatal(err)
			}
			defer r.Close()
			b, err := io.ReadAll(r)
			if err != nil {
				t.Fatal(err)
			}
			return string(b)
		}
	}
	t.Fatal("document.xml ausente")
	return ""
}

func camposPlantilla(t *testing.T, _ []byte) map[string]string {
	t.Helper()
	b, err := os.ReadFile("../../adaptadores/docx/testdata/revision_datos.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct{ Campos map[string]string }
	if err := json.Unmarshal(b, &fixture); err != nil {
		t.Fatal(err)
	}
	for clave := range fixture.Campos {
		if strings.HasPrefix(clave, "escala_") {
			delete(fixture.Campos, clave)
		}
	}
	return fixture.Campos
}
