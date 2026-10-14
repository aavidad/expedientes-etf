package estadistica_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	word "github.com/aavidad/expedientes-etf/internal/adaptadores/docx/documento"
	idiomas "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/documento"
	reglas "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/estadistica"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	actividad "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/actividad"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	documental "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/documento"
	adaptador "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/estadistica"
	expediente "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"

	documentoApp "github.com/aavidad/expedientes-etf/internal/aplicacion/documento"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	documento "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"os"
	"testing"
	"time"
)

type reloj struct{}

func (reloj) Ahora() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }

type auditoriaFallida struct{}

func (auditoriaFallida) RegistrarEnTransaccion(context.Context, auditoria.Datos) (auditoria.Evento, error) {
	return auditoria.Evento{}, errors.New("auditoria.no_disponible")
}
func (auditoriaFallida) Confirmar(context.Context, auditoria.Datos) (auditoria.Evento, error) {
	return auditoria.Evento{}, errors.New("auditoria.no_disponible")
}
func preparar(t *testing.T) (*postgres.Base, context.Context, d.ActorMemoria, string) {
	t.Helper()
	if os.Getenv("ETF_POSTGRES_INTEGRACION") != "1" {
		t.Skip("ETF_POSTGRES_INTEGRACION=1")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	b, err := postgres.Nueva(ctx, os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(b.Cerrar)
	var guid identidad.GUID
	_, err = rand.Read(guid[:])
	if err != nil {
		t.Fatal(err)
	}
	u, err := postgres.NuevosUsuarios(b).RegistrarAcceso(ctx, identidad.IdentidadDirectorio{GUID: guid, CuentaAD: "maria.memoria", NombreMostrado: "María López Aguilar"}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	a := d.ActorMemoria{UsuarioID: u.ID, Perfiles: []string{"coordinacion"}}
	var equipo string
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		tx, err := postgres.DesdeContexto(ctx)
		if err != nil {
			return err
		}
		return tx.QueryRow(ctx, `SELECT id::text FROM catalogo.equipo WHERE codigo='08'`).Scan(&equipo)
	})
	if err != nil {
		t.Fatal(err)
	}
	return b, ctx, a, equipo
}
func TestNarrativasVersionAuditoriaRLSYRollback(t *testing.T) {
	b, ctx, a, equipo := preparar(t)
	repo := &adaptador.Repositorio{}
	deps := app.DependenciasNarrativas{Repositorio: repo, Transacciones: b, Auditoria: postgres.NuevoRegistroAuditoria(b), Reloj: reloj{}}
	svc := app.NuevoNarrativas(deps)
	f := d.FiltroMemoria{Anio: 2026, EquipoID: equipo}
	vacia, err := svc.LeerNarrativas(ctx, a, f)
	if err != nil {
		t.Fatal(err)
	}
	n := vacia
	n.D = "Las visitas semanales permitieron acordar pautas de cuidado con las familias."
	n.A.EntidadLocal = "Entidad titular del servicio"
	n, err = svc.GuardarNarrativas(ctx, a, n)
	if err != nil {
		t.Fatal(err)
	}
	if n.Version != vacia.Version+1 || !d.UUIDValidoSPAF(n.ID) {
		t.Fatal("versión o identidad", n.Version, n.ID)
	}
	actual, err := svc.LeerNarrativas(ctx, a, f)
	if err != nil || actual.D != n.D || actual.Version != n.Version {
		t.Fatal("recuperación", err)
	}
	repetida := n
	repetida.Version--
	_, err = svc.GuardarNarrativas(ctx, a, repetida)
	var conflicto *d.ConflictoMemoria
	if !errors.As(err, &conflicto) || conflicto.Actual != n.Version {
		t.Fatal("conflicto esperado", err)
	}
	// El fallo de auditoría revierte tanto la versión como la historia.
	deps.Auditoria = auditoriaFallida{}
	n.D = "Se revisaron los acuerdos de asistencia escolar."
	_, err = app.NuevoNarrativas(deps).GuardarNarrativas(ctx, a, n)
	if !errors.Is(err, d.ErrDisponibleMemoria) {
		t.Fatal("auditoría", err)
	}
	actual, err = svc.LeerNarrativas(ctx, a, f)
	if err != nil || actual.Version != n.Version || actual.D == n.D {
		t.Fatal("escritura sin auditoría", err)
	}
	for _, p := range []string{"consulta", "administracion", "profesional_etf"} {
		actor := a
		actor.Perfiles = []string{p}
		if p == "profesional_etf" {
			actor.EquipoID = "00000000-0000-4000-8000-000000000012"
		}
		_, err = svc.LeerNarrativas(ctx, actor, f)
		if !errors.Is(err, d.ErrPermisoMemoria) {
			t.Fatal("perfil vio textos", p, err)
		}
		err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: actor.Perfiles, EquipoID: actor.EquipoID}, func(ctx context.Context) error {
			tx, err := postgres.DesdeContexto(ctx)
			if err != nil {
				return err
			}
			var count int
			err = tx.QueryRow(ctx, `SELECT count(*) FROM estadistica.memoria WHERE id=$1`, actual.ID).Scan(&count)
			if count != 0 {
				t.Fatal("RLS memoria", p, count)
			}
			if err != nil {
				return err
			}
			err = tx.QueryRow(ctx, `SELECT count(*) FROM estadistica.memoria_narrativa_historia WHERE memoria_id=$1`, actual.ID).Scan(&count)
			if count != 0 {
				t.Fatal("RLS historia", p, count)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

type ids struct{}

func (ids) Nuevo() (string, error) {
	var b [16]byte
	_, err := rand.Read(b[:])
	b[6] = b[6]&15 | 64
	b[8] = b[8]&63 | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:]), err
}
func TestMemoriaWordPostgresIdempotenciaYAuditoria(t *testing.T) {
	b, ctx, a, equipo := preparar(t)
	repo := &adaptador.Repositorio{}
	ficheros := idiomas.Plantillas{Raiz: "../../../../idiomas", Paquetes: postgres.NuevaPublicacionPaquetes(b)}
	fuente := app.FuenteMemoria{Expedientes: &expediente.LecturaEstadistica{Reglas: &catalogo.Repositorio{}}, Reglas: reglas.Reglas{Raiz: "../../../../datos/catalogos/memoria", SMI: &catalogo.Repositorio{}}, Equipos: &catalogo.Repositorio{}, Actividad: &actividad.Repositorio{}, Narrativas: repo}
	deps := documentoApp.DependenciasMemoria{Repositorio: documental.Repositorio{}, Fuente: fuente, Idiomas: ficheros, Ficheros: ficheros, Textos: ficheros, Motor: word.Motor{}, Transacciones: b, Registro: postgres.NuevoRegistroAuditoria(b), Reloj: reloj{}, Identificadores: ids{}}
	svc := documentoApp.NuevoMemoria(deps)
	actor := documento.Actor{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}
	p := documento.SolicitudMemoria{Anio: 2026, EquipoID: equipo, Idioma: "es", Clave: "memoria-" + a.UsuarioID}
	r, err := svc.EmitirMemoria(ctx, actor, p)
	if err != nil {
		t.Fatal(err)
	}
	if r.Repetida || r.Emision.HuellaResultado != documento.Huella(r.Contenido) {
		t.Fatal("emisión", r.Emision)
	}
	if _, err := zip.NewReader(bytes.NewReader(r.Contenido), int64(len(r.Contenido))); err != nil {
		t.Fatal("Word inválido", err)
	}
	repetida, err := svc.EmitirMemoria(ctx, actor, p)
	if err != nil || !repetida.Repetida || repetida.Emision.ID != r.Emision.ID || !bytes.Equal(repetida.Contenido, r.Contenido) {
		t.Fatal("repetición", err)
	}
	// Una edición posterior conserva la narrativa de la emisión ya confirmada.
	narrativa, err := app.NuevoNarrativas(app.DependenciasNarrativas{Repositorio: repo, Transacciones: b, Auditoria: postgres.NuevoRegistroAuditoria(b), Reloj: reloj{}}).LeerNarrativas(ctx, a, d.FiltroMemoria{Anio: 2026, EquipoID: equipo})
	if err != nil {
		t.Fatal(err)
	}
	narrativa.D = "Las reuniones con las familias se ajustaron a sus horarios."
	_, err = app.NuevoNarrativas(app.DependenciasNarrativas{Repositorio: repo, Transacciones: b, Auditoria: postgres.NuevoRegistroAuditoria(b), Reloj: reloj{}}).GuardarNarrativas(ctx, a, narrativa)
	if err != nil {
		t.Fatal(err)
	}
	repetida, err = svc.EmitirMemoria(ctx, actor, p)
	if err != nil || !bytes.Equal(repetida.Contenido, r.Contenido) {
		t.Fatal("se alteró una emisión anterior", err)
	}
	distinta := p
	distinta.Anio = 2025
	_, err = svc.EmitirMemoria(ctx, actor, distinta)
	if !errors.Is(err, documento.ErrIdempotencia) {
		t.Fatal("clave reutilizada", err)
	}
	deps.Registro = auditoriaFallida{}
	fallida := p
	fallida.Clave = "fallo-" + a.UsuarioID
	_, err = documentoApp.NuevoMemoria(deps).EmitirMemoria(ctx, actor, fallida)
	if !errors.Is(err, documento.ErrNoDisponible) {
		t.Fatal("fallo de auditoría", err)
	}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		previa, err := (documental.Repositorio{}).PeticionMemoria(ctx, a.UsuarioID, fallida.Clave)
		if previa.ID != "" {
			t.Fatal("emisión confirmada sin auditoría")
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, perfil := range []string{"consulta", "administracion"} {
		otro := actor
		otro.Perfiles = []string{perfil}
		_, err = svc.EmitirMemoria(ctx, otro, p)
		if !errors.Is(err, d.ErrPermisoMemoria) {
			t.Fatal("descarga sin permiso", perfil, err)
		}
		err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: otro.Perfiles}, func(ctx context.Context) error {
			tx, err := postgres.DesdeContexto(ctx)
			if err != nil {
				return err
			}
			var n int
			err = tx.QueryRow(ctx, `SELECT count(*) FROM documento.emision_memoria WHERE id=$1`, r.Emision.ID).Scan(&n)
			if n != 0 {
				t.Fatal("RLS emisión", perfil)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
