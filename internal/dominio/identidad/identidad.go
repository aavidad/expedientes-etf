package identidad

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"time"
)

// Error es una clave estable. Los adaptadores de entrada resolverán su texto.
type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrCredencialesInvalidas  Error = "identidad.credenciales_invalidas"
	ErrDirectorioNoDisponible Error = "identidad.directorio_no_disponible"
	ErrIntentosLimitados      Error = "identidad.intentos_limitados"
	ErrSesionNoValida         Error = "identidad.sesion_no_valida"
	ErrUsuarioNoDisponible    Error = "identidad.usuario_no_disponible"
	ErrConfiguracion          Error = "identidad.configuracion_invalida"
	ErrPersistencia           Error = "identidad.persistencia_no_disponible"
	ErrAleatoriedad           Error = "identidad.aleatoriedad_no_disponible"
)

// GUID contiene los bytes en orden UUID canónico, no en orden binario de AD.
type GUID [16]byte
type Huella [32]byte

// IP usa 16 bytes; IPv4 se representa como IPv6 mapeada. El adaptador de
// entrada valida el origen y convierte con netip.Addr.As16, sin texto libre.
type IP [16]byte

type IdentidadDirectorio struct {
	GUID           GUID
	CuentaAD       string
	NombreMostrado string
}

type Estado string

const (
	PendienteAlta Estado = "pendiente_alta"
	Activo        Estado = "activo"
	Baja          Estado = "baja"
)

type Perfil string

const (
	Coordinacion   Perfil = "coordinacion"
	ProfesionalETF Perfil = "profesional_etf"
	Consulta       Perfil = "consulta"
	Administracion Perfil = "administracion"
)

type Usuario struct {
	ID string
	IdentidadDirectorio
	Estado               Estado
	Perfiles             []Perfil
	EquipoID             string
	RevisionRestauracion bool
}

// EstadoAcceso deniega bajas, estados desconocidos y perfiles sin contrato.
func (u Usuario) EstadoAcceso() (Estado, error) {
	if u.ID == "" || u.GUID == (GUID{}) || (u.Estado != Activo && u.Estado != PendienteAlta) {
		return "", ErrUsuarioNoDisponible
	}
	if len(u.Perfiles) == 0 {
		return PendienteAlta, nil
	}
	if u.Estado != Activo {
		return "", ErrUsuarioNoDisponible
	}
	for _, p := range u.Perfiles {
		switch p {
		case Coordinacion, Consulta, Administracion:
		case ProfesionalETF:
			if u.EquipoID == "" {
				return "", ErrUsuarioNoDisponible
			}
		default:
			return "", ErrUsuarioNoDisponible
		}
	}
	return Activo, nil
}

// NormalizarCuenta admite un único formato de login corto; evita alias de AD
// que permitirían eludir el contador por usuario. LDAP añade el dominio al bind.
func NormalizarCuenta(cuenta string) (string, error) {
	cuenta = strings.ToLower(strings.TrimSpace(cuenta))
	if len(cuenta) == 0 || len(cuenta) > 128 {
		return "", ErrCredencialesInvalidas
	}
	for _, c := range cuenta {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == '-') {
			return "", ErrCredencialesInvalidas
		}
	}
	return cuenta, nil
}

type PoliticaSesion struct {
	Inactividad time.Duration
	Maxima      time.Duration
}

func (p PoliticaSesion) Validar() error {
	if p.Inactividad <= 0 || p.Maxima < p.Inactividad {
		return ErrConfiguracion
	}
	return nil
}

// Sesion nunca contiene el identificador que se entrega al cliente.
type Sesion struct {
	Huella            Huella
	UsuarioID         string
	CreadaEn          time.Time
	UltimaActividadEn time.Time
	CaducaEn          time.Time
}

func NuevaSesion(huella Huella, usuarioID string, ahora time.Time, p PoliticaSesion) (Sesion, error) {
	if err := p.Validar(); err != nil {
		return Sesion{}, err
	}
	if huella == (Huella{}) || usuarioID == "" || ahora.IsZero() {
		return Sesion{}, ErrSesionNoValida
	}
	return Sesion{huella, usuarioID, ahora, ahora, ahora.Add(p.Maxima)}, nil
}

// Tocar no revive sesiones caducadas ni desplaza la fecha máxima absoluta.
func (s *Sesion) Tocar(ahora time.Time, p PoliticaSesion) error {
	if err := p.Validar(); err != nil {
		return err
	}
	if s.Huella == (Huella{}) || s.UsuarioID == "" || s.CreadaEn.IsZero() ||
		s.UltimaActividadEn.Before(s.CreadaEn) || !s.CaducaEn.After(s.CreadaEn) ||
		ahora.Before(s.UltimaActividadEn) || !ahora.Before(s.CaducaEn) ||
		!ahora.Before(s.CreadaEn.Add(p.Maxima)) || !ahora.Before(s.UltimaActividadEn.Add(p.Inactividad)) {
		return ErrSesionNoValida
	}
	s.UltimaActividadEn = ahora
	return nil
}

func HuellaIdentificador(identificador string) (Huella, error) {
	bytes, err := base64.RawURLEncoding.DecodeString(identificador)
	if err != nil || len(bytes) != 32 || base64.RawURLEncoding.EncodeToString(bytes) != identificador {
		return Huella{}, ErrSesionNoValida
	}
	return Huella(sha256.Sum256([]byte(identificador))), nil
}

type PoliticaIntentos struct {
	Ventana    time.Duration
	PorUsuario int
	PorIP      int
	MaxClaves  int
}

func (p PoliticaIntentos) Validar() error {
	if p.Ventana <= 0 || p.PorUsuario <= 0 || p.PorIP <= 0 || p.MaxClaves < 2 {
		return ErrConfiguracion
	}
	return nil
}

type Intentos struct {
	Desde    time.Time
	Cantidad int
}

func (i Intentos) Vigente(ahora time.Time, ventana time.Duration) bool {
	return !i.Desde.IsZero() && ahora.Before(i.Desde.Add(ventana))
}

func (i Intentos) Reservar(ahora time.Time, ventana time.Duration, maximo int) (Intentos, error) {
	if !i.Vigente(ahora, ventana) {
		i = Intentos{Desde: ahora}
	}
	if ahora.Before(i.Desde) || i.Cantidad >= maximo {
		return i, ErrIntentosLimitados
	}
	i.Cantidad++
	return i, nil
}
