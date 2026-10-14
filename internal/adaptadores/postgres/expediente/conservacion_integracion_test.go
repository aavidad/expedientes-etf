package expediente_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	nombres "github.com/aavidad/expedientes-etf/internal/adaptadores/expediente/ficheros"
	reglas "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/estadistica"
	httpETF "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	actividad "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/actividad"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	memoria "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/estadistica"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	estadistica "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	est "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type relojConservacion struct{ fecha time.Time }

func (r *relojConservacion) Ahora() time.Time { return r.fecha }

type transaccionConservacionRevertida struct{ base *postgres.Base }

func (t transaccionConservacionRevertida) Ejecutar(ctx context.Context, a plataforma.Identidad, fn func(context.Context) error) error {
	return t.base.Ejecutar(ctx, a, func(ctx context.Context) error {
		if err := fn(ctx); err != nil {
			return err
		}
		return errors.New("conservacion.confirmacion_fallida")
	})
}

func TestConservacionDobleRevisionMemoriaRLSAuditoria(t *testing.T) {
	if os.Getenv("ETF_CONSERVACION_INTEGRACION") != "1" {
		t.Skip("ETF_CONSERVACION_INTEGRACION=1")
	}
	b, ctx, deps, coord, otra, equipo, p := prepararTrasladoSeguimiento(t)
	r := &repo.Repositorio{}
	deps.Conservacion = &app.ServicioConservacion{Repositorio: r, Catalogo: &catalogo.Repositorio{}, Defecto: nombres.Conservacion{Ruta: "../../../../datos/catalogos/conservacion.json"}}
	s := app.Nuevo(deps)
	f, peticion := aceptarYProponer(t, ctx, deps, coord, equipo, p, false, "OBJ_TOTAL", []d.Menor{{Orden: 1, Nombre: "Lucía", Apellidos: "Moreno Castillo", Sexo: "M", Nacimiento: texto("2016-05-14"), Escolarizacion: texto("PRIMARIA"), Subprograma: "RIESGO_SOCIAL"}})
	emisionID := datosConservacionOtrosModulos(t, b, ctx, deps, coord, equipo, f.ID)
	cierre, err := s.Cerrar(ctx, otra, f.ID, "cierre-conservar-"+f.ID, peticion)
	if err != nil {
		t.Fatal(err)
	}
	admin := d.Actor{UsuarioID: coord.UsuarioID, Perfiles: []string{"administracion"}}
	segunda := d.Actor{UsuarioID: otra.UsuarioID, Perfiles: []string{"administracion"}}
	reloj := &relojConservacion{time.Date(2051, 9, 30, 12, 0, 0, 123456789, time.UTC)}
	conservar := &app.ServicioConservacion{Repositorio: r, Catalogo: &catalogo.Repositorio{}, Defecto: nombres.Conservacion{Ruta: "../../../../datos/catalogos/conservacion.json"}, Reloj: reloj, Transacciones: b, Registro: postgres.NuevoRegistroAuditoria(b)}
	filas, err := conservar.PendientesConservacion(ctx, admin)
	if err != nil {
		t.Fatal("listar antes", err)
	}
	for _, fila := range filas {
		if fila.ExpedienteID == f.ID {
			t.Fatal("vencimiento anticipado")
		}
	}
	reloj.fecha = time.Date(2051, 10, 1, 12, 0, 0, 123456789, time.UTC)
	filas, err = conservar.PendientesConservacion(ctx, admin)
	if err != nil {
		t.Fatal("listar al corte", err)
	}
	var fila d.Conservacion
	for _, v := range filas {
		if v.ExpedienteID == f.ID {
			fila = v
		}
	}
	if fila.Fin != "2051-10-01" || fila.Estado != "pendiente_propuesta" {
		t.Fatal(fila)
	}
	for _, actor := range []d.Actor{coord, equipo, {UsuarioID: coord.UsuarioID, Perfiles: []string{"consulta"}}} {
		if _, err := conservar.PendientesConservacion(ctx, actor); !errors.Is(err, d.ErrAcceso) {
			t.Fatal("permiso lista", actor, err)
		}
	}
	if _, err := s.Consultar(ctx, admin, f.ID); !errors.Is(err, d.ErrNoEncontrada) && !errors.Is(err, d.ErrAcceso) {
		t.Fatal("Administración lee ficha", err)
	}
	if _, err := conservar.ConfirmarConservacion(ctx, segunda, f.ID, fila.Version, true); !errors.Is(err, d.ErrConservacion) {
		t.Fatal("anonimiza sin propuesta", err)
	}
	// El módulo real aplica sesión, CSRF y validación antes del caso de uso.
	handler := httpETF.Nuevo(nil, nil, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, httpETF.Opciones{Sesion: httpETF.ConfiguracionSesion{Servicio: sesionSalidaSPM{actor: admin}, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}, Modulos: []httpETF.Modulo{httpETF.ModuloConservacion{Servicio: conservar}}})
	pedir := func(metodo, ruta, csrf, cuerpo string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(metodo, "https://etf.interno"+ruta, strings.NewReader(cuerpo))
		req.AddCookie(&http.Cookie{Name: "__Host-etf-sesion", Value: "AwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwM"})
		req.Header.Set("Origin", "https://etf.interno")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", csrf)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w
	}
	var acceso struct {
		CSRF string `json:"csrf"`
	}
	if err := json.Unmarshal(pedir("GET", "/api/sesion", "", "").Body.Bytes(), &acceso); err != nil {
		t.Fatal(err)
	}
	if w := pedir("GET", "/api/conservacion/pendientes", "", ""); w.Code != 200 || strings.Contains(w.Body.String(), "Teresa") {
		t.Fatal(w.Code, w.Body.String())
	}
	ruta := "/api/conservacion/" + f.ID + "/proponer"
	if w := pedir("POST", ruta, "", `{"version":1}`); w.Code != 403 {
		t.Fatal("CSRF", w.Code)
	}
	if w := pedir("POST", ruta, acceso.CSRF, `{"version":1,"actor":"otro"}`); w.Code != 400 {
		t.Fatal("campos ajenos", w.Code)
	}
	for _, sql := range []string{`SELECT * FROM expediente.conservacion_ejecucion`, `SELECT * FROM expediente.conservacion`, `SELECT expediente.conservacion_interna(NULL,false)`, `SELECT evaluacion.api_anonimizar_expediente('` + f.ID + `',clock_timestamp())`, `SELECT documento.api_anonimizar_expediente('` + f.ID + `',clock_timestamp())`} {
		err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: admin.UsuarioID, Perfiles: admin.Perfiles}, func(ctx context.Context) error {
			tx, _ := postgres.DesdeContexto(ctx)
			_, err := tx.Exec(ctx, sql)
			return err
		})
		if err == nil {
			t.Fatal("rol app accede a datos internos", sql)
		}
	}

	calcular := func() est.InformeMemoria {
		t.Helper()
		fuente := estadistica.FuenteMemoria{Expedientes: &repo.LecturaEstadistica{}, Reglas: reglas.Reglas{Raiz: "../../../../datos/catalogos/memoria", SMI: &catalogo.Repositorio{}}, Equipos: &catalogo.Repositorio{}, Actividad: &actividad.Repositorio{}, Narrativas: &memoria.Repositorio{}}
		var informe est.InformeMemoria
		err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: coord.UsuarioID, Perfiles: coord.Perfiles}, func(ctx context.Context) error {
			var err error
			informe, err = fuente.MemoriaEnTransaccion(ctx, est.ActorMemoria{UsuarioID: coord.UsuarioID, Perfiles: coord.Perfiles}, est.FiltroMemoria{Anio: 2026, EquipoID: p.EquipoID}, reloj.fecha)
			return err
		})
		if err != nil {
			t.Fatal("calcular Memoria", err)
		}
		return informe
	}
	antes := calcular()
	encontrada := false
	for _, fuente := range antes.Fuentes {
		if fuente.ID == f.ID {
			encontrada = true
			edadConocida := false
			for _, revision := range fuente.Menores {
				for _, menor := range revision.Dato.Valor {
					if !menor.Nacimiento.IsZero() {
						edadConocida = true
					}
				}
			}
			if !edadConocida {
				t.Fatal("faltan edades conocidas en la prueba")
			}
		}
	}
	if !encontrada || len(antes.Apartados) != 33 {
		t.Fatal("Memoria sin caso preparado o apartados incompletos")
	}

	propuesta, err := conservar.ConfirmarConservacion(ctx, admin, f.ID, fila.Version, false)
	if err != nil || propuesta.Estado != "pendiente_segunda_persona" {
		t.Fatal("proponer", propuesta, err)
	}
	if _, err := conservar.ConfirmarConservacion(ctx, admin, f.ID, propuesta.Version, true); !errors.Is(err, d.ErrConservacion) {
		t.Fatal("misma persona", err)
	}
	if _, err := conservar.ConfirmarConservacion(ctx, segunda, f.ID, fila.Version, true); err == nil {
		t.Fatal("acepta versión vieja")
	}
	// Un fallo al confirmar revierte la retirada y su auditoría; no entrega éxito.
	conservar.Transacciones = transaccionConservacionRevertida{b}
	if _, err := conservar.ConfirmarConservacion(ctx, segunda, f.ID, propuesta.Version, true); !errors.Is(err, d.ErrNoDisponible) {
		t.Fatal("confirmación fallida", err)
	}
	conservar.Transacciones = b
	var identidadConservada string
	consultaSQL(t, b, ctx, coord, `SELECT string_agg(datos::text,',') FROM expediente.adulto WHERE expediente_id=$1`, []any{f.ID}, &identidadConservada)
	if !strings.Contains(identidadConservada, "Teresa") {
		t.Fatal("retirada no revertida")
	}
	const simultaneas = 4
	recibos := make([]d.Conservacion, simultaneas)
	fallos := make([]error, simultaneas)
	var grupo sync.WaitGroup
	for i := range simultaneas {
		grupo.Add(1)
		go func(i int) {
			defer grupo.Done()
			recibos[i], fallos[i] = conservar.ConfirmarConservacion(ctx, segunda, f.ID, propuesta.Version, true)
		}(i)
	}
	grupo.Wait()
	anon := recibos[0]
	for i := range simultaneas {
		if fallos[i] != nil || recibos[i].Estado != "anonimizado" || !reflect.DeepEqual(anon, recibos[i]) {
			t.Fatal("anonimización concurrente", i, recibos[i], fallos[i])
		}
	}

	repetida, err := conservar.ConfirmarConservacion(ctx, segunda, f.ID, propuesta.Version, true)
	if err != nil || !reflect.DeepEqual(anon, repetida) {
		t.Fatal("repetición", repetida, err)
	}
	despues := calcular()
	if !reflect.DeepEqual(antes.Apartados, despues.Apartados) || !reflect.DeepEqual(antes.Fuentes, despues.Fuentes) {
		t.Fatal("cambian cifras o bases de Memoria 2026")
	}
	var datos string
	consultaSQL(t, b, ctx, coord, `SELECT jsonb_build_object('derivacion',d.contenido,'familia',e.familia,'situacion',e.descripcion_situacion,'adultos',(SELECT jsonb_agg(datos) FROM expediente.adulto WHERE expediente_id=e.id),'menores',(SELECT jsonb_agg(datos) FROM expediente.menor WHERE expediente_id=e.id))::text FROM expediente.expediente e JOIN expediente.derivacion_datos d ON d.expediente_id=e.id WHERE e.id=$1`, []any{f.ID}, &datos)
	if strings.Contains(datos, "Teresa") || strings.Contains(datos, "López") || strings.Contains(datos, "telefono_contacto") || strings.Contains(datos, "direccion") {
		t.Fatal("quedan datos personales", datos)
	}

	var otros string
	consultaSQL(t, b, ctx, coord, `SELECT jsonb_build_object('instrumentos',(SELECT jsonb_agg(datos) FROM evaluacion.instrumento_aplicado WHERE expediente_id=$1),'escalas',(SELECT jsonb_agg(datos) FROM evaluacion.aplicacion_escala WHERE expediente_id=$1),'registro',(SELECT to_jsonb(r) FROM documento.registro_salida r WHERE emision_id=$2))::text`, []any{f.ID, emisionID}, &otros)
	if strings.Contains(otros, "Teresa") || strings.Contains(otros, "López") || strings.Contains(otros, "texto_otros") || strings.Contains(otros, "descripcion_cualitativa") {
		t.Fatal("quedan textos de otros módulos", otros)
	}
	var recibosPersonales int
	consultaSQL(t, b, ctx, coord, `SELECT (SELECT count(*) FROM evaluacion.peticion WHERE expediente_id=$1)+(SELECT count(*) FROM documento.registro_peticion WHERE emision_id=$2)+(SELECT count(*) FROM documento.peticion_idempotente WHERE emision_id=$2)`, []any{f.ID, emisionID}, &recibosPersonales)
	if recibosPersonales != 0 {
		t.Fatal("quedan recibos", recibosPersonales)
	}
	var puntos float64
	var versionEscala int
	consultaSQL(t, b, ctx, coord, `SELECT a.version,r.valor_numerico FROM evaluacion.aplicacion_escala a JOIN evaluacion.respuesta r ON r.aplicacion_id=a.id WHERE a.expediente_id=$1`, []any{f.ID}, &versionEscala, &puntos)
	if versionEscala != 1 || puntos != 1 {
		t.Fatal("cambian puntuaciones", versionEscala, puntos)
	}
	var emisiones int
	consultaSQL(t, b, ctx, coord, `SELECT count(*) FROM documento.emision WHERE id=$1 AND huella_datos=decode(repeat('b',64),'hex') AND versiones_datos='{"expediente":"1"}'::jsonb`, []any{emisionID}, &emisiones)
	if emisiones != 1 {
		t.Fatal("se altera emisión inmutable")
	}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: coord.UsuarioID, Perfiles: coord.Perfiles}, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		_, err := tx.Exec(ctx, `UPDATE documento.registro_salida SET numero_registro='Restaurado',version=version+1 WHERE emision_id=$1`, emisionID)
		return err
	})
	if err == nil {
		t.Fatal("permite reponer referencia documental")
	}
	if _, err := s.Reabrir(ctx, otra, f.ID, "reabrir-anon-"+f.ID, d.PeticionReapertura{Version: cierre.Version, Fecha: "2026-10-02", Motivo: "Revisión"}); err == nil {
		t.Fatal("reabre anonimizado")
	}
	roto, err := postgres.NuevoRegistroAuditoria(b).Verificar(ctx, plataforma.Identidad{UsuarioID: admin.UsuarioID, Perfiles: admin.Perfiles}, 1, 9223372036854775807)
	if err != nil || roto != nil {
		t.Fatal("cadena", roto, err)
	}
	t.Log("Memoria 2026: apartados y fuentes idénticos; doble revisión, fecha límite, RLS, repetición y cadena: OK")
}
