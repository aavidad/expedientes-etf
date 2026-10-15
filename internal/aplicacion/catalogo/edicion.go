package catalogo

import (
	"context"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo/puertos/entrada"
	paquetes "github.com/aavidad/expedientes-etf/internal/aplicacion/paquete/puertos/entrada"
	auditoria "github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	registro "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/catalogo"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"time"
)

type Edicion struct {
	repo     salida.Edicion
	paquetes paquetes.PublicacionCatalogos
	tx       plataforma.Transacciones
	registro registro.Registro
	ahora    func() time.Time
}

func NuevaEdicion(r salida.Edicion, p paquetes.PublicacionCatalogos, tx plataforma.Transacciones, a registro.Registro, ahora func() time.Time) *Edicion {
	return &Edicion{r, p, tx, a, ahora}
}
func (e *Edicion) ejecutar(ctx context.Context, a entrada.Actor, op func(context.Context) error) error {
	return e.tx.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, op)
}
func (e *Edicion) consultar(ctx context.Context, ambito string) (entrada.Administracion, error) {
	var a entrada.Administracion
	var err error
	if ambito == "firmantes" {
		var firmantes []dominio.Firmante
		firmantes, err = e.repo.Firmantes(ctx)
		if err != nil {
			return entrada.Administracion{}, err
		}
		a.Firmantes = []entrada.FirmanteVista{}
		// Las traducciones históricas se leen una vez por referencia y se reutilizan.
		cache := map[string]map[string]string{}
		for _, f := range firmantes {
			vista := entrada.FirmanteVista{Firmante: f, CargoTextos: map[string]string{}}
			for _, cod := range []string{"es", "en"} {
				contenido := f.ReferenciasTextos[cod]
				k := cod + ":" + contenido
				textos, ok := cache[k]
				if !ok {
					var err error
					textos, err = e.paquetes.TextosVersion(ctx, cod, contenido)
					if err != nil {
						return entrada.Administracion{}, err
					}
					cache[k] = textos
				}
				vista.CargoTextos[cod] = textos["cargo."+f.CargoCodigo]
			}
			a.Firmantes = append(a.Firmantes, vista)
		}
	} else {
		a.Datos, err = e.repo.Ultima(ctx, ambito)
	}
	if err != nil {
		return entrada.Administracion{}, err
	}
	a.Catalogos, err = e.paquetes.Catalogos(ctx)
	return a, err
}
func (e *Edicion) Consultar(ctx context.Context, actor entrada.Actor, ambito string) (entrada.Administracion, error) {
	if err := dominio.Autorizar(actor); err != nil {
		return entrada.Administracion{}, err
	}
	if ambito != "listas" && ambito != "plazos" && ambito != "tramos" && ambito != "firmantes" && ambito != "asignacion_municipios" {
		return entrada.Administracion{}, dominio.ErrAmbito
	}
	var a entrada.Administracion
	err := e.ejecutar(ctx, actor, func(ctx context.Context) error {
		var err error
		a, err = e.consultar(ctx, ambito)
		if err != nil {
			return err
		}
		_, err = e.registro.Confirmar(ctx, auditoria.Datos{OcurridoEn: e.ahora().UTC().Truncate(time.Microsecond), UsuarioID: actor.UsuarioID, Perfiles: actor.Perfiles, Accion: "catalogo.consultar_administracion", Entidad: "catalogo", Resultado: "ok"})
		return err
	})
	if err != nil {
		return entrada.Administracion{}, err
	}
	return a, nil
}
func (e *Edicion) Guardar(ctx context.Context, actor entrada.Actor, ambito string, c entrada.Cambio) (entrada.Administracion, error) {
	if err := dominio.Autorizar(actor); err != nil {
		return entrada.Administracion{}, err
	}
	if err := c.Validar(ambito); err != nil {
		return entrada.Administracion{}, err
	}

	if err := e.ejecutar(ctx, actor, func(ctx context.Context) error { return e.paquetes.LimpiarHuerfanos(ctx, e.ahora().Add(-24*time.Hour)) }); err != nil {
		return entrada.Administracion{}, err
	}
	c.VigenteDesde = dominio.FechaCivil(c.VigenteDesde)
	if c.Firmante != nil {
		f := *c.Firmante
		if f.VigenteHasta != nil {
			hasta := dominio.FechaCivil(*f.VigenteHasta)
			f.VigenteHasta = &hasta
		}
		c.Firmante = &f
	}
	var preparada paquetes.Preparada
	if c.ClaveTexto() != "" {
		err := e.ejecutar(ctx, actor, func(ctx context.Context) error {
			var err error
			if c.Ubicacion != nil {
				publicador, ok := e.paquetes.(paquetes.PublicacionUbicaciones)
				if !ok {
					return dominio.ErrNoDisponible
				}
				preparada, err = publicador.PrepararUbicacion(ctx, c.ClaveTexto(), c.Textos, c.Ubicacion.Variantes, c.ReferenciasEsperadas)
			} else {
				preparada, err = e.paquetes.Preparar(ctx, c.ClaveTexto(), c.Textos, c.ReferenciasEsperadas)
			}
			return err
		})
		if err != nil {
			return entrada.Administracion{}, err
		}
	}
	var a entrada.Administracion
	err := e.ejecutar(ctx, actor, func(ctx context.Context) error {
		if err := e.paquetes.Activar(ctx, preparada); err != nil {
			return err
		}
		id := ""
		if ambito == "firmantes" {
			f, err := e.repo.GuardarFirmante(ctx, c)
			if err != nil {
				return err
			}
			id = f.ID
		} else {
			v, err := e.repo.Publicar(ctx, ambito, c)
			if err != nil {
				return err
			}
			id = v.ID
		}
		_, err := e.registro.Confirmar(ctx, auditoria.Datos{OcurridoEn: e.ahora().UTC().Truncate(time.Microsecond), UsuarioID: actor.UsuarioID, Perfiles: actor.Perfiles, Accion: "catalogo.publicar", Entidad: "catalogo", EntidadID: id, Resultado: "ok"})
		if err != nil {
			return err
		}
		a, err = e.consultar(ctx, ambito)
		return err
	})
	if err != nil {
		return entrada.Administracion{}, err
	}
	return a, nil
}
