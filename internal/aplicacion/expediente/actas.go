package expediente

import (
	"context"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	registro "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"time"
)

type ServicioActas struct {
	Validador salida.ValidadorFichero
	Antivirus plataforma.Antivirus
	Lector    salida.LectorActas
	Registro  registro.Registro
	Reloj     salida.Reloj
}

var _ entrada.Actas = (*ServicioActas)(nil)

func (s *ServicioActas) registrar(ctx context.Context, a d.Actor, accion string) error {
	if s.Registro == nil || s.Reloj == nil {
		return d.ErrNoDisponible
	}
	if !d.IDValido(a.UsuarioID) {
		a = d.Actor{UsuarioID: "sistema"}
	}
	resultado := "ok"
	if accion == "expediente.acta_carga_rechazada" {
		resultado = "denegado"
	}
	_, err := s.Registro.Confirmar(ctx, auditoria.Datos{OcurridoEn: s.Reloj.Ahora().UTC().Truncate(time.Microsecond), UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, Accion: accion, Entidad: "expediente", Resultado: resultado})
	if err != nil {
		return d.ErrNoDisponible
	}
	return nil
}
func (s *ServicioActas) RechazarActa(ctx context.Context, a d.Actor) error {
	return s.registrar(ctx, a, "expediente.acta_carga_rechazada")
}
func (s *ServicioActas) LeerActa(ctx context.Context, a d.Actor, equipo, tipo string, p entrada.Subida) (r entrada.ResultadoActa, fallo error) {
	defer func() {
		if fallo != nil {
			r = entrada.ResultadoActa{}
			if err := s.RechazarActa(ctx, a); err != nil {
				fallo = err
			}
		}
	}()
	if err := autorizarCarga(a, equipo); err != nil {
		return r, err
	}
	if !d.IDValido(equipo) || tipo != "aceptacion" && tipo != "cierre" {
		return r, entrada.ErrActa
	}
	if s.Validador == nil || s.Antivirus == nil || s.Lector == nil {
		return r, d.ErrNoDisponible
	}
	formato, err := s.Validador.Validar(ctx, p.Nombre, p.Contenido)
	if err != nil {
		return r, err
	}
	estado, falloAV := s.Antivirus.Revisar(ctx, p.Contenido)
	if ctx.Err() != nil {
		return r, ctx.Err()
	}
	if estado == plataforma.AntivirusInfectado {
		return r, d.ErrInfectado
	}
	r.Antivirus = "limpio"
	if estado != plataforma.AntivirusLimpio || falloAV != nil {
		r.Antivirus = "no_revisado"
		if err := s.registrar(ctx, a, "expediente.acta_carga_no_revisada"); err != nil {
			return r, err
		}
	}
	r.Contenido, err = s.Lector.LeerActa(ctx, formato, p.Contenido)
	if err != nil {
		return r, err
	}
	if r.Contenido.Tipo != "" && r.Contenido.Tipo != tipo {
		return r, entrada.ErrActa
	}
	if err := s.registrar(ctx, a, "expediente.leer_acta_carga"); err != nil {
		return r, err
	}
	return r, nil
}
