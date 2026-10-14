package paquete

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	registro "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/paquete"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

type pendiente struct {
	contenido dominio.ContenidoInstalacion
	actor     string
	hasta     time.Time
}

// Instalador retiene como máximo dos paquetes revisados durante diez minutos.
// Nunca publica una preparación ni permite reutilizar su ticket.
type Instalador struct {
	validador   salida.ValidadorInstalacion
	antivirus   plataforma.Antivirus
	almacen     salida.AlmacenVersiones
	publicacion salida.Publicacion
	registro    registro.Registro
	ahora       func() time.Time
	mu          sync.Mutex
	pendientes  map[string]pendiente
	plazas      chan struct{}
}

func NuevoInstalador(v salida.ValidadorInstalacion, a plataforma.Antivirus, s salida.AlmacenVersiones, p salida.Publicacion, r registro.Registro, ahora func() time.Time) *Instalador {
	return &Instalador{validador: v, antivirus: a, almacen: s, publicacion: p, registro: r, ahora: ahora, pendientes: map[string]pendiente{}, plazas: make(chan struct{}, 2)}
}

func (i *Instalador) expirar() {
	i.mu.Lock()
	defer i.mu.Unlock()
	for ticket, p := range i.pendientes {
		if !i.ahora().Before(p.hasta) {
			delete(i.pendientes, ticket)
			<-i.plazas
		}
	}
}

