package expediente

import (
	"context"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	paquetes "github.com/aavidad/expedientes-etf/internal/aplicacion/paquete/puertos/entrada"
	auditoria "github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	audit "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
	salidaCatalogo "github.com/aavidad/expedientes-etf/internal/puertos/salida/catalogo"
	doc "github.com/aavidad/expedientes-etf/internal/puertos/salida/documento"
	exp "github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"slices"
	"strconv"
	"time"
)

type LecturaEquipo struct {
	Avisos               exp.ConsultaAvisosEquipo
	Asignacion           salidaCatalogo.Repositorio
	Catalogos            paquetes.PublicacionCatalogos
	IdiomaPredeterminado string
	Repositorio          exp.BandejaEquipo
	Documentos           doc.ConsultaEquipo
	Historial            audit.HistorialExpediente
	Transacciones        plataforma.Transacciones
	Registro             audit.RegistroTransaccional
	Reloj                exp.Reloj
	Catalogo             salidaCatalogo.MotivosCierre
	Capacidades          []string
}

func (s *LecturaEquipo) EtiquetasEquipo(ctx context.Context, a d.Actor, idioma string) (map[string]string, error) {
	var etiquetas map[string]string
	err := s.leer(ctx, a, "", "equipo.etiquetas_consultadas", nil, func(ctx context.Context) error {
		if len(idioma) > 63 || s.Catalogos == nil {
			return d.ErrNoDisponible
		}
		c, err := s.Catalogos.Catalogos(ctx)
		if err != nil {
			return err
		}
		etiquetas = map[string]string{}
		for _, textos := range []map[string]string{c.Textos[s.IdiomaPredeterminado], c.Textos[idioma]} {
			for k, v := range textos {
				etiquetas[k] = v
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return etiquetas, nil
}

var _ entrada.BandejaEquipo = (*LecturaEquipo)(nil)

func (s *LecturaEquipo) leer(ctx context.Context, a d.Actor, id, accion string, cantidad *int, consultar func(context.Context) error) (fallo error) {
	defer func() {
		if fallo != nil && s.Registro != nil && s.Reloj != nil {
			resultado := "error"
			if errors.Is(fallo, d.ErrAcceso) {
				resultado = "denegado"
			}
			if _, err := s.Registro.Confirmar(ctx, s.eventoLectura(a, "equipo.lectura_rechazada", "", resultado)); err != nil {
				fallo = d.ErrNoDisponible
			}
		}
	}()
	if err := a.AutorizarLecturaEquipo(); err != nil {
		return err
	}
	if id != "" && !d.IDValido(id) {
		return d.ErrPeticion
	}
	if s.Transacciones == nil || s.Registro == nil || s.Reloj == nil || s.Repositorio == nil {
		return d.ErrNoDisponible
	}
	err := s.Transacciones.Ejecutar(ctx, identidad(a), func(ctx context.Context) error {
		if err := consultar(ctx); err != nil {
			return err
		}
		if cantidad != nil {
			accion += ".filas_" + strconv.Itoa(*cantidad)
		}
		_, err := s.Registro.RegistrarEnTransaccion(ctx, s.eventoLectura(a, accion, id, "ok"))
		return err
	})
	if errors.Is(err, d.ErrNoEncontrada) || errors.Is(err, d.ErrAcceso) || errors.Is(err, d.ErrPeticion) {
		return err
	}
	if err != nil {
		return d.ErrNoDisponible
	}
	return nil
}

func (s *LecturaEquipo) eventoLectura(a d.Actor, accion, id, resultado string) auditoria.Datos {
	if !d.IDValido(a.UsuarioID) {
		a = d.Actor{UsuarioID: "sistema"}
	}
	return auditoria.Datos{OcurridoEn: s.Reloj.Ahora().UTC().Truncate(time.Microsecond), UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, Accion: accion, Entidad: "expediente", EntidadID: id, ExpedienteID: id, Resultado: resultado}
}

func (s *LecturaEquipo) ListarEquipo(ctx context.Context, a d.Actor, f d.FiltroEquipo) (d.PaginaEquipo, error) {
	var p d.PaginaEquipo
	var cantidad int
	err := s.leer(ctx, a, "", "equipo.bandeja_consultada", &cantidad, func(ctx context.Context) error {
		if err := f.Validar(); err != nil {
			return err
		}
		var err error
		p, err = s.Repositorio.ListarEquipo(ctx, f)
		cantidad = len(p.Filas)
		if err == nil {
			err = s.completarAvisosEquipo(ctx, a, p.Filas)
		}
		if err == nil && a.EquipoID != "" && s.Asignacion != nil {
			version, e := s.Asignacion.Vigente(ctx, "asignacion_municipios", s.Reloj.Ahora())
			if e != nil {
				return e
			}
			datos, e := s.Asignacion.Datos(ctx, version)
			if e != nil {
				return e
			}
			for _, equipo := range datos.Equipos {
				if equipo.ID == a.EquipoID {
					p.EquipoClave = equipo.ClaveTexto
					break
				}
			}
		}
		return err
	})
	if err != nil {
		return d.PaginaEquipo{}, err
	}
	return p, nil
}

func (s *LecturaEquipo) AbrirFichaEquipo(ctx context.Context, a d.Actor, id string) (d.FichaEquipo, error) {
	var f d.FichaEquipo
	err := s.leer(ctx, a, id, "equipo.ficha_consultada", nil, func(ctx context.Context) error {
		var err error
		f, err = s.Repositorio.FichaEquipo(ctx, id)
		if err != nil {
			return err
		}
		// Un perfil doble no convierte la bandeja profesional en acceso a otro equipo.
		if a.EquipoID != "" && f.EquipoID != a.EquipoID {
			return d.ErrAcceso
		}
		filas := []d.ResumenEquipo{f.ResumenEquipo}
		if err := s.completarAvisosEquipo(ctx, a, filas); err != nil {
			return err
		}
		f.ResumenEquipo = filas[0]
		f.Emisiones = []d.SucesoExpediente{}
		f.Historial = []d.SucesoExpediente{}
		if s.Documentos == nil || s.Historial == nil {
			return d.ErrNoDisponible
		}
		emisiones, err := s.Documentos.EmisionesEquipo(ctx, id)
		if err != nil {
			return err
		}
		for _, e := range emisiones {
			f.Emisiones = append(f.Emisiones, d.SucesoExpediente{ID: e.ID, Tipo: "emision", Codigo: e.Tipo, Fecha: e.Fecha, Autor: e.Autor, VersionPlantilla: e.VersionPlantilla})
		}
		apuntes, err := s.Historial.HistorialExpediente(ctx, id)
		if err != nil {
			return err
		}
		for _, e := range apuntes {
			f.Historial = append(f.Historial, d.SucesoExpediente{ID: e.ID, Tipo: "auditoria", Codigo: e.Accion, Fecha: e.Fecha, Autor: e.Autor, Resultado: e.Resultado})
		}
		f.Capacidades = append([]string{}, s.Capacidades...)
		f.MotivosCierre = []d.OpcionCierre{}
		f.PuedeProponer = a.AutorizarPropuesta() == nil
		if f.PuedeProponer && slices.Contains(s.Capacidades, "cierre") {
			if s.Catalogo == nil {
				return d.ErrNoDisponible
			}
			catalogo, err := s.Catalogo.CierreVigente(ctx, s.Reloj.Ahora())
			if err != nil {
				return err
			}
			for _, m := range catalogo.Motivos {
				f.MotivosCierre = append(f.MotivosCierre, d.OpcionCierre{Codigo: m.Codigo, AdmiteTexto: m.AdmiteTexto, SinConsentimiento: m.SinConsentimiento})
			}
		}
		return nil
	})
	if err != nil {
		return d.FichaEquipo{}, err
	}
	return f, nil
}

// Los avisos se calculan con el puerto y las reglas de F2.02, dentro de la
// misma transacción auditada. Sólo se proyectan los expedientes de esta página.
func (s *LecturaEquipo) completarAvisosEquipo(ctx context.Context, a d.Actor, filas []d.ResumenEquipo) error {
	if len(filas) == 0 {
		return nil
	}
	if s.Avisos == nil || s.Asignacion == nil {
		return d.ErrNoDisponible
	}
	ahora := s.Reloj.Ahora()
	reglas, _, err := Nuevo(Dependencias{Catalogo: s.Asignacion}).reglasAvisos(ctx, ahora)
	if err != nil {
		return err
	}
	ids := make([]string, len(filas))
	for i := range filas {
		ids[i] = filas[i].ID
	}
	datos, err := s.Avisos.DatosAvisosEquipo(ctx, ids, a.EquipoID, d.FechaCivil(ahora))
	if err != nil {
		return err
	}
	seleccion := make(map[string]*d.ResumenEquipo, len(filas))
	for i := range filas {
		seleccion[filas[i].ID] = &filas[i]
	}
	for _, dato := range datos {
		fila, ok := seleccion[dato.ID]
		if !ok {
			continue
		}
		if dato.EquipoID != fila.EquipoID || (a.EquipoID != "" && dato.EquipoID != a.EquipoID) {
			return d.ErrAcceso
		}
		avisos, err := d.CalcularAvisos(dato, d.FechaCivil(ahora), reglas, true)
		if err != nil {
			return err
		}
		for _, aviso := range avisos {
			clave := aviso.ClaveTexto
			if aviso.Codigo == "memoria_dato_faltante" {
				clave = "equipo.aviso.memoria"
			}
			if !slices.Contains(fila.Avisos, clave) {
				fila.Avisos = append(fila.Avisos, clave)
			}
		}
		delete(seleccion, dato.ID)
	}
	if len(seleccion) != 0 {
		return d.ErrNoDisponible
	}
	return nil
}
