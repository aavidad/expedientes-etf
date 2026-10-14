package identidad

import (
	"crypto/sha256"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"sync"
	"time"
)

// Limitador es compartido por todas las entradas de una instancia. Cuenta
// también reservas en vuelo y accesos válidos: no se puede eludir en paralelo.
// F0.09a usa una sola instancia; varios procesos necesitarán un puerto común.
type Limitador struct {
	mu           sync.Mutex
	politica     dominio.PoliticaIntentos
	intentos     map[[32]byte]dominio.Intentos
	peticiones   map[[32]byte]dominio.Intentos
	politicaHTTP dominio.PoliticaIntentos
}

func NuevoLimitador(p dominio.PoliticaIntentos, http ...dominio.PoliticaIntentos) (*Limitador, error) {
	if err := p.Validar(); err != nil {
		return nil, err
	}
	politicaHTTP := p
	if len(http) > 1 {
		return nil, dominio.ErrConfiguracion
	}
	if len(http) == 1 {
		if err := http[0].Validar(); err != nil {
			return nil, err
		}
		politicaHTTP = http[0]
	}
	return &Limitador{politica: p, intentos: make(map[[32]byte]dominio.Intentos), peticiones: make(map[[32]byte]dominio.Intentos), politicaHTTP: politicaHTTP}, nil
}

func (l *Limitador) reservar(cuenta string, ip dominio.IP, ahora time.Time) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	for clave, i := range l.intentos {
		if !i.Vigente(ahora, l.politica.Ventana) {
			delete(l.intentos, clave)
		}
	}
	u := sha256.Sum256([]byte("usuario:" + cuenta))
	r := sha256.Sum256(append([]byte("ip:"), ip[:]...))
	nuevas := 0
	if _, ok := l.intentos[u]; !ok {
		nuevas++
	}
	if _, ok := l.intentos[r]; !ok {
		nuevas++
	}
	if len(l.intentos)+nuevas > l.politica.MaxClaves {
		return dominio.ErrIntentosLimitados
	}
	usuario, err := l.intentos[u].Reservar(ahora, l.politica.Ventana, l.politica.PorUsuario)
	if err != nil {
		return err
	}
	ipIntentos, err := l.intentos[r].Reservar(ahora, l.politica.Ventana, l.politica.PorIP)
	// Aunque el usuario ya esté limitado, sus peticiones no llegan al directorio;
	// una IP limitada tampoco consume la reserva de usuario.
	if err != nil {
		return err
	}
	l.intentos[u], l.intentos[r] = usuario, ipIntentos
	return nil
}

// liberar devuelve la reserva de una entrada que AD ha aceptado: sólo los
// fallos consumen el cupo, así que entrar varias veces no bloquea a nadie.
func (l *Limitador) liberar(cuenta string, ip dominio.IP) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, clave := range [][32]byte{sha256.Sum256([]byte("usuario:" + cuenta)), sha256.Sum256(append([]byte("ip:"), ip[:]...))} {
		if i, ok := l.intentos[clave]; ok && i.Cantidad > 0 {
			i.Cantidad--
			l.intentos[clave] = i
		}
	}
}

// La solicitud fuera de presupuesto se rechaza antes de abrir una transacción
// o considerarse un intento de sesión. El mapa también tiene tamaño limitado.
func (l *Limitador) admitirPeticion(ip dominio.IP, ahora time.Time) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	p := l.politicaHTTP
	for clave, i := range l.peticiones {
		if !i.Vigente(ahora, p.Ventana) {
			delete(l.peticiones, clave)
		}
	}
	clave := sha256.Sum256(ip[:])
	if _, existe := l.peticiones[clave]; !existe && len(l.peticiones) >= p.MaxClaves {
		return dominio.ErrIntentosLimitados
	}
	nuevo, err := l.peticiones[clave].Reservar(ahora, p.Ventana, p.PorIP)
	if err != nil {
		return err
	}
	l.peticiones[clave] = nuevo
	return nil
}
