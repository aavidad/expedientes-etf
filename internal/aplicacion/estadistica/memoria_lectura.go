package estadistica

import (
	"context"
	"errors"
	"slices"
	"time"

	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	registro "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
	catalogo "github.com/aavidad/expedientes-etf/internal/puertos/salida/catalogo"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/estadistica"
	expediente "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

type DependenciasLecturaMemoria struct {
	Fuente           entrada.FuenteMemoria
	CifrasPublicadas salida.CifrasPublicadas
	Catalogos        catalogo.Repositorio
	Transacciones    plataforma.Transacciones
	Auditoria        registro.RegistroTransaccional
	Reloj            expediente.Reloj
}

type ServicioLecturaMemoria struct{ d DependenciasLecturaMemoria }

var _ entrada.ConsultaMemoria = (*ServicioLecturaMemoria)(nil)

func NuevoLecturaMemoria(dep DependenciasLecturaMemoria) *ServicioLecturaMemoria {
	return &ServicioLecturaMemoria{d: dep}
}

func (s *ServicioLecturaMemoria) ConsultarEquiposMemoria(ctx context.Context, a d.ActorMemoria) ([]entrada.EquipoCifrasMemoria, error) {
	if s.d.Transacciones == nil || s.d.Auditoria == nil || s.d.Reloj == nil {
		return nil, d.ErrDisponibleMemoria
	}
	ahora := s.d.Reloj.Ahora().UTC().Truncate(time.Microsecond)
	coordinacionOConsulta := slices.Contains(a.Perfiles, "coordinacion") || slices.Contains(a.Perfiles, "consulta")
	f := d.FiltroMemoria{Anio: ahora.Year()}
	var fallo error
	if coordinacionOConsulta {
		if !d.UUIDValidoSPAF(a.UsuarioID) {
			fallo = d.ErrPermisoMemoria
		}
	} else {
		f, fallo = d.AutorizarMemoria(a, f)
	}
	if fallo == nil && s.d.Catalogos == nil {
		fallo = d.ErrDisponibleMemoria
	}
	r := []entrada.EquipoCifrasMemoria{}
	if fallo == nil {
		fallo = s.d.Transacciones.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
			v, err := s.d.Catalogos.Vigente(ctx, "asignacion_municipios", ahora)
			if err != nil {
				return err
			}
			datos, err := s.d.Catalogos.Datos(ctx, v)
			if err != nil {
				return err
			}
			for _, equipo := range datos.Equipos {
				if coordinacionOConsulta || equipo.ID == f.EquipoID {
					r = append(r, entrada.EquipoCifrasMemoria{ID: equipo.ID, Codigo: equipo.Codigo, ClaveTexto: equipo.ClaveTexto})
				}
			}
			evento := eventoLecturaMemoria(a, ahora, f.EquipoID, "ok")
			evento.Accion, evento.Entidad = "estadistica.memoria_leer_equipos", "catalogo"
			_, err = s.d.Auditoria.RegistrarEnTransaccion(ctx, evento)
			return err
		})
	}
	if fallo == nil {
		return r, nil
	}
	resultado := "error"
	if errors.Is(fallo, d.ErrPermisoMemoria) {
		resultado = "denegado"
	}
	evento := eventoLecturaMemoria(a, ahora, "", resultado)
	evento.Accion, evento.Entidad = "estadistica.memoria_leer_equipos", "catalogo"
	ctxAuditoria, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancelar()
	if _, err := s.d.Auditoria.Confirmar(ctxAuditoria, evento); err != nil {
		return nil, d.ErrDisponibleMemoria
	}
	if errors.Is(fallo, d.ErrPermisoMemoria) {
		return nil, fallo
	}
	return nil, d.ErrDisponibleMemoria
}

