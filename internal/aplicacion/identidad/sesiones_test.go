package identidad

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
)

type relojMemoria struct{ ahora time.Time }

func (r *relojMemoria) Ahora() time.Time { return r.ahora }

type azarSeguro struct{}

func (azarSeguro) Llenar(b []byte) error { _, err := rand.Read(b); return err }

type azarFallido struct{}

func (azarFallido) Llenar([]byte) error { return errors.New("fallo-tecnico-azar") }

type directorioMemoria struct {
	mu        sync.Mutex
	identidad dominio.IdentidadDirectorio
	err       error
	llamadas  int
}

func (d *directorioMemoria) Autenticar(_ context.Context, _, _ string) (dominio.IdentidadDirectorio, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.llamadas++
	return d.identidad, d.err
}

// Los dobles aplican los contratos atómicos; nunca guardan contraseñas ni tokens.
type usuariosMemoria struct {
	mu    sync.Mutex
	porID map[string]dominio.Usuario
	err   error
}

func clonarUsuario(u dominio.Usuario) dominio.Usuario {
	u.Perfiles = append([]dominio.Perfil(nil), u.Perfiles...)
	return u
}
func (u *usuariosMemoria) RegistrarAcceso(ctx context.Context, i dominio.IdentidadDirectorio, _ time.Time) (dominio.Usuario, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if ctx.Err() != nil {
		return dominio.Usuario{}, ctx.Err()
	}
	if u.err != nil {
		return dominio.Usuario{}, u.err
	}
	for id, usuario := range u.porID {
		if usuario.GUID == i.GUID {
			usuario.IdentidadDirectorio = i
			u.porID[id] = usuario
			return clonarUsuario(usuario), nil
		}
	}
	id := fmt.Sprintf("usuario-%d", len(u.porID)+1)
	usuario := dominio.Usuario{ID: id, IdentidadDirectorio: i, Estado: dominio.PendienteAlta}
	u.porID[id] = usuario
	return usuario, nil
}
func (u *usuariosMemoria) Obtener(ctx context.Context, id string) (dominio.Usuario, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if ctx.Err() != nil {
		return dominio.Usuario{}, ctx.Err()
	}
	if u.err != nil {
		return dominio.Usuario{}, u.err
	}
	usuario, ok := u.porID[id]
	if !ok {
		return dominio.Usuario{}, dominio.ErrUsuarioNoDisponible
	}
	return clonarUsuario(usuario), nil
}
func (u *usuariosMemoria) cambiar(id string, estado dominio.Estado, perfiles ...dominio.Perfil) {
	u.mu.Lock()
	defer u.mu.Unlock()
	usuario := u.porID[id]
	usuario.Estado = estado
	usuario.Perfiles = perfiles
	u.porID[id] = usuario
}

type sesionesMemoria struct {
	mu        sync.Mutex
	porHuella map[dominio.Huella]dominio.Sesion
	err       error
}

func (s *sesionesMemoria) Crear(ctx context.Context, sesion dominio.Sesion) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if s.err != nil {
		return s.err
	}
	if _, ok := s.porHuella[sesion.Huella]; ok {
		return errors.New("colision-huella")
	}
	s.porHuella[sesion.Huella] = sesion
	return nil
}
func (s *sesionesMemoria) Modificar(ctx context.Context, h dominio.Huella, fn func(context.Context, *dominio.Sesion) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if s.err != nil {
		return s.err
	}
	sesion, ok := s.porHuella[h]
	if !ok {
		return dominio.ErrSesionNoValida
	}
	if err := fn(ctx, &sesion); err != nil {
		return err
	}
	s.porHuella[h] = sesion
	return nil
}
func (s *sesionesMemoria) Eliminar(ctx context.Context, h dominio.Huella) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if s.err != nil {
		return s.err
	}
	delete(s.porHuella, h)
	return nil
}

type entorno struct {
	s      *Servicio
	d      *directorioMemoria
	u      *usuariosMemoria
	ses    *sesionesMemoria
	r      *relojMemoria
	inicio entrada.Inicio
}

