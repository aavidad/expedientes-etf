package catalogo

import (
	"context"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	idiomas "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"slices"
	"time"
)

// ConsultaDestinosAnexo6 devuelve destinos activos y etiquetas históricas del Anexo 6.
type ConsultaDestinosAnexo6 struct{ ConsultaDecision }

func (c ConsultaDestinosAnexo6) ConsultarDestinosAnexo6(ctx context.Context, a entrada.Actor, idioma string) (entrada.DestinosAnexo6, error) {
	r := entrada.DestinosAnexo6{Servicios: []entrada.OpcionDecision{}, Textos: map[string]string{}}
	if a.UsuarioID == "" || (!slices.Contains(a.Perfiles, "coordinacion") && !(slices.Contains(a.Perfiles, "profesional_etf") && a.EquipoID != "")) {
		return r, d.ErrPermiso
	}
	if idioma == "" || len(idioma) > 16 {
		return r, d.ErrEdicion
	}
	err := c.Transacciones.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
		ahora := c.Ahora()
		v, err := c.Repositorio.Vigente(ctx, "listas", d.FechaCivil(ahora))
		if err != nil {
			return err
		}
		datos, err := c.Repositorio.Datos(ctx, v)
		if err != nil {
			return err
		}
		textos, err := c.Textos.Catalogos(ctx)
		if err != nil {
			return err
		}
		for _, o := range datos.Opciones {
			if o.Lista != "servicio_destino" {
				continue
			}
			clave := o.Lista + "." + o.Codigo
			if o.Activa {
				r.Servicios = append(r.Servicios, entrada.OpcionDecision{Codigo: o.Codigo, Clave: clave})
			}
			r.Textos[clave] = textos.Textos[idioma][clave]
			if r.Textos[clave] == "" {
				r.Textos[clave] = textos.Textos[idiomas.IdiomaPredeterminado][clave]
			}
		}
		_, err = c.Registro.RegistrarEnTransaccion(ctx, auditoria.Datos{OcurridoEn: ahora.UTC().Truncate(time.Microsecond), UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, Accion: "catalogo.consultar_destinos_anexo6", Entidad: "catalogo", Resultado: "ok"})
		return err
	})
	if err != nil {
		return entrada.DestinosAnexo6{}, err
	}
	return r, nil
}
