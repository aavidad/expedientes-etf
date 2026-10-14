package estadistica

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"slices"
)

// ConsultarOpcionesSPAF entrega sólo los equipos del ámbito de la sesión.
// Las etiquetas proceden de las versiones publicadas del paquete de idiomas.
func (s *ServicioSPAF) ConsultarOpcionesSPAF(ctx context.Context, a d.ActorSPAF) (d.OpcionesSPAF, error) {
	r := d.OpcionesSPAF{Equipos: []d.EquipoSPAF{}, Textos: map[string]map[string]string{}}
	if s.d.Transacciones == nil || s.d.Auditoria == nil || s.d.Reloj == nil {
		return d.OpcionesSPAF{}, d.ErrDisponibleSPAF
	}
	ahora := s.d.Reloj.Ahora()
	provincial := slices.Contains(a.Perfiles, "coordinacion")
	fallo := error(nil)
	if !d.UUIDValidoSPAF(a.UsuarioID) || (!provincial && (!slices.Contains(a.Perfiles, "profesional_etf") || !d.UUIDValidoSPAF(a.EquipoID))) {
		fallo = d.ErrPermisoSPAF
	} else if s.d.Catalogo == nil || s.d.Etiquetas == nil {
		fallo = d.ErrDisponibleSPAF
	} else {
		fallo = s.d.Transacciones.Ejecutar(ctx, identidadSPAF(a), func(ctx context.Context) error {
			version, err := s.d.Catalogo.Vigente(ctx, "asignacion_municipios", ahora)
			if err != nil {
				return err
			}
			datos, err := s.d.Catalogo.Datos(ctx, version)
			if err != nil {
				return err
			}
			for _, e := range datos.Equipos {
				if provincial || e.ID == a.EquipoID {
					r.Equipos = append(r.Equipos, d.EquipoSPAF{ID: e.ID, ClaveTexto: e.ClaveTexto})
				}
			}
			catalogos, err := s.d.Etiquetas.Catalogos(ctx)
			if err != nil {
				return err
			}
			r.Textos = catalogos.Textos
			apunte := datosAuditoriaSPAF(a, ahora, "", "ok")
			apunte.Accion = "estadistica.consultar_opciones_spaf"
			_, err = s.d.Auditoria.RegistrarEnTransaccion(ctx, apunte)
			return err
		})
	}
	if fallo == nil {
		return r, nil
	}
	resultado := "error"
	if fallo == d.ErrPermisoSPAF {
		resultado = "denegado"
	}
	apunte := datosAuditoriaSPAF(a, ahora, "", resultado)
	apunte.Accion = "estadistica.consultar_opciones_spaf"
	if _, err := s.d.Auditoria.Confirmar(ctx, apunte); err != nil {
		return d.OpcionesSPAF{}, d.ErrDisponibleSPAF
	}
	if fallo == d.ErrPermisoSPAF {
		return d.OpcionesSPAF{}, fallo
	}
	return d.OpcionesSPAF{}, d.ErrDisponibleSPAF
}