func preparar(t *testing.T) entorno {
	t.Helper()
	d := &directorioMemoria{identidad: dominio.IdentidadDirectorio{GUID: dominio.GUID{1, 2, 3}, CuentaAD: "ines.moreno", NombreMostrado: "Inés Moreno Castillo"}}
	u := &usuariosMemoria{porID: make(map[string]dominio.Usuario)}
	ses := &sesionesMemoria{porHuella: make(map[dominio.Huella]dominio.Sesion)}
	r := &relojMemoria{ahora: time.Date(2026, 10, 13, 8, 0, 0, 0, time.UTC)}
	l, err := NuevoLimitador(dominio.PoliticaIntentos{Ventana: time.Minute, PorUsuario: 5, PorIP: 10, MaxClaves: 100})
	if err != nil {
		t.Fatal(err)
	}
	s, err := NuevoServicio(d, u, ses, r, azarSeguro{}, dominio.PoliticaSesion{Inactividad: 30 * time.Minute, Maxima: 8 * time.Hour}, l)
	if err != nil {
		t.Fatal(err)
	}
	return entorno{s, d, u, ses, r, entrada.Inicio{Cuenta: "ines.moreno", Contrasena: "clave-local-pruebas", IP: dominio.IP(netip.MustParseAddr("192.0.2.10").As16())}}
}

func TestEntradaPendienteHuellaYCierre(t *testing.T) {
	e := preparar(t)
	abierta, err := e.s.Entrar(context.Background(), e.inicio)
	if err != nil {
		t.Fatal(err)
	}
	if abierta.Acceso.Estado != dominio.PendienteAlta || len(abierta.Identificador) != 43 {
		t.Fatalf("sesión inesperada: %+v", abierta.Acceso)
	}
	h, _ := dominio.HuellaIdentificador(abierta.Identificador)
	if len(e.ses.porHuella) != 1 || e.ses.porHuella[h].UsuarioID != abierta.Acceso.Usuario.ID {
		t.Fatal("huella no persistida")
	}
	segunda, err := e.s.Entrar(context.Background(), e.inicio)
	if err != nil || segunda.Identificador == abierta.Identificador {
		t.Fatal("identificador no renovado", err)
	}
	if err := e.s.Salir(context.Background(), abierta.Identificador); err != nil {
		t.Fatal(err)
	}
	if err := e.s.Salir(context.Background(), abierta.Identificador); err != nil {
		t.Fatal("cierre no idempotente", err)
	}
	if _, err := e.s.Consultar(context.Background(), abierta.Identificador); !errors.Is(err, dominio.ErrSesionNoValida) {
		t.Fatal(err)
	}
	if _, err := e.s.Consultar(context.Background(), segunda.Identificador); err != nil {
		t.Fatal(err)
	}
}

func TestPermisosVigentesYSesionSinAD(t *testing.T) {
	e := preparar(t)
	abierta, err := e.s.Entrar(context.Background(), e.inicio)
	if err != nil {
		t.Fatal(err)
	}
	e.d.err = dominio.ErrDirectorioNoDisponible
	e.u.cambiar(abierta.Acceso.Usuario.ID, dominio.Activo, dominio.Consulta, dominio.Administracion)
	acceso, err := e.s.Consultar(context.Background(), abierta.Identificador)
	if err != nil || acceso.Estado != dominio.Activo || len(acceso.Usuario.Perfiles) != 2 {
		t.Fatalf("%+v %v", acceso, err)
	}
	// El resultado tampoco permite mutar los permisos conservados por el servidor.
	acceso.Usuario.Perfiles[0] = dominio.ProfesionalETF
	acceso, err = e.s.Consultar(context.Background(), abierta.Identificador)
	if err != nil || acceso.Usuario.Perfiles[0] != dominio.Consulta {
		t.Fatal("perfiles compartidos")
	}
	e.u.cambiar(abierta.Acceso.Usuario.ID, dominio.Activo)
	acceso, err = e.s.Consultar(context.Background(), abierta.Identificador)
	if err != nil || acceso.Estado != dominio.PendienteAlta {
		t.Fatal("retirada sin efecto", err)
	}
	e.u.cambiar(abierta.Acceso.Usuario.ID, dominio.Baja)
	if _, err = e.s.Consultar(context.Background(), abierta.Identificador); !errors.Is(err, dominio.ErrUsuarioNoDisponible) {
		t.Fatal(err)
	}
	if _, err = e.s.Entrar(context.Background(), e.inicio); !errors.Is(err, dominio.ErrDirectorioNoDisponible) {
		t.Fatal(err)
	}
	if e.d.llamadas != 2 {
		t.Fatal("consulta recurrió a AD")
	}
}