func (i *Instalador) Descartar(ctx context.Context, a dominio.Actor, ticket string) error {
	if !a.PuedeAdministrar() {
		return dominio.ErrPermiso
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	if p, ok := i.pendientes[ticket]; ok && p.actor == a.UsuarioID {
		delete(i.pendientes, ticket)
		<-i.plazas
	}
	return nil
}

func (i *Instalador) Preparar(ctx context.Context, actor dominio.Actor, tipo, codigo string, datos []byte) (dominio.Preparacion, error) {
	vacio := dominio.Preparacion{}
	if !actor.PuedeAdministrar() {
		return vacio, dominio.ErrPermiso
	}
	if !dominio.IdentificadorValido(tipo, codigo) || len(datos) > 20<<20 {
		return vacio, dominio.ErrPaqueteInvalido
	}
	i.expirar()
	select {
	case i.plazas <- struct{}{}:
	default:
		return vacio, dominio.ErrOcupado
	}
	retener := false
	defer func() {
		if !retener {
			<-i.plazas
		}
	}()
	c, err := i.validador.Validar(ctx, tipo, datos)
	if err != nil {
		return vacio, i.fallo(ctx, actor, "paquete.revisar", err)
	}
	if c.Codigo != codigo {
		return vacio, i.fallo(ctx, actor, "paquete.revisar", dominio.ErrPaqueteInvalido)
	}
	if i.antivirus == nil {
		return vacio, i.fallo(ctx, actor, "paquete.revisar", dominio.ErrAntivirus)
	}
	estado, err := i.antivirus.Revisar(ctx, datos)
	if estado == plataforma.AntivirusInfectado {
		return vacio, i.fallo(ctx, actor, "paquete.revisar", dominio.ErrInfectado)
	}
	if err != nil || estado != plataforma.AntivirusLimpio {
		return vacio, i.fallo(ctx, actor, "paquete.revisar", dominio.ErrAntivirus)
	}
	if _, err = i.registro.Confirmar(ctx, i.evento(actor, "paquete.revisar", "ok")); err != nil {
		return vacio, dominio.ErrNoDisponible
	}
	var aleatorio [32]byte
	if _, err = rand.Read(aleatorio[:]); err != nil {
		return vacio, dominio.ErrNoDisponible
	}
	ticket := hex.EncodeToString(aleatorio[:])
	i.mu.Lock()
	i.pendientes[ticket] = pendiente{contenido: c, actor: actor.UsuarioID, hasta: i.ahora().Add(10 * time.Minute)}
	i.mu.Unlock()
	retener = true
	return dominio.Preparacion{Ticket: ticket, VersionInstalada: dominio.VersionInstalada{Tipo: c.Tipo, Codigo: c.Codigo, Version: c.Version, Huella: c.Huella}}, nil
}

func (i *Instalador) Actual(ctx context.Context, actor dominio.Actor, tipo, codigo string) (dominio.Referencia, error) {
	if !actor.PuedeAdministrar() {
		return dominio.Referencia{}, dominio.ErrPermiso
	}
	if !dominio.IdentificadorValido(tipo, codigo) {
		return dominio.Referencia{}, dominio.ErrPaqueteInvalido
	}
	refs, err := i.publicacion.Referencias(ctx, tipo)
	if err != nil {
		return dominio.Referencia{}, err
	}
	actual := dominio.Referencia{VersionInstalada: dominio.VersionInstalada{Tipo: tipo, Codigo: codigo}}
	for _, r := range refs {
		if r.Codigo == codigo {
			actual = r
			break
		}
	}
	if _, err = i.registro.Confirmar(ctx, i.evento(actor, "paquete.consultar", "ok")); err != nil {
		return dominio.Referencia{}, dominio.ErrNoDisponible
	}
	return actual, nil
}

func (i *Instalador) Activar(ctx context.Context, actor dominio.Actor, ticket string, revision int64) (dominio.Referencia, error) {
	vacio := dominio.Referencia{}
	if !actor.PuedeAdministrar() {
		return vacio, dominio.ErrPermiso
	}
	if revision < 0 || revision > 9007199254740991 {
		return vacio, dominio.ErrPaqueteInvalido
	}
	i.expirar()
	i.mu.Lock()
	p, ok := i.pendientes[ticket]
	if ok && p.actor == actor.UsuarioID {
		delete(i.pendientes, ticket)
	} else {
		ok = false
	}
	i.mu.Unlock()
	if !ok {
		return vacio, dominio.ErrPreparacion
	}
	defer func() { <-i.plazas }()
	ctx, cancelar := context.WithTimeout(ctx, 30*time.Second)
	defer cancelar()
	v, err := i.almacen.Guardar(ctx, p.contenido)
	if err != nil {
		return vacio, i.fallo(ctx, actor, "paquete.activar", err)
	}
	r, err := i.publicacion.Activar(ctx, actor, v, revision)
	if err == nil {
		return r, nil
	}
	// Un error de COMMIT puede ser indeterminado. Nunca borramos sin comprobar
	// en otra transacción si esa ruta llegó a registrarse, incluso si ya no está activa.
	limpieza, cancelarLimpieza := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancelarLimpieza()
	registrada, consultaErr := i.publicacion.Registrada(limpieza, v)
	if consultaErr == nil && !registrada {
		if limpiarErr := i.almacen.Eliminar(limpieza, v); limpiarErr != nil {
			return vacio, i.fallo(ctx, actor, "paquete.activar", dominio.ErrNoDisponible)
		}
	}
	return vacio, i.fallo(ctx, actor, "paquete.activar", err)
}

// Limpiar sólo elimina rutas propias, antiguas y no registradas. Una caída de
// PostgreSQL interrumpe la limpieza; conservar un huérfano es preferible a perder una versión.
func (i *Instalador) Limpiar(ctx context.Context) error {
	i.expirar()
	versiones, err := i.almacen.Huerfanas(ctx, i.ahora().Add(-time.Hour))
	if err != nil {
		return err
	}
	for _, v := range versiones {
		registrada, err := i.publicacion.Registrada(ctx, v)
		if err != nil {
			return err
		}
		if !registrada {
			if err := i.almacen.Eliminar(ctx, v); err != nil {
				return err
			}
		}
	}
	return nil
}

func (i *Instalador) evento(a dominio.Actor, accion, resultado string) auditoria.Datos {
	return auditoria.Datos{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, OcurridoEn: i.ahora().UTC().Truncate(time.Microsecond), Accion: accion, Entidad: "paquete", Resultado: resultado}
}

func (i *Instalador) fallo(ctx context.Context, a dominio.Actor, accion string, fallo error) error {
	ctx, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancelar()
	if _, err := i.registro.Confirmar(ctx, i.evento(a, accion, "error")); err != nil {
		return dominio.ErrNoDisponible
	}
	return fallo
}
