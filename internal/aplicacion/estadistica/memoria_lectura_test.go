package estadistica_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	app "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica"
	catalogo "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/estadistica"
)

type repositorioFotoCerrada struct {
	salida.Cierres
	revision           d.RevisionMemoria
	cerrada            bool
	bloqueos, lecturas int
}

func (r *repositorioFotoCerrada) BloquearMemorias(ctx context.Context) error {
	if ctx.Value(claveTxSPAF{}) != true {
		return errors.New("cierre.sin_transaccion")
	}
	r.bloqueos++
	return nil
}
func (r *repositorioFotoCerrada) RevisionConocida(ctx context.Context, _ d.FiltroMemoria, _ time.Time) (d.RevisionMemoria, bool, error) {
	if ctx.Value(claveTxSPAF{}) != true {
		return d.RevisionMemoria{}, false, errors.New("cierre.sin_transaccion")
	}
	r.lecturas++
	return r.revision, r.cerrada, nil
}

func TestLecturaMemoriaUsaFotoCerradaSinRecalcularYPublicaCifrasCongeladas(t *testing.T) {
	f := d.FiltroMemoria{Anio: 2026, EquipoID: equipoSPAF}
	conocido := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	informe := d.InformeMemoria{FiltroMemoria: f, ConocidoEn: conocido, Corte: conocido,
		Narrativas: d.Narrativas{D: "Narrativa guardada al cerrar."},
		Apartados:  []d.ApartadoMemoria{{Codigo: d.CasosAtendidos, Total: d.Recuento{Familias: 4}}}}
	repo := &repositorioFotoCerrada{cerrada: true, revision: d.RevisionMemoria{Informe: informe}}
	actual, fotos, tx, auditoria := &fuenteLecturaMemoria{}, &cifrasPublicadasPrueba{}, &transaccionesSPAF{}, &auditoriaSPAF{}
	svc := app.NuevoLecturaMemoria(app.DependenciasLecturaMemoria{Fuente: app.FuenteConCierre{Fuente: actual, Repositorio: repo}, CifrasPublicadas: fotos, Transacciones: tx, Auditoria: auditoria, Reloj: relojSPAF{}})
	a := d.ActorMemoria{UsuarioID: usuarioSPAF, EquipoID: equipoSPAF, Perfiles: []string{"profesional_etf"}}
	for range 2 {
		got, err := svc.ConsultarMemoria(context.Background(), a, f)
		if err != nil || actual.llamadas != 0 || got.Apartados[0].Total.Familias != 4 || !got.ConocidoEn.Equal(conocido) || fotos.cifras.Apartados[0].Total.Familias != 4 {
			t.Fatal(got, err, actual, fotos)
		}
		b, err := json.Marshal(fotos.cifras)
		if err != nil || strings.Contains(string(b), "Narrativa") || strings.Contains(string(b), "narrativas") {
			t.Fatal("publicación filtra narrativa cerrada", err, string(b))
		}
	}
	if repo.bloqueos != 2 || repo.lecturas != 2 || fotos.publicaciones != 2 || len(auditoria.datos) != 2 {
		t.Fatal(repo, fotos, auditoria)
	}
	repo.cerrada = false
	got, err := svc.ConsultarMemoria(context.Background(), a, f)
	if err != nil || actual.llamadas != 1 || got.Apartados[0].Total.Familias != 7 || fotos.cifras.Apartados[0].Total.Familias != 7 || !got.ConocidoEn.After(conocido) {
		t.Fatal("reabierta no usa fuente viva", got, err, actual, fotos)
	}
}

type catalogosLecturaMemoria struct{ llamadas int }