func (s *ServicioLecturaMemoria) ConsultarMemoria(ctx context.Context, a d.ActorMemoria, f d.FiltroMemoria) (entrada.CifrasMemoria, error) {
	if s.d.Transacciones == nil || s.d.Auditoria == nil || s.d.Reloj == nil {
		return entrada.CifrasMemoria{}, d.ErrDisponibleMemoria
	}
	ahora := s.d.Reloj.Ahora().UTC().Truncate(time.Microsecond)
	consulta := slices.Contains(a.Perfiles, "consulta") && !slices.Contains(a.Perfiles, "coordinacion") && (!slices.Contains(a.Perfiles, "profesional_etf") || f.EquipoID != a.EquipoID)
	var fallo error
	if consulta {
		if !d.UUIDValidoSPAF(a.UsuarioID) {
			fallo = d.ErrPermisoMemoria
		} else if f.Anio < 1 || f.Anio > 9998 || (f.EquipoID != "" && !d.UUIDValidoSPAF(f.EquipoID)) {
			fallo = d.ErrEntrada
		}
	} else {
		f, fallo = d.AutorizarMemoria(a, f)
	}
	if fallo == nil && ((!consulta && s.d.Fuente == nil) || (consulta && s.d.CifrasPublicadas == nil)) {
		fallo = d.ErrDisponibleMemoria
	}
	var cifras entrada.CifrasMemoria
	if fallo == nil {
		fallo = s.d.Transacciones.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
			var err error
			if consulta {
				cifras, err = s.d.CifrasPublicadas.Consultar(ctx, f)
				if err != nil {
					return err
				}
				if cifras.Anio != f.Anio || cifras.EquipoID != f.EquipoID {
					return d.ErrPermisoMemoria
				}
			} else {
				informe, err := s.d.Fuente.MemoriaEnTransaccion(ctx, a, f, ahora)
				if err != nil {
					return err
				}
				if informe.FiltroMemoria != f {
					return d.ErrPermisoMemoria
				}
				cifras = proyectarCifrasMemoria(informe)
				if s.d.CifrasPublicadas != nil {
					if err := s.d.CifrasPublicadas.Publicar(ctx, cifras); err != nil {
						return err
					}
				}
			}
			_, err = s.d.Auditoria.RegistrarEnTransaccion(ctx, eventoLecturaMemoria(a, ahora, f.EquipoID, "ok"))
			return err
		})
	}
	if fallo == nil {
		return cifras, nil
	}
	resultado := "error"
	if errors.Is(fallo, d.ErrPermisoMemoria) {
		resultado = "denegado"
	}
	ctxAuditoria, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancelar()
	if _, err := s.d.Auditoria.Confirmar(ctxAuditoria, eventoLecturaMemoria(a, ahora, "", resultado)); err != nil {
		return entrada.CifrasMemoria{}, d.ErrDisponibleMemoria
	}
	if errors.Is(fallo, d.ErrPermisoMemoria) || errors.Is(fallo, d.ErrEntrada) || errors.Is(fallo, d.ErrCifrasNoPublicadasMemoria) {
		return entrada.CifrasMemoria{}, fallo
	}
	return entrada.CifrasMemoria{}, d.ErrDisponibleMemoria
}

func eventoLecturaMemoria(a d.ActorMemoria, ahora time.Time, equipo, resultado string) auditoria.Datos {
	usuario, perfiles := a.UsuarioID, a.Perfiles
	if !d.UUIDValidoSPAF(usuario) {
		usuario, perfiles = "sistema", nil
	}
	return auditoria.Datos{OcurridoEn: ahora, UsuarioID: usuario, Perfiles: perfiles, Accion: "estadistica.memoria_leer_cifras", Entidad: "estadistica.memoria", EntidadID: equipo, Resultado: resultado}
}

func proyectarCifrasMemoria(informe d.InformeMemoria) entrada.CifrasMemoria {
	r := entrada.CifrasMemoria{Anio: informe.Anio, EquipoID: informe.EquipoID, EquipoCodigo: informe.EquipoCodigo, Corte: informe.Corte, ConocidoEn: informe.ConocidoEn, Apartados: []entrada.CifrasApartadoMemoria{}}
	for _, ap := range informe.Apartados {
		c, total := ap.Contrato, ap.Total
		apartado := entrada.CifrasApartadoMemoria{Codigo: ap.Codigo,
			Contrato: entrada.ContratoCifrasMemoria{Poblacion: c.Poblacion, Denominador: c.Denominador, Desconocidos: c.Desconocidos, FechaDatos: c.FechaDatos, Corte: c.Corte, FechaEdad: c.FechaEdad, VersionRegla: c.VersionRegla, DivisionCero: c.DivisionCero},
			Total:    entrada.TotalCifrasMemoria{Familias: total.Familias, Menores: total.Menores, Adultos: total.Adultos, Comisiones: total.Comisiones, Informes: total.Informes, FamiliasSinRelacionMenores: total.FamiliasSinRelacionMenores}, Celdas: []entrada.CeldaCifrasMemoria{}, Avisos: ap.Avisos}
		for _, ce := range ap.Celdas {
			g, c := ce.Grupo, ce.Celda
			celda := entrada.CeldaCifrasMemoria{Grupo: entrada.GrupoCifrasMemoria{Subprograma: g.Subprograma, Origen: g.Origen, Sexo: g.Sexo, Tramo: g.Tramo, Motivo: g.Motivo, Papel: g.Papel, Estado: g.Estado, Causa: g.Causa, Organismo: g.Organismo}, Unidad: ce.Unidad, Numerador: c.Numerador, Denominador: entrada.DenominadorCifrasMemoria{Conocido: c.Denominador.Conocido, Valor: c.Denominador.Valor}, SinDato: c.SinDato, Poblacion: c.Poblacion, VersionRegla: c.VersionRegla}
			if c.Porcentaje != nil {
				celda.Porcentaje = &entrada.PorcentajeCifrasMemoria{Centesimas: c.Porcentaje.Centesimas, Aviso: c.Porcentaje.Aviso}
			}
			apartado.Celdas = append(apartado.Celdas, celda)
		}
		r.Apartados = append(r.Apartados, apartado)
	}
	return r
}
