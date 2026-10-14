package identidad

import (
	"context"
	"encoding/base64"
	"errors"
	"time"

	"github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/identidad"
)

type Servicio struct {
	directorio  salida.Directorio
	usuarios    salida.Usuarios
	sesiones    salida.Sesiones
	reloj       salida.Reloj
	azar        salida.Aleatoriedad
	politica    dominio.PoliticaSesion
	limitador   *Limitador
	operaciones salida.Operaciones
}

var _ entrada.Sesiones = (*Servicio)(nil)

func NuevoServicio(d salida.Directorio, u salida.Usuarios, s salida.Sesiones, r salida.Reloj, a salida.Aleatoriedad, p dominio.PoliticaSesion, l *Limitador, operaciones ...salida.Operaciones) (*Servicio, error) {
	if err := p.Validar(); err != nil {
		return nil, err
	}
	if d == nil || u == nil || s == nil || r == nil || a == nil || l == nil {
		return nil, dominio.ErrConfiguracion
	}
	servicio := &Servicio{directorio: d, usuarios: u, sesiones: s, reloj: r, azar: a, politica: p, limitador: l}
	if len(operaciones) > 1 || len(operaciones) == 1 && operaciones[0] == nil {
		return nil, dominio.ErrConfiguracion
	}
	if len(operaciones) == 1 {
		servicio.operaciones = operaciones[0]
	}
	return servicio, nil
}

func (s *Servicio) Entrar(ctx context.Context, inicio entrada.Inicio) (abierta entrada.SesionAbierta, err error) {
	defer func() { err = s.registrarFallo(ctx, "identidad.sesion_entrar", err) }()
	return s.entrar(ctx, inicio)
}
func (s *Servicio) entrar(ctx context.Context, inicio entrada.Inicio) (entrada.SesionAbierta, error) {
	if err := ctx.Err(); err != nil {
		return entrada.SesionAbierta{}, err
	}
	cuenta, err := dominio.NormalizarCuenta(inicio.Cuenta)
	if err != nil {
		return entrada.SesionAbierta{}, err
	}
	if inicio.IP == (dominio.IP{}) {
		return entrada.SesionAbierta{}, dominio.ErrCredencialesInvalidas
	}
	if err := s.limitador.reservar(cuenta, inicio.IP, s.reloj.Ahora()); err != nil {
		return entrada.SesionAbierta{}, err
	}
	if inicio.Contrasena == "" {
		return entrada.SesionAbierta{}, dominio.ErrCredencialesInvalidas
	}
	identidad, err := s.directorio.Autenticar(ctx, cuenta, inicio.Contrasena)
	if err != nil {
		return entrada.SesionAbierta{}, errorDirectorio(err)
	}
	login, err := dominio.NormalizarCuenta(identidad.CuentaAD)
	if err != nil || login != cuenta || identidad.GUID == (dominio.GUID{}) {
		return entrada.SesionAbierta{}, dominio.ErrCredencialesInvalidas
	}
	s.limitador.liberar(cuenta, inicio.IP)
	identidad.CuentaAD = login
	if s.operaciones == nil {
		return s.abrir(ctx, identidad)
	}
	var abierta entrada.SesionAbierta
	err = s.operaciones.EjecutarIdentidad(ctx, func(ctx context.Context) error {
		var fallo error
		abierta, fallo = s.abrir(ctx, identidad)
		return fallo
	})
	if err != nil {
		return entrada.SesionAbierta{}, errorRepositorio(ctx, err)
	}
	return abierta, nil
}

func (s *Servicio) abrir(ctx context.Context, identidad dominio.IdentidadDirectorio) (entrada.SesionAbierta, error) {
	ahora := s.reloj.Ahora()
	usuario, err := s.usuarios.RegistrarAcceso(ctx, identidad, ahora)
	if err != nil {
		return entrada.SesionAbierta{}, errorRepositorio(ctx, err)
	}
	if usuario.GUID != identidad.GUID {
		return entrada.SesionAbierta{}, dominio.ErrUsuarioNoDisponible
	}
	estado, err := usuario.EstadoAcceso()
	if err != nil {
		return entrada.SesionAbierta{}, err
	}
	var bytes [32]byte
	if err := s.azar.Llenar(bytes[:]); err != nil {
		return entrada.SesionAbierta{}, dominio.ErrAleatoriedad
	}
	if bytes == ([32]byte{}) {
		return entrada.SesionAbierta{}, dominio.ErrAleatoriedad
	}
	identificador := base64.RawURLEncoding.EncodeToString(bytes[:])
	huella, _ := dominio.HuellaIdentificador(identificador)
	sesion, err := dominio.NuevaSesion(huella, usuario.ID, ahora, s.politica)
	if err != nil {
		return entrada.SesionAbierta{}, err
	}
	if err := ctx.Err(); err != nil {
		return entrada.SesionAbierta{}, err
	}
	if err := s.sesiones.Crear(ctx, sesion); err != nil {
		return entrada.SesionAbierta{}, errorRepositorio(ctx, err)
	}
	return entrada.SesionAbierta{Identificador: identificador, Acceso: entrada.Acceso{Usuario: usuario, Estado: estado}}, nil
}

