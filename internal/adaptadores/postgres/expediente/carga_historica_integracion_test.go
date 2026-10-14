package expediente_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	nombres "github.com/aavidad/expedientes-etf/internal/adaptadores/expediente/ficheros"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	est "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"github.com/jackc/pgx/v5/pgconn"
)

func dependenciasHistoricas(deps app.Dependencias) app.Dependencias {
	r := &repo.Repositorio{}
	deps.CargasHistoricas = r
	deps.Decisiones = r
	deps.Cierres = r
	deps.MotivosCierre = &catalogo.Repositorio{}
	deps.Familias = r
	deps.CatalogosFamilia = nombres.CatalogosFamilia{Listas: "../../../../idiomas/es/catalogos/listas.json", Familia: "../../../../idiomas/es/catalogos/familia.json", Parametros: "../../../../datos/catalogos/familia.json"}
	deps.Menores = r
	deps.CatalogosMenores = nombres.CatalogosMenores{Ruta: "../../../../idiomas/es/catalogos/menores.json"}
	deps.Reloj = relojCierreFijo{}
	deps.Bandeja = r
	return deps
}
func casoHistorico(p d.Aceptacion, numero string) d.CasoHistorico {
	p.Menores = []d.SexoMenor{{Orden: 1, Sexo: "M"}}
	return d.CasoHistorico{NumeroHistorico: d.NumeroHistorico{Numero: numero}, FechaDerivacion: "2026-01-15", Estado: d.Cerrada,
		Datos:      d.DatosDerivacion{EsquemaVersion: "1", Extraccion: d.Extraccion{Tipo: "anexo3", Campos: []d.Campo{{Codigo: "menores.persona.nombre", Valor: "Lucía"}, {Codigo: "familia.contacto.apellidos", Valor: "Serrano Ruiz"}}}},
		Aceptacion: &p, Episodios: []d.EpisodioHistorico{{Apertura: p.FechaActa, Cierre: "2026-03-01", MotivoCierre: "OBJ_TOTAL"}, {Apertura: "2026-04-01", MotivoReapertura: "La familia vuelve a necesitar apoyo.", Cierre: "2026-05-01", MotivoCierre: "OBJ_TOTAL"}},
		Familia: []d.CambioFamilia{{Desde: p.FechaActa, Familia: ptrFamilia(datosFamilia())}, {Desde: p.FechaActa, Adultos: &[]d.Adulto{{Orden: 1, Nombre: "Carmen", Apellidos: "Serrano Ruiz", Papel: "MADRE", Sexo: "M", Edad: ptrEdadHistorica(35)}}}, {Desde: "2026-04-01", Familia: ptrFamilia(d.Familia{Composicion: texto("MONOPARENTAL_MADRE")})}},
		Menores: []d.CambioMenores{{Desde: p.FechaActa, Menores: &[]d.Menor{{Orden: 1, Nombre: "Lucía", Apellidos: "Serrano Ruiz", Sexo: "M", Subprograma: "RIESGO_SOCIAL"}}}, {Desde: "2026-04-01", Menores: &[]d.Menor{{Orden: 1, Nombre: "Lucía", Apellidos: "Serrano Ruiz", Sexo: "M", Subprograma: "RIESGO_SOCIAL", Nacimiento: texto("2014-06-10")}}}},
	}
}
func ptrFamilia(f d.Familia) *d.Familia { return &f }

