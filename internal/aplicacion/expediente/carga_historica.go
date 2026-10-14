package expediente

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"time"

	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

var errVistaPrevia = errors.New("expediente.vista_previa_revertida")

var _ entrada.CargasHistoricas = (*Servicio)(nil)

func autorizarCarga(a d.Actor, equipo string) error {
	if err := a.AutorizarConsulta(); err != nil {
		return err
	}
	if !slices.Contains(a.Perfiles, "coordinacion") && a.EquipoID != equipo {
		return d.ErrAcceso
	}
	return nil
}
func (s *Servicio) RechazarCargaHistorica(ctx context.Context, a d.Actor) error {
	if s.d.Registro == nil || s.d.Reloj == nil {
		return d.ErrNoDisponible
	}
	if !d.IDValido(a.UsuarioID) {
		a = d.Actor{}
	}
	_, err := s.d.Registro.Confirmar(ctx, s.evento(a, "expediente.carga_historica_rechazada", "", "error"))
	if err != nil {
		return d.ErrNoDisponible
	}
	return nil
}
func (s *Servicio) ValidarCargaHistorica(ctx context.Context, a d.Actor, p d.CargaHistorica) (r d.ValidacionCargaHistorica, fallo error) {
	defer func() {
		if fallo != nil && s.RechazarCargaHistorica(ctx, a) != nil {
			fallo = d.ErrNoDisponible
		}
	}()
	if err := autorizarCarga(a, p.EquipoID); err != nil {
		return r, err
	}
	if s.d.Reloj == nil || s.d.Identificadores == nil || s.d.Registro == nil {
		return r, d.ErrNoDisponible
	}
	if err := p.Validar(s.d.Reloj.Ahora()); err != nil {
		return r, err
	}
	id, err := s.d.Identificadores.Nuevo()
	if err != nil {
		return r, d.ErrNoDisponible
	}
	// Ensaya los mismos servicios y revierte toda la transacción antes de auditar
	// la validación. Los números y vigencias también se comprueban en la vista previa.
	if _, err = s.cargarHistoricos(ctx, a, id, "validar-"+id, p, false); err != nil {
		return r, err
	}
	if _, err = s.d.Registro.Confirmar(ctx, s.evento(a, "expediente.validar_carga_historica", "", "ok")); err != nil {
		return r, d.ErrNoDisponible
	}
	return d.ValidacionCargaHistorica{LoteID: id, Casos: len(p.Casos)}, nil
}
func (s *Servicio) CargarHistoricos(ctx context.Context, a d.Actor, lote, clave string, p d.CargaHistorica) (d.ResultadoCargaHistorica, error) {
	if lote == "" {
		h := sha256.Sum256([]byte("expediente.carga_manual:" + a.UsuarioID + ":" + clave))
		h[6] = (h[6] & 15) | 80
		h[8] = (h[8] & 63) | 128
		v := hex.EncodeToString(h[:16])
		lote = v[:8] + "-" + v[8:12] + "-" + v[12:16] + "-" + v[16:20] + "-" + v[20:]
	}
	return s.cargarHistoricos(ctx, a, lote, clave, p, true)
}
func (s *Servicio) cargarHistoricos(ctx context.Context, a d.Actor, lote, clave string, p d.CargaHistorica, confirmar bool) (r d.ResultadoCargaHistorica, fallo error) {
	defer func() {
		if fallo != nil {
			r = d.ResultadoCargaHistorica{}
			if s.RechazarCargaHistorica(ctx, a) != nil {
				fallo = d.ErrNoDisponible
			}
		}
	}()
	if err := autorizarCarga(a, p.EquipoID); err != nil {
		return r, err
	}
	if !d.IDValido(lote) {
		return r, d.ErrPeticion
	}
	if !d.ClaveValida(clave) {
		return r, d.ErrClave
	}
	if s.d.CargasHistoricas == nil || s.d.Aceptaciones == nil || s.d.Decisiones == nil || s.d.Cierres == nil || s.d.Catalogo == nil || s.d.Numeracion == nil || s.d.MotivosCierre == nil || !s.disponibleFamilia() || !s.disponibleMenores() || s.d.Identificadores == nil {
		return r, d.ErrNoDisponible
	}
	instante := s.d.Reloj.Ahora()
	ahora := instante.UTC().Truncate(time.Microsecond)
	if err := p.Validar(instante); err != nil {
		return r, err
	}
	b, err := json.Marshal(p)
	if err != nil {
		return r, d.ErrPeticion
	}
	defer clear(b)
	if p.FuenteHoja != nil {
		fuente, err := json.Marshal(p.FuenteHoja)
		if err != nil {
			return r, d.ErrPeticion
		}
		b = append(b, fuente...)
	}
	h := sha256.Sum256(b)
	huella := hex.EncodeToString(h[:])
	ruta := "/api/carga-2026/" + lote + "/confirmar"
	err = s.d.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		anterior, repetida, err := s.d.CargasHistoricas.RepetirCargaHistorica(ctx, a.UsuarioID, clave, ruta, huella, ahora)
		if err != nil {
			return err
		}
		if repetida {
			r = anterior
			return nil
		}
		if p.FuenteHoja != nil {
			err = s.d.CargasHistoricas.CrearLoteHojaHistorico(ctx, lote, p.EquipoID, a.UsuarioID, ahora, len(p.Casos), *p.FuenteHoja)
		} else {
			err = s.d.CargasHistoricas.CrearLoteHistorico(ctx, lote, p.EquipoID, a.UsuarioID, ahora, len(p.Casos))
		}
		if err != nil {
			return err
		}
		r = d.ResultadoCargaHistorica{LoteID: lote, Expedientes: []string{}, Marca: "expediente.cargado_para_revision"}
		// El ámbito del lote se ha autorizado; nunca se cambian perfiles de sesión.
		actorFicha := a
		actorFicha.EquipoID = p.EquipoID
		for indice, c := range p.Casos {
			falloCaso := func() error {
				id, err := s.d.Identificadores.Nuevo()
				if err != nil {
					return err
				}
				c.Datos.EsquemaVersion = "1"
				c.Datos.Extraccion = d.DatosPermitidos(c.Datos.Extraccion)
				if err = s.d.CargasHistoricas.CrearCasoHistorico(ctx, id, lote, p.EquipoID, c, a.UsuarioID, ahora); err != nil {
					return err
				}
				subclave := func(accion string) string { return "historico:" + id + ":" + accion }
				if c.NoAceptacion != nil {
					decision := *c.NoAceptacion
					decision.Version = 1
					decision, err = decision.Validar(d.Derivacion{Estado: d.DerivacionRecibida, Datos: c.Datos})
					if err != nil {
						return err
					}
					recibo := d.ResultadoDecision{ID: id, Estado: d.NoAceptada, Version: 2, Fecha: decision.Fecha, Subprograma: decision.Subprograma}
					if err = s.d.Decisiones.Decidir(ctx, decision, recibo, d.DerivacionRecibida, a.UsuarioID, ahora); err != nil {
						return err
					}
				} else {
					aceptacion := *c.Aceptacion
					aceptacion.Version = 1
					aceptada, err := s.aceptacionEnTransaccion(ctx, a, id, subclave("aceptar"), aceptacion, ruta, huella, ahora, instante, &c.NumeroHistorico)
					if err != nil {
						return err
					}
					version := aceptada.Version
					if err = s.cargarVigenciasHistoricas(ctx, actorFicha, id, ruta, huella, ahora, instante, c, &version); err != nil {
						return err
					}
					for j, e := range c.Episodios {
						situacion, err := s.d.Cierres.BloquearCierre(ctx, id)
						if err != nil {
							return err
						}
						if j > 0 {
							_, err = s.d.Cierres.Reabrir(ctx, id, situacion, d.PeticionReapertura{Version: situacion.Version, Fecha: e.Apertura, Motivo: e.MotivoReapertura}, d.Aceptada, a.UsuarioID, ahora)
							if err != nil {
								return err
							}
							situacion, err = s.d.Cierres.BloquearCierre(ctx, id)
							if err != nil {
								return err
							}
						}
						if e.Cierre != "" {
							fecha, _ := time.Parse("2006-01-02", e.Cierre)
							cat, err := s.d.MotivosCierre.CierreVigente(ctx, fecha)
							if err != nil {
								return err
							}
							encontrado := false
							for _, m := range cat.Motivos {
								if m.Codigo == e.MotivoCierre {
									situacion.Motivo = m.Codigo
									situacion.MotivoActa = m.Acta
									situacion.MotivoSPAF = m.SPAF
									encontrado = true
								}
							}
							if !encontrado {
								return d.ErrCargaHistorica
							}
							if _, err = s.d.Cierres.ConfirmarCierre(ctx, id, situacion, d.PeticionCierre{Version: situacion.Version, Fecha: e.Cierre, MotivoActaTexto: e.MotivoActaTexto}, a.UsuarioID, ahora); err != nil {
								return err
							}
						}
					}
					if c.Estado == d.EnIntervencion {
						if err = s.d.CargasHistoricas.EstadoHistorico(ctx, id, c.Estado, a.UsuarioID, ahora); err != nil {
							return err
						}
					}
				}
				if _, err = s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "expediente.cargado_para_revision", id, "ok")); err != nil {
					return err
				}
				r.Expedientes = append(r.Expedientes, id)
				return nil
			}()
			if falloCaso != nil {
				return d.ErrorFilaHistorica{Indice: indice, Causa: falloCaso}
			}
		}
		if _, err = s.d.RegistroTransaccional.RegistrarEnTransaccion(ctx, s.evento(a, "expediente.confirmar_carga_historica", "", "ok")); err != nil {
			return err
		}
		if !confirmar {
			return errVistaPrevia
		}
		return s.d.CargasHistoricas.RecordarCargaHistorica(ctx, a.UsuarioID, clave, ruta, huella, r, ahora)
	})
	if errors.Is(err, errVistaPrevia) {
		return r, nil
	}
	if err != nil {
		for _, conocido := range []error{d.ErrCargaHistorica, d.ErrNumeroHistorico, d.ErrNumeroDuplicado, d.ErrIdempotencia, d.ErrAcceso, d.ErrFamilia, d.ErrMenores, d.ErrObligatorios, d.ErrProfesional, d.ErrSexo, d.ErrPeticion, d.ErrEstado} {
			if errors.Is(err, conocido) {
				return r, err
			}
		}
		return r, d.ErrNoDisponible
	}
	return r, nil
}