func (s *catalogosLecturaMemoria) Vigente(ctx context.Context, ambito string, _ time.Time) (catalogo.Version, error) {
	if ctx.Value(claveTxSPAF{}) != true || ambito != "asignacion_municipios" {
		return catalogo.Version{}, errors.New("catalogo.sin_transaccion")
	}
	return catalogo.Version{}, nil
}
func (s *catalogosLecturaMemoria) Versiones(context.Context, string) ([]catalogo.Version, error) {
	return nil, nil
}
func (s *catalogosLecturaMemoria) Datos(ctx context.Context, _ catalogo.Version) (catalogo.Datos, error) {
	s.llamadas++
	if ctx.Value(claveTxSPAF{}) != true {
		return catalogo.Datos{}, errors.New("catalogo.sin_transaccion")
	}
	return catalogo.Datos{Equipos: []catalogo.Equipo{{ID: equipoSPAF, Codigo: "02", ClaveTexto: "equipo.02"}, {ID: otroEquipoSPAF, Codigo: "03", ClaveTexto: "equipo.03"}}}, nil
}

func TestEquiposMemoriaFiltraPerfilYConfirmaAntesDeEntregar(t *testing.T) {
	for _, perfil := range []string{"coordinacion", "consulta", "profesional_etf", "administracion"} {
		t.Run(perfil, func(t *testing.T) {
			catalogos, tx, registro := &catalogosLecturaMemoria{}, &transaccionesSPAF{}, &auditoriaSPAF{}
			svc := app.NuevoLecturaMemoria(app.DependenciasLecturaMemoria{Catalogos: catalogos, Transacciones: tx, Auditoria: registro, Reloj: relojSPAF{}})
			a := d.ActorMemoria{UsuarioID: usuarioSPAF, EquipoID: equipoSPAF, Perfiles: []string{perfil}}
			equipos, err := svc.ConsultarEquiposMemoria(context.Background(), a)
			if perfil == "administracion" {
				if !errors.Is(err, d.ErrPermisoMemoria) || catalogos.llamadas != 0 || equipos != nil || len(registro.fallos) != 1 {
					t.Fatal(equipos, err, catalogos, registro)
				}
				return
			}
			esperados := 2
			if perfil == "profesional_etf" {
				esperados = 1
			}
			if err != nil || len(equipos) != esperados || equipos[0].ID != equipoSPAF || len(registro.datos) != 1 || registro.datos[0].Accion != "estadistica.memoria_leer_equipos" {
				t.Fatal(equipos, err, registro)
			}
			tx.errorCommit = true
			equipos, err = svc.ConsultarEquiposMemoria(context.Background(), a)
			if !errors.Is(err, d.ErrDisponibleMemoria) || equipos != nil {
				t.Fatal("equipos entregados sin commit", equipos, err)
			}
		})
	}
}

type fuenteLecturaMemoria struct {
	llamadas     int
	filtro       d.FiltroMemoria
	falla, ajena bool
}

type cifrasPublicadasPrueba struct {
	lecturas, publicaciones int
	cifras                  d.CifrasMemoria
	filtro                  d.FiltroMemoria
	err                     error
}

func (s *cifrasPublicadasPrueba) Consultar(ctx context.Context, f d.FiltroMemoria) (d.CifrasMemoria, error) {
	s.lecturas++
	s.filtro = f
	if ctx.Value(claveTxSPAF{}) != true {
		return d.CifrasMemoria{}, errors.New("cifras.sin_transaccion")
	}
	return s.cifras, s.err
}
func (s *cifrasPublicadasPrueba) Publicar(ctx context.Context, c d.CifrasMemoria) error {
	s.publicaciones++
	if ctx.Value(claveTxSPAF{}) != true {
		return errors.New("cifras.sin_transaccion")
	}
	s.cifras = c
	return s.err
}

