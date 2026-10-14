package evaluacion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	aud "github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/evaluacion"
	exp "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	registro "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/evaluacion"
	expediente "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"strings"
	"time"
)

type Servicio struct {
	MenoresDocumento lecturaMenoresDocumento
	SujetosDocumento expediente.SujetosDocumento
	Repo             salida.Repositorio
	Catalogo         salida.Catalogo
	Familias         expediente.Familias
	Adultos          expediente.ConsultaAdultos
	Menores          salida.ConsultaMenores
	Profesionales    salida.ConsultaProfesionales
	Tx               plataforma.Transacciones
	Auditoria        registro.Registro
	Ahora            func() time.Time
}

func (s *Servicio) ejecutar(ctx context.Context, a d.Actor, id string, escribir bool, fn func(context.Context) error) error {
	identidad := plataforma.Identidad{UsuarioID: a.UsuarioID, EquipoID: a.EquipoID, Perfiles: a.Perfiles}
	accion := "evaluacion.leer"
	if escribir {
		accion = "evaluacion.guardar"
	}
	auditar := func(ctx context.Context, resultado string) error {
		usuario := a.UsuarioID
		perfiles := a.Perfiles
		if !d.IDValido(usuario) {
			usuario = "sistema"
			perfiles = nil
		}
		entidad := id
		if !d.IDValido(entidad) {
			entidad = ""
		}
		_, err := s.Auditoria.Confirmar(ctx, aud.Datos{UsuarioID: usuario, Perfiles: perfiles, OcurridoEn: s.Ahora().UTC().Truncate(time.Microsecond), Accion: accion, Entidad: "evaluacion", EntidadID: entidad, Resultado: resultado})
		return err
	}
	err := a.Autorizar(escribir)
	if err == nil && !d.IDValido(id) {
		err = d.ErrPeticion
	}
	if err == nil {
		err = s.Tx.Ejecutar(ctx, identidad, func(ctx context.Context) error {
			if err := fn(ctx); err != nil {
				return err
			}
			return auditar(ctx, "ok")
		})
	}
	if err != nil {
		resultado := "error"
		if errors.Is(err, d.ErrAcceso) {
			resultado = "denegado"
		}
		if !d.IDValido(a.UsuarioID) {
			identidad = plataforma.Identidad{}
		}
		if fallo := s.Tx.Ejecutar(ctx, identidad, func(ctx context.Context) error { return auditar(ctx, resultado) }); fallo != nil {
			return d.ErrNoDisponible
		}
	}
	return err
}
func (s *Servicio) Leer(ctx context.Context, a d.Actor, id string) (r d.Registro, err error) {
	err = s.ejecutar(ctx, a, id, false, func(ctx context.Context) error {
		if _, e := s.Familias.ObtenerFamilia(ctx, id); e != nil {
			return d.ErrNoEncontrada
		}
		var e error
		r, e = s.Repo.Consultar(ctx, id)
		if e != nil {
			return e
		}
		if catalogo, ok := s.Catalogo.(salida.CatalogoPublicado); ok {
			definiciones, err := catalogo.Definiciones(ctx)
			if err != nil {
				return err
			}
			r.Definiciones = &definiciones
		}
		return nil
	})
	if err != nil {
		return d.Registro{}, err
	}
	return
}
func (s *Servicio) Consultar(ctx context.Context, a d.Actor, id string) ([]d.Aplicacion, error) {
	r, e := s.Leer(ctx, a, id)
	return r.Escalas, e
}
func (s *Servicio) Guardar(ctx context.Context, a d.Actor, id string, c d.Cambio) (r d.Cambio, err error) {
	err = s.ejecutar(ctx, a, id, true, func(ctx context.Context) error {
		if (c.Aplicacion == nil) == (c.Instrumento == nil) || c.VersionEsperada < 0 || len(c.Clave) < 1 || len(c.Clave) > 128 || strings.TrimSpace(c.Clave) != c.Clave {
			return d.ErrPeticion
		}
		ficha, e := s.Familias.BloquearFamilia(ctx, id)
		if e != nil {
			return d.ErrNoEncontrada
		}
		if ficha.Estado != exp.Aceptada && ficha.Estado != "en_intervencion" {
			return d.ErrAcceso
		}
		b, e := json.Marshal(c)
		if e != nil {
			return d.ErrPeticion
		}
		h := sha256.Sum256(b)
		huella := hex.EncodeToString(h[:])
		previo, existe, e := s.Repo.Repetir(ctx, a.UsuarioID, id, c.Clave, huella)
		if e != nil {
			return e
		}
		if existe {
			r = previo
			return nil
		}
		objetivo := ""
		if c.Aplicacion != nil {
			ap := *c.Aplicacion
			if ap.Momento == "derivacion" {
				return d.ErrAcceso
			}
			objetivo = ap.ID
			if ap.ExpedienteID != id || ap.EquipoID != ficha.EquipoID || len(ap.DescripcionCualitativa) > 4000 {
				return d.ErrPeticion
			}
			escala, e := s.Catalogo.Escala(ctx, ap.VersionEscalaID)
			if e != nil {
				return e
			}
			ap.Version = c.VersionEsperada + 1
			if e = ap.Validar(escala); e != nil {
				return e
			}
			if ap.Fecha == nil || ap.Fecha.Year() < 1 || ap.Fecha.Year() > 9998 {
				return d.ErrPeticion
			}
			switch ap.Sujeto.Tipo {
			case "figura_parental":
				adultos, e := s.Adultos.AdultosVigentes(ctx, id, ap.Fecha.Format("2006-01-02"))
				if e != nil {
					return d.ErrNoDisponible
				}
				pertenece := false
				for _, adulto := range adultos {
					if adulto.ID == ap.Sujeto.AdultoID {
						pertenece = true
					}
				}
				if !pertenece {
					return d.ErrSujeto
				}
			case "menor":
				if s.Menores == nil {
					return d.ErrNoDisponible
				}
				ok, e := s.Menores.Pertenece(ctx, id, ap.Sujeto.MenorID, ap.Fecha.Format("2006-01-02"))
				if e != nil {
					return e
				}
				if !ok {
					return d.ErrSujeto
				}
			case "profesional":
				if s.Profesionales == nil {
					return d.ErrNoDisponible
				}
				ok, e := s.Profesionales.Pertenece(ctx, ap.Sujeto.ProfesionalID, ficha.EquipoID, ap.Fecha.Format("2006-01-02"))
				if e != nil {
					return e
				}
				if !ok {
					return d.ErrSujeto
				}
			}
			c.Aplicacion = &ap
		} else {
			ins := c.Instrumento
			objetivo = ins.ID
			if !d.IDValido(ins.ID) || ins.ExpedienteID != id || ins.EquipoID != ficha.EquipoID || ins.Fecha.IsZero() || ins.Fecha.Year() > 9998 || len(ins.TextoOtros) > 500 || c.VersionEsperada != 0 {
				return d.ErrPeticion
			}
			if ins.Sujeto != nil {
				if ins.Sujeto.Validar() != nil || ins.Sujeto.Tipo != "figura_parental" || (ins.Momento != "inicial" && ins.Momento != "intermedia" && ins.Momento != "final") {
					return d.ErrSujeto
				}
				adultos, err := s.Adultos.AdultosVigentes(ctx, id, ins.Fecha.Format("2006-01-02"))
				if err != nil {
					return d.ErrNoDisponible
				}
				pertenece := false
				for _, adulto := range adultos {
					if adulto.ID == ins.Sujeto.AdultoID {
						pertenece = true
					}
				}
				if !pertenece {
					return d.ErrSujeto
				}
			} else if ins.Momento != "" {
				return d.ErrPeticion
			}
			ok, e := s.Catalogo.Instrumento(ctx, ins.Codigo)
			if e != nil {
				return e
			}
			if !ok || (ins.Codigo == "OTROS") != (strings.TrimSpace(ins.TextoOtros) != "") {
				return d.ErrPeticion
			}
		}
		version, e := s.Repo.Version(ctx, id, objetivo)
		if e != nil {
			return e
		}
		if version != c.VersionEsperada {
			return d.Conflicto{VersionActual: version}
		}
		if e = s.Repo.Guardar(ctx, c); e != nil {
			return e
		}
		if e = s.Repo.Recordar(ctx, a.UsuarioID, id, c.Clave, huella, c); e != nil {
			return e
		}
		r = c
		return nil
	})
	if err != nil {
		return d.Cambio{}, err
	}
	return
}
