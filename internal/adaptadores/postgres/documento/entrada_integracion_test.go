package documento_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	firmantes "github.com/aavidad/expedientes-etf/internal/adaptadores/documento/catalogo"
	manual "github.com/aavidad/expedientes-etf/internal/adaptadores/documento/registro_manual"
	word "github.com/aavidad/expedientes-etf/internal/adaptadores/docx/documento"
	ficheros "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/documento"
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/documento"
	expedienteRepo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	paqueteRepo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/paquete"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/documento"
	expedienteApp "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

func xmlDocumento(t *testing.T, b []byte) string {
	t.Helper()
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range z.File {
		if f.Name == "word/document.xml" {
			r, err := f.Open()
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
	t.Fatal("XML ausente")
	return ""
}

func TestEntradaPostgresPuertoRealTresTiposYRegistroHTTP(t *testing.T) {
	if os.Getenv("ETF_DOCUMENTO_POSTGRES_INTEGRACION") != "1" {
		t.Skip("ETF_DOCUMENTO_POSTGRES_INTEGRACION=1")
	}
	ctx, cancelar := context.WithTimeout(t.Context(), 70*time.Second)
	defer cancelar()
	b, err := postgres.Nueva(ctx, os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Cerrar()
	r := postgres.NuevoRegistroAuditoria(b)
	archivos := ficheros.Plantillas{Raiz: "../../../../idiomas", Paquetes: postgres.NuevaPublicacionPaquetes(b)}
	deps := app.Dependencias{Repositorio: repo.Repositorio{}, RepositorioRegistro: repo.Repositorio{}, RegistroSalida: manual.Registro{}, Datos: app.DatosEntrada{Expedientes: expedienteApp.ConsultaDocumentos{Repositorio: &expedienteRepo.Repositorio{}}, Catalogos: &catalogo.Repositorio{}, Referencias: &paqueteRepo.Referencias{}, Presentacion: archivos, Reloj: reloj{}}, Firmantes: firmantes.Firmantes{Catalogo: &catalogo.Repositorio{}}, Motor: word.Motor{}, Idiomas: archivos, Ficheros: archivos, Transacciones: b, Registro: r, RegistroTransaccional: r, Reloj: reloj{}, Identificadores: ids{}}
	s := app.Nuevo(deps)
	a := d.Actor{UsuarioID: usuario, Perfiles: []string{"coordinacion"}}
	var oficio d.Emision
	for _, origen := range []string{"anexo3", "anexo4"} {
		for _, n := range []int{1, 6, 15} {
			indice := n
			if origen == "anexo4" {
				indice += 100
			}
			id := fmt.Sprintf("77777777-7777-4777-8777-%012d", indice)
			tipos := []string{d.ActaAceptacion, d.OficioSSCC}
			if origen == "anexo4" {
				tipos[1] = d.OficioSPM
			}
			for _, tipo := range tipos {
				p := d.Solicitud{Tipo: tipo, ExpedienteID: id, Idioma: "es", Clave: fmt.Sprintf("real-%s-%d-%s", origen, n, strings.ReplaceAll(tipo, "/", "_"))}
				result, err := s.Emitir(ctx, a, p)
				if err != nil {
					t.Fatal(origen, n, tipo, err)
				}
				xml := xmlDocumento(t, result.Contenido)
				if strings.Contains(xml, "{{") || !strings.Contains(xml, "Nerea Ortega Salas") || !strings.Contains(xml, "Inés Moreno Castillo") || result.Emision.VersionesDatos["expediente"] != "2" {
					t.Fatal("contrato/plantilla", result.Emision)
				}
				if n == 15 && !strings.Contains(xml, "Eva Ortega Salas") {
					t.Fatal("menor 15 ausente")
				}
				if tipo == d.ActaAceptacion && !strings.Contains(xml, "Elena Pérez Gil") {
					t.Fatal("Otros 5 ausente")
				}
				if tipo == d.OficioSPM && !strings.Contains(xml, "Clara Serrano Rivas") {
					t.Fatal("destinatario no obtenido del catálogo")
				}
				repetida, err := s.Emitir(ctx, a, p)
				if err != nil || !repetida.Repetida || !bytes.Equal(result.Contenido, repetida.Contenido) {
					t.Fatal("repetición", err)
				}
				if d.EsOficio(tipo) {
					oficio = result.Emision
				}
			}
			incorrecto := d.OficioSPM
			if origen == "anexo4" {
				incorrecto = d.OficioSSCC
			}
			if _, err := s.Emitir(ctx, a, d.Solicitud{Tipo: incorrecto, ExpedienteID: id, Idioma: "es", Clave: fmt.Sprintf("origen-incorrecto-%d", indice)}); !errors.Is(err, d.ErrPeticion) {
				t.Fatal("oficio de otro origen admitido", err)
			}
		}
	}
	h := adaptadorHTTP.Nuevo(nil, nil, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, adaptadorHTTP.Opciones{Sesion: adaptadorHTTP.ConfiguracionSesion{Servicio: sesiones{}, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}, Documentos: adaptadorHTTP.ConfiguracionDocumentos{Servicio: s, Registros: s}})
	cookie := &http.Cookie{Name: "__Host-etf-sesion", Value: "AwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwM"}
	consulta := httptest.NewRequest("GET", "https://etf.interno/api/sesion", nil)
	consulta.AddCookie(cookie)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, consulta)
	var acceso struct {
		CSRF string `json:"csrf"`
	}
	if json.Unmarshal(w.Body.Bytes(), &acceso) != nil || acceso.CSRF == "" {
		t.Fatal("sesión", w.Code, w.Body.String())
	}
	registrar := func(clave, cuerpo, csrf string) *httptest.ResponseRecorder {
		peticion := httptest.NewRequest("POST", "https://etf.interno/api/documentos/emisiones/"+oficio.ID+"/registro-salida", strings.NewReader(cuerpo))
		peticion.AddCookie(cookie)
		peticion.Header.Set("Content-Type", "application/json")
		peticion.Header.Set("Origin", "https://etf.interno")
		peticion.Header.Set("X-CSRF-Token", csrf)
		peticion.Header.Set("Idempotency-Key", clave)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticion)
		return w
	}
	body := `{"version":1,"numero_registro":"2026/1842","fecha_registro":"2026-10-13"}`
	respuesta := registrar("registro-real", body, acceso.CSRF)
	if respuesta.Code != 200 {
		t.Fatal("registro HTTP", respuesta.Code, respuesta.Body.String())
	}
	repetida := registrar("registro-real", body, acceso.CSRF)
	if repetida.Code != 200 || !bytes.Equal(respuesta.Body.Bytes(), repetida.Body.Bytes()) {
		t.Fatal("repetición registro distinta", repetida.Code)
	}
	if w := registrar("registro-real", strings.Replace(body, "1842", "1843", 1), acceso.CSRF); w.Code != 422 {
		t.Fatal("misma clave alterada", w.Code)
	}
	if w := registrar("registro-conflicto", strings.Replace(body, "1842", "1843", 1), acceso.CSRF); w.Code != 409 {
		t.Fatal("versión ignorada", w.Code)
	}
	if w := registrar("registro-csrf", body, ""); w.Code != 403 {
		t.Fatal("CSRF ignorado", w.Code)
	}
	// No se entrega una anotación si falla la auditoría: se revierte también la clave.
	fallidas := deps
	fallidas.RegistroTransaccional = registroFallido{}
	p := d.SolicitudRegistroSalida{EmisionID: oficio.ID, Clave: "registro-fallo", Version: 2, Numero: "2026/1843", Fecha: "2026-10-13"}
	if _, err := app.Nuevo(fallidas).AnotarRegistroSalida(ctx, a, p); !errors.Is(err, d.ErrNoDisponible) {
		t.Fatal("registro sin auditoría", err)
	}
	identidad := plataforma.Identidad{UsuarioID: usuario, Perfiles: a.Perfiles}
	if err := b.Ejecutar(ctx, identidad, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		var version int
		var numero string
		if err := tx.QueryRow(ctx, `SELECT version,numero_registro FROM documento.registro_salida WHERE emision_id=$1`, oficio.ID).Scan(&version, &numero); err != nil {
			return err
		}
		if version != 2 || numero != "2026/1842" {
			t.Fatal("registro parcial")
		}
		var n int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM documento.registro_peticion WHERE clave='registro-fallo'`).Scan(&n); err != nil {
			return err
		}
		if n != 0 {
			t.Fatal("clave sin registro")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, perfil := range []string{"administracion", "consulta", "equipo"} {
		if err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: usuario, Perfiles: []string{perfil}}, func(ctx context.Context) error {
			tx, _ := postgres.DesdeContexto(ctx)
			var n int
			if err := tx.QueryRow(ctx, `SELECT count(*) FROM documento.registro_salida`).Scan(&n); err != nil {
				return err
			}
			if n != 0 {
				t.Fatal("RLS de registro", perfil, n)
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	t.Log("Puerto expediente en misma transacción: 12 emisiones, 1/6/15 menores, catálogo real; registro HTTP 200/422/409/403; repetición exacta, RLS y auditoría atómica")
}

func TestFirmanteCatalogoCambiadoPostgres(t *testing.T) {
	if os.Getenv("ETF_DOCUMENTO_FIRMANTE_CAMBIADO") != "1" {
		t.Skip("ETF_DOCUMENTO_FIRMANTE_CAMBIADO=1")
	}
	b, err := postgres.Nueva(t.Context(), os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Cerrar()
	r := postgres.NuevoRegistroAuditoria(b)
	archivos := ficheros.Plantillas{Raiz: "../../../../idiomas", Paquetes: postgres.NuevaPublicacionPaquetes(b)}
	s := app.Nuevo(app.Dependencias{Repositorio: repo.Repositorio{}, Datos: app.DatosEntrada{Expedientes: expedienteApp.ConsultaDocumentos{Repositorio: &expedienteRepo.Repositorio{}}, Catalogos: &catalogo.Repositorio{}, Referencias: &paqueteRepo.Referencias{}, Presentacion: archivos, Reloj: reloj{}}, Firmantes: firmantes.Firmantes{Catalogo: &catalogo.Repositorio{}}, Motor: word.Motor{}, Idiomas: archivos, Ficheros: archivos, Transacciones: b, Registro: r, RegistroTransaccional: r, Reloj: reloj{}, Identificadores: ids{}})
	a := d.Actor{UsuarioID: usuario, Perfiles: []string{"coordinacion"}}
	p := d.Solicitud{Tipo: d.ActaAceptacion, ExpedienteID: "77777777-7777-4777-8777-000000000001", Idioma: "es", Clave: "real-anexo3-1-entrada_aceptacion"}
	if _, err := s.Emitir(t.Context(), a, p); !errors.Is(err, d.ErrIdempotencia) {
		t.Fatal("clave antigua permitió firmante nuevo", err)
	}
	p.Clave = "catalogo-firmante-nuevo"
	nueva, err := s.Emitir(t.Context(), a, p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(xmlDocumento(t, nueva.Contenido), "Elena Núñez Romero") || strings.Contains(xmlDocumento(t, nueva.Contenido), "Inés Moreno Castillo") || nueva.Emision.Aviso != "documento.regenerado" {
		t.Fatal("catálogo actualizado no reflejado")
	}
	t.Log("Firmante de catálogo v2 en DOCX nuevo; misma clave antigua rechazada; nueva emisión con aviso")
}
