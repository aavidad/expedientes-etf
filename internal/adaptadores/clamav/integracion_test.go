package clamav

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

// ETF_PRUEBA_CLAMAV_DIRECCION apunta al clamd local de compose. La prueba
// es omitible sin servicio configurado; con dirección explícita debe pasar.
func TestIntegracionClamd(t *testing.T) {
	direccion := os.Getenv("ETF_PRUEBA_CLAMAV_DIRECCION")
	if direccion == "" {
		t.Skip("clamd no configurado: ETF_PRUEBA_CLAMAV_DIRECCION")
	}
	a := nuevo(t, direccion, 10*time.Second)
	// Firma de prueba oficial, sin contenido ejecutable ni datos personales.
	eicar := "X5O!P%@AP[4\\PZX54(P^)7CC)7}$" + "EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*"
	var comprimido bytes.Buffer
	archivo := zip.NewWriter(&comprimido)
	f, err := archivo.Create("firma.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte(eicar)); err != nil {
		t.Fatal(err)
	}
	if err := archivo.Close(); err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		nombre, contenido string
		estado            plataforma.EstadoAntivirus
	}{
		{"limpio", `{"codigo":"es","version":"1.0.0"}`, plataforma.AntivirusLimpio},
		{"EICAR", eicar, plataforma.AntivirusInfectado},
		{"EICAR_en_ZIP", comprimido.String(), plataforma.AntivirusInfectado},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			estado, err := a.Revisar(context.Background(), []byte(caso.contenido))
			if err != nil || estado != caso.estado {
				t.Fatalf("estado=%v error=%v", estado, err)
			}
		})
	}
}
