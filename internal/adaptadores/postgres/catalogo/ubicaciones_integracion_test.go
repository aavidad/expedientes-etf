package catalogo_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	repo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo/puertos/entrada"
	paquetes "github.com/aavidad/expedientes-etf/internal/aplicacion/paquete"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

// Comparte los archivos y referencias del ensayo de edición; no deja referencias
// activas apuntando a directorios temporales de otra prueba.
func probarUbicaciones(t *testing.T, b *postgres.Base, ctx context.Context, editor *app.Edicion, publicador *paquetes.Publicacion) {
	t.Helper()
	actor := entrada.Actor{UsuarioID: actorID, Perfiles: []string{"administracion"}}
	identidad := plataforma.Identidad{UsuarioID: actorID, Perfiles: actor.Perfiles}
	estado, err := editor.Consultar(ctx, actor, "asignacion_municipios")
	if err != nil {
		t.Fatal(err)
	}
	anterior := estado.Datos
	var municipio dominio.Municipio
	for _, m := range anterior.Municipios {
		if m.INE == "18102" {
			municipio = m
		}
	}
	if municipio.ID == "" {
		t.Fatal("Íllora ausente")
	}
	equipo := anterior.Equipos[0].ID
	centro := anterior.Centros[0].ID
	desde := time.Date(2036, 1, 1, 0, 0, 0, 0, time.UTC)
	cambio := entrada.Cambio{VersionEsperada: anterior.Version.Numero, VigenteDesde: desde, ReferenciasEsperadas: estado.Referencias, Textos: map[string]string{"es": "Íllora", "en": "Íllora"}, Ubicacion: &dominio.Ubicacion{Tipo: "municipio", ID: municipio.ID, Codigo: municipio.INE, CentroID: &centro, EquipoID: &equipo, AmbitoDiputacion: !municipio.AmbitoDiputacion, Variantes: map[string][]string{"es": {"Villa de Íllora"}, "en": {"Illora town"}}}}
	for _, perfil := range []string{"coordinacion", "profesional_etf", "consulta", "sistema", ""} {
		if _, err := editor.Guardar(ctx, entrada.Actor{UsuarioID: actorID, Perfiles: []string{perfil}}, "asignacion_municipios", cambio); !errors.Is(err, dominio.ErrPermiso) {
			t.Fatal("permiso aplicación", perfil, err)
		}
		err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actorID, Perfiles: []string{perfil}}, func(ctx context.Context) error {
			tx, _ := postgres.DesdeContexto(ctx)
			_, err := tx.Exec(ctx, `SELECT catalogo.editar_ubicacion($1,$2,'{}')`, cambio.VersionEsperada, desde)
			return err
		})
		if err == nil {
			t.Fatal("permiso SQL", perfil)
		}
	}
	fallido := app.NuevaEdicion(&repo.Repositorio{}, publicador, b, auditoriaFallida{}, time.Now)
	if _, err := fallido.Guardar(ctx, actor, "asignacion_municipios", cambio); err == nil {
		t.Fatal("auditoría fallida admitida")
	}
	despues, err := editor.Consultar(ctx, actor, "asignacion_municipios")
	if err != nil || !reflect.DeepEqual(estado, despues) {
		t.Fatal("rollback datos e idiomas", err)
	}
	estado, err = editor.Guardar(ctx, actor, "asignacion_municipios", cambio)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range estado.Datos.Municipios {
		if m.ID == municipio.ID && (m.CentroID == nil || *m.CentroID != centro || m.EquipoID == nil || *m.EquipoID != equipo || m.AmbitoDiputacion == municipio.AmbitoDiputacion) {
			t.Fatal("asignaciones y ámbito")
		}
	}
	nombres := app.NombresUbicaciones{Publicacion: publicador, Idioma: "es"}
	err = b.Ejecutar(ctx, identidad, func(ctx context.Context) error {
		m, err := nombres.Nombres(ctx)
		if err != nil {
			return err
		}
		if m[municipio.ClaveTexto+".variante.v1"] != "Villa de Íllora" {
			t.Fatal("grafía no activada")
		}
		historicos, err := (&repo.Repositorio{}).Datos(ctx, anterior.Version)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(historicos.Municipios, anterior.Municipios) || !reflect.DeepEqual(historicos.Equipos, anterior.Equipos) || !reflect.DeepEqual(historicos.Centros, anterior.Centros) {
			t.Fatal("instantánea histórica alterada")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var conflicto *dominio.Conflicto
	if _, err := editor.Guardar(ctx, actor, "asignacion_municipios", cambio); err == nil {
		t.Fatal("doble publicación admitida")
	} else if !errors.As(err, &conflicto) && err.Error() != "catalogo.conflicto" {
		t.Fatal(err)
	}
	// Alta nueva, baja posterior y conservación de la versión previa.
	cambio = entrada.Cambio{VersionEsperada: estado.Datos.Version.Numero, VigenteDesde: desde.AddDate(0, 0, 1), ReferenciasEsperadas: estado.Referencias, Textos: map[string]string{"es": "Centro de la Vega Norte", "en": "North Vega Centre"}, Ubicacion: &dominio.Ubicacion{Tipo: "centro", Codigo: "VEGA_NORTE", ActivoDesde: desde.AddDate(0, 0, 1)}}
	estado, err = editor.Guardar(ctx, actor, "asignacion_municipios", cambio)
	if err != nil {
		t.Fatal(err)
	}
	var nuevo dominio.Centro
	for _, c := range estado.Datos.Centros {
		if c.Codigo == "VEGA_NORTE" {
			nuevo = c
		}
	}
	if nuevo.ID == "" {
		t.Fatal("alta centro")
	}
	versionAlta := estado.Datos.Version
	baja := desde.AddDate(0, 0, 3)
	cambio.Ubicacion.ID = nuevo.ID
	cambio.Ubicacion.ActivoHasta = &baja
	cambio.VersionEsperada = estado.Datos.Version.Numero
	cambio.ReferenciasEsperadas = estado.Referencias
	cambio.VigenteDesde = desde.AddDate(0, 0, 2)
	estado, err = editor.Guardar(ctx, actor, "asignacion_municipios", cambio)
	if err != nil {
		t.Fatal(err)
	}
	err = b.Ejecutar(ctx, identidad, func(ctx context.Context) error {
		r := &repo.Repositorio{}
		alta, err := r.Datos(ctx, versionAlta)
		if err != nil {
			return err
		}
		for _, c := range alta.Centros {
			if c.ID == nuevo.ID && c.ActivoHasta != nil {
				t.Fatal("baja reescribe alta")
			}
		}
		originales, err := r.Datos(ctx, anterior.Version)
		if err != nil {
			return err
		}
		for _, c := range originales.Centros {
			if c.ID == nuevo.ID {
				t.Fatal("alta contamina versión anterior")
			}
		}
		for _, c := range estado.Datos.Centros {
			if c.ID == nuevo.ID && (c.ActivoHasta == nil || !c.ActivoHasta.Equal(baja)) {
				t.Fatal("baja perdida")
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	tx := migrador(t, ctx)
	rechaza(t, ctx, tx, `UPDATE catalogo.ubicacion_vigencia SET activo_hasta=DATE '2039-01-01' WHERE version_id=$1`, "P0001", estado.Datos.Version.ID)
	rechaza(t, ctx, tx, `DELETE FROM catalogo.municipio_ambito WHERE version_id=$1`, "P0001", estado.Datos.Version.ID)
	t.Log("UBICACIONES-OK asignaciones independientes, ámbito, grafías publicadas, alta/baja e historia; permisos y auditoría fallida revierten datos y textos")
}
