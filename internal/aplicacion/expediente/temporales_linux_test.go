package expediente

import (
	"context"
	"errors"
	"os"
	"syscall"
	"testing"

	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

// Inotify detecta incluso un fichero creado y borrado antes de la respuesta.
// No basta con comprobar que el directorio queda vacío al terminar.
func TestNoSeCreanTemporalesEnExitoErroresYCancelacion(t *testing.T) {
	for _, n := range []string{"docx", "pdf", "antivirus_caido", "infectado", "macro", "cancelada"} {
		t.Run(n, func(t *testing.T) {
			s, e, a, b := preparar(t)
			if n == "pdf" {
				var err error
				b, err = os.ReadFile("../../adaptadores/anexos/pdf/testdata/anexo3.pdf")
				if err != nil {
					t.Fatal(err)
				}
			}
			temporal := t.TempDir()
			t.Setenv("TMPDIR", temporal)
			fd, err := syscall.InotifyInit1(syscall.IN_NONBLOCK | syscall.IN_CLOEXEC)
			if err != nil {
				t.Fatal(err)
			}
			defer syscall.Close(fd)
			if _, err := syscall.InotifyAddWatch(fd, temporal, syscall.IN_CREATE|syscall.IN_MOVED_TO); err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			nombre := "Derivacion.docx"
			switch n {
			case "pdf":
				nombre = "Derivacion.pdf"
			case "antivirus_caido":
				e.antivirus, e.falloAV = plataforma.AntivirusNoDisponible, errors.New("antivirus.no_disponible")
			case "infectado":
				e.antivirus = plataforma.AntivirusInfectado
			case "macro":
				nombre = "Derivacion.docm"
			case "cancelada":
				var cancelar context.CancelFunc
				ctx, cancelar = context.WithCancel(ctx)
				cancelar()
			}
			_, fallo := s.Subir(ctx, a, entrada.Subida{Nombre: nombre, Contenido: b})
			siExito := n == "docx" || n == "pdf" || n == "antivirus_caido"
			if (fallo == nil) != siExito {
				t.Fatal(fallo)
			}
			var eventos [4096]byte
			if n, err := syscall.Read(fd, eventos[:]); n > 0 || (err != nil && !errors.Is(err, syscall.EAGAIN)) {
				t.Fatal("se creó un temporal", n, err)
			}
		})
	}
}
