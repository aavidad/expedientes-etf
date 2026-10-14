package clamav

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

func servidor(t *testing.T, atender func(net.Conn)) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	terminado := make(chan struct{})
	go func() {
		defer close(terminado)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		atender(conn)
	}()
	t.Cleanup(func() { _ = listener.Close(); <-terminado })
	return listener.Addr().String()
}

func recibir(t *testing.T, conn net.Conn) []byte {
	t.Helper()
	_ = conn.SetDeadline(time.Now().Add(time.Second))
	comando := make([]byte, len("zINSTREAM\x00"))
	if _, err := io.ReadFull(conn, comando); err != nil {
		t.Error(err)
		return nil
	}
	if string(comando) != "zINSTREAM\x00" {
		t.Errorf("comando: %q", comando)
		return nil
	}
	var b bytes.Buffer
	var cabecera [4]byte
	for {
		if _, err := io.ReadFull(conn, cabecera[:]); err != nil {
			t.Error(err)
			return nil
		}
		n := binary.BigEndian.Uint32(cabecera[:])
		if n == 0 {
			return b.Bytes()
		}
		if n > 32<<10 {
			t.Errorf("fragmento: %d", n)
			return nil
		}
		if _, err := io.CopyN(&b, conn, int64(n)); err != nil {
			t.Error(err)
			return nil
		}
	}
}

func nuevo(t *testing.T, direccion string, plazo time.Duration) *Antivirus {
	t.Helper()
	a, err := Nuevo(Configuracion{Direccion: direccion, TiempoLimite: plazo, MaximoBytes: 20 << 20})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestINSTREAM(t *testing.T) {
	for _, caso := range []struct {
		nombre, respuesta string
		estado            plataforma.EstadoAntivirus
		fallo             bool
	}{
		{"limpio", "stream: OK\x00", plataforma.AntivirusLimpio, false},
		{"infectado", "stream: Eicar-Signature FOUND\x00", plataforma.AntivirusInfectado, false},
		{"error", "stream: size limit exceeded ERROR\x00", plataforma.AntivirusNoDisponible, true},
		{"desconocido", "OK\x00", plataforma.AntivirusNoDisponible, true},
		{"sin_terminador", "stream: OK", plataforma.AntivirusNoDisponible, true},
		{"respuesta_grande", strings.Repeat("a", 4096) + "\x00", plataforma.AntivirusNoDisponible, true},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			contenido := bytes.Repeat([]byte("configuracion"), 6000)
			direccion := servidor(t, func(conn net.Conn) {
				if b := recibir(t, conn); !bytes.Equal(b, contenido) {
					t.Error("contenido distinto")
				}
				_, _ = io.WriteString(conn, caso.respuesta)
			})
			estado, err := nuevo(t, direccion, time.Second).Revisar(context.Background(), contenido)
			if estado != caso.estado || (err != nil) != caso.fallo {
				t.Fatalf("estado=%v error=%v", estado, err)
			}
		})
	}
}

func TestCaida(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	direccion := l.Addr().String()
	_ = l.Close()
	estado, err := nuevo(t, direccion, time.Second).Revisar(context.Background(), []byte("datos"))
	if estado != plataforma.AntivirusNoDisponible || !errors.Is(err, ErrNoDisponible) {
		t.Fatalf("%v %v", estado, err)
	}
}

func TestPlazoYCancelacion(t *testing.T) {
	for _, cancelarAntes := range []bool{false, true} {
		t.Run(map[bool]string{false: "plazo", true: "cancelacion"}[cancelarAntes], func(t *testing.T) {
			leido := make(chan struct{})
			direccion := servidor(t, func(conn net.Conn) {
				_ = recibir(t, conn)
				close(leido)
				// Espera el cierre del cliente; nunca responde al análisis.
				_, _ = io.Copy(io.Discard, conn)
			})
			ctx, cancelar := context.WithCancel(context.Background())
			defer cancelar()
			if cancelarAntes {
				go func() { <-leido; cancelar() }()
			}
			inicio := time.Now()
			estado, err := nuevo(t, direccion, 100*time.Millisecond).Revisar(ctx, nil)
			if estado != plataforma.AntivirusNoDisponible || !errors.Is(err, ErrNoDisponible) {
				t.Fatalf("%v %v", estado, err)
			}
			if time.Since(inicio) > time.Second {
				t.Fatal("no respetó el plazo")
			}
		})
	}
}

func TestLimiteEntrada(t *testing.T) {
	a := nuevo(t, "127.0.0.1:1", time.Second)
	estado, err := a.Revisar(context.Background(), make([]byte, (20<<20)+1))
	if estado != plataforma.AntivirusNoDisponible || !errors.Is(err, ErrTamano) {
		t.Fatalf("%v %v", estado, err)
	}
}

func TestCancelacionDuranteEnvio(t *testing.T) {
	conectado := make(chan struct{})
	liberar := make(chan struct{})
	direccion := servidor(t, func(conn net.Conn) {
		close(conectado)
		// No lee: el envío grande agota el búfer TCP del cliente.
		<-liberar
	})
	defer close(liberar)
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	go func() { <-conectado; cancelar() }()
	inicio := time.Now()
	estado, err := nuevo(t, direccion, time.Second).Revisar(ctx, make([]byte, 20<<20))
	if estado != plataforma.AntivirusNoDisponible || !errors.Is(err, ErrNoDisponible) || time.Since(inicio) > time.Second {
		t.Fatalf("estado=%v error=%v tiempo=%v", estado, err, time.Since(inicio))
	}
}

func TestConfiguracion(t *testing.T) {
	for _, cfg := range []Configuracion{
		{}, {"localhost", time.Second, 20 << 20}, {":3310", time.Second, 20 << 20},
		{"localhost:3310", 0, 20 << 20}, {"localhost:3310", time.Second, 0},
		{"localhost:3310", time.Second, (100 << 20) + 1},
		{"localhost:0", time.Second, 20 << 20}, {"localhost:65536", time.Second, 20 << 20},
		{"localhost:clamd", time.Second, 20 << 20},
	} {
		if _, err := Nuevo(cfg); !errors.Is(err, ErrConfiguracion) {
			t.Fatalf("%+v: %v", cfg, err)
		}
	}
}