func TestConsultaMemoriaRecibeSoloFotoSinLeerFuenteNiElevarPerfiles(t *testing.T) {
	for _, equipo := range []string{"", equipoSPAF} {
		fuente, tx, registro := &fuenteLecturaMemoria{}, &transaccionesSPAF{}, &auditoriaSPAF{}
		foto := &cifrasPublicadasPrueba{cifras: d.CifrasMemoria{Anio: 2026, EquipoID: equipo, ConocidoEn: time.Date(2026, 1, 31, 12, 0, 0, 0, time.UTC), Apartados: []d.CifrasApartadoMemoria{{Codigo: d.CasosAtendidos, Total: d.TotalCifrasMemoria{Familias: 7}}}}}
		svc := app.NuevoLecturaMemoria(app.DependenciasLecturaMemoria{Fuente: fuente, CifrasPublicadas: foto, Transacciones: tx, Auditoria: registro, Reloj: relojSPAF{}})
		got, err := svc.ConsultarMemoria(context.Background(), d.ActorMemoria{UsuarioID: usuarioSPAF, Perfiles: []string{"consulta"}}, d.FiltroMemoria{Anio: 2026, EquipoID: equipo})
		if err != nil || fuente.llamadas != 0 || foto.lecturas != 1 || foto.publicaciones != 0 || len(tx.identidad.Perfiles) != 1 || tx.identidad.Perfiles[0] != "consulta" || got.Apartados[0].Total.Familias != 7 || !got.ConocidoEn.Equal(foto.cifras.ConocidoEn) || len(registro.datos) != 1 {
			t.Fatal(got, err, fuente, foto, tx, registro)
		}
		tx.errorCommit = true
		got, err = svc.ConsultarMemoria(context.Background(), d.ActorMemoria{UsuarioID: usuarioSPAF, Perfiles: []string{"consulta"}}, d.FiltroMemoria{Anio: 2026, EquipoID: equipo})
		if !errors.Is(err, d.ErrDisponibleMemoria) || got.Apartados != nil {
			t.Fatal("Consulta recibe sin commit", got, err)
		}
	}
}

func TestConsultaMemoriaSinFotoNoInventaCeros(t *testing.T) {
	foto, tx, registro := &cifrasPublicadasPrueba{err: d.ErrCifrasNoPublicadasMemoria}, &transaccionesSPAF{}, &auditoriaSPAF{}
	svc := app.NuevoLecturaMemoria(app.DependenciasLecturaMemoria{CifrasPublicadas: foto, Transacciones: tx, Auditoria: registro, Reloj: relojSPAF{}})
	got, err := svc.ConsultarMemoria(context.Background(), d.ActorMemoria{UsuarioID: usuarioSPAF, Perfiles: []string{"consulta"}}, d.FiltroMemoria{Anio: 2026})
	if !errors.Is(err, d.ErrCifrasNoPublicadasMemoria) || got.Apartados != nil || len(registro.fallos) != 1 || len(registro.datos) != 0 {
		t.Fatal(got, err, registro)
	}
}

func TestMemoriaProfesionalYConsultaPuedeLeerFotoAjenaSinElevarPermisos(t *testing.T) {
	fuente, tx, registro := &fuenteLecturaMemoria{}, &transaccionesSPAF{}, &auditoriaSPAF{}
	foto := &cifrasPublicadasPrueba{cifras: d.CifrasMemoria{Anio: 2026, EquipoID: otroEquipoSPAF, Apartados: []d.CifrasApartadoMemoria{}}}
	svc := app.NuevoLecturaMemoria(app.DependenciasLecturaMemoria{Fuente: fuente, CifrasPublicadas: foto, Transacciones: tx, Auditoria: registro, Reloj: relojSPAF{}})
	a := d.ActorMemoria{UsuarioID: usuarioSPAF, EquipoID: equipoSPAF, Perfiles: []string{"profesional_etf", "consulta"}}
	got, err := svc.ConsultarMemoria(context.Background(), a, d.FiltroMemoria{Anio: 2026, EquipoID: otroEquipoSPAF})
	if err != nil || fuente.llamadas != 0 || foto.lecturas != 1 || got.EquipoID != otroEquipoSPAF || tx.identidad.EquipoID != equipoSPAF {
		t.Fatal(got, err, fuente, tx)
	}
}

func TestMemoriaPublicaSoloCifrasEnMismaTransaccion(t *testing.T) {
	fuente, foto, tx, registro := &fuenteLecturaMemoria{}, &cifrasPublicadasPrueba{}, &transaccionesSPAF{}, &auditoriaSPAF{}
	svc := app.NuevoLecturaMemoria(app.DependenciasLecturaMemoria{Fuente: fuente, CifrasPublicadas: foto, Transacciones: tx, Auditoria: registro, Reloj: relojSPAF{}})
	a := d.ActorMemoria{UsuarioID: usuarioSPAF, Perfiles: []string{"coordinacion"}}
	got, err := svc.ConsultarMemoria(context.Background(), a, d.FiltroMemoria{Anio: 2026})
	if err != nil || foto.publicaciones != 1 || foto.lecturas != 0 || tx.llamadas != 1 || foto.cifras.Apartados[0].Total.Familias != got.Apartados[0].Total.Familias || len(registro.datos) != 1 {
		t.Fatal(got, err, foto, tx, registro)
	}
	foto.err = errors.New("publicacion.error")
	got, err = svc.ConsultarMemoria(context.Background(), a, d.FiltroMemoria{Anio: 2026})
	if !errors.Is(err, d.ErrDisponibleMemoria) || got.Apartados != nil {
		t.Fatal("entrega tras fallo publicación", got, err)
	}
}

