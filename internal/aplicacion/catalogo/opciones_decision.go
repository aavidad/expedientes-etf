package catalogo

import (
	"context"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo/puertos/entrada"
	paquete "github.com/aavidad/expedientes-etf/internal/aplicacion/paquete/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	idiomas "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	registro "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/catalogo"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"slices"
	"time"
)

type ConsultaDecision struct {
	Repositorio   salida.Repositorio
	Textos        paquete.PublicacionCatalogos
	Transacciones plataforma.Transacciones
	Registro      registro.RegistroTransaccional
	Ahora         func() time.Time
}

func (c ConsultaDecision) ConsultarOpcionesDecision(ctx context.Context, a entrada.Actor, idioma string) (entrada.OpcionesDecision, error) {
	return c.consultarOpciones(ctx, a, idioma, false)
}

func (c ConsultaDecision) consultarOpciones(ctx context.Context, a entrada.Actor, idioma string, carga bool) (entrada.OpcionesDecision, error) {
	coordinacion := slices.Contains(a.Perfiles, "coordinacion")
	if a.UsuarioID == "" || !coordinacion && (!carga || !slices.Contains(a.Perfiles, "profesional_etf") || a.EquipoID == "") {
		return entrada.OpcionesDecision{}, dominio.ErrPermiso
	}
	if idioma == "" || len(idioma) > 16 {
		return entrada.OpcionesDecision{}, dominio.ErrEdicion
	}
	var r entrada.OpcionesDecision
	err := c.Transacciones.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
		fecha := dominio.FechaCivil(c.Ahora())
		v, err := c.Repositorio.Vigente(ctx, "asignacion_municipios", fecha)
		if err != nil {
			return err
		}
		d, err := c.Repositorio.Datos(ctx, v)
		if err != nil {
			return err
		}
		textos, err := c.Textos.Catalogos(ctx)
		if err != nil {
			return err
		}
		r.Textos = map[string]string{}
		for clave, valor := range textos.Textos[idiomas.IdiomaPredeterminado] {
			r.Textos[clave] = valor
		}
		for clave, valor := range textos.Textos[idioma] {
			if valor != "" {
				r.Textos[clave] = valor
			}
		}
		agregar := func(clave string) {
			texto := textos.Textos[idioma][clave]
			if texto == "" {
				texto = textos.Textos[idiomas.IdiomaPredeterminado][clave]
			}
			r.Textos[clave] = texto
		}
		r.Equipos = []entrada.EquipoDecision{}
		r.Centros = []entrada.EquipoDecision{}
		r.Municipios = []entrada.MunicipioDecision{}
		r.Subprogramas = []entrada.OpcionDecision{}
		for _, e := range d.Equipos {
			if (!carga || coordinacion || e.ID == a.EquipoID) && !fecha.Before(e.ActivoDesde) && (e.ActivoHasta == nil || fecha.Before(*e.ActivoHasta)) {
				r.Equipos = append(r.Equipos, entrada.EquipoDecision{ID: e.ID, Clave: e.ClaveTexto, Codigo: e.Codigo})
				agregar(e.ClaveTexto)
			}
		}
		for _, e := range d.Centros {
			if !fecha.Before(e.ActivoDesde) && (e.ActivoHasta == nil || fecha.Before(*e.ActivoHasta)) {
				r.Centros = append(r.Centros, entrada.EquipoDecision{ID: e.ID, Clave: e.ClaveTexto, Codigo: e.Codigo})
				agregar(e.ClaveTexto)
			}
		}
		for _, m := range d.Municipios {
			if carga && !coordinacion && (m.EquipoID == nil || *m.EquipoID != a.EquipoID) {
				continue
			}
			r.Municipios = append(r.Municipios, entrada.MunicipioDecision{ID: m.ID, INE: m.INE, Clave: m.ClaveTexto, CentroID: m.CentroID, EquipoID: m.EquipoID, Ambito: m.AmbitoDiputacion})
			agregar(m.ClaveTexto)
		}
		v, err = c.Repositorio.Vigente(ctx, "listas", fecha)
		if err != nil {
			return err
		}
		d, err = c.Repositorio.Datos(ctx, v)
		if err != nil {
			return err
		}
		for _, o := range d.Opciones {
			if o.Lista == "subprograma" && o.Activa {
				clave := o.Lista + "." + o.Codigo
				r.Subprogramas = append(r.Subprogramas, entrada.OpcionDecision{Codigo: o.Codigo, Clave: clave})
				agregar(clave)
			}
		}
		accion := "catalogo.consultar_opciones_decision"
		if carga {
			accion = "catalogo.consultar_opciones_carga"
		}
		_, err = c.Registro.RegistrarEnTransaccion(ctx, auditoria.Datos{OcurridoEn: c.Ahora().UTC().Truncate(time.Microsecond), UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, Accion: accion, Entidad: "catalogo", Resultado: "ok"})
		return err
	})
	if err != nil {
		return entrada.OpcionesDecision{}, err
	}
	return r, nil
}
