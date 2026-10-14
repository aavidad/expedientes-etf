package documento_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	word "github.com/aavidad/expedientes-etf/internal/adaptadores/docx/documento"
	ficheros "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/documento"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/documento"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/documento"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

type antivirusLimpio struct{}

func (antivirusLimpio) Revisar(context.Context, []byte) (plataforma.EstadoAntivirus, error) {
	return plataforma.AntivirusLimpio, nil
}
func TestPlantillasPublicacionPostgres(t *testing.T) {
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
	identidad := plataforma.Identidad{UsuarioID: usuario, Perfiles: a.Perfiles}
	r := repo.Repositorio{}
	reg := postgres.NuevoRegistroAuditoria(base)
	lectura := ficheros.Plantillas{Raiz: "../../../../idiomas", Paquetes: postgres.NuevaPublicacionPaquetes(base)}
	almacen := ficheros.Plantillas{Raiz: t.TempDir()}
	servicio := app.NuevaAdministracionPlantillas(app.DependenciasPlantillas{Repositorio: r, Almacen: almacen, Validador: word.ValidadorPlantillas{}, Antivirus: antivirusLimpio{}, Idiomas: lectura, Ficheros: almacen, Transacciones: base, Registro: reg, RegistroTransaccional: reg, Reloj: reloj{}, Identificadores: ids{}})
	actual, err := servicio.Actual(ctx, a, d.OficioSSCC, "en")
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("../../../../idiomas/en/plantillas/entrada/aceptacion/oficio_sscc/v1.0.1.docx")
	if err != nil {
		t.Fatal(err)
	}
	if d.PlantillaPublicada(actual.Plantilla) {
		if err := almacen.Guardar(ctx, actual.Plantilla, b); err != nil {
			t.Fatal(err)
		}
	} else {
		ruta := filepath.Join(almacen.Raiz, "en", actual.Plantilla.Ruta)
		if err := os.MkdirAll(filepath.Dir(ruta), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(ruta, b, 0400); err != nil {
			t.Fatal(err)
		}
	}
	versionNueva := func(n int) string { return fmt.Sprintf("%d.%d.0", n, time.Now().UnixNano()) }
	// Dos conexiones publican a partir de la misma referencia. Una sola gana.
	var wg sync.WaitGroup
	resultados := make(chan d.ReferenciaPlantilla, 2)
	fallos := make(chan error, 2)
	for _, v := range []string{versionNueva(2), versionNueva(3)} {
		wg.Add(1)
		go func(v string) {
			defer wg.Done()
			res, err := servicio.Publicar(ctx, a, d.PublicarPlantilla{Tipo: d.OficioSSCC, Idioma: "en", Version: v, VersionPrevia: actual.Revision}, b)
			resultados <- res
			fallos <- err
		}(v)
	}
	wg.Wait()
	close(resultados)
	close(fallos)
	ok, conflictos := 0, 0
	for err := range fallos {
		var c *d.ConflictoPlantilla
		if err == nil {
			ok++
		} else if errors.As(err, &c) && c.VersionActual == actual.Revision+1 {
			conflictos++
		} else {
			t.Fatal(err)
		}
	}
	if ok != 1 || conflictos != 1 {
		t.Fatal(ok, conflictos)
	}
	var ganadora d.ReferenciaPlantilla
	for r := range resultados {
		if r.Revision != 0 {
			ganadora = r
		}
	}
	var eventos, archivos int
	if err := base.Ejecutar(ctx, identidad, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM documento.activacion_plantilla WHERE tipo_documento=(SELECT tipo_documento FROM documento.plantilla WHERE id=$1) AND plantilla_id=$2`, actual.Plantilla.ID, ganadora.Plantilla.ID).Scan(&archivos); err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	exportacion, err := reg.Exportar(ctx, identidad, 1, 9007199254740991)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range exportacion.Eventos {
		if e.Accion == "documento.plantilla_activar" && e.EntidadID == ganadora.Plantilla.ID {
			eventos++
		}
	}
	if archivos != 1 || eventos != 1 {
		t.Fatal(archivos, eventos)
	}
	recuperada, err := almacen.Leer(ctx, d.IdiomaPlantilla{Ruta: ".versiones/en/4.0.0/huella/id/en"}, ganadora.Plantilla)
	if err != nil || !bytes.Equal(recuperada, b) {
		t.Fatal(err)
	}
	// Una versión repetida no cambia la activa y elimina sólo su ruta huérfana.
	repetida := servicio
	_, err = repetida.Publicar(ctx, a, d.PublicarPlantilla{Tipo: d.OficioSSCC, Idioma: "en", Version: ganadora.Plantilla.Version, VersionPrevia: ganadora.Revision}, b)
	if !errors.Is(err, d.ErrVersionPlantilla) {
		t.Fatal(err)
	}
	// Un fallo posterior a activar revierte referencia, archivo y auditoría.
	id, err := (ids{}).Nuevo()
	if err != nil {
		t.Fatal(err)
	}
	nueva := ganadora.Plantilla
	nueva.ID = id
	nueva.Version = versionNueva(9)
	nueva.Ruta = d.RutaPublicada(nueva)
	forzar := errors.New("prueba.rollback")
	err = base.Ejecutar(ctx, identidad, func(ctx context.Context) error {
		if _, err := r.Publicar(ctx, nueva, ganadora.Revision); err != nil {
			return err
		}
		return forzar
	})
	if !errors.Is(err, forzar) {
		t.Fatal(err)
	}
	if err := base.Ejecutar(ctx, identidad, func(ctx context.Context) error {
		actual, err := r.Actual(ctx, d.OficioSSCC, "en")
		if err != nil {
			return err
		}
		if actual.Plantilla.ID != ganadora.Plantilla.ID {
			t.Fatal("referencia confirmada tras rollback")
		}
		tx, _ := postgres.DesdeContexto(ctx)
		var n int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM documento.plantilla WHERE id=$1`, id).Scan(&n); err != nil {
			return err
		}
		if n != 0 {
			t.Fatal("plantilla persistida tras rollback")
		}

		return nil
	}); err != nil {
		t.Fatal(err)
	}
	exportacion, err = reg.Exportar(ctx, identidad, 1, 9007199254740991)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range exportacion.Eventos {
		if e.Accion == "documento.plantilla_activar" && e.EntidadID == id {
			t.Fatal("auditoría persistida tras rollback")
		}
	}
	// La función SQL comprueba perfil aunque se eluda la capa de aplicación.
	for _, perfil := range []string{"coordinacion", "consulta", "equipo", ""} {
		err := base.Ejecutar(ctx, plataforma.Identidad{UsuarioID: usuario, Perfiles: []string{perfil}}, func(ctx context.Context) error { _, err := r.Publicar(ctx, nueva, ganadora.Revision); return err })
		if err == nil {
			t.Fatalf("SQL aceptó perfil %q", perfil)
		}
	}
	if err := base.Ejecutar(ctx, identidad, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		_, err := tx.Exec(ctx, `UPDATE documento.plantilla SET huella_sha256=decode(repeat('a',64),'hex') WHERE id=$1`, ganadora.Plantilla.ID)
		return err
	}); err == nil {
		t.Fatal("huella modificable por etf_app")
	}
	if err := base.Ejecutar(ctx, identidad, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		_, err := tx.Exec(ctx, `DELETE FROM documento.archivo_plantilla WHERE nueva_id=$1`, ganadora.Plantilla.ID)
		return err
	}); err == nil {
		t.Fatal("historial borrable por etf_app")
	}
	t.Log("PUBLICACION-OK PostgreSQL: una activación, archivo, auditoría, rollback y permisos")
}
