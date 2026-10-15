package main

import (
	"testing"
	"time"
)

func TestPoliticaSesionConfigurada(t *testing.T) {
	for _, caso := range []struct {
		valores             map[string]string
		inactividad, maxima time.Duration
		falla               bool
	}{
		{map[string]string{}, 30 * time.Minute, 8 * time.Hour, false},
		{map[string]string{"ETF_SESION_INACTIVIDAD": "15m", "ETF_SESION_MAXIMA": "4h"}, 15 * time.Minute, 4 * time.Hour, false},
		{map[string]string{"ETF_SESION_MAXIMA": "0"}, 0, 0, true},
		{map[string]string{"ETF_SESION_MAXIMA": "500h"}, 0, 0, true},
		{map[string]string{"ETF_SESION_INACTIVIDAD": "2m"}, 0, 0, true},
		{map[string]string{"ETF_SESION_INACTIVIDAD": "9h"}, 0, 0, true},
	} {
		p, err := politicaSesion(func(k string) string { return caso.valores[k] })
		if (err != nil) != caso.falla || (!caso.falla && (p.Inactividad != caso.inactividad || p.Maxima != caso.maxima)) {
			t.Fatal(caso, p, err)
		}
	}
}
