package expediente_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	httpETF "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	identidadEntrada "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	identidadDominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
)

type auditoriaFallidaSPM struct{}

func (auditoriaFallidaSPM) RegistrarEnTransaccion(context.Context, auditoria.Datos) (auditoria.Evento, error) {
	return auditoria.Evento{}, errors.New("fallo_controlado_auditoria")
}

type sesionSalidaSPM struct {
	sesionAceptacion
	actor d.Actor
}

func (s sesionSalidaSPM) Consultar(context.Context, string) (identidadEntrada.Acceso, error) {
	u := identidadDominio.Usuario{ID: s.actor.UsuarioID, Estado: identidadDominio.Activo, EquipoID: s.actor.EquipoID}
	for _, p := range s.actor.Perfiles {
		u.Perfiles = append(u.Perfiles, identidadDominio.Perfil(p))
	}
	return identidadEntrada.Acceso{Estado: identidadDominio.Activo, Usuario: u}, nil
}

func TestSPMPostgresVersionIdempotenciaPermisosYAuditoria(t *testing.T) {
	b, ctx, deps, a, p := prepararAceptacion(t)
	deps.SalidasSPM = &repo.Repositorio{}
	derivada := subir(t, ctx, deps, a)
	s := app.Nuevo(deps)
	aceptada, err := s.Aceptar(ctx, a, derivada.ID, "spm-aceptar", conMenores(p, derivada))
	if err != nil {
		t.Fatal(err)
	}
	profesional := d.Actor{UsuarioID: a.UsuarioID, Perfiles: []string{"profesional_etf"}, EquipoID: p.EquipoID}
	consulta := app.ConsultaCierreSPM{Repositorio: &repo.Repositorio{}, Salidas: &repo.Repositorio{}}
	permite := func(esperado bool) {
		t.Helper()
		err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
			v, err := consulta.PermiteCierreOrdinarioSPM(ctx, a, derivada.ID)
			if err == nil && v != esperado {
				t.Fatalf("cierre %v, esperado %v", v, esperado)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	permite(true)
	peticion := d.PeticionSPM{Version: aceptada.Version, Motivo: "La familia requiere valoración de protección.", PropuestaMedida: "Valorar apoyo especializado."}
	// La misma petición concurrente crea una sola salida.
	var wg sync.WaitGroup
	var rs [2]d.SalidaSPM
	var es [2]error
	for i := range rs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			rs[i], es[i] = s.OperarSPM(ctx, profesional, derivada.ID, "spm-crear", "crear", peticion)
		}(i)
	}
	wg.Wait()
	if es[0] != nil || es[1] != nil || !reflect.DeepEqual(rs[0], rs[1]) {
		t.Fatal(rs, es)
	}
	r := rs[0]
	for _, actorHTTP := range []d.Actor{a, profesional} {
		handler := httpETF.Nuevo(nil, nil, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, httpETF.Opciones{Sesion: httpETF.ConfiguracionSesion{Servicio: sesionSalidaSPM{actor: actorHTTP}, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}, Expedientes: httpETF.ConfiguracionAceptaciones{SPM: s}})
		pedir := func(metodo, ruta, csrf, clave, cuerpo string) *httptest.ResponseRecorder {
			req := httptest.NewRequest(metodo, "https://etf.interno"+ruta, strings.NewReader(cuerpo))
			req.AddCookie(&http.Cookie{Name: "__Host-etf-sesion", Value: "AwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwM"})
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Origin", "https://etf.interno")
			req.Header.Set("X-CSRF-Token", csrf)
			req.Header.Set("Idempotency-Key", clave)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			return w
		}
		var acceso struct {
			CSRF string `json:"csrf"`
		}
		sesion := pedir("GET", "/api/sesion", "", "", "")
		if json.Unmarshal(sesion.Body.Bytes(), &acceso) != nil {
			t.Fatal(sesion.Body.String())
		}
		vieja := pedir("POST", "/api/expedientes/"+derivada.ID+"/salida-spm/fecha", acceso.CSRF, "http-spm-vieja", `{"version":9,"fecha":"2026-10-12"}`)
		if vieja.Code != 409 || !strings.Contains(vieja.Body.String(), `"version_actual":1`) {
			t.Fatal(vieja.Code, vieja.Body.String())
		}
		for _, metodo := range []string{"DELETE", "PUT"} {
			w := pedir(metodo, "/api/expedientes/"+derivada.ID+"/salida-spm", acceso.CSRF, "", "{}")
			if w.Code != 405 {
				t.Fatal(metodo, w.Code)
			}
		}
		sinCSRF := pedir("PATCH", "/api/expedientes/"+derivada.ID+"/salida-spm/respuesta", "", "http-spm-csrf", `{"version":1,"respuesta":"a_la_espera"}`)
		if sinCSRF.Code != 403 {
			t.Fatal(sinCSRF.Code)
		}

		leidaHTTP := pedir("GET", "/api/expedientes/"+derivada.ID+"/salida-spm", "", "", "")
		if leidaHTTP.Code != 200 || !strings.Contains(leidaHTTP.Body.String(), r.ID) {
			t.Fatal("lectura HTTP propia", actorHTTP, leidaHTTP.Code, leidaHTTP.Body.String())
		}
	}
	permite(false)
	peticion.Motivo = "Otro contenido"
	if _, err := s.OperarSPM(ctx, profesional, derivada.ID, "spm-crear", "crear", peticion); !errors.Is(err, d.ErrIdempotencia) {
		t.Fatal(err)
	}
	if _, err := s.OperarSPM(ctx, profesional, derivada.ID, "spm-vieja", "confirmar", d.PeticionSPM{Version: 9, Fecha: "2026-10-12"}); err == nil {
		t.Fatal("versión antigua")
	} else {
		var c d.Conflicto
		if !errors.As(err, &c) || c.VersionActual != 1 {
			t.Fatal(err)
		}
	}
	r, err = s.OperarSPM(ctx, a, derivada.ID, "spm-forzado-sin-fecha", "forzar", d.PeticionSPM{Version: r.Version, Motivo: "La coordinación documenta la excepción mientras se verifica la fecha de salida."})
	if err != nil || r.ForzadoID == "" {
		t.Fatal(r, err)
	}
	r, err = s.OperarSPM(ctx, profesional, derivada.ID, "spm-fecha", "confirmar", d.PeticionSPM{Version: r.Version, Fecha: "2026-10-12"})
	if err != nil {
		t.Fatal(err)
	}
	for _, perfil := range []string{"administracion", "consulta"} {
		actor := d.Actor{UsuarioID: a.UsuarioID, Perfiles: []string{perfil}}
		if _, err := s.ConsultarSalidaSPM(ctx, actor, derivada.ID); !errors.Is(err, d.ErrAcceso) {
			t.Fatal(perfil, err)
		}
	}
	ajeno := profesional
	ajeno.EquipoID = "88888888-8888-4888-8888-888888888888"
	if _, err := s.ConsultarSalidaSPM(ctx, ajeno, derivada.ID); err == nil {
		t.Fatal("equipo ajeno")
	}
	if _, err := s.OperarSPM(ctx, profesional, derivada.ID, "spm-forzar-denegado", "forzar", d.PeticionSPM{Version: r.Version, Motivo: "Existe un acuerdo de protección urgente."}); !errors.Is(err, d.ErrAcceso) {
		t.Fatal(err)
	}
	forzado, err := s.OperarSPM(ctx, a, derivada.ID, "spm-forzar", "forzar", d.PeticionSPM{Version: r.Version, Motivo: "Existe un acuerdo de protección urgente."})
	if err != nil || forzado.ForzadoID == "" {
		t.Fatal(forzado, err)
	}
	permite(false)
	repetido, err := s.OperarSPM(ctx, a, derivada.ID, "spm-forzar", "forzar", d.PeticionSPM{Version: r.Version, Motivo: "Existe un acuerdo de protección urgente."})
	if err != nil || !reflect.DeepEqual(forzado, repetido) {
		t.Fatal(repetido, err)
	}
	leida, err := s.ConsultarSalidaSPM(ctx, a, derivada.ID)
	if err != nil || leida.ForzadoID != forzado.ForzadoID || leida.MotivoForzado == "" {
		t.Fatal("forzado vigente no leído", leida, err)
	}
	r = forzado
	for i, respuesta := range []string{d.SPMInicioConocido, d.SPMAlaEspera, d.SPMMedidaNoNecesaria} {
		fecha := "2026-10-13"
		if respuesta == d.SPMAlaEspera {
			fecha = ""
		}
		r, err = s.OperarSPM(ctx, profesional, derivada.ID, fmt.Sprintf("spm-responder-%d", i), "responder", d.PeticionSPM{Version: r.Version, Respuesta: respuesta, Fecha: fecha})
		if err != nil {
			t.Fatal(err)
		}
		permite(respuesta != d.SPMAlaEspera)
	}
	// El fallo de auditoría revierte la nueva respuesta y su historia.
	fallidas := deps
	fallidas.RegistroTransaccional = auditoriaFallidaSPM{}
	if _, err := app.Nuevo(fallidas).OperarSPM(ctx, profesional, derivada.ID, "spm-auditoria-fallida", "responder", d.PeticionSPM{Version: r.Version, Respuesta: d.SPMAlaEspera}); !errors.Is(err, d.ErrNoDisponible) {
		t.Fatal(err)
	}
	leida, err = s.ConsultarSalidaSPM(ctx, a, derivada.ID)
	if err != nil || leida.ForzadoID != "" {
		t.Fatal("forzado anterior todavía vigente", leida, err)
	}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		var n int
		var motivo string
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM expediente.spm_forzado WHERE expediente_id=$1`, derivada.ID).Scan(&n); err != nil {
			return err
		}
		if n != 2 {
			t.Fatal("número de acciones de forzado incorrecto", n)
		}
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM expediente.spm_historia WHERE salida_id=$1`, r.ID).Scan(&n); err != nil {
			return err
		}
		if n != r.Version {
			t.Fatal("historia perdida", n, r.Version)
		}
		if err := tx.QueryRow(ctx, `SELECT motivo FROM expediente.spm_forzado WHERE id=$1`, forzado.ForzadoID).Scan(&motivo); err != nil {
			return err
		}
		if motivo != "Existe un acuerdo de protección urgente." {
			t.Fatal("motivo perdido")
		}
		ficha, err := deps.Repositorio.Obtener(ctx, derivada.ID)
		if err != nil {
			return err
		}
		if ficha.Estado != d.Aceptada || ficha.Version != aceptada.Version {
			t.Fatal("SPM cambia ficha", ficha.Version)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	eventos, err := postgres.NuevoRegistroAuditoria(b).Exportar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: []string{"administracion"}}, 1, 1000000)
	if err != nil {
		t.Fatal(err)
	}
	totalForzados := 0
	for _, ev := range eventos.Eventos {
		if ev.EntidadID == forzado.ForzadoID && ev.Accion == "expediente.spm_forzar" {
			totalForzados++
		}
	}
	if totalForzados != 1 {
		t.Fatal("forzado sin auditoría única", totalForzados)
	}
	// RLS independiente del caso de uso: perfiles ajenos no ven datos.
	for _, actor := range []d.Actor{ajeno, {UsuarioID: a.UsuarioID, Perfiles: []string{"consulta"}}, {UsuarioID: a.UsuarioID, Perfiles: []string{"administracion"}}} {
		err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: actor.Perfiles, EquipoID: actor.EquipoID}, func(ctx context.Context) error {
			tx, _ := postgres.DesdeContexto(ctx)
			var n int
			if err := tx.QueryRow(ctx, `SELECT count(*) FROM expediente.salida_spm`).Scan(&n); err != nil {
				return err
			}
			if n != 0 {
				t.Fatal("RLS expone salida", actor, n)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
