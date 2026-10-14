package registro_manual

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	"time"
)

type Registro struct{}

func (Registro) Anotar(ctx context.Context, r d.RegistroSalida, s d.SolicitudRegistroSalida, a d.Actor, ahora time.Time) (d.RegistroSalida, error) {
	if err := ctx.Err(); err != nil {
		return d.RegistroSalida{}, d.ErrNoDisponible
	}
	if err := s.Validar(); err != nil {
		return d.RegistroSalida{}, err
	}
	r.Adaptador = "manual"
	r.Estado = "anotado"
	r.Numero = s.Numero
	r.Fecha = s.Fecha
	r.AnotadoPor = a.UsuarioID
	r.AnotadoEn = ahora.UTC().Truncate(time.Microsecond)
	r.Version++
	return r, nil
}