func (s *fuenteLecturaMemoria) MemoriaEnTransaccion(ctx context.Context, _ d.ActorMemoria, f d.FiltroMemoria, ahora time.Time) (d.InformeMemoria, error) {
	s.llamadas++
	s.filtro = f
	if ctx.Value(claveTxSPAF{}) != true || s.falla {
		return d.InformeMemoria{}, errors.New("fuente.error")
	}
	if s.ajena {
		f.EquipoID = otroEquipoSPAF
	}
	return d.InformeMemoria{FiltroMemoria: f, Corte: ahora, ConocidoEn: ahora,
		Narrativas: d.Narrativas{ID: "identificador-privado", D: "contenido narrativo privado", A: d.GeneralesMemoria{Observaciones: "observaciones privadas"}},
		Apartados: []d.ApartadoMemoria{{Codigo: d.CasosAtendidos, Total: d.Recuento{Familias: 7, Menores: 11, FamiliasSinRelacionMenores: 2},
			Celdas: []d.CeldaMemoria{{Grupo: d.Grupo{Subprograma: d.Riesgo}, Unidad: d.Familias, Celda: d.Celda{Numerador: 7, SinDato: 2, Denominador: d.Dato[int]{Valor: 11, Conocido: true}, Porcentaje: &d.Porcentaje{Centesimas: 6364}}}}, Avisos: map[string]int{"estadistica.denominador_pendiente": 1}}}}, nil
}

func TestLecturaMemoriaAutorizaReutilizaFuenteYExcluyeNarrativas(t *testing.T) {
	for _, caso := range []struct {
		nombre                       string
		perfiles                     []string
		equipo, solicitado, esperado string
	}{
		{"profesional implícito", []string{"profesional_etf"}, equipoSPAF, "", equipoSPAF},
		{"profesional explícito", []string{"profesional_etf"}, equipoSPAF, equipoSPAF, equipoSPAF},
		{"coordinación equipo", []string{"coordinacion"}, "", otroEquipoSPAF, otroEquipoSPAF},
		{"coordinación provincial", []string{"coordinacion"}, "", "", ""},
		{"perfiles combinados", []string{"coordinacion", "administracion"}, "", equipoSPAF, equipoSPAF},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			fuente, tx, registro := &fuenteLecturaMemoria{}, &transaccionesSPAF{}, &auditoriaSPAF{}
			svc := app.NuevoLecturaMemoria(app.DependenciasLecturaMemoria{Fuente: fuente, Transacciones: tx, Auditoria: registro, Reloj: relojSPAF{}})
			cifras, err := svc.ConsultarMemoria(context.Background(), d.ActorMemoria{UsuarioID: usuarioSPAF, EquipoID: caso.equipo, Perfiles: caso.perfiles}, d.FiltroMemoria{Anio: 2026, EquipoID: caso.solicitado})
			if err != nil || fuente.llamadas != 1 || fuente.filtro.EquipoID != caso.esperado || cifras.EquipoID != caso.esperado || tx.identidad.EquipoID != caso.equipo || len(registro.datos) != 1 || registro.datos[0].Resultado != "ok" || registro.datos[0].EntidadID != caso.esperado {
				t.Fatal(cifras, err, fuente, tx, registro)
			}
			if cifras.Apartados[0].Total.Familias != 7 || cifras.Apartados[0].Total.Menores != 11 || cifras.Apartados[0].Celdas[0].Porcentaje.Centesimas != 6364 || cifras.Apartados[0].Celdas[0].SinDato != 2 {
				t.Fatal("proyección cambia las cifras", cifras)
			}
			b, err := json.Marshal(cifras)
			if err != nil {
				t.Fatal(err)
			}
			for _, prohibido := range []string{"privad", "narrativas", "observaciones", "expediente", "identificador"} {
				if strings.Contains(string(b), prohibido) {
					t.Fatalf("respuesta filtra %q: %s", prohibido, b)
				}
			}
		})
	}
}

