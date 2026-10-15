package estadistica

import (
	"context"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	audit "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/estadistica"
	exp "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"slices"
	"time"
)

type DependenciasConsulta struct {
	Reglas        salida.ReglasConsulta
	Cerradas      salida.CerradasConsulta
	Calculo       salida.CalculoConsulta
	Word          salida.WordConsulta
	Transacciones plataforma.Transacciones
	Auditoria     audit.RegistroTransaccional
	Reloj         exp.Reloj
}
type ServicioConsulta struct{ d DependenciasConsulta }

var _ entrada.ConsultaCifrasMemoria = (*ServicioConsulta)(nil)

func NuevoConsulta(d DependenciasConsulta) *ServicioConsulta { return &ServicioConsulta{d} }

func (s *ServicioConsulta) consultar(ctx context.Context, a d.ActorMemoria, f d.FiltroMemoria, idioma string, word bool) (r d.CifrasConsulta, b []byte, err error) {
	if s.d.Cerradas == nil || s.d.Transacciones == nil || s.d.Auditoria == nil || s.d.Reloj == nil {
		return r, nil, d.ErrDisponibleMemoria
	}
	fallo := d.AutorizarConsultaMemoria(a, f)
	ahora := s.d.Reloj.Ahora().UTC().Truncate(time.Microsecond)
	if fallo == nil && f.Anio > ahora.Year() {
		fallo = d.ErrEntrada
	}
	identidad := plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}
	if !d.UUIDValidoSPAF(a.UsuarioID) {
		identidad = plataforma.Identidad{}
	}
	accion := "estadistica.memoria_consulta"
	if word {
		accion = "estadistica.memoria_consulta_word"
	}
	evento := func(resultado string) auditoria.Datos {
		usuario := identidad.UsuarioID
		if usuario == "" {
			usuario = "sistema"
		}
		return auditoria.Datos{OcurridoEn: ahora, UsuarioID: usuario, Perfiles: identidad.Perfiles, Accion: accion, Entidad: "estadistica", Resultado: resultado}
	}
	// La lectura de la revisión cerrada usa una sentencia MVCC. La conexión
	// se devuelve antes del cálculo técnico. El Word usa otra transacción,
	// sin bloqueo asesor, para resolver sus referencias y auditar la descarga.
	var cerrada bool
	err = s.d.Transacciones.Ejecutar(ctx, identidad, func(ctx context.Context) error {
		if fallo != nil {
			_, err := s.d.Auditoria.RegistrarEnTransaccion(ctx, evento("denegado"))
			return err
		}
		var err error
		r, cerrada, err = s.d.Cerradas.CifrasCerradas(ctx, f)
		return err
	})
	if err == nil && fallo == nil && !cerrada {
		ahora = s.d.Reloj.Ahora().UTC().Truncate(time.Microsecond)
		if s.d.Calculo == nil {
			err = d.ErrDisponibleMemoria
		} else {
			r, err = s.d.Calculo.CalcularConsulta(ctx, a, f, ahora)
		}
	}
	if err == nil && fallo == nil {
		// El carácter provisional corresponde al estado observado al empezar.
		// Un cierre concurrente no transforma ese cálculo en una foto cerrada.
		err = s.d.Transacciones.Ejecutar(ctx, identidad, func(ctx context.Context) error {
			if s.d.Reglas == nil {
				return d.ErrDisponibleMemoria
			}
			umbral, err := s.d.Reglas.UmbralConsulta(ctx, ahora)
			if err != nil || umbral < 3 {
				return d.ErrDisponibleMemoria
			}
			r = d.SuprimirCifras(r, umbral)
			if word {
				if s.d.Word == nil {
					return d.ErrDisponibleMemoria
				}
				var err error
				b, err = s.d.Word.GenerarCifras(ctx, r, idioma)
				if err != nil {
					return err
				}
			}
			_, err = s.d.Auditoria.RegistrarEnTransaccion(ctx, evento("ok"))
			return err
		})
	}
	if err != nil {
		clear(b)
		ctxAudit, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if _, e := s.d.Auditoria.Confirmar(ctxAudit, evento("error")); e != nil {
			return d.CifrasConsulta{}, nil, d.ErrDisponibleMemoria
		}
		return d.CifrasConsulta{}, nil, d.ErrDisponibleMemoria
	}
	if fallo != nil {
		return d.CifrasConsulta{}, nil, fallo
	}
	return r, b, nil
}
func (s *ServicioConsulta) ConsultarCifras(ctx context.Context, a d.ActorMemoria, f d.FiltroMemoria) (d.CifrasConsulta, error) {
	r, _, err := s.consultar(ctx, a, f, "", false)
	return r, err
}
func (s *ServicioConsulta) DescargarCifras(ctx context.Context, a d.ActorMemoria, f d.FiltroMemoria, idioma string) ([]byte, error) {
	_, b, err := s.consultar(ctx, a, f, idioma, true)
	return b, err
}

