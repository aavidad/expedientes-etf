// Package clamav implementa el protocolo clamd INSTREAM por la red interna.
package clamav

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

var (
	ErrConfiguracion = errors.New("antivirus.configuracion_invalida")
	ErrNoDisponible  = errors.New("antivirus.no_disponible")
	ErrTamano        = errors.New("antivirus.tamano_excedido")
)

type Configuracion struct {
	Direccion    string
	TiempoLimite time.Duration
	MaximoBytes  int
}

type Antivirus struct{ cfg Configuracion }

var _ plataforma.Antivirus = (*Antivirus)(nil)

func Nuevo(cfg Configuracion) (*Antivirus, error) {
	host, puerto, err := net.SplitHostPort(cfg.Direccion)
	numeroPuerto, errPuerto := strconv.Atoi(puerto)
	if err != nil || host == "" || errPuerto != nil || numeroPuerto < 1 || numeroPuerto > 65535 || cfg.TiempoLimite <= 0 || cfg.MaximoBytes <= 0 || cfg.MaximoBytes > 100<<20 {
		return nil, ErrConfiguracion
	}
	return &Antivirus{cfg: cfg}, nil
}

func (a *Antivirus) Revisar(ctx context.Context, contenido []byte) (plataforma.EstadoAntivirus, error) {
	if a == nil {
		return plataforma.AntivirusNoDisponible, ErrNoDisponible
	}
	if len(contenido) > a.cfg.MaximoBytes {
		return plataforma.AntivirusNoDisponible, ErrTamano
	}
	ctx, cancelar := context.WithTimeout(ctx, a.cfg.TiempoLimite)
	defer cancelar()
	conexion, err := (&net.Dialer{}).DialContext(ctx, "tcp", a.cfg.Direccion)
	if err != nil {
		return plataforma.AntivirusNoDisponible, ErrNoDisponible
	}
	defer conexion.Close()
	// La cancelación corta también una escritura bloqueada, no sólo la espera
	// de respuesta. El plazo abarca conexión, envío y análisis completos.
	detener := context.AfterFunc(ctx, func() { _ = conexion.Close() })
	defer detener()
	plazo, _ := ctx.Deadline()
	if err := conexion.SetDeadline(plazo); err != nil {
		return plataforma.AntivirusNoDisponible, ErrNoDisponible
	}
	if err := escribir(conexion, []byte("zINSTREAM\x00")); err != nil {
		return plataforma.AntivirusNoDisponible, ErrNoDisponible
	}
	var cabecera [4]byte
	for len(contenido) > 0 {
		tamano := min(len(contenido), 32<<10)
		binary.BigEndian.PutUint32(cabecera[:], uint32(tamano))
		if err := escribir(conexion, cabecera[:]); err != nil {
			return plataforma.AntivirusNoDisponible, ErrNoDisponible
		}
		if err := escribir(conexion, contenido[:tamano]); err != nil {
			return plataforma.AntivirusNoDisponible, ErrNoDisponible
		}
		contenido = contenido[tamano:]
	}
	binary.BigEndian.PutUint32(cabecera[:], 0)
	if err := escribir(conexion, cabecera[:]); err != nil {
		return plataforma.AntivirusNoDisponible, ErrNoDisponible
	}
	// clamd termina la respuesta con NUL. Nunca se publica el diagnóstico ni
	// el nombre de la firma; una respuesta desconocida no acredita limpieza.
	respuesta, err := bufio.NewReader(io.LimitReader(conexion, 4096)).ReadString(0)
	if err != nil || ctx.Err() != nil {
		return plataforma.AntivirusNoDisponible, ErrNoDisponible
	}
	respuesta = strings.TrimSuffix(respuesta, "\x00")
	if respuesta == "stream: OK" {
		return plataforma.AntivirusLimpio, nil
	}
	if strings.HasPrefix(respuesta, "stream: ") && strings.HasSuffix(respuesta, " FOUND") && len(respuesta) > len("stream:  FOUND") {
		return plataforma.AntivirusInfectado, nil
	}
	return plataforma.AntivirusNoDisponible, ErrNoDisponible
}

func escribir(w io.Writer, b []byte) error {
	for len(b) > 0 {
		n, err := w.Write(b)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		b = b[n:]
	}
	return nil
}
