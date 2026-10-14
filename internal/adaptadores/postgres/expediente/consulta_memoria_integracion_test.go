package expediente_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	word "github.com/aavidad/expedientes-etf/internal/adaptadores/docx/documento"
	nombres "github.com/aavidad/expedientes-etf/internal/adaptadores/expediente/ficheros"
	plantillas "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/documento"
	reglas "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/estadistica"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	actividad "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/actividad"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	documento "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/documento"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/estadistica"
	exp "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	docApp "github.com/aavidad/expedientes-etf/internal/aplicacion/documento"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica"
	familia "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"github.com/jackc/pgx/v5/pgconn"
	"io"
	"os"
	"strings"
	"testing"
)

func TestMemoriaConsultaCalculoFotoRLSYWordSinTextos(t *testing.T) {
	if os.Getenv("ETF_POSTGRES_INTEGRACION") == "1" && os.Getenv("ETF_CALCULO_MEMORIA_CONTRASENA") == "" {
		t.Skip("ETF_CALCULO_MEMORIA_CONTRASENA; ensayo completo en migraciones/estadistica/ensayar.py")
	}
	b, ctx, deps, actor, p := prepararAceptacion(t)
	subida := subir(t, ctx, deps, actor)
	p = conMenores(p, subida)
	aceptada, err := familia.Nuevo(deps).Aceptar(ctx, actor, subida.ID, "consulta-"+subida.ID, p)
	if err != nil {
		t.Fatal(err)
	}
	// FX1 alimenta ambas ramas del motor con las mismas colecciones históricas.
	deps.Menores = &exp.Repositorio{}
	deps.CatalogosMenores = nombres.CatalogosMenores{Ruta: "../../../../idiomas/es/catalogos/menores.json"}
	deps.Intervencion = &exp.Repositorio{}
	deps.CatalogosIntervencion = &catalogo.Repositorio{}
	equipoActor := actor
	equipoActor.Perfiles = []string{"profesional_etf"}
	equipoActor.EquipoID = p.EquipoID
	menoresNuevos := datosMenores()
	edicion, err := familia.Nuevo(deps).EditarMenores(ctx, equipoActor, aceptada.ID, "consulta-menores-"+subida.ID, dominio.CambioMenores{Version: 2, Desde: "2026-02-03", Menores: &menoresNuevos})
	if err != nil {
		t.Fatal(err)
	}
	intervencion := datosIntervencion()
	_, err = familia.Nuevo(deps).EditarIntervencion(ctx, equipoActor, aceptada.ID, "consulta-intervencion-"+subida.ID, dominio.CambioIntervencion{Version: edicion.Version, Desde: "2026-02-03", Intervencion: &intervencion})
	if err != nil {
		t.Fatal(err)
	}
	coordinacion := d.ActorMemoria{UsuarioID: actor.UsuarioID, Perfiles: []string{"coordinacion"}}
	consulta := d.ActorMemoria{UsuarioID: actor.UsuarioID, Perfiles: []string{"consulta"}}
	f := d.FiltroMemoria{Anio: 2026, EquipoID: p.EquipoID}
	persistencia := &repo.Repositorio{}
	fuente := app.FuenteMemoria{Expedientes: &exp.LecturaEstadistica{}, Reglas: reglas.Reglas{Raiz: "../../../../datos/catalogos/memoria", SMI: &catalogo.Repositorio{}}, Actividad: &actividad.Repositorio{}, Equipos: &catalogo.Repositorio{}, Narrativas: persistencia}
	cierres := app.NuevoCierre(app.DependenciasCierre{Repositorio: persistencia, Fuente: fuente, Transacciones: b, Auditoria: postgres.NuevoRegistroAuditoria(b), Reloj: relojMemoria{}})
	estado, err := cierres.EstadoMemoria(ctx, coordinacion, f)
	if err != nil {
		t.Fatal(err)
	}
	if estado.Estado == "cerrada" {
		_, err = cierres.ReabrirMemoria(ctx, coordinacion, f, "consulta-reabrir-"+subida.ID, d.ActoMemoria{Version: estado.Version, Motivo: "Se revisan las cifras de atención de este año."})
		if err != nil {
			t.Fatal(err)
		}
	}
	textos := app.NuevoNarrativas(app.DependenciasNarrativas{Repositorio: persistencia, Transacciones: b, Auditoria: postgres.NuevoRegistroAuditoria(b), Reloj: relojMemoria{}})
	n, err := textos.LeerNarrativas(ctx, coordinacion, f)
	if err != nil {
		t.Fatal(err)
	}
	privado := "Se acordó acompañar a la familia de Ana Beltrán en sus visitas."
	n.A.EntidadLocal = privado
	n.A.Denominacion = privado
	n.A.Observaciones = privado
	n.D = privado
	n.E = privado
	n.F = privado
	n.G = privado
	_, err = textos.GuardarNarrativas(ctx, coordinacion, n)
	if err != nil {
		t.Fatal(err)
	}
	tecnico, err := postgres.NuevaCalculoMemoria(ctx, os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	defer tecnico.Cerrar()
	if err := tecnico.Comprobar(ctx); err != nil {
		t.Fatal("conexión técnica", err)
	}
	fuenteTecnica := app.FuenteMemoria{Expedientes: &exp.LecturaEstadistica{CalculoInterno: true}, Reglas: fuente.Reglas, Equipos: &catalogo.Repositorio{}, Actividad: &actividad.Repositorio{CalculoInterno: true}}
	ficheros := plantillas.Plantillas{Raiz: "../../../../idiomas", Paquetes: postgres.NuevaPublicacionPaquetes(b)}
	dep := app.DependenciasConsulta{Cerradas: persistencia, Calculo: app.EjecutorConsulta{Transacciones: tecnico, Fuente: fuenteTecnica}, Word: docApp.WordCifras{Repositorio: documento.Repositorio{}, Idiomas: ficheros, Ficheros: ficheros, Textos: ficheros, Motor: word.Motor{}, Metadatos: reglas.TextosCifras{Raiz: "../../../../idiomas"}}, Transacciones: b, Auditoria: postgres.NuevoRegistroAuditoria(b), Reloj: relojMemoria{}}
	svc := app.NuevoConsulta(dep)
	abierta, err := svc.ConsultarCifras(ctx, consulta, f)
	if err != nil {
		t.Fatal("cálculo real", err)
	}
	if abierta.Definitivas || abierta.Fecha.IsZero() || len(abierta.Apartados) != 33 || abierta.EquipoCodigo != "08" {
		t.Fatal("contrato abierto", abierta)
	}
	var informe d.InformeMemoria
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: coordinacion.Perfiles}, func(ctx context.Context) error {
		var e error
		informe, e = fuente.MemoriaEnTransaccion(ctx, coordinacion, f, abierta.Fecha)
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
	porCodigo := map[d.Apartado]d.ApartadoMemoria{}
	for _, a := range informe.Apartados {
		porCodigo[a.Codigo] = a
	}
	for _, a := range abierta.Apartados {
		esperado := porCodigo[a.Codigo].Total
		if a.Total.Familias != esperado.Familias || a.Total.Menores != esperado.Menores || a.Total.Comisiones != esperado.Comisiones || a.Total.Informes != esperado.Informes {
			t.Fatal("motor diferente", a.Codigo)
		}
	}
	if porCodigo[d.CasosAtendidos].Total.Menores < 2 {
		t.Fatal("FX1 no incorpora menores")
	}
	intervencionLeida := false
	for _, expediente := range informe.Fuentes {
		if expediente.ID == aceptada.ID {
			for _, revision := range expediente.Factores {
				for _, factor := range revision.Dato.Valor {
					if factor.Codigo == "ABSENTISMO" {
						intervencionLeida = true
					}
				}
			}
		}
	}
	if !intervencionLeida {
		t.Fatal("FX1 no incorpora intervención")
	}
	if porCodigo[d.CasosAtendidos].Total.Familias < 1 {
		t.Fatal("control positivo sin familias")
	}
	comprobar := func(r d.CifrasConsulta) {
		t.Helper()
		j, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range []string{privado, aceptada.ID, "Narrativas", "Fuentes", "Composicion", "HuellaFuentes", "HuellasComposicion", "Moreno Castillo", "Centro de acogida de la Vega", "Servicio de apoyo de la Vega"} {
			if strings.Contains(string(j), s) {
				t.Fatal("filtración JSON", s)
			}
		}
	}
	comprobar(abierta)
	provincial, err := svc.ConsultarCifras(ctx, consulta, d.FiltroMemoria{Anio: 2026})
	if err != nil || len(provincial.Apartados) != 33 || provincial.EquipoID != "" {
		t.Fatal("provincia", err)
	}
	comprobar(provincial)
	for _, tabla := range []string{"expediente.expediente", "expediente.derivacion_datos", "expediente.episodio", "expediente.hito", "expediente.familia_revision", "expediente.adultos_revision", "estadistica.memoria", "estadistica.memoria_narrativa_historia", "estadistica.memoria_revision", "expediente.menores_revision", "expediente.menor_revision", "expediente.intervencion_revision", "expediente.intervencion_marca", "expediente.intervencion_exclusion", "expediente.marca"} {
		err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: consulta.Perfiles}, func(ctx context.Context) error {
			tx, e := postgres.DesdeContexto(ctx)
			if e != nil {
				return e
			}
			var cantidad int
			e = tx.QueryRow(ctx, "SELECT count(*) FROM "+tabla).Scan(&cantidad)
			if e == nil && cantidad != 0 {
				t.Fatalf("RLS %s: %d filas", tabla, cantidad)
			}
			return e
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, conexion := range []*postgres.Base{b, tecnico} {
		for _, sql := range []string{`SELECT * FROM estadistica.calculo_expedientes`, `SELECT * FROM estadistica.calculo_menores`, `SELECT * FROM estadistica.calculo_intervencion`, `SELECT contenido FROM expediente.derivacion_datos`} {
			if (conexion == tecnico && strings.Contains(sql, "estadistica.calculo_")) || (conexion == b && strings.Contains(sql, "derivacion_datos")) {
				continue
			}
			err = conexion.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: consulta.Perfiles}, func(ctx context.Context) error {
				tx, e := postgres.DesdeContexto(ctx)
				if e != nil {
					return e
				}
				rows, e := tx.Query(ctx, sql)
				if e == nil {
					rows.Close()
					e = rows.Err()
				}
				return e
			})
			var pg *pgconn.PgError
			if !errors.As(err, &pg) || pg.Code != "42501" {
				t.Fatal("permiso técnico abierto", sql, err)
			}
		}
	}
	// Un perfil distinto tampoco activa las vistas técnicas, aunque conozca su nombre.
	err = tecnico.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: []string{"administracion"}}, func(ctx context.Context) error {
		tx, e := postgres.DesdeContexto(ctx)
		if e != nil {
			return e
		}
		var n int
		e = tx.QueryRow(ctx, `SELECT count(*) FROM estadistica.calculo_expedientes`).Scan(&n)
		if e == nil && n != 0 {
			t.Fatal("vistas sin autorización")
		}
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
	nav := app.ServicioNavegacion{Fuente: app.FuenteConCierre{Repositorio: persistencia, Fuente: fuente}, Cierres: persistencia, Numeros: &exp.LecturaEstadistica{}, Transacciones: b, Auditoria: postgres.NuevoRegistroAuditoria(b), Reloj: relojMemoria{}}
	lista, err := nav.ComposicionMemoria(ctx, coordinacion, f, d.CasosAtendidos)
	if err != nil {
		t.Fatal(err)
	}
	encontrado := false
	for _, v := range lista {
		for _, numero := range v.Numeros {
			if numero == aceptada.Numero {
				encontrado = true
			}
			if strings.Contains(numero, aceptada.ID) {
				t.Fatal("composición con UUID")
			}
		}
	}
	if !encontrado {
		t.Fatal("no incluye número aceptado")
	}
	profesional := d.ActorMemoria{UsuarioID: actor.UsuarioID, Perfiles: []string{"profesional_etf"}, EquipoID: p.EquipoID}
	propia, err := nav.ComposicionMemoria(ctx, profesional, f, d.CasosAtendidos)
	if err != nil || len(propia) == 0 {
		t.Fatal("profesional su equipo", err)
	}
	_, err = nav.ComposicionMemoria(ctx, profesional, d.FiltroMemoria{Anio: 2026, EquipoID: "00000000-0000-4000-8000-000000000012"}, d.CasosAtendidos)
	if !errors.Is(err, d.ErrPermisoMemoria) {
		t.Fatal("equipo ajeno", err)
	}
	_, err = nav.ComposicionMemoria(ctx, consulta, f, d.CasosAtendidos)
	if !errors.Is(err, d.ErrPermisoMemoria) {
		t.Fatal("Consulta compone", err)
	}
	estado, err = cierres.EstadoMemoria(ctx, coordinacion, f)
	if err != nil {
		t.Fatal(err)
	}
	_, err = cierres.CerrarMemoria(ctx, coordinacion, f, "consulta-cerrar-"+subida.ID, d.ActoMemoria{Version: estado.Version})
	if err != nil {
		t.Fatal(err)
	}
	// Foto cerrada disponible incluso si el ejecutor de cálculo está retirado.
	sinMotor := dep
	sinMotor.Calculo = nil
	cerrada, err := app.NuevoConsulta(sinMotor).ConsultarCifras(ctx, consulta, f)
	if err != nil || !cerrada.Definitivas || len(cerrada.Apartados) != 33 {
		t.Fatal("foto", err)
	}
	comprobar(cerrada)
	for _, idioma := range []string{"es", "en"} {
		err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: consulta.Perfiles}, func(ctx context.Context) error { _, e := sinMotor.Word.GenerarCifras(ctx, cerrada, idioma); return e })
		if err != nil {
			t.Fatal("generador Word", err)
		}
		contenido, err := app.NuevoConsulta(sinMotor).DescargarCifras(ctx, consulta, f, idioma)
		if err != nil {
			t.Fatal("Word Consulta", err)
		}
		z, err := zip.NewReader(bytes.NewReader(contenido), int64(len(contenido)))
		if err != nil {
			t.Fatal(err)
		}
		for _, entrada := range z.File {
			r, err := entrada.Open()
			if err != nil {
				t.Fatal(err)
			}
			texto, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				t.Fatal(err)
			}
			for _, s := range []string{privado, aceptada.ID, n.ID, "Moreno Castillo", "Centro de acogida de la Vega", "Servicio de apoyo de la Vega"} {
				if s != "" && bytes.Contains(texto, []byte(s)) {
					t.Fatal("filtración Word", entrada.Name, s)
				}
			}
		}
	}
	fallo := dep
	fallo.Auditoria = auditMemoriaFallida{}
	r, err := app.NuevoConsulta(fallo).ConsultarCifras(ctx, consulta, f)
	if err == nil || r.Anio != 0 {
		t.Fatal("datos sin auditoría")
	}
	// No dejar una foto de equipo cerrada que interfiera con otras pruebas
	// que añaden fuentes provinciales después de esta comprobación.
	estado, err = cierres.EstadoMemoria(ctx, coordinacion, f)
	if err != nil {
		t.Fatal(err)
	}
	_, err = cierres.ReabrirMemoria(ctx, coordinacion, f, "consulta-fin-"+subida.ID, d.ActoMemoria{Version: estado.Version, Motivo: "Se continúa la revisión de los datos inventados de la Memoria."})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("F4.07-OK: motor compartido; 33 códigos; foto cerrada; números con ámbito; RLS negativa; JSON y Word sin textos libres")
}
