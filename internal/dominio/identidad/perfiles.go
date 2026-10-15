package identidad

import "time"

const (
	ErrCambioPropioDenegado Error = "identidad.cambio_propio_denegado"
	ErrAccesoDenegado       Error = "identidad.acceso_denegado"
	ErrPerfilesInvalidos    Error = "identidad.perfiles_invalidos"
	ErrConflicto            Error = "identidad.conflicto"
)

type Concesion struct {
	Perfil Perfil     `json:"perfil"`
	Desde  time.Time  `json:"desde"`
	Hasta  *time.Time `json:"hasta"`
}

type Adscripcion struct {
	EquipoID string     `json:"equipo_id"`
	Puesto   string     `json:"puesto"`
	Desde    time.Time  `json:"desde"`
	Hasta    *time.Time `json:"hasta"`
}

type CambioPerfiles struct {
	Version  int64        `json:"version"`
	Estado   Estado       `json:"estado"`
	Perfiles []Concesion  `json:"perfiles"`
	Equipo   *Adscripcion `json:"equipo"`
}

func (u Usuario) Permite(perfil Perfil) bool {
	if u.Estado != Activo {
		return false
	}
	for _, p := range u.Perfiles {
		if p == perfil {
			return p != ProfesionalETF || u.EquipoID != ""
		}
	}
	return false
}

func (c CambioPerfiles) Validar() error {
	if c.Version < 1 || (c.Estado != Activo && c.Estado != Baja) || len(c.Perfiles) > 4 {
		return ErrPerfilesInvalidos
	}
	if c.Estado == Baja && (len(c.Perfiles) != 0 || c.Equipo != nil) {
		return ErrPerfilesInvalidos
	}
	vistos := map[Perfil]bool{}
	var profesional *Concesion
	for i, p := range c.Perfiles {
		if vistos[p.Perfil] || !intervaloValido(p.Desde, p.Hasta) {
			return ErrPerfilesInvalidos
		}
		switch p.Perfil {
		case Coordinacion, Consulta, Administracion:
		case ProfesionalETF:
			profesional = &c.Perfiles[i]
		default:
			return ErrPerfilesInvalidos
		}
		vistos[p.Perfil] = true
	}
	if (profesional == nil) != (c.Equipo == nil) {
		return ErrPerfilesInvalidos
	}
	if c.Equipo != nil {
		e := c.Equipo
		if e.EquipoID == "" || (e.Puesto != "ts" && e.Puesto != "es" && e.Puesto != "ps") || !intervaloValido(e.Desde, e.Hasta) || !e.Desde.Equal(profesional.Desde) || (e.Hasta == nil) != (profesional.Hasta == nil) {
			return ErrPerfilesInvalidos
		}
		if e.Hasta != nil && !e.Hasta.Equal(*profesional.Hasta) {
			return ErrPerfilesInvalidos
		}
	}
	return nil
}

func intervaloValido(desde time.Time, hasta *time.Time) bool {
	return !desde.IsZero() && (hasta == nil || hasta.After(desde))
}

type UsuarioAdministrado struct {
	ID        string       `json:"id"`
	Propio    bool         `json:"propio"`
	Cuenta    string       `json:"cuenta"`
	Nombre    string       `json:"nombre"`
	Iniciales string       `json:"iniciales"`
	Estado    Estado       `json:"estado"`
	Version   int64        `json:"version"`
	Perfiles  []Concesion  `json:"perfiles"`
	Equipo    *Adscripcion `json:"equipo"`
}