func TestCambioLoginYLoginReutilizado(t *testing.T) {
	e := preparar(t)
	primera, err := e.s.Entrar(context.Background(), e.inicio)
	if err != nil {
		t.Fatal(err)
	}
	e.u.cambiar(primera.Acceso.Usuario.ID, dominio.Activo, dominio.Coordinacion)
	e.inicio.Cuenta = "ines.castillo"
	e.d.identidad.CuentaAD = e.inicio.Cuenta
	segunda, err := e.s.Entrar(context.Background(), e.inicio)
	if err != nil || segunda.Acceso.Usuario.ID != primera.Acceso.Usuario.ID || segunda.Acceso.Estado != dominio.Activo {
		t.Fatal("renombrado perdió identidad", err)
	}
	e.d.identidad.GUID = dominio.GUID{9}
	e.inicio.Cuenta = "ines.moreno"
	e.d.identidad.CuentaAD = e.inicio.Cuenta
	tercera, err := e.s.Entrar(context.Background(), e.inicio)
	if err != nil || tercera.Acceso.Usuario.ID == primera.Acceso.Usuario.ID || tercera.Acceso.Estado != dominio.PendienteAlta {
		t.Fatal("login heredó permisos", err)
	}
}

func TestCaducidadEnAplicacion(t *testing.T) {
	for _, tipo := range []string{"inactividad", "maxima"} {
		t.Run(tipo, func(t *testing.T) {
			e := preparar(t)
			abierta, err := e.s.Entrar(context.Background(), e.inicio)
			if err != nil {
				t.Fatal(err)
			}
			if tipo == "maxima" {
				for i := 0; i < 23; i++ {
					e.r.ahora = e.r.ahora.Add(20 * time.Minute)
					if _, err = e.s.Consultar(context.Background(), abierta.Identificador); err != nil {
						t.Fatal(err)
					}
				}
				e.r.ahora = e.r.ahora.Add(20 * time.Minute)
			} else {
				e.r.ahora = e.r.ahora.Add(30 * time.Minute)
			}
			if _, err = e.s.Consultar(context.Background(), abierta.Identificador); !errors.Is(err, dominio.ErrSesionNoValida) {
				t.Fatal(err)
			}
		})
	}
}

func TestIntentosPorUsuarioIPYVentana(t *testing.T) {
	e := preparar(t)
	e.d.err = dominio.ErrCredencialesInvalidas
	for i := 0; i < 5; i++ {
		e.inicio.IP[15] = byte(10 + i)
		if _, err := e.s.Entrar(context.Background(), e.inicio); !errors.Is(err, dominio.ErrCredencialesInvalidas) {
			t.Fatal(err)
		}
	}
	e.inicio.Cuenta = "  INES.MORENO "
	if _, err := e.s.Entrar(context.Background(), e.inicio); !errors.Is(err, dominio.ErrIntentosLimitados) {
		t.Fatal(err)
	}
	if e.d.llamadas != 5 {
		t.Fatal("límite después del directorio")
	}
	e.r.ahora = e.r.ahora.Add(time.Minute)
	if _, err := e.s.Entrar(context.Background(), e.inicio); !errors.Is(err, dominio.ErrCredencialesInvalidas) {
		t.Fatal("ventana no renovada", err)
	}
	e = preparar(t)
	e.d.err = dominio.ErrCredencialesInvalidas
	for i := 0; i < 10; i++ {
		e.inicio.Cuenta = fmt.Sprintf("persona-%d", i)
		if _, err := e.s.Entrar(context.Background(), e.inicio); !errors.Is(err, dominio.ErrCredencialesInvalidas) {
			t.Fatal(err)
		}
	}
	e.inicio.Cuenta = "pablo.ruiz"
	if _, err := e.s.Entrar(context.Background(), e.inicio); !errors.Is(err, dominio.ErrIntentosLimitados) {
		t.Fatal(err)
	}
	if e.d.llamadas != 10 {
		t.Fatal("IP no limitada")
	}
}

