package actividad

import (
	"context"
	"errors"
	paquetes "github.com/aavidad/expedientes-etf/internal/aplicacion/paquete/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/actividad"
	auditoria "github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/actividad"
	registro "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"strconv"
	"strings"
	"time"
)

type Servicio struct {
	repo      salida.Repositorio
	tx        plataforma.Transacciones
	auditoria registro.Registro
	ahora     func() time.Time
	equipos   salida.Equipos
	etiquetas paquetes.PublicacionCatalogos
}

type Opciones struct {
	Equipos   salida.Equipos
	Etiquetas paquetes.PublicacionCatalogos
}

func Nuevo(repo salida.Repositorio, tx plataforma.Transacciones, audit registro.Registro, ahora func() time.Time, opciones ...Opciones) *Servicio {
	s := &Servicio{repo: repo, tx: tx, auditoria: audit, ahora: ahora}
	if len(opciones) > 0 {
		s.equipos = opciones[0].Equipos
		s.etiquetas = opciones[0].Etiquetas
	}
	return s
}

func (s *Servicio) auditar(ctx context.Context, a dominio.Actor, accion, id, resultado string) error {
	usuario, perfiles := a.UsuarioID, a.Perfiles
	if !dominio.UUIDValido(usuario) {
		usuario, perfiles = "sistema", nil
	}
	_, err := s.auditoria.Confirmar(ctx, auditoria.Datos{OcurridoEn: s.ahora().UTC().Truncate(time.Microsecond),
		UsuarioID: usuario, Perfiles: perfiles, Accion: accion, Entidad: "actividad.registro", EntidadID: id, Resultado: resultado})
	return err
}

func (s *Servicio) fallo(ctx context.Context, a dominio.Actor, accion, id string, err error) error {
	resultado := "error"
	if errors.Is(err, dominio.ErrPermiso) {
		resultado = "denegado"
	}
	identidad := plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}
	if !dominio.UUIDValido(a.UsuarioID) {
		identidad = plataforma.Identidad{}
	}
	if !dominio.UUIDValido(id) {
		id = ""
	}
	if fallo := s.tx.Ejecutar(ctx, identidad, func(ctx context.Context) error {
		return s.auditar(ctx, a, accion, id, resultado)
	}); fallo != nil {
		return dominio.ErrPersistencia
	}
	return err
}

func (s *Servicio) ejecutar(ctx context.Context, a dominio.Actor, accion, objetivo string, escritura bool, operacion func(context.Context) (string, error)) error {
	if err := a.Autorizar(escritura); err != nil {
		return s.fallo(ctx, a, accion, objetivo, err)
	}
	id := objetivo
	err := s.tx.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
		resultadoID, err := operacion(ctx)
		if resultadoID != "" {
			id = resultadoID
		}
		if err != nil {
			return err
		}
		return s.auditar(ctx, a, accion, id, "ok")
	})
	if err != nil {
		return s.fallo(ctx, a, accion, id, err)
	}
	return nil
}

func filtroActor(a dominio.Actor, f dominio.Filtro) (dominio.Filtro, error) {
	if err := f.Validar(); err != nil {
		return f, err
	}
	for _, p := range a.Perfiles {
		if p == "coordinacion" {
			return f, nil
		}
	}
	if f.EquipoID != "" && f.EquipoID != a.EquipoID {
		return f, dominio.ErrPermiso
	}
	f.EquipoID = a.EquipoID
	return f, nil
}

func (s *Servicio) Consultar(ctx context.Context, a dominio.Actor, tipo dominio.Tipo, f dominio.Filtro) ([]dominio.Registro, error) {
	var filas []dominio.Registro
	err := s.ejecutar(ctx, a, "actividad.consultar_"+string(tipo), "", false, func(ctx context.Context) (string, error) {
		f, err := filtroActor(a, f)
		if err != nil {
			return "", err
		}
		filas, err = s.repo.Consultar(ctx, tipo, f)
		return "", err
	})
	if err != nil {
		return nil, err
	}
	return filas, nil
}