func (s *Servicio) Consultar(ctx context.Context, identificador string) (acceso entrada.Acceso, err error) {
	defer func() {
		// Una cookie inventada o caducada sólo deja apunte dentro del
		// presupuesto de su origen; fuera de él responde «limitado» y deja
		// únicamente el registro técnico HTTP, como /api/sesion.
		if origen := entrada.Origen(ctx); errors.Is(err, dominio.ErrSesionNoValida) && origen != (dominio.IP{}) && s.limitador.admitirPeticion(origen, s.reloj.Ahora()) != nil {
			err = dominio.ErrIntentosLimitados
			return
		}
		err = s.registrarFallo(ctx, "identidad.sesion_consultar", err)
	}()
	return s.consultar(ctx, identificador)
}
func (s *Servicio) consultar(ctx context.Context, identificador string) (entrada.Acceso, error) {
	huella, err := dominio.HuellaIdentificador(identificador)
	if err != nil {
		return entrada.Acceso{}, err
	}
	var acceso entrada.Acceso
	err = s.sesiones.Modificar(ctx, huella, func(ctx context.Context, sesion *dominio.Sesion) error {
		if sesion.Huella != huella {
			return dominio.ErrSesionNoValida
		}
		if err := sesion.Tocar(s.reloj.Ahora(), s.politica); err != nil {
			return err
		}
		usuario, err := s.usuarios.Obtener(ctx, sesion.UsuarioID)
		if err != nil {
			return errorRepositorio(ctx, err)
		}
		if usuario.ID != sesion.UsuarioID {
			return dominio.ErrUsuarioNoDisponible
		}
		estado, err := usuario.EstadoAcceso()
		if err != nil {
			return err
		}
		acceso = entrada.Acceso{Usuario: usuario, Estado: estado}
		return nil
	})
	if err != nil {
		return entrada.Acceso{}, errorRepositorio(ctx, err)
	}
	return acceso, nil
}

func (s *Servicio) Salir(ctx context.Context, identificador string) (err error) {
	defer func() { err = s.registrarFallo(ctx, "identidad.sesion_salir", err) }()
	return s.salir(ctx, identificador)
}
func (s *Servicio) salir(ctx context.Context, identificador string) error {
	huella, err := dominio.HuellaIdentificador(identificador)
	if err != nil {
		return err
	}
	return errorRepositorio(ctx, s.sesiones.Eliminar(ctx, huella))
}

// No propagar errores de conectores: pueden incluir DN, cuentas o secretos.
func errorDirectorio(err error) error {
	for _, conocido := range []error{dominio.ErrCredencialesInvalidas, dominio.ErrDirectorioNoDisponible, context.Canceled, context.DeadlineExceeded} {
		if errors.Is(err, conocido) {
			return conocido
		}
	}
	return dominio.ErrDirectorioNoDisponible
}

func errorRepositorio(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	for _, conocido := range []error{dominio.ErrSesionNoValida, dominio.ErrUsuarioNoDisponible} {
		if errors.Is(err, conocido) {
			return conocido
		}
	}
	return dominio.ErrPersistencia
}

// La operación fallida ya ha revertido antes de confirmar este apunte. No se
// conserva un evento dentro de una transacción que deba revertirse.
func (s *Servicio) registrarFallo(ctx context.Context, accion string, fallo error) error {
	if fallo == nil || s.operaciones == nil {
		return fallo
	}
	resultado := "error"
	for _, denegacion := range []error{dominio.ErrCredencialesInvalidas, dominio.ErrIntentosLimitados, dominio.ErrSesionNoValida, dominio.ErrUsuarioNoDisponible} {
		if errors.Is(fallo, denegacion) {
			resultado = "denegado"
		}
	}
	registro, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancelar()
	if err := s.operaciones.RegistrarFallo(registro, accion, resultado); err != nil {
		return dominio.ErrPersistencia
	}
	return fallo
}

func (s *Servicio) RegistrarRechazo(ctx context.Context, accion, resultado string) error {
	if (accion != "identidad.sesion_entrar" && accion != "identidad.sesion_salir" && accion != "identidad.autorizacion") || (resultado != "denegado" && resultado != "error") {
		return dominio.ErrConfiguracion
	}
	if s.operaciones == nil {
		return dominio.ErrPersistencia
	}
	registro, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancelar()
	if accion == "identidad.autorizacion" {
		registro = salida.ConActorAuditoria(registro, entrada.AccesoActual(ctx).Usuario)
	}
	if err := s.operaciones.RegistrarFallo(registro, accion, resultado); err != nil {
		return dominio.ErrPersistencia
	}
	return nil
}

func (s *Servicio) AdmitirPeticion(ctx context.Context) error {
	return s.limitador.admitirPeticion(entrada.Origen(ctx), s.reloj.Ahora())
}
