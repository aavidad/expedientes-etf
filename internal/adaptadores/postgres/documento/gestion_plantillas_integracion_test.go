package documento_test

import (
	"context"
	"errors"
	"fmt"
	word "github.com/aavidad/expedientes-etf/internal/adaptadores/docx/documento"
	ficheros "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/documento"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/documento"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/documento"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGestionPlantillasPostgres(t *testing.T) {
	if os.Getenv("ETF_DOCUMENTO_POSTGRES_INTEGRACION") != "1" {
		t.Skip("ETF_DOCUMENTO_POSTGRES_INTEGRACION=1")
	}
	ctx := t.Context()
	base, err := postgres.Nueva(ctx, os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	defer base.Cerrar()
	a := d.Actor{UsuarioID: usuario, Perfiles: []string{"administracion"}}
	r := repo.Repositorio{}
	reg := postgres.NuevoRegistroAuditoria(base)
	idiomas := ficheros.Plantillas{Raiz: "../../../../idiomas", Paquetes: postgres.NuevaPublicacionPaquetes(base)}
	almacen := ficheros.Plantillas{Raiz: t.TempDir()}
	servicio := app.NuevaAdministracionPlantillas(app.DependenciasPlantillas{Repositorio: r, Gestion: r, Almacen: almacen, Validador: word.ValidadorPlantillas{}, Antivirus: antivirusLimpio{}, Idiomas: idiomas, Ficheros: almacen, Transacciones: base, Registro: reg, RegistroTransaccional: reg, Reloj: reloj{}, Identificadores: ids{}})
	original, err := servicio.Actual(ctx, a, d.ActaNoAceptacion, "en")
	if err != nil {
		t.Fatal(err)
	}
	version := traduccionesEN(t)[d.ActaNoAceptacion].Nueva
	b, err := os.ReadFile("../../../../idiomas/en/plantillas/entrada/no_aceptacion/v" + version + ".docx")
	if err != nil {
		t.Fatal(err)
	}
	if d.PlantillaPublicada(original.Plantilla) {
		if err := almacen.Guardar(ctx, original.Plantilla, b); err != nil {
			t.Fatal(err)
		}
	} else {
		ruta := filepath.Join(almacen.Raiz, "en", original.Plantilla.Ruta)
		if err := os.MkdirAll(filepath.Dir(ruta), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(ruta, b, 0400); err != nil {
			t.Fatal(err)
		}
	}
	p := d.PublicarPlantilla{Tipo: original.Plantilla.Tipo, Idioma: "en", Version: fmt.Sprintf("9.%d.0", time.Now().UnixNano()), VersionPrevia: original.Revision}
	nueva, err := servicio.Subir(ctx, a, p, b)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := servicio.Actual(ctx, a, p.Tipo, p.Idioma)
	if err != nil || actual.Plantilla.ID != original.Plantilla.ID || actual.Revision != original.Revision {
		t.Fatal("subir cambio activa", actual, err)
	}
	if _, err := servicio.Subir(ctx, a, p, b); !errors.Is(err, d.ErrVersionPlantilla) {
		t.Fatal("version duplicada", err)
	}
	historia, err := servicio.Listar(ctx, a)
	if err != nil {
		t.Fatal(err)
	}
	encontrada := false
	for _, v := range historia {
		if v.Plantilla.ID == nueva.Plantilla.ID {
			encontrada = true
			if v.Activa || len(v.Activaciones) != 0 || v.Antivirus != "limpio" || v.SubidaPor != usuario {
				t.Fatal(v)
			}
		}
	}
	if !encontrada {
		t.Fatal("version subida ausente")
	}
	activada, err := servicio.Activar(ctx, a, d.ActivarPlantilla{Idioma: "en", ID: nueva.Plantilla.ID, VersionPrevia: original.Revision})
	if err != nil || activada.Revision != original.Revision+1 {
		t.Fatal(activada, err)
	}
	_, err = servicio.Activar(ctx, a, d.ActivarPlantilla{Idioma: "en", ID: original.Plantilla.ID, VersionPrevia: original.Revision})
	var conflicto *d.ConflictoPlantilla
	if !errors.As(err, &conflicto) || conflicto.VersionActual != activada.Revision {
		t.Fatal("conflicto ausente", err)
	}
	if _, err := servicio.Subir(ctx, a, d.PublicarPlantilla{Tipo: p.Tipo, Idioma: p.Idioma, Version: "8.0.0", VersionPrevia: original.Revision}, b); !errors.As(err, &conflicto) {
		t.Fatal("subida obsoleta", err)
	}
	restaurada, err := servicio.Activar(ctx, a, d.ActivarPlantilla{Idioma: "en", ID: original.Plantilla.ID, VersionPrevia: activada.Revision})
	if err != nil {
		t.Fatal(err)
	}
	otra, err := servicio.Activar(ctx, a, d.ActivarPlantilla{Idioma: "en", ID: nueva.Plantilla.ID, VersionPrevia: restaurada.Revision})
	if err != nil {
		t.Fatal("reactivacion", err)
	}
	repetida, err := servicio.Activar(ctx, a, d.ActivarPlantilla{Idioma: "en", ID: nueva.Plantilla.ID, VersionPrevia: otra.Revision})
	if err != nil || repetida.Revision != otra.Revision {
		t.Fatal("activar activa no es repetible", err)
	}
	historia, err = servicio.Listar(ctx, a)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range historia {
		if v.Plantilla.ID == nueva.Plantilla.ID {
			if !v.Activa || len(v.Activaciones) != 2 || v.Activaciones[0].Por != usuario || v.Activaciones[0].En.IsZero() {
				t.Fatal(v)
			}
		}
	}
	for _, perfil := range []string{"coordinacion", "consulta", "profesional_etf"} {
		intruso := d.Actor{UsuarioID: usuario, Perfiles: []string{perfil}}
		if _, err := servicio.Listar(ctx, intruso); !errors.Is(err, d.ErrAcceso) {
			t.Fatal(perfil, err)
		}
		if _, err := servicio.Activar(ctx, intruso, d.ActivarPlantilla{Idioma: "en", ID: nueva.Plantilla.ID, VersionPrevia: otra.Revision}); !errors.Is(err, d.ErrAcceso) {
			t.Fatal(perfil, err)
		}
		// La funcion SQL mantiene la barrera aunque se omita el caso de uso.
		err := base.Ejecutar(ctx, plataforma.Identidad{UsuarioID: usuario, Perfiles: []string{perfil}}, func(ctx context.Context) error {
			tx, e := postgres.DesdeContexto(ctx)
			if e != nil {
				return e
			}
			var cuenta int
			if e := tx.QueryRow(ctx, `SELECT count(*) FROM documento.activacion_plantilla`).Scan(&cuenta); e != nil {
				return e
			}
			if cuenta != 0 {
				t.Fatal("RLS expuso historial", perfil, cuenta)
			} //etf:tecnico
			_, e = tx.Exec(ctx, `SELECT * FROM documento.activar_version_plantilla($1,$2)`, nueva.Plantilla.ID, otra.Revision)
			return e //etf:tecnico
		})
		if err == nil {
			t.Fatal("SQL permitio perfil", perfil)
		}
	}
	// Un fichero modificado se rechaza antes de cambiar la referencia.
	ruta := filepath.Join(almacen.Raiz, "en", nueva.Plantilla.Ruta)
	if err := os.Chmod(ruta, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte("alterada"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := servicio.Activar(ctx, a, d.ActivarPlantilla{Idioma: "en", ID: nueva.Plantilla.ID, VersionPrevia: otra.Revision}); err == nil {
		t.Fatal("fichero alterado activado")
	}
}
