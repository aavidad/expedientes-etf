package actividad

import (
	"context"
	"errors"
	paquetes "github.com/aavidad/expedientes-etf/internal/aplicacion/paquete/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/actividad"
	auditoria "github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/actividad"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"testing"
	"time"
)

type transaccionEtiquetas struct{ identidad plataforma.Identidad }

func (t *transaccionEtiquetas) Ejecutar(ctx context.Context, identidad plataforma.Identidad, f func(context.Context) error) error {
	t.identidad = identidad
	return f(ctx)
}

type auditoriaEtiquetas struct {
	fallar bool
	accion string
}

func (a *auditoriaEtiquetas) Confirmar(_ context.Context, dato auditoria.Datos) (auditoria.Evento, error) {
	a.accion = dato.Accion
	if a.fallar {
		return auditoria.Evento{}, errors.New("auditoria.no_disponible")
	}
	return auditoria.Evento{}, nil
}

type catalogosEtiquetas struct {
	paquetes.PublicacionCatalogos
	llamadas int
}

func (c *catalogosEtiquetas) Catalogos(context.Context) (paquetes.Catalogos, error) {
	c.llamadas++
	return paquetes.Catalogos{Textos: map[string]map[string]string{"es": {"equipo.02": "Vega Temple / Santa Fe", "actividad.organismo.JUZGADOS": "Juzgados", "municipio.18001": "Agrón"}}}, nil
}

func TestEtiquetasAutorizaProyectaYConfirmaAuditoria(t *testing.T) {
	for _, caso := range []struct {
		perfil         string
		fallaAuditoria bool
		permitido      bool
	}{
		{"profesional_etf", false, true}, {"coordinacion", false, true}, {"consulta", false, false}, {"administracion", false, false}, {"coordinacion", true, false},
	} {
		t.Run(caso.perfil, func(t *testing.T) {
			tx := &transaccionEtiquetas{}
			audit := &auditoriaEtiquetas{fallar: caso.fallaAuditoria}
			catalogos := &catalogosEtiquetas{}
			s := Nuevo(struct{ salida.Repositorio }{}, tx, audit, time.Now, Opciones{Etiquetas: catalogos})
			actor := dominio.Actor{UsuarioID: "11111111-1111-4111-8111-111111111111", EquipoID: "22222222-2222-4222-8222-222222222222", Perfiles: []string{caso.perfil}}
			textos, err := s.Etiquetas(context.Background(), actor)
			if !caso.permitido {
				if err == nil || textos != nil {
					t.Fatal("datos entregados sin permiso o auditoría")
				}
				if !caso.fallaAuditoria && catalogos.llamadas != 0 {
					t.Fatal("catálogo leído sin permiso")
				}
				return
			}
			if err != nil || len(textos["es"]) != 2 || textos["es"]["equipo.02"] == "" || textos["es"]["municipio.18001"] != "" {
				t.Fatal("proyección incorrecta", err, textos)
			}
			if audit.accion != "actividad.consultar_etiquetas" || tx.identidad.UsuarioID != actor.UsuarioID || tx.identidad.EquipoID != actor.EquipoID {
				t.Fatal("lectura sin identidad o auditoría")
			}
		})
	}
}