// EjecutorConsulta mantiene las credenciales técnicas fuera de la sesión web.
type EjecutorConsulta struct {
	Transacciones plataforma.Transacciones
	Fuente        FuenteMemoria
}

func (s EjecutorConsulta) CalcularConsulta(ctx context.Context, a d.ActorMemoria, f d.FiltroMemoria, ahora time.Time) (r d.CifrasConsulta, err error) {
	if err = d.AutorizarConsultaMemoria(a, f); err != nil {
		return r, err
	}
	if s.Transacciones == nil {
		return r, d.ErrDisponibleMemoria
	}
	// El ámbito solicitado limita el ejecutor técnico; no cambia los perfiles.
	err = s.Transacciones.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: f.EquipoID}, func(ctx context.Context) error {
		var e error
		r, e = s.Fuente.CalcularCifrasEnTransaccion(ctx, a, f, ahora)
		return e
	})
	if err != nil {
		return d.CifrasConsulta{}, err
	}
	return r, nil
}

type ServicioNavegacion struct {
	Cierres       salida.Cierres
	Fuente        entrada.FuenteMemoria
	Numeros       exp.NumerosMemoria
	Transacciones plataforma.Transacciones
	Auditoria     audit.RegistroTransaccional
	Reloj         exp.Reloj
}

