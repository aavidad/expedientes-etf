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
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	c "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type relojTraslado struct{ fecha time.Time }

func (r relojTraslado) Ahora() time.Time { return r.fecha }

func prepararTrasladoSeguimiento(t *testing.T) (*postgres.Base, context.Context, app.Dependencias, d.Actor, d.Actor, d.Actor, d.Aceptacion) {
	t.Helper()
	b, ctx, deps, coord, p := prepararAceptacion(t)
	r := &repo.Repositorio{}
	deps.Cierres, deps.Propuestas, deps.SalidasSPM, deps.Hitos, deps.Traslados, deps.Seguimientos = r, r, r, r, r, r
	deps.Familias = r
	deps.Menores = r
	deps.CatalogosMenores = nombres.CatalogosMenores{Ruta: "../../../../idiomas/es/catalogos/menores.json"}
	deps.SMI = &catalogo.Repositorio{}
	deps.CatalogosFamilia = nombres.CatalogosFamilia{Listas: "../../../../idiomas/es/catalogos/listas.json", Familia: "../../../../idiomas/es/catalogos/familia.json", Parametros: "../../../../datos/catalogos/familia.json"}
	deps.MotivosCierre = &catalogo.Repositorio{}
	deps.EmisionesFinalizacion = documentos.Repositorio{}
	deps.ConsultaCierreSPM = app.ConsultaCierreSPM{Repositorio: r, Salidas: r}
	zona, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatal(err)
	}
	deps.Reloj = relojTraslado{time.Now().In(zona)}
	_, _, _, otra := preparar(t)
	equipo := d.Actor{UsuarioID: coord.UsuarioID, EquipoID: p.EquipoID, Perfiles: []string{"profesional_etf"}}
	return b, ctx, deps, coord, otra, equipo, p
}
func aceptarYProponer(t *testing.T, ctx context.Context, deps app.Dependencias, coord, equipo d.Actor, p d.Aceptacion, reunificacion bool, motivo string, coleccion ...[]d.Menor) (d.Derivacion, d.PeticionCierre) {
	t.Helper()
	s := app.Nuevo(deps)
	var ficha d.Derivacion
	if reunificacion {
		contenido, err := os.ReadFile("../../anexos/pdf/testdata/anexo4.pdf")
		if err != nil {
			t.Fatal(err)
		}
		leida, err := s.Subir(ctx, coord, entrada.Subida{Nombre: "Derivacion.pdf", Contenido: contenido})
		if err != nil {
			t.Fatal(err)
		}
		ficha = confirmarLectura(t, ctx, deps, coord, leida)
		p.Subprograma = "REUNIFICACION"
	} else {
		ficha = subir(t, ctx, deps, coord)
	}
	a, err := s.Aceptar(ctx, coord, ficha.ID, "aceptar-"+ficha.ID, conMenores(p, ficha))
	if err != nil {
		t.Fatal(err)
	}
	h, err := s.ConfirmarHito(ctx, equipo, ficha.ID, "consentir-"+ficha.ID, d.ConfirmacionHito{Version: a.Version, Tipo: "consentimiento", Fecha: "2026-02-04"})
	if err != nil {
		t.Fatal(err)
	}
	familia := datosFamilia()
	guardada, err := s.EditarFamilia(ctx, equipo, ficha.ID, "familia-"+ficha.ID, d.CambioFamilia{Version: h.Version, Desde: "2026-02-04", Familia: &familia})
	if err != nil {
		t.Fatal(err)
	}
	adultos := []d.Adulto{{Orden: 1, Nombre: "Teresa", Apellidos: "López Rivas", Papel: "MADRE", Sexo: "M"}}
	guardada, err = s.EditarFamilia(ctx, equipo, ficha.ID, "adultos-"+ficha.ID, d.CambioFamilia{Version: guardada.Version, Desde: "2026-02-04", Adultos: &adultos})
	if err != nil {
		t.Fatal(err)
	}
	if len(coleccion) > 0 {
		guardada, err = s.EditarMenores(ctx, equipo, ficha.ID, "menores-"+ficha.ID, d.CambioMenores{Version: guardada.Version, Desde: "2026-02-04", Menores: &coleccion[0]})
		if err != nil {
			t.Fatal(err)
		}
	}
	final, err := s.GuardarFinalizacion(ctx, equipo, ficha.ID, "final-"+ficha.ID, d.Finalizacion{Version: guardada.Version, Motivo: motivo})
	if err != nil {
		t.Fatal(err)
	}
	propuesta, err := s.ProponerCierre(ctx, equipo, ficha.ID, "proponer-"+ficha.ID, d.VersionCierre{Version: final.Version})
	if err != nil {
		t.Fatal(err)
	}
	return ficha, d.PeticionCierre{Version: propuesta.Version, Fecha: "2026-10-01"}
}
func consultaSQL(t *testing.T, b *postgres.Base, ctx context.Context, a d.Actor, sql string, args []any, destinos ...any) {
	t.Helper()
	err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		return tx.QueryRow(ctx, sql, args...).Scan(destinos...)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestTrasladoPostgresAtomicidadNumeroDestinoRLSYEpisodios(t *testing.T) {
	b, ctx, deps, coord, otra, equipo, p := prepararTrasladoSeguimiento(t)
	s := app.Nuevo(deps)
	ficha, cierre := aceptarYProponer(t, ctx, deps, coord, equipo, p, false, "TRASLADO")
	var destinoEquipo string
	consultaSQL(t, b, ctx, coord, `SELECT id::text FROM catalogo.equipo WHERE codigo='02'`, nil, &destinoEquipo)
	peticion := d.PeticionTraslado{PeticionCierre: cierre, EquipoDestinoID: destinoEquipo}
	if _, err := s.Trasladar(ctx, coord, ficha.ID, "misma-persona", peticion); !errors.Is(err, d.ErrDobleIntervencion) {
		t.Fatal(err)
	}
	if _, err := s.Trasladar(ctx, equipo, ficha.ID, "sin-permiso", peticion); !errors.Is(err, d.ErrAcceso) {
		t.Fatal(err)
	}
	fallo := deps
	fallo.RegistroTransaccional = registroFallido{}
	if _, err := app.Nuevo(fallo).Trasladar(ctx, otra, ficha.ID, "fallo-traslado", peticion); !errors.Is(err, d.ErrNoDisponible) {
		t.Fatal(err)
	}
	var cuenta int
	consultaSQL(t, b, ctx, coord, `SELECT count(*) FROM expediente.traslado WHERE origen_id=$1`, []any{ficha.ID}, &cuenta)
	if cuenta != 0 {
		t.Fatal("traslado parcial", cuenta)
	}
	const n = 6
	var wg sync.WaitGroup
	var rs [n]d.ResultadoTraslado
	var es [n]error
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			rs[i], es[i] = s.Trasladar(ctx, otra, ficha.ID, "traslado-"+ficha.ID, peticion)
		}(i)
	}
	wg.Wait()
	for i := range n {
		if es[i] != nil || !reflect.DeepEqual(rs[i], rs[0]) {
			t.Fatal(i, rs[i], es[i])
		}
	}
	traslado := rs[0]
	if traslado.Cierre.Estado != d.Cerrada || traslado.VersionDestino != 1 {
		t.Fatal(traslado)
	}
	alterada := peticion
	alterada.Fecha = "2026-10-02"
	if _, err := s.Trasladar(ctx, otra, ficha.ID, "traslado-"+ficha.ID, alterada); !errors.Is(err, d.ErrIdempotencia) {
		t.Fatal(err)
	}
	destino, err := s.Consultar(ctx, coord, traslado.DestinoID)
	if err != nil || destino.Estado != d.DerivacionRecibida || destino.Numero != nil || destino.EquipoID != nil || destino.ServicioDerivante != "OTRO_ETF" || destino.Traslado == nil {
		t.Fatal(destino, err)
	}
	nuevoP := conMenores(p, destino)
	nuevoP.EquipoID = destinoEquipo
	nuevoP.FechaActa = "2026-10-02"
	incorrecta := nuevoP
	incorrecta.EquipoID = p.EquipoID
	if _, err = s.Aceptar(ctx, otra, destino.ID, "equipo-equivocado", incorrecta); !errors.Is(err, d.ErrTraslado) {
		t.Fatal(err)
	}
	a, err := s.Aceptar(ctx, otra, destino.ID, "aceptar-destino", nuevoP)
	if err != nil || !strings.HasPrefix(a.Numero, "02-") {
		t.Fatal(a, err)
	}
	if _, err = s.Consultar(ctx, equipo, destino.ID); !errors.Is(err, d.ErrNoEncontrada) {
		t.Fatal("origen ve destino", err)
	}
	destinoActor := equipo
	destinoActor.EquipoID = destinoEquipo
	if _, err = s.Consultar(ctx, destinoActor, destino.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Consultar(ctx, destinoActor, ficha.ID); !errors.Is(err, d.ErrNoEncontrada) {
		t.Fatal("destino ve ficha origen", err)
	}
	copia, err := s.ConsultarFamilia(ctx, destinoActor, destino.ID)
	if err != nil || copia.Familia.Direccion != "Calle Olivo 12" || len(copia.Adultos) != 1 || copia.Adultos[0].Nombre != "Teresa" {
		t.Fatal(copia, err)
	}
	// El puerto de documentos recibe la misma copia vigente, no el anexo antiguo.
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: otra.UsuarioID, Perfiles: otra.Perfiles}, func(ctx context.Context) error {
		datos, err := (&repo.Repositorio{}).ParaDocumentos(ctx, destino.ID)
		if err != nil {
			return err
		}
		if datos.CopiaTraslado == nil || datos.CopiaTraslado.Familia.Direccion != "Calle Olivo 12" || len(datos.CopiaTraslado.Adultos) != 1 || datos.CopiaTraslado.Adultos[0].Nombre != "Teresa" {
			t.Fatal("documento sin la copia vigente", datos.CopiaTraslado)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	fuente, err := s.ConsultarFamilia(ctx, equipo, ficha.ID)
	if err != nil || len(fuente.Adultos) != 1 || fuente.Adultos[0].ID == copia.Adultos[0].ID {
		t.Fatal(fuente, err)
	}
	var episodios, consentimientos int
	consultaSQL(t, b, ctx, coord, `SELECT count(*),(SELECT count(*) FROM expediente.hito WHERE expediente_id=$1 AND tipo='consentimiento') FROM expediente.episodio WHERE expediente_id=$1 AND cierre_fecha IS NULL`, []any{destino.ID}, &episodios, &consentimientos)
	if episodios != 1 || consentimientos != 0 {
		t.Fatal(episodios, consentimientos)
	}
	var fecha, motivo string
	consultaSQL(t, b, ctx, coord, `SELECT to_char(cierre_fecha,'YYYY-MM-DD'),motivo_cierre FROM expediente.episodio WHERE expediente_id=$1`, []any{ficha.ID}, &fecha, &motivo)
	if fecha != cierre.Fecha || motivo != "TRASLADO" {
		t.Fatal(fecha, motivo)
	}
	consultaSQL(t, b, ctx, equipo, `SELECT count(*) FROM expediente.derivacion_datos WHERE expediente_id=$1`, []any{destino.ID}, &cuenta)
	if cuenta != 0 {
		t.Fatal("RLS permite datos nuevos", cuenta)
	}
	// La modificación del destino no cambia el snapshot ni los datos de origen.
	familia := datosFamilia()
	familia.Direccion = "Calle Almendro 8"
	editado, err := s.EditarFamilia(ctx, destinoActor, destino.ID, "editar-destino", d.CambioFamilia{Version: a.Version, Desde: "2026-10-02", Familia: &familia})
	if err != nil {
		t.Fatal(editado, err)
	}
	var original []byte
	consultaSQL(t, b, ctx, equipo, `SELECT familia FROM expediente.expediente WHERE id=$1`, []any{ficha.ID}, &original)
	var origenFamilia d.Familia
	if err = json.Unmarshal(original, &origenFamilia); err != nil {
		t.Fatal(err)
	}
	if origenFamilia.Direccion != "Calle Olivo 12" {
		t.Fatal("origen alterado", string(original))
	}
	exportacion, err := postgres.NuevoRegistroAuditoria(b).Exportar(ctx, plataforma.Identidad{UsuarioID: otra.UsuarioID, Perfiles: []string{"administracion"}}, 1, 1000000)
	if err != nil {
		t.Fatal(err)
	}
	cuenta = 0
	for _, ev := range exportacion.Eventos {
		if ev.ExpedienteID == ficha.ID && ev.Accion == "expediente.trasladar" && ev.Resultado == "ok" {
			cuenta++
		}
	}
	if cuenta != 1 {
		t.Fatal("auditoria duplicada", cuenta)
	}
	// Un equipo intermedio conserva sus enlaces de entrada y salida.
	hito, err := s.ConfirmarHito(ctx, destinoActor, destino.ID, "consentimiento-destino", d.ConfirmacionHito{Version: editado.Version, Tipo: "consentimiento", Fecha: "2026-10-03"})
	if err != nil {
		t.Fatal(err)
	}
	final, err := s.GuardarFinalizacion(ctx, destinoActor, destino.ID, "final-destino", d.Finalizacion{Version: hito.Version, Motivo: "TRASLADO"})
	if err != nil {
		t.Fatal(err)
	}
	propuesta, err := s.ProponerCierre(ctx, destinoActor, destino.ID, "proponer-destino", d.VersionCierre{Version: final.Version})
	if err != nil {
		t.Fatal(err)
	}
	var tercerEquipo string
	consultaSQL(t, b, ctx, coord, `SELECT id::text FROM catalogo.equipo WHERE codigo='03'`, nil, &tercerEquipo)
	siguiente, err := s.Trasladar(ctx, otra, destino.ID, "segundo-traslado", d.PeticionTraslado{PeticionCierre: d.PeticionCierre{Version: propuesta.Version, Fecha: "2026-10-14"}, EquipoDestinoID: tercerEquipo})
	if err != nil {
		t.Fatal(err)
	}
	intermedio, err := s.Consultar(ctx, destinoActor, destino.ID)
	if err != nil || len(intermedio.Traslados) != 2 || intermedio.Traslados[0].DestinoID != siguiente.DestinoID || intermedio.Traslados[1].OrigenID != ficha.ID {
		t.Fatal("cadena incompleta", intermedio.Traslados, err)
	}
	if _, err = s.Consultar(ctx, equipo, siguiente.DestinoID); !errors.Is(err, d.ErrNoEncontrada) {
		t.Fatal("origen ve tercer expediente", err)
	}
	t.Log("traslado atómico, seis repeticiones iguales, número del destino, RLS sin datos nuevos en origen, episodios y ambos enlaces en traslado encadenado")
}

func TestTrasladoHTTP409CSRFYEntradaEstricta(t *testing.T) {
	_, ctx, deps, coord, otra, equipo, p := prepararTrasladoSeguimiento(t)
	s := app.Nuevo(deps)
	ficha, cierre := aceptarYProponer(t, ctx, deps, coord, equipo, p, false, "TRASLADO")
	h := httpETF.Nuevo(nil, nil, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, httpETF.Opciones{Sesion: httpETF.ConfiguracionSesion{Servicio: sesionSalidaSPM{actor: otra}, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}, Expedientes: httpETF.ConfiguracionAceptaciones{TrasladoSeguimiento: s}})
	pedir := func(ruta, csrf, clave, cuerpo string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "https://etf.interno"+ruta, strings.NewReader(cuerpo))
		if ruta == "/api/sesion" {
			req.Method = "GET"
		}
		req.AddCookie(&http.Cookie{Name: "__Host-etf-sesion", Value: "AwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwM"})
		req.Header.Set("Origin", "https://etf.interno")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", csrf)
		req.Header.Set("Idempotency-Key", clave)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	var acceso struct {
		CSRF string `json:"csrf"`
	}
	w := pedir("/api/sesion", "", "", "")
	if err := json.Unmarshal(w.Body.Bytes(), &acceso); err != nil {
		t.Fatal(err)
	}
	ruta := "/api/expedientes/" + ficha.ID + "/trasladar"
	peticion := d.PeticionTraslado{PeticionCierre: cierre, EquipoDestinoID: p.EquipoID}
	peticion.Version--
	b, _ := json.Marshal(peticion)
	if w = pedir(ruta, "", "sin-csrf", string(b)); w.Code != 403 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w = pedir(ruta, acceso.CSRF, "version-obsoleta", string(b)); w.Code != 409 || !strings.Contains(w.Body.String(), "version_actual") {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, cuerpo := range []string{`{"version":2,"perfiles":["coordinacion"]}`, `{} {}`, `{`} {
		if w = pedir(ruta, acceso.CSRF, "json-invalido", cuerpo); w.Code != 400 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	t.Log("HTTP con PostgreSQL: 409 y versión actual, CSRF obligatorio y JSON estricto")
}

func TestTrasladoPostgresAceptacionDesdeEspera(t *testing.T) {
	b, ctx, deps, coord, otra, equipo, p := prepararTrasladoSeguimiento(t)
	s := app.Nuevo(deps)
	ficha, cierre := aceptarYProponer(t, ctx, deps, coord, equipo, p, false, "TRASLADO")
	var destinoEquipo string
	consultaSQL(t, b, ctx, coord, `SELECT id::text FROM catalogo.equipo WHERE codigo='02'`, nil, &destinoEquipo)
	r, err := s.Trasladar(ctx, otra, ficha.ID, "traslado-espera", d.PeticionTraslado{PeticionCierre: cierre, EquipoDestinoID: destinoEquipo})
	if err != nil {
		t.Fatal(err)
	}
	deps.Decisiones = &repo.Repositorio{}
	s = app.Nuevo(deps)
	espera, err := s.DejarEnEspera(ctx, otra, r.DestinoID, "espera-traslado", d.Espera{Version: 1})
	if err != nil {
		t.Fatal(err)
	}
	destino, err := s.Consultar(ctx, coord, r.DestinoID)
	if err != nil {
		t.Fatal(err)
	}
	p = conMenores(p, destino)
	p.Version = espera.Version
	p.EquipoID = destinoEquipo
	p.FechaActa = "2026-10-14"
	a, err := s.Aceptar(ctx, otra, destino.ID, "aceptar-traslado-espera", p)
	if err != nil || a.Version != 3 {
		t.Fatal(a, err)
	}
	destinoActor := equipo
	destinoActor.EquipoID = destinoEquipo
	familia, err := s.ConsultarFamilia(ctx, destinoActor, destino.ID)
	if err != nil || len(familia.Adultos) != 1 || familia.Adultos[0].Nombre != "Teresa" {
		t.Fatal(familia, err)
	}
	t.Log("destino en lista de espera: aceptación con versión 3 materializa copia familiar y adultos")
}

func TestTrasladoPostgresCopiaMenoresVigentesConAltasYBajas(t *testing.T) {
	for _, cantidad := range []int{1, 2, 3} {
		t.Run(strconv.Itoa(cantidad), func(t *testing.T) {
			b, ctx, deps, coord, otra, equipo, p := prepararTrasladoSeguimiento(t)
			s := app.Nuevo(deps)
			si := true
			menores := []d.Menor{
				{Orden: 1, Nombre: "Lucía", Apellidos: "Moreno Castillo", Sexo: "M", Nacimiento: texto("2016-05-14"), Escolarizacion: texto("PRIMARIA"), Subprograma: "RIESGO_SOCIAL"},
				{Orden: 2, Nombre: "Pablo", Apellidos: "Moreno Castillo", Sexo: "H", Subprograma: "RIESGO_SOCIAL"},
				{Orden: 3, Nombre: "Irene", Apellidos: "Moreno Castillo", Sexo: "M", Subprograma: "REUNIFICACION", AtencionTemprana: &si},
			}[:cantidad]
			if cantidad == 2 {
				// Identidades nuevas sin nombres: los textos del anexo anterior
				// no deben atribuirse a menores añadidos o reordenados.
				for i := range menores {
					menores[i].Nombre, menores[i].Apellidos = "", ""
					menores[i].Nacimiento, menores[i].Escolarizacion = nil, nil
				}
			}
			ficha, cierre := aceptarYProponer(t, ctx, deps, coord, equipo, p, false, "TRASLADO", menores)
			origen, err := s.ConsultarMenores(ctx, equipo, ficha.ID)
			if err != nil {
				t.Fatal(err)
			}
			var destinoEquipo string
			consultaSQL(t, b, ctx, coord, `SELECT id::text FROM catalogo.equipo WHERE codigo='02'`, nil, &destinoEquipo)
			traslado, err := s.Trasladar(ctx, otra, ficha.ID, "traslado-menores", d.PeticionTraslado{PeticionCierre: cierre, EquipoDestinoID: destinoEquipo})
			if err != nil {
				t.Fatal(err)
			}
			destino, err := s.Consultar(ctx, coord, traslado.DestinoID)
			if err != nil {
				t.Fatal(err)
			}
			p = conMenores(p, destino)
			p.EquipoID, p.FechaActa = destinoEquipo, "2026-10-14"
			for i := range p.Menores {
				p.Menores[i].Sexo = menores[i].Sexo
			}
			if len(p.Menores) != cantidad {
				t.Fatal("colección del anexo antiguo", p.Menores)
			}
			// La confirmación puede llegar en un orden distinto al de la copia.
			// Cada sexo debe seguir al orden del menor, no a su posición JSON.
			for i, j := 0, len(p.Menores)-1; i < j; i, j = i+1, j-1 {
				p.Menores[i], p.Menores[j] = p.Menores[j], p.Menores[i]
			}
			if _, err = s.Aceptar(ctx, otra, destino.ID, "aceptar-menores-traslado", p); err != nil {
				t.Fatal(err)
			}
			destinoActor := equipo
			destinoActor.EquipoID = destinoEquipo
			copia, err := s.ConsultarMenores(ctx, destinoActor, destino.ID)
			if err != nil || len(copia.Menores) != cantidad {
				t.Fatal(copia, err)
			}
			for i, m := range copia.Menores {
				if m.ID == origen.Menores[i].ID {
					t.Fatal("identidad compartida")
				}
				m.ID = origen.Menores[i].ID
				if !reflect.DeepEqual(m, origen.Menores[i]) {
					t.Fatal("datos perdidos", i)
				}
			}
			if _, err = s.ConsultarMenores(ctx, equipo, destino.ID); !errors.Is(err, d.ErrNoEncontrada) {
				t.Fatal("origen ve menores destino", err)
			}
			t.Log("menores vigentes copiados con identidades nuevas, historia propia y RLS del destino")
		})
	}
}

// PublicarPlazo usa el adaptador del catálogo: ninguna prueba del módulo
// expediente escribe directamente en tablas de otro módulo.
func publicarPlazo(t *testing.T, b *postgres.Base, ctx context.Context, a d.Actor, clave, unidad string, valor int) {
	t.Helper()
	a.Perfiles = []string{"administracion"}
	err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		r := &catalogo.Repositorio{}
		ultima, err := r.Ultima(ctx, "plazos")
		if err != nil {
			return err
		}
		_, err = r.Publicar(ctx, "plazos", c.Cambio{VersionEsperada: ultima.Version.Numero, VigenteDesde: ultima.Version.VigenteDesde.AddDate(0, 0, 1), Plazo: &c.Plazo{Clave: clave, Unidad: unidad, Valor: valor}})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSeguimientoPostgresVigenciaFinAnticipadoReaperturaYAuditoria(t *testing.T) {
	b, ctx, deps, coord, otra, equipo, p := prepararTrasladoSeguimiento(t)
	s := app.Nuevo(deps)
	ficha, cierre := aceptarYProponer(t, ctx, deps, coord, equipo, p, true, "OBJ_TOTAL")
	cerrado, err := s.Cerrar(ctx, otra, ficha.ID, "cerrar-seguimiento", cierre)
	if err != nil {
		t.Fatal(err)
	}
	peticion := d.PeticionSeguimiento{Version: cerrado.Version, Fecha: "2026-10-01"}
	fallo := deps
	fallo.RegistroTransaccional = registroFallido{}
	if _, err = app.Nuevo(fallo).Seguimiento(ctx, equipo, ficha.ID, "fallo-inicio", "iniciar", peticion); !errors.Is(err, d.ErrNoDisponible) {
		t.Fatal(err)
	}
	if r, err := s.ConsultarSeguimiento(ctx, equipo, ficha.ID); err != nil || r != nil {
		t.Fatal(r, err)
	}
	const n = 6
	var wg sync.WaitGroup
	var rs [n]d.Seguimiento
	var es [n]error
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			rs[i], es[i] = s.Seguimiento(ctx, equipo, ficha.ID, "iniciar-seguimiento", "iniciar", peticion)
		}(i)
	}
	wg.Wait()
	for i := range n {
		if es[i] != nil || rs[i] != rs[0] {
			t.Fatal(i, rs[i], es[i])
		}
	}
	r := rs[0]
	if r.FechaFin != "2027-10-01" || r.Version != 1 {
		t.Fatal(r)
	}
	ajeno := equipo
	ajeno.EquipoID = "88888888-8888-4888-8888-888888888888"
	if _, err = s.ConsultarSeguimiento(ctx, ajeno, ficha.ID); !errors.Is(err, d.ErrNoEncontrada) {
		t.Fatal(err)
	}
	if avisos, err := s.ConsultarAvisosSeguimiento(ctx, equipo); err != nil || len(avisos) != 0 {
		t.Fatal(avisos, err)
	}
	proximo := deps
	proximo.Reloj = relojTraslado{time.Date(2027, 9, 20, 12, 0, 0, 0, time.UTC)}
	avisos, err := app.Nuevo(proximo).ConsultarAvisosSeguimiento(ctx, equipo)
	if err != nil || len(avisos) != 1 || avisos[0].Desde != "2027-09-01" || avisos[0].Hasta != "2027-10-01" {
		t.Fatal(avisos, err)
	}
	proximo.Reloj = relojTraslado{time.Date(2027, 10, 1, 12, 0, 0, 0, time.UTC)}
	if avisos, err = app.Nuevo(proximo).ConsultarAvisosSeguimiento(ctx, equipo); err != nil || len(avisos) != 0 {
		t.Fatal("aviso fuera de vigencia", avisos, err)
	}
	fin := d.PeticionSeguimiento{Version: 1, Fecha: "2026-10-14", Motivo: "CAMBIO_SITUACION"}
	finInvalido := fin
	finInvalido.Motivo = "NO_PUBLICADO"
	if _, err = s.Seguimiento(ctx, equipo, ficha.ID, "motivo-no-publicado", "terminar", finInvalido); !errors.Is(err, d.ErrSeguimiento) {
		t.Fatal(err)
	}
	r, err = s.Seguimiento(ctx, equipo, ficha.ID, "terminar-seguimiento", "terminar", fin)
	if err != nil || r.Version != 2 || r.Estado != "terminado" {
		t.Fatal(r, err)
	}
	reapertura := d.PeticionSeguimiento{Version: 2, Motivo: "La familia solicita retomar el acompañamiento."}
	if _, err = s.Seguimiento(ctx, equipo, ficha.ID, "reabrir-sin-permiso", "reabrir", reapertura); !errors.Is(err, d.ErrAcceso) {
		t.Fatal(err)
	}
	r, err = s.Seguimiento(ctx, otra, ficha.ID, "reabrir-seguimiento", "reabrir", reapertura)
	hoy := deps.Reloj.Ahora().Format("2006-01-02")
	finReapertura := finCalendarioEsperado(deps.Reloj.Ahora(), 12)
	if err != nil || r.Version != 3 || r.Estado != "activo" || r.TramoDesde != hoy || r.FechaFin != finReapertura || r.MotivoReapertura != reapertura.Motivo {
		t.Fatal(r, err)
	}
	retroactiva := d.PeticionSeguimiento{Version: 3, Fecha: "2026-10-13", Motivo: "CAMBIO_SITUACION"}
	if _, err = s.Seguimiento(ctx, equipo, ficha.ID, "fin-anterior-reapertura", "terminar", retroactiva); !errors.Is(err, d.ErrSeguimiento) {
		t.Fatal("intervalo invertido", err)
	}
	obsoleta := d.PeticionSeguimiento{Version: 1, Fecha: "2026-10-14", Motivo: "CAMBIO_SITUACION"}
	if _, err = s.Seguimiento(ctx, equipo, ficha.ID, "obsoleta-seguimiento", "terminar", obsoleta); err == nil {
		t.Fatal("falta 409")
	} else {
		var c d.Conflicto
		if !errors.As(err, &c) || c.VersionActual != 3 {
			t.Fatal(err)
		}
	}
	var historia, tramos int
	consultaSQL(t, b, ctx, equipo, `SELECT count(*) FROM expediente.seguimiento_historia WHERE expediente_id=$1`, []any{ficha.ID}, &historia)
	consultaSQL(t, b, ctx, equipo, `SELECT count(*) FROM expediente.consulta_seguimientos WHERE expediente_id=$1`, []any{ficha.ID}, &tramos)
	if historia != 3 || tramos != 2 {
		t.Fatal(historia, tramos)
	}
	lecturaFallida := deps
	lecturaFallida.RegistroTransaccional = registroFallido{}
	if datos, err := app.Nuevo(lecturaFallida).ConsultarAvisosSeguimiento(ctx, equipo); !errors.Is(err, d.ErrNoDisponible) || datos != nil {
		t.Fatal("datos sin auditoria", datos, err)
	}
	familia := datosFamilia()
	if _, err = s.EditarFamilia(ctx, equipo, ficha.ID, "editar-durante-seguimiento", d.CambioFamilia{Version: cerrado.Version, Desde: "2026-10-14", Familia: &familia}); !errors.Is(err, d.ErrEstado) {
		t.Fatal(err)
	}
	if err = s.VencerSeguimientos(ctx); err != nil {
		t.Fatal(err)
	}
	actual, err := s.ConsultarSeguimiento(ctx, equipo, ficha.ID)
	if err != nil || actual.Version != 3 {
		t.Fatal("vencimiento prematuro", actual, err)
	}
	t.Log("seguimiento: catálogo versionado, avisos [desde,hasta), motivo publicado, seis repeticiones, 409, RLS, fin anticipado y reapertura con historia")
}

func TestSeguimientoPostgresVencimientoAutomaticoAtomico(t *testing.T) {
	b, ctx, deps, coord, otra, equipo, p := prepararTrasladoSeguimiento(t)
	defer publicarPlazo(t, b, ctx, coord, "seguimiento_tras_cese", "meses", 12)
	s := app.Nuevo(deps)
	ficha, cierre := aceptarYProponer(t, ctx, deps, coord, equipo, p, true, "OBJ_TOTAL")
	cerrado, err := s.Cerrar(ctx, otra, ficha.ID, "cerrar-vencimiento", cierre)
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.Seguimiento(ctx, equipo, ficha.ID, "seguimiento-vencido", "iniciar", d.PeticionSeguimiento{Version: cerrado.Version, Fecha: "2025-09-01"})
	if err != nil || r.FechaFin != "2026-09-01" {
		t.Fatal(r, err)
	}
	fallo := deps
	fallo.RegistroTransaccional = registroFallido{}
	if err = app.Nuevo(fallo).VencerSeguimientos(ctx); !errors.Is(err, d.ErrNoDisponible) {
		t.Fatal(err)
	}
	actual, err := s.ConsultarSeguimiento(ctx, equipo, ficha.ID)
	if err != nil || actual.Version != 1 || actual.Estado != "activo" {
		t.Fatal("vencimiento parcial", actual, err)
	}
	const n = 4
	var wg sync.WaitGroup
	var es [n]error
	for i := range n {
		wg.Add(1)
		go func(i int) { defer wg.Done(); es[i] = s.VencerSeguimientos(ctx) }(i)
	}
	wg.Wait()
	for _, err := range es {
		if err != nil {
			t.Fatal(err)
		}
	}
	actual, err = s.ConsultarSeguimiento(ctx, equipo, ficha.ID)
	if err != nil || actual.Version != 2 || actual.Estado != "terminado" || actual.FechaTerminacion != "2026-09-01" || actual.Motivo != "PLAZO_CUMPLIDO" {
		t.Fatal(actual, err)
	}
	var historia int
	consultaSQL(t, b, ctx, equipo, `SELECT count(*) FROM expediente.seguimiento_historia WHERE expediente_id=$1`, []any{ficha.ID}, &historia)
	if historia != 2 {
		t.Fatal(historia)
	}
	// Tras el vencimiento, la reapertura aplica el plazo vigente completo y
	// conserva el tramo anterior para la estadística.
	publicarPlazo(t, b, ctx, coord, "seguimiento_tras_cese", "meses", 2)
	reapertura := d.PeticionSeguimiento{Version: 2, Motivo: "La familia necesita retomar el seguimiento tras el cese."}
	fallo = deps
	fallo.RegistroTransaccional = registroFallido{}
	if _, err = app.Nuevo(fallo).Seguimiento(ctx, otra, ficha.ID, "reapertura-fallida", "reabrir", reapertura); !errors.Is(err, d.ErrNoDisponible) {
		t.Fatal(err)
	}
	actual, err = s.ConsultarSeguimiento(ctx, equipo, ficha.ID)
	if err != nil || actual.Version != 2 || actual.Estado != "terminado" {
		t.Fatal("reapertura parcial", actual, err)
	}
	zona, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatal(err)
	}
	// El servidor inyecta la fecha civil de Madrid: a las 23:30 UTC ya es
	// el día siguiente. SQL y Go deben calcular el mismo fin de plazo.
	fechaPublicacion := deps.Reloj.Ahora()
	deps.Reloj = relojTraslado{time.Date(fechaPublicacion.Year(), fechaPublicacion.Month(), fechaPublicacion.Day(), 23, 30, 0, 0, time.UTC).In(zona)}
	s = app.Nuevo(deps)
	reabierto, err := s.Seguimiento(ctx, otra, ficha.ID, "reabrir-vencido", "reabrir", reapertura)
	nuevoDesde := deps.Reloj.Ahora().Format("2006-01-02")
	nuevoFin := finCalendarioEsperado(deps.Reloj.Ahora(), 2)
	if err != nil || reabierto.Version != 3 || reabierto.Estado != "activo" || reabierto.FechaFin != nuevoFin || reabierto.TramoDesde != nuevoDesde || reabierto.ReglaID == r.ReglaID || reabierto.FechaCese != r.FechaCese {
		t.Fatal(reabierto, err)
	}
	repetido, err := s.Seguimiento(ctx, otra, ficha.ID, "reabrir-vencido", "reabrir", reapertura)
	if err != nil || repetido != reabierto {
		t.Fatal("reapertura duplicada", repetido, err)
	}
	consultaSQL(t, b, ctx, equipo, `SELECT count(*) FROM expediente.seguimiento_historia WHERE expediente_id=$1`, []any{ficha.ID}, &historia)
	var tramos int
	consultaSQL(t, b, ctx, equipo, `SELECT count(*) FROM expediente.consulta_seguimientos WHERE expediente_id=$1 AND ((vigente_desde='2025-09-01' AND vigente_hasta='2026-09-01') OR (vigente_desde=$2::date AND vigente_hasta=$3::date))`, []any{ficha.ID, nuevoDesde, nuevoFin}, &tramos)
	if historia != 3 || tramos != 2 {
		t.Fatal("historia perdida", historia, tramos)
	}
	if err = s.VencerSeguimientos(ctx); err != nil {
		t.Fatal(err)
	}
	actual, err = s.ConsultarSeguimiento(ctx, equipo, ficha.ID)
	if err != nil || *actual != reabierto {
		t.Fatal("vencimiento con el plazo anterior", actual, err)
	}
	for _, actor := range []d.Actor{coord, equipo} {
		err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: actor.Perfiles, EquipoID: actor.EquipoID}, func(ctx context.Context) error {
			tx, _ := postgres.DesdeContexto(ctx)
			_, err := tx.Exec(ctx, `SELECT * FROM expediente.vencer_seguimientos()`)
			return err
		})
		if err == nil {
			t.Fatal("vencimiento técnico admitió actor humano")
		}
	}
	t.Log("vencimiento automático: cuatro ejecutores, un acto; reapertura tras vencimiento con regla vigente, plazo completo, idempotencia e historia; auditoría atómica")
}

func finCalendarioEsperado(desde time.Time, meses int) string {
	fin := desde.AddDate(0, meses, 0)
	if fin.Day() != desde.Day() {
		fin = time.Date(fin.Year(), fin.Month(), 0, 12, 0, 0, 0, fin.Location())
	}
	return fin.Format("2006-01-02")
}