func TestIntentosConcurrentesYMemoriaAcotada(t *testing.T) {
	e := preparar(t)
	e.d.err = dominio.ErrCredencialesInvalidas
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := e.s.Entrar(context.Background(), e.inicio)
			if !errors.Is(err, dominio.ErrCredencialesInvalidas) && !errors.Is(err, dominio.ErrIntentosLimitados) {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if e.d.llamadas != 5 {
		t.Fatalf("llamadas concurrentes: %d", e.d.llamadas)
	}
	l, err := NuevoLimitador(dominio.PoliticaIntentos{Ventana: time.Minute, PorUsuario: 2, PorIP: 2, MaxClaves: 2})
	if err != nil {
		t.Fatal(err)
	}
	if err = l.reservar("ines.moreno", e.inicio.IP, e.r.ahora); err != nil {
		t.Fatal(err)
	}
	if err = l.reservar("pablo.ruiz", e.inicio.IP, e.r.ahora); !errors.Is(err, dominio.ErrIntentosLimitados) {
		t.Fatal(err)
	}
	if len(l.intentos) != 2 {
		t.Fatal("memoria sin límite")
	}
	if err = l.reservar("pablo.ruiz", e.inicio.IP, e.r.ahora.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
}

func TestRechazosSinSesionNiFiltracion(t *testing.T) {
	for _, caso := range []string{"vacia", "directorio", "azar", "usuarios", "sesiones", "baja", "guid", "cancelado"} {
		t.Run(caso, func(t *testing.T) {
			e := preparar(t)
			ctx := context.Background()
			esperado := error(dominio.ErrCredencialesInvalidas)
			switch caso {
			case "vacia":
				e.inicio.Contrasena = ""
			case "directorio":
				e.d.err = errors.New("fallo con datos privados")
				esperado = dominio.ErrDirectorioNoDisponible
			case "azar":
				e.s.azar = azarFallido{}
				esperado = dominio.ErrAleatoriedad
			case "usuarios":
				e.u.err = errors.New("fallo con datos privados")
				esperado = dominio.ErrPersistencia
			case "sesiones":
				e.ses.err = errors.New("fallo con datos privados")
				esperado = dominio.ErrPersistencia
			case "baja":
				e.u.porID["usuario-1"] = dominio.Usuario{ID: "usuario-1", IdentidadDirectorio: e.d.identidad, Estado: dominio.Baja}
				esperado = dominio.ErrUsuarioNoDisponible
			case "guid":
				e.d.identidad.GUID = dominio.GUID{}
			case "cancelado":
				c, cancel := context.WithCancel(ctx)
				cancel()
				ctx = c
				esperado = context.Canceled
			}
			abierta, err := e.s.Entrar(ctx, e.inicio)
			if !errors.Is(err, esperado) || abierta.Identificador != "" || len(e.ses.porHuella) != 0 {
				t.Fatalf("error: %v", err)
			}
			if (caso == "vacia" || caso == "cancelado") && e.d.llamadas != 0 {
				t.Fatal("llamó al directorio")
			}
		})
	}
}

func TestCierreConcurrenteNoReviveSesion(t *testing.T) {
	e := preparar(t)
	abierta, err := e.s.Entrar(context.Background(), e.inicio)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := e.s.Consultar(context.Background(), abierta.Identificador)
			if err != nil && !errors.Is(err, dominio.ErrSesionNoValida) {
				t.Error(err)
			}
		}()
	}
	if err = e.s.Salir(context.Background(), abierta.Identificador); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	if len(e.ses.porHuella) != 0 {
		t.Fatal("sesión revivida")
	}
}
