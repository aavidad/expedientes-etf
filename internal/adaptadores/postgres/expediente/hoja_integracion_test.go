package expediente_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/clamav"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/hoja"
	httpETF "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func csvHistorico(t *testing.T, casos ...d.CasoHistorico) []byte {
	t.Helper()
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"numero", "fecha_derivacion", "estado", "tipo_anexo", "fecha_aceptacion", "municipio_id", "centro_ss_id", "subprograma", "profesional_referencia_puesto", "profesional_referencia_id", "descripcion_situacion", "sexos_menores", "fecha_no_aceptacion", "causa_no_aceptacion", "campos_derivacion", "episodios", "familia", "menores"})
	j := func(v any) string {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		if string(b) == "null" {
			return ""
		}
		return string(b)
	}
	for _, c := range casos {
		r := make([]string, 18)
		r[0], r[1], r[2], r[3] = c.Numero, c.FechaDerivacion, c.Estado, c.Datos.Extraccion.Tipo
		if a := c.Aceptacion; a != nil {
			r[4], r[5], r[6], r[7], r[8], r[10], r[11] = a.FechaActa, a.MunicipioID, a.CentroID, a.Subprograma, a.ProfesionalPuesto, a.Descripcion, j(a.Menores)
			if a.ProfesionalID != nil {
				r[9] = *a.ProfesionalID
			}
		}
		if n := c.NoAceptacion; n != nil {
			r[7], r[12], r[13] = n.Subprograma, n.Fecha, n.Motivo
		}
		r[14], r[15], r[16], r[17] = j(c.Datos.Extraccion.Campos), j(c.Episodios), j(c.Familia), j(c.Menores)
		if err := w.Write(r); err != nil {
			t.Fatal(err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestHojaHTTPPostgresClamAVVistaPreviaLoteIdempotente(t *testing.T) {
	b, ctx, deps, coord, p := prepararAceptacion(t)
	deps = dependenciasHistoricas(deps)
	s := app.Nuevo(deps)
	av, err := clamav.Nuevo(clamav.Configuracion{Direccion: os.Getenv("ETF_PRUEBA_CLAMAV_DIRECCION"), TiempoLimite: 3 * time.Second, MaximoBytes: d.MaxBytes})
	if err != nil {
		t.Fatal(err)
	}
	sheet := &app.ServicioHoja{Historicos: s, Lector: hoja.Lector{}, Antivirus: av, Reloj: deps.Reloj}
	textos := ficheros.NuevosPaquetes("../../../../idiomas", "../../../../temas")
	ss := &sesionFamilia{sesionAceptacion: sesionAceptacion{coord.UsuarioID}, equipo: p.EquipoID, perfil: identidad.ProfesionalETF}
	h := httpETF.Nuevo(nil, fstest.MapFS{}, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, httpETF.Opciones{Sesion: httpETF.ConfiguracionSesion{Servicio: ss, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}, Expedientes: httpETF.ConfiguracionAceptaciones{Servicio: s, CargasHistoricas: s, Hoja: httpETF.ConfiguracionHoja{Servicio: sheet, Textos: textos, Idioma: "es"}}})
	cookie := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32))
	pedir := func(ruta, csrf, clave, huella, equipo, idioma string, cuerpo []byte) *httptest.ResponseRecorder {
		metodo := "POST"
		if ruta == "/api/sesion" {
			metodo = "GET"
		}
		r := httptest.NewRequest(metodo, "https://etf.interno"+ruta, bytes.NewReader(cuerpo)).WithContext(ctx)
		r.AddCookie(&http.Cookie{Name: "__Host-etf-sesion", Value: cookie})
		r.Header.Set("Content-Type", "text/csv; charset=utf-8")
		r.Header.Set("Origin", "https://etf.interno")
		if csrf != "" {
			r.Header.Set("X-CSRF-Token", csrf)
		}
		if clave != "" {
			r.Header.Set("Idempotency-Key", clave)
		}
		r.Header.Set("X-Equipo-Carga", equipo)
		r.Header.Set("X-Confirmar-Duplicados", "true")
		r.Header.Set("Accept-Language", idioma)
		if huella != "" {
			r.Header.Set("X-Huella-Hoja", huella)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	var acceso struct {
		CSRF string `json:"csrf"`
	}
	_ = json.Unmarshal(pedir("/api/sesion", "", "", "", p.EquipoID, "es", nil).Body.Bytes(), &acceso)
	caso := casoHistorico(p, numeroHistoricoLibre(t, b, ctx, coord, p))
	numeroAntes := contador(t, b, ctx, coord, p)
	antes := contarCargas(t, b, ctx, coord)
	csv := csvHistorico(t, caso)
	v := pedir("/api/carga-2026/hoja", acceso.CSRF, "", "", p.EquipoID, "es", csv)
	var informe d.InformeHoja
	if v.Code != 200 || json.Unmarshal(v.Body.Bytes(), &informe) != nil || !informe.Valida || !d.IDValido(informe.LoteID) {
		t.Fatal(v.Code, v.Body.String())
	}
	if contarCargas(t, b, ctx, coord) != antes || contador(t, b, ctx, coord, p) != numeroAntes {
		t.Fatal("vista previa persiste o consume correlativos")
	}
	if bytes.Contains(v.Body.Bytes(), []byte("Serrano")) {
		t.Fatal("vista previa expone datos personales")
	}
	// Un segundo caso incorrecto invalida todo el lote y obtiene su fila en llano.
	invalido := casoHistorico(p, "")
	invalido.FechaDerivacion = "2026-11-31"
	rechazo := pedir("/api/carga-2026/hoja", acceso.CSRF, "", "", p.EquipoID, "es", csvHistorico(t, caso, invalido))
	var errores d.InformeHoja
	if rechazo.Code != 422 || json.Unmarshal(rechazo.Body.Bytes(), &errores) != nil || len(errores.Incidencias) == 0 || errores.Incidencias[0].Fila != 3 || errores.Incidencias[0].Mensaje == "" || contarCargas(t, b, ctx, coord) != antes {
		t.Fatal(rechazo.Code, rechazo.Body.String())
	}
	// Los errores que dependen de catálogo también vuelven a su fila original.
	invalido = casoHistorico(p, "")
	invalido.Aceptacion.MunicipioID = "90000000-0000-4000-8000-000000000999"
	rechazo = pedir("/api/carga-2026/hoja", acceso.CSRF, "", "", p.EquipoID, "en", csvHistorico(t, caso, invalido))
	if rechazo.Code != 422 || json.Unmarshal(rechazo.Body.Bytes(), &errores) != nil || len(errores.Incidencias) == 0 || errores.Incidencias[0].Fila != 3 || errores.Incidencias[0].Mensaje == "" {
		t.Fatal(rechazo.Code, rechazo.Body.String())
	}
	ruta := "/api/carga-2026/hoja/" + informe.LoteID + "/confirmar"
	clave := "hoja-" + informe.LoteID
	if r := pedir(ruta, "", clave, informe.Huella, p.EquipoID, "es", csv); r.Code != 403 {
		t.Fatal(r.Code)
	}
	if r := pedir(ruta, acceso.CSRF, "", informe.Huella, p.EquipoID, "es", csv); r.Code != 400 {
		t.Fatal(r.Code)
	}
	if r := pedir(ruta, acceso.CSRF, clave, informe.Huella, p.EquipoID, "es", append(csv, '\n')); r.Code != 409 {
		t.Fatal(r.Code, r.Body.String())
	}
	if r := pedir("/api/carga-2026/hoja", acceso.CSRF, "", "", p.EquipoID, "es", bytes.Repeat([]byte{'a'}, d.MaxBytes+1)); r.Code != 413 {
		t.Fatal(r.Code)
	}
	ss.equipo = "90000000-0000-4000-8000-000000000999"
	if r := pedir(ruta, acceso.CSRF, clave, informe.Huella, p.EquipoID, "es", csv); r.Code != 403 {
		t.Fatal(r.Code)
	}
	ss.equipo = p.EquipoID
	sheet.Antivirus = antivirusCaido{}
	if r := pedir(ruta, acceso.CSRF, clave, informe.Huella, p.EquipoID, "es", csv); r.Code != 503 {
		t.Fatal(r.Code)
	}
	sheet.Antivirus = av
	eicar := []byte("X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*")
	if r := pedir("/api/carga-2026/hoja", acceso.CSRF, "", "", p.EquipoID, "es", eicar); r.Code != 422 {
		t.Fatal(r.Code, r.Body.String())
	}
	primera := pedir(ruta, acceso.CSRF, clave, informe.Huella, p.EquipoID, "es", csv)
	if primera.Code != 200 {
		t.Fatal(primera.Code, primera.Body.String())
	}
	// Recuperación con una instancia nueva: no hay caché de hojas o recibos.
	sheet.Historicos = app.Nuevo(deps)
	repetida := pedir(ruta, acceso.CSRF, clave, informe.Huella, p.EquipoID, "es", csv)
	if repetida.Code != 200 || repetida.Body.String() != primera.Body.String() || contarCargas(t, b, ctx, coord) != antes+1 {
		t.Fatal(repetida.Code, repetida.Body.String())
	}
	if contador(t, b, ctx, coord, p) != numeroAntes {
		t.Fatal("hoja consume correlativos")
	}
	var recibo d.ResultadoCargaHistorica
	_ = json.Unmarshal(primera.Body.Bytes(), &recibo)
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: coord.UsuarioID, Perfiles: coord.Perfiles}, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		var origen, huella, formato, av string
		var n int
		if err := tx.QueryRow(ctx, `SELECT origen,encode(huella_fuente,'hex'),formato_fuente,bytes_fuente,antivirus FROM expediente.carga_inicial WHERE id=$1`, recibo.LoteID).Scan(&origen, &huella, &formato, &n, &av); err != nil {
			return err
		}
		if origen != "hoja" || huella != informe.Huella || formato != "csv" || n != len(csv) || av != "limpio" {
			t.Fatal("procedencia incompleta")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	eventos, err := postgres.NuevoRegistroAuditoria(b).Exportar(ctx, plataforma.Identidad{UsuarioID: coord.UsuarioID, Perfiles: []string{"administracion"}}, 1, 100000)
	if err != nil {
		t.Fatal(err)
	}
	nEventos := 0
	for _, e := range eventos.Eventos {
		if e.Accion == "expediente.cargado_para_revision" && e.ExpedienteID == recibo.Expedientes[0] {
			nEventos++
		}
	}
	if nEventos != 1 {
		t.Fatal("repetición duplica auditoría")
	}

	ss.perfil = identidad.Administracion
	if r := pedir("/api/carga-2026/hoja", acceso.CSRF, "", "", p.EquipoID, "es", csv); r.Code != 403 {
		t.Fatal(r.Code)
	}
	t.Log("HTTP + PostgreSQL 18 + ClamAV: vista previa sin datos guardados; errores por fila traducidos; CSRF y equipo; EICAR y antivirus caído; lote idempotente, histórico y auditado")
}

func TestHojaPostgresAuditoriaFallidaRevierteTodosLosCasos(t *testing.T) {
	b, ctx, deps, coord, p := prepararAceptacion(t)
	deps = dependenciasHistoricas(deps)
	s := app.Nuevo(deps)
	av := antivirusHojaLimpio{}
	sheet := &app.ServicioHoja{Historicos: s, Lector: hoja.Lector{}, Antivirus: av, Reloj: deps.Reloj}
	csv := csvHistorico(t, casoHistorico(p, ""), casoHistorico(p, ""))
	u := entrada.SubidaHoja{ConfirmarDuplicados: true, EquipoID: p.EquipoID, Formato: "csv", Contenido: csv}
	v, err := sheet.ValidarHoja(ctx, coord, u)
	if err != nil || !v.Valida {
		t.Fatal(err)
	}
	antes := contarCargas(t, b, ctx, coord)
	deps.RegistroTransaccional = registroFallido{}
	sheet.Historicos = app.Nuevo(deps)
	if _, _, err := sheet.ConfirmarHoja(ctx, coord, v.LoteID, "fallo-"+v.LoteID, v.Huella, u); !errors.Is(err, d.ErrNoDisponible) {
		t.Fatal(err)
	}
	if contarCargas(t, b, ctx, coord) != antes {
		t.Fatal("fallo de auditoría deja carga parcial")
	}
	t.Log("Fallo de auditoría revierte el lote completo")
}

type antivirusHojaLimpio struct{}

func (antivirusHojaLimpio) Revisar(context.Context, []byte) (plataforma.EstadoAntivirus, error) {
	return plataforma.AntivirusLimpio, nil
}

func TestHojaCSVNoAdmiteEquipoNiUsuarioDeFila(t *testing.T) {
	b := []byte("fecha_derivacion,estado,tipo_anexo,equipo_id\n2026-01-01,no_aceptada,anexo3,otro\n")
	_, err := (hoja.Lector{}).LeerHoja(context.Background(), "csv", b)
	if !errors.Is(err, d.ErrHojaFormato) || strings.Contains(err.Error(), "otro") {
		t.Fatal(err)
	}
}
