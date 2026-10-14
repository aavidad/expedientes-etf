package expediente_test

import (
	"context"
	"encoding/json"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	adaptador "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	estadistica "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"strings"
	"testing"
	"time"
)

func TestLecturaHistoricaEstadisticaRLSYConocimiento(t *testing.T) {
	b, ctx, deps, a, p := prepararAceptacion(t)
	recibida := subir(t, ctx, deps, a)
	p = conMenores(p, recibida)
	anterior := time.Now().UTC().Add(-time.Second)
	aceptada, err := app.Nuevo(deps).Aceptar(ctx, a, recibida.ID, "historia-"+recibida.ID, p)
	if err != nil {
		t.Fatal(err)
	}
	lector := &adaptador.LecturaEstadistica{Reglas: &catalogo.Repositorio{}}
	corte := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	for _, caso := range []struct {
		perfil, equipo string
		visible        bool
	}{{"coordinacion", "", true}, {"profesional_etf", p.EquipoID, true}, {"profesional_etf", "00000000-0000-4000-8000-000000000012", false}, {"consulta", "", false}, {"administracion", "", false}, {"", "", false}} {
		t.Run(caso.perfil+caso.equipo, func(t *testing.T) {
			perfiles := []string{}
			if caso.perfil != "" {
				perfiles = append(perfiles, caso.perfil)
			}
			err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: perfiles, EquipoID: caso.equipo}, func(ctx context.Context) error {
				rows, err := lector.ConsultarMemoria(ctx, "", corte, time.Now().UTC())
				if err != nil {
					return err
				}
				encontrado := false
				for _, exp := range rows {
					if exp.ID != aceptada.ID {
						continue
					}
					encontrado = true
					if len(exp.Episodios) != 1 || exp.Episodios[0].Aceptacion.Format("2006-01-02") != "2026-02-03" {
						t.Fatal("episodio no confirmado", exp.Episodios)
					}
					dato, _, err := exp.Subprograma.ValorEn(corte, time.Now().UTC())
					if err != nil || !dato.Conocido || dato.Valor != d.Riesgo {
						t.Fatal("subprograma", dato, err)
					}
					servicio, _, fallo := exp.ServicioDerivante.ValorEn(corte, time.Now().UTC())
					if fallo != nil || !servicio.Conocido || servicio.Valor != "SSSSCC" {
						t.Fatal("servicio fuera del catálogo", servicio, fallo)
					}
					menores, _, err := exp.Menores.ValorEn(corte, time.Now().UTC())
					if err != nil || !menores.Conocido || len(menores.Valor) != len(p.Menores) {
						t.Fatal("colección aceptada", menores, err)
					}
				}
				if encontrado != caso.visible {
					t.Fatal("aislamiento", encontrado, caso.visible)
				}
				antiguas, err := lector.ConsultarMemoria(ctx, "", corte, anterior)
				if err != nil {
					return err
				}
				for _, exp := range antiguas {
					if exp.ID == aceptada.ID && len(exp.Episodios) > 0 {
						t.Fatal("se filtró aceptación posterior al conocimiento")
					}
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
	servicio := estadistica.NuevoSPAF(estadistica.DependenciasSPAF{Lectura: lector, Transacciones: b, Auditoria: postgres.NuevoRegistroAuditoria(b), Reloj: reloj{}})
	tabla, err := servicio.ConsultarSPAF(ctx, d.ActorSPAF{UsuarioID: a.UsuarioID, Perfiles: []string{"profesional_etf"}, EquipoID: p.EquipoID}, d.FiltroSPAF{Anio: 2026, Mes: 2})
	if err != nil {
		t.Fatal(err)
	}
	encontrada := false
	for _, fila := range tabla.Filas {
		if fila.ExpedienteID == aceptada.ID {
			encontrada = true
			if !fila.Alta || !fila.NumeroMenores.Conocido || fila.NumeroMenores.Valor != len(p.Menores) {
				t.Fatal("alta o dato ausente incorrectos")
			}
		}
	}
	if !encontrada {
		t.Fatal("SPAF no conectado")
	}
}

func TestLecturaEstadisticaFamiliaRetroactivaYSinIdentidadPersonal(t *testing.T) {
	b, ctx, deps, a, id := prepararFamilia(t)
	familia := datosFamilia()
	adultos := []dominio.Adulto{{Orden: 1, Nombre: "Carlos", Apellidos: "Ortega Serrano", Papel: "PADRE", Sexo: "H", Nacimiento: texto("1980-01-02"), Educacion: texto("SECUNDARIOS"), Trabajo: texto("EMPLEO_FIJO")}}
	servicio := app.Nuevo(deps)
	primero, err := servicio.EditarFamilia(ctx, a, id, "estadistica-febrero-"+id, dominio.CambioFamilia{Version: 2, Desde: "2026-02-03", Familia: &familia})
	if err != nil {
		t.Fatal(err)
	}
	primero, err = servicio.EditarFamilia(ctx, a, id, "estadistica-adultos-"+id, dominio.CambioFamilia{Version: primero.Version, Desde: "2026-02-03", Adultos: &adultos})
	if err != nil {
		t.Fatal(err)
	}

	familia.Composicion = texto("MONOPARENTAL_PADRE")
	segundo, err := servicio.EditarFamilia(ctx, a, id, "estadistica-mayo-"+id, dominio.CambioFamilia{Version: primero.Version, Desde: "2026-05-01", Familia: &familia})
	if err != nil {
		t.Fatal(err)
	}
	conocido := time.Now().UTC()
	familia.Composicion = texto("RECONSTITUIDA")
	ultimo, err := servicio.EditarFamilia(ctx, a, id, "estadistica-correccion-"+id, dominio.CambioFamilia{Version: segundo.Version, Desde: "2026-05-01", Familia: &familia})
	if err != nil {
		t.Fatal(err)
	}
	lector := &adaptador.LecturaEstadistica{Reglas: &catalogo.Repositorio{}}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
		for _, caso := range []struct {
			conocido        time.Time
			corte, esperado string
		}{{time.Now().UTC(), "2026-04-30", "BIPARENTAL"}, {conocido, "2026-06-01", "MONOPARENTAL_PADRE"}, {time.Now().UTC(), "2026-06-01", "RECONSTITUIDA"}} {
			corte, err := time.Parse("2006-01-02", caso.corte)
			if err != nil {
				return err
			}
			filas, err := lector.ConsultarMemoria(ctx, a.EquipoID, corte, caso.conocido)
			if err != nil {
				return err
			}
			encontrado := false
			for _, exp := range filas {
				if exp.ID != id {
					continue
				}
				encontrado = true
				dato, _, err := exp.Composicion.ValorEn(corte, caso.conocido)
				if err != nil || !dato.Conocido || dato.Valor != caso.esperado {
					t.Fatal("vigencia/conocimiento", dato, caso.esperado, err)
				}
				aa, _, err := exp.Adultos.ValorEn(corte, caso.conocido)
				if err != nil || !aa.Conocido || len(aa.Valor) != 1 || aa.Valor[0].Sexo != d.Nino || aa.Valor[0].Nacimiento.Year() != 1980 || aa.Valor[0].NivelEducativo != "SECUNDARIOS" {
					t.Fatal("dimensiones parentales", aa, err)
				}
				bytes, err := json.Marshal(exp)
				if err != nil {
					return err
				}
				for _, prohibido := range []string{"Carlos", "Ortega Serrano", "Calle Olivo", "958123456"} {
					if strings.Contains(string(bytes), prohibido) {
						t.Fatal("identidad o contacto en estadística")
					}
				}
				estado, _, err := exp.Estados.ValorEn(corte, caso.conocido)
				if err != nil {
					t.Fatal("estados", err)
				}
				if corteEstado := time.Now().UTC(); !corteEstado.IsZero() {
					est, _, fallo := exp.Estados.ValorEn(corteEstado, caso.conocido)
					if fallo != nil || !est.Conocido || est.Valor != "aceptada" {
						t.Fatal("historia de estado", est, fallo)
					}
				}
				_ = estado
			}
			if !encontrado {
				t.Fatal("familia no visible para su equipo")
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	antesFirma := time.Now().UTC()
	deps.Hitos = &adaptador.Repositorio{}
	_, err = app.Nuevo(deps).ConfirmarHito(ctx, a, id, "estadistica-consentimiento-"+id, dominio.ConfirmacionHito{Tipo: "consentimiento", Fecha: "2026-02-07", Version: ultimo.Version})
	if err != nil {
		t.Fatal(err)
	}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
		corte, _ := time.Parse("2006-01-02", "2026-06-01")
		for _, known := range []time.Time{antesFirma, time.Now().UTC()} {
			rows, err := lector.ConsultarMemoria(ctx, a.EquipoID, corte, known)
			if err != nil {
				return err
			}
			for _, exp := range rows {
				if exp.ID == id {
					cons, _, err := exp.Consentimiento.ValorEn(corte, known)
					if err != nil {
						return err
					}
					if cons.Conocido == known.Equal(antesFirma) {
						t.Fatal("corte de conocimiento del consentimiento", cons)
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func comprobarLecturaEstadisticaCierre(t *testing.T, b *postgres.Base, ctx context.Context, a dominio.Actor, id string, reabierto bool) {
	t.Helper()
	lector := &adaptador.LecturaEstadistica{Reglas: &catalogo.Repositorio{}}
	err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
		corte, _ := time.Parse("2006-01-02", "2026-03-31")
		filas, err := lector.ConsultarMemoria(ctx, "", corte, time.Now().UTC())
		if err != nil {
			return err
		}
		for _, exp := range filas {
			if exp.ID != id {
				continue
			}
			n := 1
			if reabierto {
				n = 2
			}
			if len(exp.Episodios) != n {
				t.Fatal("cadena de episodios", len(exp.Episodios), n)
			}
			encontrados := false
			for _, ep := range exp.Episodios {
				if ep.ReaperturaDe == "" {
					encontrados = true
					if ep.Cierre.Format("2006-01-02") != "2026-03-01" || ep.Motivo == "" {
						t.Fatal("cierre confirmado sin fecha o categoría", ep)
					}
				} else if ep.Aceptacion.IsZero() {
					t.Fatal("Acta de aceptación no conservada en reapertura")
				}
			}
			if !encontrados {
				t.Fatal("episodio raíz ausente")
			}
			if reabierto {
				cons, _, err := exp.Consentimiento.ValorEn(corte, time.Now().UTC())
				if err != nil || cons.Conocido {
					t.Fatal("firma anterior se aplicó a reapertura", cons, err)
				}
			}
			_, _, err = lector.ConsultarSPAF(ctx, d.FiltroSPAF{Anio: 2026, Mes: 3, EquipoID: exp.EquipoID}, corte, time.Now().UTC())
			if err != nil {
				return err
			}
			return nil
		}
		t.Fatal("expediente cerrado no visible en fuente")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