func TestLecturaMemoriaDeniegaSinLeerYAudita(t *testing.T) {
	for _, caso := range []struct {
		nombre   string
		actor    d.ActorMemoria
		filtro   d.FiltroMemoria
		esperado error
	}{
		{"equipo ajeno", d.ActorMemoria{UsuarioID: usuarioSPAF, EquipoID: equipoSPAF, Perfiles: []string{"profesional_etf"}}, d.FiltroMemoria{Anio: 2026, EquipoID: otroEquipoSPAF}, d.ErrPermisoMemoria},
		{"sin equipo", d.ActorMemoria{UsuarioID: usuarioSPAF, Perfiles: []string{"profesional_etf"}}, d.FiltroMemoria{Anio: 2026}, d.ErrPermisoMemoria},
		{"administración", d.ActorMemoria{UsuarioID: usuarioSPAF, Perfiles: []string{"administracion"}}, d.FiltroMemoria{Anio: 2026}, d.ErrPermisoMemoria},
		{"anónimo", d.ActorMemoria{Perfiles: []string{"coordinacion"}}, d.FiltroMemoria{Anio: 2026}, d.ErrPermisoMemoria},
		{"año inválido", d.ActorMemoria{UsuarioID: usuarioSPAF, Perfiles: []string{"coordinacion"}}, d.FiltroMemoria{Anio: 0}, d.ErrEntrada},
		{"equipo mal formado", d.ActorMemoria{UsuarioID: usuarioSPAF, Perfiles: []string{"coordinacion"}}, d.FiltroMemoria{Anio: 2026, EquipoID: "otro"}, d.ErrEntrada},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			fuente, tx, registro := &fuenteLecturaMemoria{}, &transaccionesSPAF{}, &auditoriaSPAF{}
			svc := app.NuevoLecturaMemoria(app.DependenciasLecturaMemoria{Fuente: fuente, Transacciones: tx, Auditoria: registro, Reloj: relojSPAF{}})
			got, err := svc.ConsultarMemoria(context.Background(), caso.actor, caso.filtro)
			if !errors.Is(err, caso.esperado) || fuente.llamadas != 0 || tx.llamadas != 0 || got.Apartados != nil || len(registro.fallos) != 1 {
				t.Fatal(got, err, fuente, tx, registro)
			}
		})
	}
}

func TestLecturaMemoriaNoEntregaSiFallaFuenteAuditoriaOCommit(t *testing.T) {
	for _, caso := range []string{"fuente", "ámbito ajeno", "auditoría", "commit", "auditoría de fallo"} {
		t.Run(caso, func(t *testing.T) {
			fuente, tx, registro := &fuenteLecturaMemoria{}, &transaccionesSPAF{}, &auditoriaSPAF{}
			switch caso {
			case "fuente":
				fuente.falla = true
			case "ámbito ajeno":
				fuente.ajena = true
			case "auditoría":
				registro.errorRegistro = true
			case "commit":
				tx.errorCommit = true
			case "auditoría de fallo":
				fuente.falla, registro.errorFallo = true, true
			}
			svc := app.NuevoLecturaMemoria(app.DependenciasLecturaMemoria{Fuente: fuente, Transacciones: tx, Auditoria: registro, Reloj: relojSPAF{}})
			got, err := svc.ConsultarMemoria(context.Background(), d.ActorMemoria{UsuarioID: usuarioSPAF, EquipoID: equipoSPAF, Perfiles: []string{"profesional_etf"}}, d.FiltroMemoria{Anio: 2026})
			if err == nil || got.Apartados != nil || got.Anio != 0 {
				t.Fatal("entrega sin confirmación", got, err)
			}
		})
	}
}
