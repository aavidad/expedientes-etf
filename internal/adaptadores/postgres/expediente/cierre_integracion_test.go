package expediente_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	nombres "github.com/aavidad/expedientes-etf/internal/adaptadores/expediente/ficheros"
	httpETF "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	documentos "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/documento"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type relojCierreFijo struct{}

func (relojCierreFijo) Ahora() time.Time { return time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC) }

func TestCierrePostgresDobleIntervencionIdempotenciaEpisodiosYBloqueo(t *testing.T) {
	b, ctx, deps, coord, p := prepararAceptacion(t)
	r := &repo.Repositorio{}
	deps.Reloj = relojCierreFijo{}
	deps.Familias = r
	deps.SMI = &catalogo.Repositorio{}
	deps.CatalogosFamilia = nombres.CatalogosFamilia{Listas: "../../../../idiomas/es/catalogos/listas.json", Familia: "../../../../idiomas/es/catalogos/familia.json", Parametros: "../../../../datos/catalogos/familia.json"}
	deps.Cierres = r
	deps.Propuestas = r
	deps.MotivosCierre = &catalogo.Repositorio{}
	deps.SalidasSPM = r
	deps.Hitos = r
	deps.EmisionesFinalizacion = documentos.Repositorio{}
	deps.ConsultaCierreSPM = app.ConsultaCierreSPM{Repositorio: r, Salidas: r}
	s := app.Nuevo(deps)
	_, _, _, otra := preparar(t)
	equipo := d.Actor{UsuarioID: coord.UsuarioID, EquipoID: p.EquipoID, Perfiles: []string{"profesional_etf"}}
	for _, consiente := range []bool{false, true} {
		ficha := subir(t, ctx, deps, coord)
		aceptada, err := s.Aceptar(ctx, coord, ficha.ID, "aceptar-"+ficha.ID, conMenores(p, ficha))
		if err != nil {
			t.Fatal(err)
		}
		version := aceptada.Version
		if consiente {
			h, err := s.ConfirmarHito(ctx, equipo, ficha.ID, "consentir-"+ficha.ID, d.ConfirmacionHito{Version: version, Tipo: "consentimiento", Fecha: "2026-02-04"})
			if err != nil {
				t.Fatal(err)
			}
			version = h.Version
		}
		f := d.Finalizacion{Version: version, Motivo: "SIN_CONSENTIMIENTO", SinConsentimiento: true}
		if consiente {
			f.Motivo = "OBJ_TOTAL"
			f.SinConsentimiento = false
		}
		final, err := s.GuardarFinalizacion(ctx, equipo, ficha.ID, "final-"+ficha.ID, f)
		if err != nil {
			t.Fatal(err)
		}
		propuesta, err := s.ProponerCierre(ctx, equipo, ficha.ID, "proponer-"+ficha.ID, d.VersionCierre{Version: final.Version})
		if err != nil {
			t.Fatal(err)
		}
		peticion := d.PeticionCierre{Version: propuesta.Version, Fecha: "2026-03-01", MotivoActaTexto: "La familia mantiene los cuidados con apoyo de su red familiar."}
		if _, err = s.Cerrar(ctx, coord, ficha.ID, "misma-persona", peticion); !errors.Is(err, d.ErrDobleIntervencion) {
			t.Fatal(err)
		}
		for _, perfil := range []string{"profesional_etf", "consulta", "administracion"} {
			a := otra
			a.Perfiles = []string{perfil}
			a.EquipoID = p.EquipoID
			if _, err = s.Cerrar(ctx, a, ficha.ID, "sin-permiso-"+perfil, peticion); !errors.Is(err, d.ErrAcceso) {
				t.Fatal(perfil, err)
			}
		}
		ajeno := equipo
		ajeno.EquipoID = "88888888-8888-4888-8888-888888888888"
		if _, err = s.Consultar(ctx, ajeno, ficha.ID); !errors.Is(err, d.ErrNoEncontrada) {
			t.Fatal("equipo ajeno", err)
		}
		fallo := deps
		fallo.RegistroTransaccional = registroFallido{}
		if _, err = app.Nuevo(fallo).Cerrar(ctx, otra, ficha.ID, "fallo-"+ficha.ID, peticion); !errors.Is(err, d.ErrNoDisponible) {
			t.Fatal(err)
		}
		vista, err := s.Consultar(ctx, coord, ficha.ID)
		if err != nil || vista.Version != peticion.Version || vista.Estado == d.Cerrada {
			t.Fatal(vista, err)
		}
		obsoleta := peticion
		obsoleta.Version--
		if _, err = s.Cerrar(ctx, otra, ficha.ID, "obsoleta-"+ficha.ID, obsoleta); err == nil {
			t.Fatal("falta conflicto")
		} else {
			var c d.Conflicto
			if !errors.As(err, &c) || c.VersionActual != peticion.Version {
				t.Fatal(err)
			}
		}
		const n = 8
		var wg sync.WaitGroup
		var rs [n]d.ResultadoCierre
		var es [n]error
		for i := range n {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				rs[i], es[i] = s.Cerrar(ctx, otra, ficha.ID, "cerrar-"+ficha.ID, peticion)
			}(i)
		}
		wg.Wait()
		for i := range n {
			if es[i] != nil || rs[i] != rs[0] {
				t.Fatal(i, rs[i], es[i])
			}
		}
		cerrado := rs[0]
		comprobarLecturaEstadisticaCierre(t, b, ctx, otra, ficha.ID, false)
		if cerrado.Estado != d.Cerrada || cerrado.Version != peticion.Version+1 || cerrado.Aviso != "expediente.cierre_sin_anexo" {
			t.Fatal(cerrado)
		}
		alterada := peticion
		alterada.Fecha = "2026-03-02"
		if _, err = s.Cerrar(ctx, otra, ficha.ID, "cerrar-"+ficha.ID, alterada); !errors.Is(err, d.ErrIdempotencia) {
			t.Fatal(err)
		}
		if _, err = s.GuardarFinalizacion(ctx, equipo, ficha.ID, "editar-cerrado-"+ficha.ID, d.Finalizacion{Version: cerrado.Version, Motivo: "OBJ_TOTAL"}); !errors.Is(err, d.ErrEstado) {
			t.Fatal(err)
		}
		familia := datosFamilia()
		if _, err = s.EditarFamilia(ctx, equipo, ficha.ID, "familia-cerrada-"+ficha.ID, d.CambioFamilia{Version: cerrado.Version, Desde: "2026-03-01", Familia: &familia}); !errors.Is(err, d.ErrEstado) {
			t.Fatal("familia cerrada", err)
		}
		// La segunda barrera rechaza incluso SQL directo por Coordinación.
		for _, sql := range []string{`UPDATE expediente.expediente SET familia='{}'::jsonb WHERE id=$1`, `UPDATE expediente.expediente SET descripcion_situacion='Cambio' WHERE id=$1`, `UPDATE expediente.expediente SET version=version+1 WHERE id=$1`, `UPDATE expediente.finalizacion SET modificado_en=clock_timestamp() WHERE expediente_id=$1`, `UPDATE expediente.menor SET sexo='H' WHERE expediente_id=$1`} {
			err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: otra.UsuarioID, Perfiles: otra.Perfiles}, func(ctx context.Context) error {
				tx, _ := postgres.DesdeContexto(ctx)
				tag, err := tx.Exec(ctx, sql, ficha.ID)
				if err == nil && tag.RowsAffected() != 0 {
					t.Fatal("escritura cerrada permitida", sql)
				}
				return err
			})

		}
		err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: otra.UsuarioID, Perfiles: otra.Perfiles}, func(ctx context.Context) error {
			tx, _ := postgres.DesdeContexto(ctx)
			_, err := tx.Exec(ctx, `INSERT INTO expediente.episodio(expediente_id,equipo_id,apertura_fecha,creado_por,creado_en) VALUES($1,$2,DATE '2026-03-02',$3,clock_timestamp())`, ficha.ID, p.EquipoID, otra.UsuarioID)
			return err
		})
		if err == nil {
			t.Fatal("episodio directo en cerrado")
		}
		reapertura := d.PeticionReapertura{Version: cerrado.Version, Fecha: "2026-03-02", Motivo: "La familia solicita retomar el acompañamiento."}
		nuevo, err := s.Reabrir(ctx, otra, ficha.ID, "reabrir-"+ficha.ID, reapertura)
		if err != nil {
			t.Fatal(err)
		}
		estado := d.Aceptada
		if consiente {
			estado = d.EnIntervencion
		}
		if nuevo.Estado != estado || nuevo.EpisodioID == cerrado.EpisodioID {
			t.Fatal(nuevo)
		}
		comprobarLecturaEstadisticaCierre(t, b, ctx, otra, ficha.ID, true)
		repetido, err := s.Reabrir(ctx, otra, ficha.ID, "reabrir-"+ficha.ID, reapertura)
		if err != nil || repetido != nuevo {
			t.Fatal(repetido, err)
		}
		editada, err := s.EditarFamilia(ctx, equipo, ficha.ID, "familia-reabierta-"+ficha.ID, d.CambioFamilia{Version: nuevo.Version, Desde: "2026-03-02", Familia: &familia})
		if err != nil {
			t.Fatal("familia reabierta", err)
		}
		nuevo.Version = editada.Version
		if consiente {
			final2, err := s.GuardarFinalizacion(ctx, equipo, ficha.ID, "final2-"+ficha.ID, d.Finalizacion{Version: nuevo.Version, Motivo: "OBJ_TOTAL"})
			if err != nil {
				t.Fatal(err)
			}
			propuesta2, err := s.ProponerCierre(ctx, equipo, ficha.ID, "proponer2-"+ficha.ID, d.VersionCierre{Version: final2.Version})
			if err != nil {
				t.Fatal(err)
			}
			cerrado2, err := s.Cerrar(ctx, otra, ficha.ID, "cerrar2-"+ficha.ID, d.PeticionCierre{Version: propuesta2.Version, Fecha: "2026-03-03"})
			if err != nil || cerrado2.EpisodioID != nuevo.EpisodioID {
				t.Fatal(cerrado2, err)
			}
			if _, err := s.Reabrir(ctx, otra, ficha.ID, "reabrir2-"+ficha.ID, d.PeticionReapertura{Version: cerrado2.Version, Fecha: "2026-03-04", Motivo: "Se acuerda continuar el acompañamiento."}); err != nil {
				t.Fatal(err)
			}
		}
		err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: otra.UsuarioID, Perfiles: otra.Perfiles}, func(ctx context.Context) error {
			acta, err := r.LeerActaCierre(ctx, ficha.ID)
			if err == nil && acta != nil {
				t.Fatal("Acta anterior expuesta tras reapertura")
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		if !consiente {
			// El consentimiento posterior se asigna sólo al nuevo episodio activo.
			if _, err := s.ConfirmarHito(ctx, equipo, ficha.ID, "consentimiento-reabierto", d.ConfirmacionHito{Version: nuevo.Version, Tipo: "consentimiento", Fecha: "2026-03-03"}); err != nil {
				t.Fatal(err)
			}
		}
		err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: otra.UsuarioID, Perfiles: otra.Perfiles}, func(ctx context.Context) error {
			tx, _ := postgres.DesdeContexto(ctx)
			var total, activos int
			var fecha, motivo string
			if err := tx.QueryRow(ctx, `SELECT count(*),count(*) FILTER(WHERE cierre_fecha IS NULL) FROM expediente.episodio WHERE expediente_id=$1`, ficha.ID).Scan(&total, &activos); err != nil {
				return err
			}
			esperados := 2
			if consiente {
				esperados = 3
			}
			if total != esperados || activos != 1 {
				t.Fatal(total, activos)
			}
			if err := tx.QueryRow(ctx, `SELECT to_char(cierre_fecha,'YYYY-MM-DD'),motivo_cierre FROM expediente.episodio WHERE id=$1`, cerrado.EpisodioID).Scan(&fecha, &motivo); err != nil {
				return err
			}
			if fecha != peticion.Fecha || motivo != f.Motivo {
				t.Fatal(fecha, motivo)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		exportacion, err := postgres.NuevoRegistroAuditoria(b).Exportar(ctx, plataforma.Identidad{UsuarioID: otra.UsuarioID, Perfiles: []string{"administracion"}}, 1, 1000000)
		if err != nil {
			t.Fatal(err)
		}
		eventos := 0
		for _, ev := range exportacion.Eventos {
			if ev.ExpedienteID == ficha.ID && (ev.Accion == "expediente.cerrar" || ev.Accion == "expediente.reabrir") && ev.Resultado == "ok" {
				eventos++
			}
		}
		esperados := 2
		if consiente {
			esperados = 4
		}
		if eventos != esperados {
			t.Fatal(eventos)
		}
	}
	t.Log("8 cierres concurrentes: un cierre y un evento; 409 e idempotencia; segunda persona, perfiles y equipo ajeno; rollback de auditoría; cerrado bloquea SQL; reapertura conserva episodio y asigna nuevo consentimiento al activo")
}
func TestCierrePostgresSPMPendienteForzadoYHTTP(t *testing.T) {
	b, ctx, deps, coord, p := prepararAceptacion(t)
	r := &repo.Repositorio{}
	deps.Cierres = r
	deps.Propuestas = r
	deps.SalidasSPM = r
	deps.MotivosCierre = &catalogo.Repositorio{}
	deps.EmisionesFinalizacion = documentos.Repositorio{}
	deps.ConsultaCierreSPM = app.ConsultaCierreSPM{Repositorio: r, Salidas: r}
	s := app.Nuevo(deps)
	_, _, _, otra := preparar(t)
	ficha := subir(t, ctx, deps, coord)
	a, err := s.Aceptar(ctx, coord, ficha.ID, "aceptar", conMenores(p, ficha))
	if err != nil {
		t.Fatal(err)
	}
	equipo := d.Actor{UsuarioID: coord.UsuarioID, EquipoID: p.EquipoID, Perfiles: []string{"profesional_etf"}}
	spm, err := s.OperarSPM(ctx, equipo, ficha.ID, "crear-spm", "crear", d.PeticionSPM{Version: a.Version, Motivo: "Se requiere valoración de protección.", PropuestaMedida: "Valorar una medida protectora."})
	if err != nil {
		t.Fatal(err)
	}
	spm, err = s.OperarSPM(ctx, equipo, ficha.ID, "confirmar-spm", "confirmar", d.PeticionSPM{Version: spm.Version, Fecha: "2026-02-10"})
	if err != nil {
		t.Fatal(err)
	}
	final, err := s.GuardarFinalizacion(ctx, equipo, ficha.ID, "final", d.Finalizacion{Version: a.Version, Motivo: "SIN_CONSENTIMIENTO", SinConsentimiento: true})
	if err != nil {
		t.Fatal(err)
	}
	prop, err := s.ProponerCierre(ctx, equipo, ficha.ID, "proponer", d.VersionCierre{Version: final.Version})
	if err != nil {
		t.Fatal(err)
	}
	pC := d.PeticionCierre{Version: prop.Version, Fecha: "2026-03-01"}
	if _, err = s.Cerrar(ctx, otra, ficha.ID, "bloqueado", pC); !errors.Is(err, d.ErrCierreSPM) {
		t.Fatal(err)
	}
	spm, err = s.OperarSPM(ctx, otra, ficha.ID, "forzar-spm", "forzar", d.PeticionSPM{Version: spm.Version, Motivo: "Coordinación confirma que finaliza la intervención mientras continúa la valoración del SPM."})
	if err != nil {
		t.Fatal(err)
	}
	pC.ForzadoID = spm.ForzadoID
	forzadoID := spm.ForzadoID
	handler := httpETF.Nuevo(nil, nil, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, httpETF.Opciones{Sesion: httpETF.ConfiguracionSesion{Servicio: sesionSalidaSPM{actor: otra}, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}, Expedientes: httpETF.ConfiguracionAceptaciones{Cierres: s}})
	pedir := func(metodo, ruta, csrf, clave, cuerpo string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(metodo, "https://etf.interno"+ruta, strings.NewReader(cuerpo))
		req.AddCookie(&http.Cookie{Name: "__Host-etf-sesion", Value: "AwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwM"})
		req.Header.Set("Origin", "https://etf.interno")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", csrf)
		req.Header.Set("Idempotency-Key", clave)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w
	}
	var acceso struct {
		CSRF string `json:"csrf"`
	}
	w := pedir("GET", "/api/sesion", "", "", "")
	if err = json.Unmarshal(w.Body.Bytes(), &acceso); err != nil {
		t.Fatal(err)
	}
	cuerpo, _ := json.Marshal(pC)
	ruta := "/api/expedientes/" + ficha.ID + "/cerrar"
	if w := pedir("POST", ruta, "", "cierre-http", string(cuerpo)); w.Code != 403 {
		t.Fatal(w.Code, w.Body.String())
	}
	obsoleta := pC
	obsoleta.Version--
	bc, _ := json.Marshal(obsoleta)
	if w := pedir("POST", ruta, acceso.CSRF, "obsoleta-http", string(bc)); w.Code != 409 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = pedir("POST", ruta, acceso.CSRF, "cierre-http", string(cuerpo))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var cerrado d.ResultadoCierre
	if err = json.Unmarshal(w.Body.Bytes(), &cerrado); err != nil {
		t.Fatal(err)
	}
	if rep := pedir("POST", ruta, acceso.CSRF, "cierre-http", string(cuerpo)); rep.Code != 200 || rep.Body.String() != w.Body.String() {
		t.Fatal(rep.Code, rep.Body.String())
	}
	nuevo, err := s.Reabrir(ctx, otra, ficha.ID, "reabrir-spm", d.PeticionReapertura{Version: cerrado.Version, Fecha: "2026-03-02", Motivo: "Se retoma la atención mientras responde el SPM."})
	if err != nil || nuevo.Estado != d.DerivadaSPM {
		t.Fatal(nuevo, err)
	}
	// La nueva propuesta conserva la falta de consentimiento y el bloqueo SPM.
	nuevaPropuesta, err := s.ProponerCierre(ctx, equipo, ficha.ID, "proponer-spm-reabierto", d.VersionCierre{Version: nuevo.Version})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Cerrar(ctx, otra, ficha.ID, "ordinario-spm-reabierto", d.PeticionCierre{Version: nuevaPropuesta.Version, Fecha: "2026-03-03"}); !errors.Is(err, d.ErrCierreSPM) {
		t.Fatal(err)
	}
	spm, err = s.OperarSPM(ctx, equipo, ficha.ID, "respuesta-spm-reabierto", "responder", d.PeticionSPM{Version: spm.Version, Respuesta: d.SPMMedidaNoNecesaria, Fecha: "2026-03-03"})
	if err != nil {
		t.Fatal(err)
	}
	// La respuesta SPM restaura intervención e incrementa la versión compartida.
	if _, err := s.Cerrar(ctx, otra, ficha.ID, "ordinario-spm-version-antigua", d.PeticionCierre{Version: nuevaPropuesta.Version, Fecha: "2026-03-03"}); err == nil {
		t.Fatal("cierre con versión anterior a la respuesta")
	} else {
		var c d.Conflicto
		if !errors.As(err, &c) || c.VersionActual != nuevaPropuesta.Version+1 {
			t.Fatal(err)
		}
	}
	if _, err := s.Cerrar(ctx, otra, ficha.ID, "ordinario-spm-resuelto", d.PeticionCierre{Version: nuevaPropuesta.Version + 1, Fecha: "2026-03-03"}); err != nil {
		t.Fatal(err)
	}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: otra.UsuarioID, Perfiles: otra.Perfiles}, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		var fecha, motivo string
		if err := tx.QueryRow(ctx, `SELECT to_char(cierre_fecha,'YYYY-MM-DD'),f.motivo FROM expediente.episodio e JOIN expediente.spm_forzado f ON f.id=e.forzado_id WHERE e.id=$1`, cerrado.EpisodioID).Scan(&fecha, &motivo); err != nil {
			return err
		}
		if fecha != "2026-03-01" || motivo == "" {
			t.Fatal(fecha, motivo)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	exportacion, err := postgres.NuevoRegistroAuditoria(b).Exportar(ctx, plataforma.Identidad{UsuarioID: otra.UsuarioID, Perfiles: []string{"administracion"}}, 1, 1000000)
	if err != nil {
		t.Fatal(err)
	}
	forzados := 0
	for _, ev := range exportacion.Eventos {
		if ev.EntidadID == forzadoID && ev.Accion == "expediente.spm_forzar" && ev.Resultado == "ok" {
			forzados++
		}
	}
	if forzados != 1 {
		t.Fatal(forzados)
	}
	t.Log("SPM a la espera bloquea; forzado enlazado a motivo y evento; HTTP real 403 CSRF, 409 versión y 200 idempotente; reapertura DERIVADA A SPM conserva cierre")
}
