package identidad

import (
	"errors"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"testing"
	"time"
)

func TestLimitePeticionesSeparaIPsYReabreVentana(t *testing.T) {
	politica := dominio.PoliticaIntentos{Ventana: time.Minute, PorUsuario: 5, PorIP: 3, MaxClaves: 2}
	l, err := NuevoLimitador(politica)
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC)
	var ip, otra, tercera dominio.IP
	ip[15] = 1
	otra[15] = 2
	tercera[15] = 3
	for n := 0; n < 3; n++ {
		if err := l.admitirPeticion(ip, ahora); err != nil {
			t.Fatal(err)
		}
	}
	if err := l.admitirPeticion(ip, ahora); !errors.Is(err, dominio.ErrIntentosLimitados) {
		t.Fatal("presupuesto superado", err)
	}
	if err := l.admitirPeticion(otra, ahora); err != nil {
		t.Fatal("IP independiente limitada", err)
	}
	if err := l.admitirPeticion(tercera, ahora); !errors.Is(err, dominio.ErrIntentosLimitados) {
		t.Fatal("mapa sin límite", err)
	}
	if err := l.admitirPeticion(ip, ahora.Add(time.Minute)); err != nil {
		t.Fatal("ventana no reiniciada", err)
	}
}