func (s *Servicio) guardar(ctx context.Context, a dominio.Actor, tipo dominio.Tipo, id string, c dominio.Cambio, nuevo bool) (dominio.Registro, error) {
	var fila dominio.Registro
	accion := "actividad.corregir_" + string(tipo)
	if nuevo {
		accion = "actividad.registrar_" + string(tipo)
	}
	err := s.ejecutar(ctx, a, accion, id, true, func(ctx context.Context) (string, error) {
		if err := c.Validar(tipo, nuevo); err != nil {
			return "", err
		}
		if !nuevo && !dominio.UUIDValido(id) {
			return "", dominio.ErrDatos
		}
		if tipo == dominio.Informe {
			opciones, err := s.repo.Organismos(ctx)
			if err != nil {
				return "", err
			}
			valido := false
			for _, o := range opciones {
				if o.Codigo == c.Organismo {
					valido = true
				}
			}
			if !valido {
				return "", dominio.ErrDatos
			}
		}
		var err error
		if nuevo {
			fila, err = s.repo.Registrar(ctx, tipo, a.EquipoID, c)
		} else {
			fila, err = s.repo.Corregir(ctx, tipo, id, c)
		}
		return fila.ID, err
	})
	if err != nil {
		return dominio.Registro{}, err
	}
	return fila, nil
}

func (s *Servicio) Registrar(ctx context.Context, a dominio.Actor, tipo dominio.Tipo, c dominio.Cambio) (dominio.Registro, error) {
	return s.guardar(ctx, a, tipo, "", c, true)
}
func (s *Servicio) Corregir(ctx context.Context, a dominio.Actor, tipo dominio.Tipo, id string, c dominio.Cambio) (dominio.Registro, error) {
	return s.guardar(ctx, a, tipo, id, c, false)
}
func (s *Servicio) Retirar(ctx context.Context, a dominio.Actor, tipo dominio.Tipo, id string, version int) (dominio.Registro, error) {
	var fila dominio.Registro
	err := s.ejecutar(ctx, a, "actividad.retirar_"+string(tipo), id, true, func(ctx context.Context) (string, error) {
		if !dominio.UUIDValido(id) || version < 1 {
			return "", dominio.ErrDatos
		}
		var err error
		fila, err = s.repo.Retirar(ctx, tipo, id, version)
		return fila.ID, err
	})
	if err != nil {
		return dominio.Registro{}, err
	}
	return fila, nil
}
func (s *Servicio) Organismos(ctx context.Context, a dominio.Actor) ([]dominio.Organismo, error) {
	var filas []dominio.Organismo
	err := s.ejecutar(ctx, a, "actividad.consultar_organismos", "", false, func(ctx context.Context) (string, error) {
		var err error
		filas, err = s.repo.Organismos(ctx)
		return "", err
	})
	if err != nil {
		return nil, err
	}
	return filas, nil
}
func (s *Servicio) Recontar(ctx context.Context, a dominio.Actor, f dominio.Filtro) (dominio.Recuento, error) {
	var cifras dominio.Recuento
	err := s.ejecutar(ctx, a, "actividad.recontar_"+strconv.Itoa(f.Anio), "", false, func(ctx context.Context) (string, error) {
		f, err := filtroActor(a, f)
		if err != nil {
			return "", err
		}
		cifras, err = s.repo.Recontar(ctx, f)
		return "", err
	})
	if err != nil {
		return dominio.Recuento{}, err
	}
	return cifras, nil
}

func (s *Servicio) Equipos(ctx context.Context, a dominio.Actor) ([]dominio.Equipo, error) {
	filas := []dominio.Equipo{}
	err := s.ejecutar(ctx, a, "actividad.consultar_equipos", "", false, func(ctx context.Context) (string, error) {
		if s.equipos == nil {
			return "", dominio.ErrPersistencia
		}
		opciones, err := s.equipos.ConsultarEquipos(ctx)
		if err != nil {
			return "", err
		}
		provincial := false
		for _, p := range a.Perfiles {
			if p == "coordinacion" {
				provincial = true
			}
		}
		for _, e := range opciones {
			if provincial || e.ID == a.EquipoID {
				filas = append(filas, e)
			}
		}
		return "", nil
	})
	if err != nil {
		return nil, err
	}
	return filas, nil
}

func (s *Servicio) Etiquetas(ctx context.Context, a dominio.Actor) (map[string]map[string]string, error) {
	textos := make(map[string]map[string]string)
	err := s.ejecutar(ctx, a, "actividad.consultar_etiquetas", "", false, func(ctx context.Context) (string, error) {
		if s.etiquetas == nil {
			return "", dominio.ErrPersistencia
		}
		catalogos, err := s.etiquetas.Catalogos(ctx)
		if err != nil {
			return "", dominio.ErrPersistencia
		}
		for idioma, catalogo := range catalogos.Textos {
			textos[idioma] = make(map[string]string)
			for clave, valor := range catalogo {
				if strings.HasPrefix(clave, "equipo.") || strings.HasPrefix(clave, "actividad.organismo.") {
					textos[idioma][clave] = valor
				}
			}
		}
		return "", nil
	})
	if err != nil {
		return nil, err
	}
	return textos, nil
}