func TestCargaHistoricaRESP2ReservaPorEquipoYAnio(t *testing.T) {
	b, ctx, deps, a, p := prepararAceptacion(t)
	deps = dependenciasHistoricas(deps)
	s := app.Nuevo(deps)
	antes := contador(t, b, ctx, a, p)
	if antes >= 98 {
		t.Fatal("correlativos insuficientes en el entorno desechable")
	}
	anterior := casoHistorico(p, "08-01-22")
	anterior.FechaDerivacion = "2022-02-01"
	anterior.Aceptacion.FechaActa = "2022-03-09"
	anterior.Estado = d.EnIntervencion
	anterior.Episodios = []d.EpisodioHistorico{{Apertura: "2022-03-09"}}
	anterior.Familia = nil
	anterior.Menores = nil
	actual := casoHistorico(p, fmt.Sprintf("08-%02d-26", antes+1))
	actual.Estado = d.EnIntervencion
	actual.Episodios = []d.EpisodioHistorico{{Apertura: p.FechaActa}}
	actual.Familia = nil
	actual.Menores = nil
	carga := d.CargaHistorica{EquipoID: p.EquipoID, Casos: []d.CasoHistorico{anterior, actual}}
	v, err := s.ValidarCargaHistorica(ctx, a, carga)
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.CargarHistoricos(ctx, a, v.LoteID, "resp2-"+v.LoteID, carga)
	if err != nil {
		t.Fatal(err)
	}
	if contador(t, b, ctx, a, p) != antes {
		t.Fatal("carga consume correlativos nuevos")
	}
	ficha, err := s.Consultar(ctx, a, r.Expedientes[0])
	if err != nil || ficha.Numero == nil || *ficha.Numero != "08-01-22" || !ficha.CargadoParaRevision {
		t.Fatal("número histórico perdido", ficha, err)
	}
	derivacion := subir(t, ctx, deps, a)
	nueva, err := s.Aceptar(ctx, a, derivacion.ID, "resp2-nueva-"+derivacion.ID, conMenores(p, derivacion))
	if err != nil {
		t.Fatal(err)
	}
	if nueva.Numero != fmt.Sprintf("08-%02d-26", antes+2) || nueva.Anio != 2026 {
		t.Fatal("colisión o año incorrecto", nueva)
	}
	t.Log("RESP2_POSTGRES_OK: abierto 2022 conserva 08-01-22; carga no consume contador; nueva aceptación 2026 salta correlativo histórico ocupado")
}
func numeroHistoricoLibre(t *testing.T, b *postgres.Base, ctx context.Context, a d.Actor, p d.Aceptacion) string {
	t.Helper()
	var n int
	err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		return tx.QueryRow(ctx, `SELECT n FROM generate_series(99,1,-1) n WHERE NOT EXISTS(SELECT 1 FROM expediente.expediente WHERE equipo_id=$1 AND anio_numero=2026 AND correlativo=n) LIMIT 1`, p.EquipoID).Scan(&n)
	})
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("08-%02d-26", n)
}
func contarCargas(t *testing.T, b *postgres.Base, ctx context.Context, a d.Actor) int {
	t.Helper()
	var n int
	err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		return tx.QueryRow(ctx, `SELECT count(*) FROM expediente.carga_inicial`).Scan(&n)
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}
func TestCargaHistoricaPostgresNumerosVigenciasEpisodiosRLSAuditoria(t *testing.T) {
	b, ctx, deps, coord, p := prepararAceptacion(t)
	deps = dependenciasHistoricas(deps)
	s := app.Nuevo(deps)
	equipo := coord
	equipo.EquipoID = p.EquipoID
	equipo.Perfiles = []string{"profesional_etf"}
	numero := numeroHistoricoLibre(t, b, ctx, coord, p)
	cerrado := casoHistorico(p, numero)
	vivo := casoHistorico(p, "")
	vivo.Estado = d.EnIntervencion
	vivo.Episodios = vivo.Episodios[:1]
	vivo.Episodios[0].Cierre = ""
	vivo.Episodios[0].MotivoCierre = ""
	no := d.CasoHistorico{Estado: d.NoAceptada, FechaDerivacion: "2026-01-15", Datos: cerrado.Datos, NoAceptacion: &d.NoAceptacion{Fecha: "2026-02-01", Motivo: "Se mantiene el apoyo desde los Servicios Sociales Comunitarios.", Subprograma: "RIESGO_SOCIAL"}}
	carga := d.CargaHistorica{EquipoID: p.EquipoID, Casos: []d.CasoHistorico{cerrado, vivo, no}}
	antes := contador(t, b, ctx, coord, p)
	vista, err := s.ValidarCargaHistorica(ctx, equipo, carga)
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.CargarHistoricos(ctx, equipo, vista.LoteID, "alta-historica-"+vista.LoteID, carga)
	if err != nil {
		t.Fatal(err)
	}
	otra, err := s.CargarHistoricos(ctx, equipo, vista.LoteID, "alta-historica-"+vista.LoteID, carga)
	if err != nil || !reflect.DeepEqual(r, otra) {
		t.Fatal(r, otra, err)
	}
	if contador(t, b, ctx, coord, p) != antes || len(r.Expedientes) != 3 {
		t.Fatal("contador o lote", r)
	}
	for i, id := range r.Expedientes {
		ficha, err := s.Consultar(ctx, equipo, id)
		if err != nil || !ficha.CargadoParaRevision || ficha.Estado != carga.Casos[i].Estado {
			t.Fatal(ficha, err)
		}
		if i == 0 && (ficha.Numero == nil || *ficha.Numero != numero) || i > 0 && ficha.Numero != nil {
			t.Fatal("número", ficha.Numero)
		}
	}
	filtro := d.FiltroBandeja{EquipoID: p.EquipoID, Limite: 100, Estado: d.NoAceptada}
	bandeja, err := s.ConsultarBandeja(ctx, coord, filtro)
	if err != nil {
		t.Fatal(err)
	}
	encontrado := false
	for _, fila := range bandeja.Filas {
		if fila.ID == r.Expedientes[2] {
			encontrado = fila.CargadoParaRevision && fila.EquipoID == p.EquipoID
		}
	}
	if !encontrado {
		t.Fatal("no aceptado histórico ausente en bandeja")
	}
	id := r.Expedientes[0]
	historica, err := s.ConsultarHistoriaFamilia(ctx, equipo, id, "2026-03-01", 2026)
	if err != nil || historica.Familia.Composicion == nil || *historica.Familia.Composicion != "BIPARENTAL" {
		t.Fatal(historica, err)
	}
	hmenores, err := s.ConsultarHistoriaMenores(ctx, equipo, id, "2026-03-01")
	if err != nil || len(hmenores.Menores) != 1 || hmenores.Menores[0].Nacimiento != nil {
		t.Fatal(hmenores, err)
	}
	var episodios []est.Episodio
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: coord.UsuarioID, Perfiles: coord.Perfiles}, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		rows, err := tx.Query(ctx, `SELECT id::text,apertura_fecha,cierre_fecha,coalesce(reapertura_de_id::text,'') FROM expediente.episodio WHERE expediente_id=$1 ORDER BY apertura_fecha`, id)
		if err != nil {
			return err
		}
		for rows.Next() {
			var e est.Episodio
			if err = rows.Scan(&e.ID, &e.Apertura, &e.Cierre, &e.ReaperturaDe); err != nil {
				rows.Close()
				return err
			}
			e.Aceptacion = e.Apertura
			episodios = append(episodios, e)
		}
		rows.Close()
		if err = rows.Err(); err != nil {
			return err
		}
		var docs, lecturas int
		if err = tx.QueryRow(ctx, `SELECT count(*) FROM expediente.lectura_anexo WHERE expediente_id=ANY($1::uuid[])`, r.Expedientes).Scan(&lecturas); err != nil {
			return err
		}
		if err = tx.QueryRow(ctx, `SELECT count(*) FROM documento.emision WHERE expediente_id=ANY($1::uuid[])`, r.Expedientes).Scan(&docs); err != nil {
			return err
		}
		if docs != 0 || lecturas != 0 {
			return errors.New("documentos ficticios")
		}
		return nil
	})
	if err != nil || len(episodios) != 2 || episodios[1].ReaperturaDe != episodios[0].ID {
		t.Fatal(episodios, err)
	}
	comprobarEstadisticaHistorica(t, p.EquipoID, id, numero, episodios)
	for _, perfil := range []string{"consulta", "administracion", "profesional_etf"} {
		a := coord
		a.Perfiles = []string{perfil}
		a.EquipoID = "88888888-8888-4888-8888-888888888888"
		err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
			tx, _ := postgres.DesdeContexto(ctx)
			var n int
			err := tx.QueryRow(ctx, `SELECT count(*) FROM expediente.expediente WHERE id=ANY($1::uuid[])`, r.Expedientes).Scan(&n)
			if n != 0 {
				t.Error("RLS", perfil, n)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		lote, _ := ids{}.Nuevo()
		if _, err = s.CargarHistoricos(ctx, a, lote, "sin-permiso-"+perfil, carga); !errors.Is(err, d.ErrAcceso) {
			t.Fatal(perfil, err)
		}
	}
	// La excepción de carga termina al confirmar: el equipo no reabre ni cambia el número.
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: equipo.UsuarioID, Perfiles: equipo.Perfiles, EquipoID: equipo.EquipoID}, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		_, err := tx.Exec(ctx, `UPDATE expediente.expediente SET estado='aceptada',version=version+1,modificado_por=$2 WHERE id=$1`, id, equipo.UsuarioID)
		return err
	})
	if err == nil {
		t.Fatal("reapertura ordinaria por equipo")
	}
	eventos, err := postgres.NuevoRegistroAuditoria(b).Exportar(ctx, plataforma.Identidad{UsuarioID: coord.UsuarioID, Perfiles: []string{"administracion"}}, 1, 100000)
	if err != nil {
		t.Fatal(err)
	}
	marcas := map[string]int{}
	for _, e := range eventos.Eventos {
		if e.Accion == "expediente.cargado_para_revision" {
			marcas[e.ExpedienteID]++
		}
	}
	for _, id := range r.Expedientes {
		if marcas[id] != 1 {
			t.Fatal("marca auditada", id, marcas[id])
		}
	}
	antesCargas := contarCargas(t, b, ctx, coord)
	lote, _ := ids{}.Nuevo()
	if _, err = s.CargarHistoricos(ctx, coord, lote, "duplicado-"+lote, d.CargaHistorica{EquipoID: p.EquipoID, Casos: []d.CasoHistorico{vivo, cerrado}}); !errors.Is(err, d.ErrNumeroDuplicado) {
		t.Fatal("duplicado", err)
	}
	if contarCargas(t, b, ctx, coord) != antesCargas {
		t.Fatal("lote parcial")
	}
	fallo := deps
	fallo.RegistroTransaccional = registroFallido{}
	lote, _ = ids{}.Nuevo()
	if _, err = app.Nuevo(fallo).CargarHistoricos(ctx, equipo, lote, "fallo-"+lote, d.CargaHistorica{EquipoID: p.EquipoID, Casos: []d.CasoHistorico{vivo}}); !errors.Is(err, d.ErrNoDisponible) {
		t.Fatal(err)
	}
	if contarCargas(t, b, ctx, coord) != antesCargas {
		t.Fatal("carga sin auditoría")
	}
	// Sin contexto, la base no expone los casos cargados.
	err = b.Ejecutar(ctx, plataforma.Identidad{}, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		var n int
		err := tx.QueryRow(ctx, `SELECT count(*) FROM expediente.expediente WHERE id=ANY($1::uuid[])`, r.Expedientes).Scan(&n)
		if n != 0 {
			t.Error("sin contexto", n)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}
func comprobarEstadisticaHistorica(t *testing.T, equipo, id, numero string, episodios []est.Episodio) {
	t.Helper()
	fecha := func(s string) time.Time { f, _ := time.Parse("2006-01-02", s); return f }
	entrada := est.EntradaSPAF{Filtro: est.FiltroSPAF{Anio: 2026, Mes: 3, EquipoID: equipo}, FechaCalculo: time.Now(), ConocidoEn: time.Now().Add(-time.Second),
		Reglas:      est.Historia[est.ReglasSPAF]{{ID: "reglas-2026", Vigencia: est.Vigencia{Desde: fecha("2026-01-01")}, RegistradoEn: fecha("2026-01-01"), Dato: est.Dato[est.ReglasSPAF]{Conocido: true, Valor: est.ReglasSPAF{MotivosCierre: map[est.Motivo]string{}}}}},
		Expedientes: []est.ExpedienteSPAF{{ID: id, EquipoID: equipo, Numero: numero, Episodios: episodios}}}
	for _, mes := range []int{3, 5} {
		entrada.Filtro.Mes = mes
		tabla, err := est.CalcularSPAF(entrada)
		if err != nil || tabla.Resumen.Bajas != 1 {
			t.Fatal("estadística de episodios", mes, tabla.Resumen, err)
		}
	}
}

func ptrEdadHistorica(n int) *int { return &n }

func TestCargaHistoricaAceptadaAntes2026YAltaOrdinariaSaltaNumeroOcupado(t *testing.T) {
	b, ctx, deps, coord, p := prepararAceptacion(t)
	deps = dependenciasHistoricas(deps)
	s := app.Nuevo(deps)
	antiguo := casoHistorico(p, "08-99-25")
	antiguo.FechaDerivacion = "2025-10-20"
	antiguo.Aceptacion.FechaActa = "2025-11-01"
	antiguo.Episodios[0].Apertura = "2025-11-01"
	antiguo.Familia[0].Desde = "2025-11-01"
	antiguo.Familia[1].Desde = "2025-11-01"
	antiguo.Menores[0].Desde = "2025-11-01"
	antes := contador(t, b, ctx, coord, p)
	lote, _ := ids{}.Nuevo()
	r, err := s.CargarHistoricos(ctx, coord, lote, "antiguo-"+lote, d.CargaHistorica{EquipoID: p.EquipoID, Casos: []d.CasoHistorico{antiguo}})
	if err != nil {
		t.Fatal(err)
	}
	ficha, err := s.Consultar(ctx, coord, r.Expedientes[0])
	if err != nil || ficha.Numero == nil || *ficha.Numero != "08-99-25" || ficha.Aceptacion.ReglaID != "" || ficha.Aceptacion.FechaActa != "2025-11-01" {
		t.Fatal(ficha, err)
	}
	if contador(t, b, ctx, coord, p) != antes {
		t.Fatal("histórico consumió contador")
	}
	if antes >= 97 {
		t.Fatal("preparación excede rango reservado para la prueba")
	}
	historico := casoHistorico(p, fmt.Sprintf("08-%02d-26", antes+1))
	lote, _ = ids{}.Nuevo()
	if _, err = s.CargarHistoricos(ctx, coord, lote, "numero-ocupado-"+lote, d.CargaHistorica{EquipoID: p.EquipoID, Casos: []d.CasoHistorico{historico}}); err != nil {
		t.Fatal(err)
	}
	if contador(t, b, ctx, coord, p) != antes {
		t.Fatal("número original consumió contador")
	}
	derivacion := subir(t, ctx, deps, coord)
	ordinaria, err := s.Aceptar(ctx, coord, derivacion.ID, "alta-tras-historico-"+derivacion.ID, conMenores(p, derivacion))
	if err != nil || ordinaria.Correlativo != antes+2 || ordinaria.Numero == historico.Numero {
		t.Fatal(ordinaria, err)
	}
}

func TestCargaHistoricaPostgresRechazaFechasPorSQLYExplicaEpisodios(t *testing.T) {
	b, ctx, _, coord, p := prepararAceptacion(t)
	repositorio := &repo.Repositorio{}
	lote, _ := ids{}.Nuevo()
	id, _ := ids{}.Nuevo()
	antes := contarCargas(t, b, ctx, coord)
	err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: coord.UsuarioID, Perfiles: coord.Perfiles}, func(ctx context.Context) error {
		if err := repositorio.CrearLoteHistorico(ctx, lote, p.EquipoID, coord.UsuarioID, time.Now(), 1); err != nil {
			return err
		}
		c := casoHistorico(p, "")
		if err := repositorio.CrearCasoHistorico(ctx, id, lote, p.EquipoID, c, coord.UsuarioID, time.Now()); err != nil {
			return err
		}
		// Bypass deliberado de la validación Go: la segunda barrera debe rechazarlo.
		p.FechaActa = "2026-01-01"
		p.Menores = []d.SexoMenor{{Orden: 1, Sexo: "M"}}
		return repositorio.Aceptar(ctx, p, d.ResultadoAceptacion{ID: id, EquipoID: p.EquipoID, Version: 2, Estado: d.Aceptada}, coord.UsuarioID, time.Now())
	})
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || pg.Message != "expediente.carga_historica_invalida" {
		t.Fatal("PostgreSQL admitió fechas incoherentes", err)
	}
	if contarCargas(t, b, ctx, coord) != antes {
		t.Fatal("fallo SQL dejó lote parcial")
	}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: coord.UsuarioID, Perfiles: coord.Perfiles}, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		rows, err := tx.Query(ctx, `EXPLAIN SELECT id FROM expediente.episodio WHERE expediente_id=$1 AND cierre_fecha IS NOT NULL`, id)
		if err != nil {
			return err
		}
		defer rows.Close()
		n := 0
		for rows.Next() {
			var plan string
			if err := rows.Scan(&plan); err != nil {
				return err
			}
			n++
		}
		if n == 0 {
			t.Error("EXPLAIN sin plan")
		}
		return rows.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("PostgreSQL rechaza aceptación anterior a derivación y revierte el lote; EXPLAIN de episodios ejecutado")
}
