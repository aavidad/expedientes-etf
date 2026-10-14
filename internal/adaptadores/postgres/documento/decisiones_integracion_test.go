package documento_test

import (
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
	expedienteEntrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

func TestDecisionesDocumentosPostgresHTTPPuertoReal(t *testing.T) {
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
	datos := app.DatosEntrada{Expedientes: consulta, Catalogos: &catalogo.Repositorio{}, Referencias: &paqueteRepo.Referencias{}, Presentacion: archivos, Reloj: reloj{}}
	s := app.Nuevo(app.Dependencias{Repositorio: repo.Repositorio{}, RepositorioRegistro: repo.Repositorio{}, RegistroSalida: manual.Registro{}, Datos: datos, Firmantes: firmantes.Firmantes{Catalogo: &catalogo.Repositorio{}}, Motor: word.Motor{}, Idiomas: archivos, Ficheros: archivos, Transacciones: b, Registro: r, RegistroTransaccional: r, Reloj: reloj{}, Identificadores: ids{}})
	a := d.Actor{UsuarioID: usuario, Perfiles: []string{"coordinacion"}}
	ea := expedienteEntrada.Actor{UsuarioID: usuario, Perfiles: a.Perfiles}
	if _, err := consulta.ParaDocumentos(ctx, ea, "88888888-7777-4777-8777-000000000201"); !errors.Is(err, expedienteEntrada.ErrNoDisponible) {
		t.Fatal("lectura sin transacción", err)
	}
	h := adaptadorHTTP.Nuevo(nil, nil, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, adaptadorHTTP.Opciones{Sesion: adaptadorHTTP.ConfiguracionSesion{Servicio: sesiones{}, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}, Documentos: adaptadorHTTP.ConfiguracionDocumentos{Servicio: s, Registros: s}})
	cookie := &http.Cookie{Name: "__Host-etf-sesion", Value: "AwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwM"}
	inicio := httptest.NewRequest("GET", "https://etf.interno/api/sesion", nil)
	inicio.AddCookie(cookie)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, inicio)
	var sesion struct {
		CSRF string `json:"csrf"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &sesion); err != nil || sesion.CSRF == "" {
		t.Fatal("sesión", err)
	}
	post := func(tipo, id, idioma, clave string) *httptest.ResponseRecorder {
		cuerpo, _ := json.Marshal(d.Solicitud{Tipo: tipo, ExpedienteID: id, Idioma: idioma})
		peticion := httptest.NewRequest("POST", "https://etf.interno/api/documentos/emisiones", bytes.NewReader(cuerpo))
		peticion.AddCookie(cookie)
		peticion.Header.Set("Content-Type", "application/json")
		peticion.Header.Set("Origin", "https://etf.interno")
		peticion.Header.Set("X-CSRF-Token", sesion.CSRF)
		peticion.Header.Set("Idempotency-Key", clave)
		respuesta := httptest.NewRecorder()
		h.ServeHTTP(respuesta, peticion)
		return respuesta
	}
	for _, tipo := range []string{d.ActaNoAceptacion, d.OficioListaEspera} {
		for _, n := range []int{1, 6, 15} {
			origenes := []string{"anexo3"}
			if tipo == d.ActaNoAceptacion {
				origenes = append(origenes, "anexo4")
			}
			for _, origen := range origenes {
				base, prefijo, estado := 200, "88888888", "no_aceptada"
				if tipo == d.OficioListaEspera {
					base, prefijo, estado = 300, "99999999", "lista_espera"
				}
				if origen == "anexo4" {
					base += 100
				}
				id := fmt.Sprintf("%s-7777-4777-8777-%012d", prefijo, base+n)
				for _, idioma := range []string{"es", "en"} {
					t.Run(fmt.Sprintf("%s/%s/%d/%s", tipo, origen, n, idioma), func(t *testing.T) {
						clave := fmt.Sprintf("decision-%s-%d-%s", prefijo, base+n, idioma)
						respuesta := post(tipo, id, idioma, clave)
						if respuesta.Code != 201 {
							t.Fatal(respuesta.Code, respuesta.Body.String())
						}
						xml := xmlDocumento(t, respuesta.Body.Bytes())
						if strings.Contains(xml, "{{") || !strings.Contains(xml, "Inés Moreno Castillo") {
							t.Fatal("relleno incompleto")
						}
						nombres := []string{"Nerea", "Álvaro", "Celia", "Mateo", "Irene", "Hugo", "Paula", "Bruno", "Aitana", "Leo", "Claudia", "Mario", "Vera", "Lucas", "Eva"}
						for _, nombre := range nombres[:n] {
							if strings.Count(xml, nombre+" Ortega Salas") != 1 {
								t.Fatal("menor omitido o duplicado", nombre)
							}
						}

						if tipo == d.ActaNoAceptacion && !strings.Contains(xml, "La familia recibe atención especializada de otro recurso.") {
							t.Fatal("motivo ausente")
						}
						if tipo == d.OficioListaEspera && !strings.Contains(xml, "07/10/2026") {
							t.Fatal("recepción confundida con decisión")
						}
						repetida := post(tipo, id, idioma, clave)
						if repetida.Code != 200 || !bytes.Equal(respuesta.Body.Bytes(), repetida.Body.Bytes()) || respuesta.Header().Get("X-Emision-ID") != repetida.Header().Get("X-Emision-ID") {
							t.Fatal("repetición", repetida.Code)
						}
						err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: usuario, Perfiles: a.Perfiles}, func(ctx context.Context) error {
							lectura, err := consulta.ParaDocumentos(ctx, ea, id)
							if err != nil {
								return err
							}
							if lectura.FechaRecepcion != "2026-10-07" {
								t.Fatal("recepción fuera de Madrid", lectura.FechaRecepcion)
							}
							if lectura.Estado != estado || lectura.Version != 2 || lectura.Numero != nil || lectura.EquipoID != nil || lectura.MunicipioID != nil {
								t.Fatal("emisión cambió la decisión", lectura)
							}
							if tipo == d.ActaNoAceptacion && (lectura.NoAceptacion == nil || lectura.NoAceptacion.Motivo == "" || lectura.NoAceptacion.Subprograma == "") {
								t.Fatal("decisión no leída")
							}
							if tipo == d.OficioListaEspera && (lectura.FechaEspera == nil || *lectura.FechaEspera != "2026-10-13") {
								t.Fatal("espera no leída")
							}
							entrada, err := datos.Obtener(ctx, a, d.Solicitud{Tipo: tipo, ExpedienteID: id, Idioma: idioma, IdiomaPaquete: d.IdiomaPlantilla{Ruta: idioma}})
							if err != nil {
								return err
							}
							if entrada.Aceptacion.Numero != "" || entrada.Aceptacion.Municipio != "" || entrada.Aceptacion.Equipo != "" {
								t.Fatal("asignación inventada")
							}
							for _, menor := range entrada.Aceptacion.Menores {
								if menor.Sexo != "" || menor.CodigoSexo != "" {
									t.Fatal("sexo inventado")
								}
							}
							if tipo == d.ActaNoAceptacion {
								esperado := "RIESGO_SOCIAL"
								if origen == "anexo4" {
									esperado = "REUNIFICACION"
								}
								if entrada.Aceptacion.Decision.Subprograma != esperado || entrada.Aceptacion.Decision.Fecha != "13/10/2026" {
									t.Fatal("decisión incorrecta")
								}
							}
							_, registro, err := (repo.Repositorio{}).LeerRegistro(ctx, respuesta.Header().Get("X-Emision-ID"))
							if err == nil && ((tipo == d.OficioListaEspera && registro.Estado != "pendiente") || (tipo == d.ActaNoAceptacion && registro.EmisionID != "")) {
								t.Fatal("registro de salida incorrecto")
							}
							return err
						})
						if err != nil {
							t.Fatal(err)
						}
						if n == 1 && idioma == "es" && origen == "anexo3" {
							for _, perfil := range []string{"equipo", "administracion", "consulta"} {
								err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: usuario, Perfiles: []string{perfil}}, func(ctx context.Context) error {
									emision, _, err := (repo.Repositorio{}).LeerRegistro(ctx, respuesta.Header().Get("X-Emision-ID"))
									if err != nil {
										return err
									}
									if emision.ID != "" {
										t.Fatal("RLS expone emisión", perfil)
									}
									_, err = (&expedienteRepo.Repositorio{}).ParaDocumentos(ctx, id)
									if !errors.Is(err, expedienteEntrada.ErrNoEncontrada) {
										t.Fatal("RLS expone decisión", perfil, err)
									}
									return nil
								})
								if err != nil {
									t.Fatal(err)
								}
							}
						}
					})
				}
			}
		}
	}
	for _, caso := range []struct{ tipo, id, clave string }{
		{d.ActaNoAceptacion, "77777777-7777-4777-8777-000000000001", "documento.requiere_no_aceptada"},
		{d.ActaNoAceptacion, "99999999-7777-4777-8777-000000000301", "documento.requiere_no_aceptada"},
		{d.OficioListaEspera, "77777777-7777-4777-8777-000000000001", "documento.requiere_lista_espera"},
		{d.OficioListaEspera, "88888888-7777-4777-8777-000000000201", "documento.requiere_lista_espera"},
	} {
		for _, idioma := range []string{"es", "en"} {
			respuesta := post(caso.tipo, caso.id, idioma, "estado-"+idioma+"-"+strings.ReplaceAll(caso.tipo, "/", "_")+caso.id)
			if respuesta.Code != 409 || !strings.Contains(respuesta.Body.String(), caso.clave) {
				t.Fatal("estado incorrecto admitido", respuesta.Code, respuesta.Body.String())
			}
		}
	}
	t.Log("POST real: No Aceptación anexo3/anexo4 y Espera anexo3; 1/6/15 menores; es/en; repetición idéntica; estado y fechas conservados; estado incorrecto 409")
}