// Reutiliza las ediciones de ficha dentro de la transacción del lote. Los UUID
// de entrada identifican personas dentro del caso, nunca recursos ya existentes.
func (s *Servicio) cargarVigenciasHistoricas(ctx context.Context, a d.Actor, id, ruta, huella string, ahora, instante time.Time, c d.CasoHistorico, version *int) error {
	adultos := map[string]string{}
	menores := map[string]string{}
	ficha, err := s.d.Menores.ObtenerMenores(ctx, id)
	if err != nil {
		return err
	}
	ordenes := map[int]string{}
	for _, m := range ficha.Menores {
		ordenes[m.Orden] = m.ID
	}
	for i, cambio := range c.Familia {
		cambio.Version = *version
		origen := []d.Adulto{}
		if cambio.Adultos != nil {
			origen = slices.Clone(*cambio.Adultos)
			destino := slices.Clone(origen)
			for j := range destino {
				k := origen[j].ID
				if k == "" {
					k = strconv.Itoa(origen[j].Orden)
				}
				destino[j].ID = adultos[k]
			}
			cambio.Adultos = &destino
		}
		r, err := s.familiaEnTransaccion(ctx, a, id, "historico:"+id+":familia:"+strconv.Itoa(i), cambio, ruta, huella, ahora, instante)
		if err != nil {
			return err
		}
		*version = r.Version
		if cambio.Adultos != nil {
			for j, adulto := range *cambio.Adultos {
				k := origen[j].ID
				if k == "" {
					k = strconv.Itoa(origen[j].Orden)
				}
				adultos[k] = adulto.ID
			}
		}
	}
	for i, cambio := range c.Menores {
		cambio.Version = *version
		if cambio.Menores == nil {
			return d.ErrMenores
		}
		origen := slices.Clone(*cambio.Menores)
		destino := slices.Clone(origen)
		for j := range destino {
			k := origen[j].ID
			if k == "" {
				k = strconv.Itoa(origen[j].Orden)
			}
			destino[j].ID = menores[k]
			if i == 0 {
				destino[j].ID = ordenes[destino[j].Orden]
			}
		}
		cambio.Menores = &destino
		r, err := s.menoresEnTransaccion(ctx, a, id, "historico:"+id+":menores:"+strconv.Itoa(i), cambio, ruta, huella, ahora, instante)
		if err != nil {
			return err
		}
		*version = r.Version
		for j, menor := range *cambio.Menores {
			k := origen[j].ID
			if k == "" {
				k = strconv.Itoa(origen[j].Orden)
			}
			menores[k] = menor.ID
		}
	}
	return nil
}