func (s ServicioNavegacion) leer(ctx context.Context, a d.ActorMemoria, f d.FiltroMemoria, apartado d.Apartado) (r d.VistaMemoria, desglose []d.DesgloseMemoria, err error) {
	if s.Transacciones == nil || s.Auditoria == nil || s.Reloj == nil || s.Fuente == nil {
		return r, nil, d.ErrDisponibleMemoria
	}
	ahora := s.Reloj.Ahora().UTC().Truncate(time.Microsecond)
	accion := "estadistica.memoria_navegar"
	if apartado != "" {
		accion = "estadistica.memoria_composicion"
	}
	registrarFallo := func(resultado string) error {
		usuario := a.UsuarioID
		if !d.UUIDValidoSPAF(usuario) {
			usuario = "sistema"
		}
		ctxAudit, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_, e := s.Auditoria.Confirmar(ctxAudit, auditoria.Datos{OcurridoEn: ahora, UsuarioID: usuario, Perfiles: a.Perfiles, Accion: accion, Entidad: "estadistica", Resultado: resultado})
		return e
	}
	f, fallo := d.AutorizarMemoria(a, f)
	if fallo == nil && f.Anio > ahora.Year() {
		fallo = d.ErrEntrada
	}
	if fallo != nil {
		if registrarFallo("denegado") != nil {
			return r, nil, d.ErrDisponibleMemoria
		}
		return r, nil, fallo
	}
	err = s.Transacciones.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
		informe, e := s.Fuente.MemoriaEnTransaccion(ctx, a, f, ahora)
		if e != nil {
			return e
		}
		r = d.VistaMemoria{FiltroMemoria: f, Fecha: informe.ConocidoEn, Apartados: informe.Apartados}
		// La fuente cerrada conserva el instante original del cálculo.
		if s.Cierres != nil {
			estado, e := s.Cierres.EstadoMemoria(ctx, f, false, a.UsuarioID, ahora)
			if e != nil {
				return e
			}
			r.Definitivas = estado.Estado == "cerrada"
		}
		if apartado != "" {
			valido := false
			for _, ap := range informe.Apartados {
				if ap.Codigo == apartado {
					valido = true
				}
			}
			if !valido {
				return d.ErrEntrada
			}
			if s.Numeros == nil {
				return d.ErrDisponibleMemoria
			}
			ids := []string{}
			for _, c := range informe.Composicion {
				if c.Apartado == apartado {
					ids = append(ids, c.Expedientes...)
				}
			}
			slices.Sort(ids)
			ids = slices.Compact(ids)
			numeros, e := s.Numeros.NumerosMemoria(ctx, ids)
			if e != nil {
				return e
			}
			desglose = []d.DesgloseMemoria{}
			for _, c := range informe.Composicion {
				if c.Apartado != apartado {
					continue
				}
				v := d.DesgloseMemoria{Apartado: apartado, Grupo: c.Grupo, Unidad: c.Unidad, Numeros: []string{}}
				for _, id := range c.Expedientes {
					n, ok := numeros[id]
					if !ok {
						return d.ErrPermisoMemoria
					}
					if n != "" {
						v.Numeros = append(v.Numeros, n)
					}
				}
				slices.Sort(v.Numeros)
				v.Numeros = slices.Compact(v.Numeros)
				desglose = append(desglose, v)
			}
		}
		accion := "estadistica.memoria_navegar"
		if apartado != "" {
			accion = "estadistica.memoria_composicion"
		}
		_, e = s.Auditoria.RegistrarEnTransaccion(ctx, auditoria.Datos{OcurridoEn: ahora, UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, Accion: accion, Entidad: "estadistica", Resultado: "ok"})
		return e
	})
	if err != nil {
		if registrarFallo("error") != nil {
			return d.VistaMemoria{}, nil, d.ErrDisponibleMemoria
		}
		return d.VistaMemoria{}, nil, err
	}
	return r, desglose, nil
}
func (s ServicioNavegacion) ConsultarMemoria(ctx context.Context, a d.ActorMemoria, f d.FiltroMemoria) (d.VistaMemoria, error) {
	r, _, err := s.leer(ctx, a, f, "")
	return r, err
}
func (s ServicioNavegacion) ComposicionMemoria(ctx context.Context, a d.ActorMemoria, f d.FiltroMemoria, ap d.Apartado) ([]d.DesgloseMemoria, error) {
	if ap == "" {
		return nil, d.ErrEntrada
	}
	_, r, err := s.leer(ctx, a, f, ap)
	return r, err
}

func rechazarPeticionMemoria(ctx context.Context, a d.ActorMemoria, registro audit.RegistroTransaccional, reloj exp.Reloj) error {
	if registro == nil || reloj == nil {
		return d.ErrDisponibleMemoria
	}
	usuario := a.UsuarioID
	if !d.UUIDValidoSPAF(usuario) {
		usuario = "sistema"
	}
	ctxAudit, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	_, err := registro.Confirmar(ctxAudit, auditoria.Datos{OcurridoEn: reloj.Ahora().UTC().Truncate(time.Microsecond), UsuarioID: usuario, Perfiles: a.Perfiles, Accion: "estadistica.memoria_peticion_rechazada", Entidad: "estadistica", Resultado: "denegado"})
	if err != nil {
		return d.ErrDisponibleMemoria
	}
	return nil
}
func (s *ServicioConsulta) RechazarPeticionMemoria(ctx context.Context, a d.ActorMemoria) error {
	return rechazarPeticionMemoria(ctx, a, s.d.Auditoria, s.d.Reloj)
}
func (s ServicioNavegacion) RechazarPeticionMemoria(ctx context.Context, a d.ActorMemoria) error {
	return rechazarPeticionMemoria(ctx, a, s.Auditoria, s.Reloj)
}
