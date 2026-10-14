package documento_test

import (
	"bytes"
	"context"
	"encoding/json"
	firmantes "github.com/aavidad/expedientes-etf/internal/adaptadores/documento/catalogo"
	word "github.com/aavidad/expedientes-etf/internal/adaptadores/docx/documento"
	ficheros "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/documento"
	h "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/documento"
	expedienteRepo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	paqueteRepo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/paquete"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/documento"
	expedienteApp "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	identidadEntrada "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

type relojSPM struct{ fecha time.Time }

func (r *relojSPM) Ahora() time.Time { return r.fecha }

type sesionesSPM struct {
	sesiones
	perfil identidad.Perfil
	equipo string
}

func (s sesionesSPM) Consultar(context.Context, string) (identidadEntrada.Acceso, error) {
	return identidadEntrada.Acceso{Estado: identidad.Activo, Usuario: identidad.Usuario{ID: usuario, Estado: identidad.Activo, Perfiles: []identidad.Perfil{s.perfil}, EquipoID: s.equipo}}, nil
}

func TestSPMAnexo5DescargaHTTPPostgres(t *testing.T) {
	if os.Getenv("ETF_DOCUMENTO_POSTGRES_INTEGRACION") != "1" {
		t.Skip("ETF_DOCUMENTO_POSTGRES_INTEGRACION=1")
	}
	ctx, cancelar := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancelar()
	b, err := postgres.Nueva(ctx, os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Cerrar()
	r := postgres.NuevoRegistroAuditoria(b)
	archivos := ficheros.Plantillas{Raiz: "../../../../idiomas", Paquetes: postgres.NuevaPublicacionPaquetes(b)}
	consulta := expedienteApp.ConsultaDocumentos{Repositorio: &expedienteRepo.Repositorio{}}
	relojCivil := &relojSPM{fecha: time.Date(2026, 10, 13, 21, 59, 0, 0, time.UTC)}
	datos := app.DatosEntrada{Expedientes: consulta, Catalogos: &catalogo.Repositorio{}, Referencias: &paqueteRepo.Referencias{}, Presentacion: archivos, Reloj: relojCivil}
	s := app.Nuevo(app.Dependencias{Repositorio: repo.Repositorio{}, Datos: datos, Firmantes: firmantes.Firmantes{Catalogo: &catalogo.Repositorio{}}, Motor: word.Motor{}, Idiomas: archivos, Ficheros: archivos, Transacciones: b, Registro: r, RegistroTransaccional: r, Reloj: relojCivil, Identificadores: ids{}})
	id := "77777777-7777-4777-8777-000000000015"
	var equipo string
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: usuario, Perfiles: []string{"coordinacion"}}, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		return tx.QueryRow(ctx, `SELECT equipo_id::text FROM expediente.expediente WHERE id=$1`, id).Scan(&equipo)
	})
	if err != nil {
		t.Fatal(err)
	}
	post := func(perfil identidad.Perfil, eq, idioma, clave, tipo string, csrf bool) *httptest.ResponseRecorder {
		handler := h.Nuevo(nil, nil, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, h.Opciones{Sesion: h.ConfiguracionSesion{Servicio: sesionesSPM{perfil: perfil, equipo: eq}, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}, Documentos: h.ConfiguracionDocumentos{Servicio: s}})
		cookie := &http.Cookie{Name: "__Host-etf-sesion", Value: "AwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwM"}
		inicio := httptest.NewRequest("GET", "https://etf.interno/api/sesion", nil)
		inicio.AddCookie(cookie)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, inicio)
		var sesion struct {
			CSRF string `json:"csrf"`
		}
		if json.Unmarshal(w.Body.Bytes(), &sesion) != nil {
			t.Fatal(w.Body.String())
		}
		cuerpo, _ := json.Marshal(d.Solicitud{Tipo: tipo, ExpedienteID: id, Idioma: idioma})
		peticion := httptest.NewRequest("POST", "https://etf.interno/api/documentos/emisiones", bytes.NewReader(cuerpo))
		peticion.AddCookie(cookie)
		peticion.Header.Set("Content-Type", "application/json")
		peticion.Header.Set("Origin", "https://etf.interno")
		if csrf {
			peticion.Header.Set("X-CSRF-Token", sesion.CSRF)
		}
		peticion.Header.Set("Idempotency-Key", clave)
		respuesta := httptest.NewRecorder()
		handler.ServeHTTP(respuesta, peticion)
		return respuesta
	}
	for _, perfil := range []identidad.Perfil{identidad.Coordinacion, identidad.ProfesionalETF} {
		t.Run(string(perfil), func(t *testing.T) {
			for _, idioma := range []string{"es", "en"} {
				relojCivil.fecha = time.Date(2026, 10, 13, 21, 59, 0, 0, time.UTC)
				respuesta := post(perfil, equipo, idioma, "anexo5-"+idioma+"-"+string(perfil), d.Anexo5, true)
				if respuesta.Code != 201 {
					t.Fatal(respuesta.Code, respuesta.Body.String())
				}
				if !strings.Contains(respuesta.Header().Get("Content-Disposition"), "attachment") {
					t.Fatal("no descarga")
				}
				xml := xmlDocumento(t, respuesta.Body.Bytes())
				if strings.Contains(xml, "{{") || !strings.Contains(xml, "Lucía") || !strings.Contains(xml, "Eva Ortega Salas") {
					t.Fatal("Word incompleto", xml[:500])
				}
				if strings.Contains(xml, "Haga clic") || strings.Contains(xml, "Seleccione fecha") || strings.Contains(xml, "Elija un elemento") {
					t.Fatal("marcadores oficiales sin sustituir")
				}
				relojCivil.fecha = time.Date(2026, 10, 13, 22, 1, 0, 0, time.UTC)
				repetida := post(perfil, equipo, idioma, "anexo5-"+idioma+"-"+string(perfil), d.Anexo5, true)
				if repetida.Code != 200 || !bytes.Equal(respuesta.Body.Bytes(), repetida.Body.Bytes()) {
					t.Fatal("repetición", repetida.Code, repetida.Body.String())
				}
			}
		})
	}

	for _, c := range []struct {
		perfil identidad.Perfil
		equipo string
		tipo   string
		csrf   bool
	}{{identidad.Consulta, equipo, d.Anexo5, true}, {identidad.Administracion, equipo, d.Anexo5, true}, {identidad.ProfesionalETF, "88888888-8888-4888-8888-888888888888", d.Anexo5, true}, {identidad.ProfesionalETF, equipo, d.ActaAceptacion, true}, {identidad.ProfesionalETF, equipo, d.Anexo5, false}} {
		respuesta := post(c.perfil, c.equipo, "es", "anexo5-denegado", c.tipo, c.csrf)
		if respuesta.Code != 403 {
			t.Fatal("permiso", c, respuesta.Code, respuesta.Body.String())
		}
	}
	// Emitir conserva el estado y la versión de la ficha; no confirma ninguna fecha.
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: usuario, Perfiles: []string{"coordinacion"}}, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		var estado string
		var version, hitos int
		if err := tx.QueryRow(ctx, `SELECT estado,version FROM expediente.expediente WHERE id=$1`, id).Scan(&estado, &version); err != nil {
			return err
		}
		if estado != "aceptada" || version != 2 {
			t.Fatal("emisión modifica expediente")
		}
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM expediente.salida_spm WHERE expediente_id=$1 AND fecha_derivacion IS NOT NULL`, id).Scan(&hitos); err != nil {
			return err
		}
		if hitos != 0 {
			t.Fatal("emisión confirma fecha")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
