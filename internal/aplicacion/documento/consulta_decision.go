package documento

import (
	"context"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/documento/puertos/entrada"
	paquete "github.com/aavidad/expedientes-etf/internal/aplicacion/paquete/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	registro "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/documento"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"time"
)

// ConsultaDecision usa los puertos del propietario; no añade SQL ni escrituras.
type ConsultaDecision struct {
	Idiomas       paquete.Paquetes
	Repositorio   salida.Repositorio
	Registros     salida.RepositorioRegistroSalida
	Transacciones plataforma.Transacciones
	Auditoria     registro.RegistroTransaccional
	Ahora         func() time.Time
}

func (c ConsultaDecision) ConsultarEmisionesDecision(ctx context.Context, a entrada.Actor, id string) ([]entrada.EmisionDecision, error) {
	if err := a.AutorizarEntrada(); err != nil {
		return nil, err
	}
	if !d.IDValido(id) {
		return nil, d.ErrPeticion
	}
	idiomas, err := c.Idiomas.Idiomas(ctx)
	if err != nil {
		return nil, d.ErrNoDisponible
	}
	r := []entrada.EmisionDecision{}
	err = c.Transacciones.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
		for _, tipo := range []string{d.ActaAceptacion, d.OficioSSCC, d.OficioSPM, "entrada/no_aceptacion", "entrada/no_aceptacion/lista_espera"} {
			var e d.Emision
			for _, idioma := range idiomas {
				candidata, err := c.Repositorio.Ultima(ctx, d.Solicitud{ExpedienteID: id, Tipo: tipo, Idioma: idioma.Codigo})
				if err != nil {
					return err
				}
				if candidata.GeneradaEn.After(e.GeneradaEn) {
					e = candidata
				}
			}
			if e.ID == "" {
				continue
			}
			fila := entrada.EmisionDecision{ID: e.ID, Tipo: e.Tipo, GeneradaEn: e.GeneradaEn, PlantillaVersion: e.Plantilla.Version}
			if d.EsOficio(e.Tipo) {
				_, registro, err := c.Registros.LeerRegistro(ctx, e.ID)
				if err != nil {
					return err
				}
				fila.Registro = &registro
			}
			r = append(r, fila)
		}
		_, err := c.Auditoria.RegistrarEnTransaccion(ctx, auditoria.Datos{OcurridoEn: c.Ahora().UTC().Truncate(time.Microsecond), UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, Accion: "documento.consultar_emisiones_decision", Entidad: "expediente", EntidadID: id, ExpedienteID: id, Resultado: "ok"})
		return err
	})
	if err != nil {
		return nil, d.ErrNoDisponible
	}
	return r, nil
}
