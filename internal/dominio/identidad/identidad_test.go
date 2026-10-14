package identidad

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"
)

func TestSesionCaducidadYHuella(t *testing.T) {
	ahora := time.Date(2026, 10, 13, 8, 0, 0, 0, time.UTC)
	p := PoliticaSesion{Inactividad: 30 * time.Minute, Maxima: 8 * time.Hour}
	id := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	huella, err := HuellaIdentificador(id)
	if err != nil {
		t.Fatal(err)
	}
	s, err := NuevaSesion(huella, "usuario-1", ahora, p)
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		nombre string
		fecha  time.Time
		error  error
	}{
		{"antes_inactividad", ahora.Add(29 * time.Minute), nil},
		{"limite_inactividad", ahora.Add(30 * time.Minute), ErrSesionNoValida},
		{"reloj_retrocede", ahora.Add(-time.Second), ErrSesionNoValida},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			copia := s
			if err := copia.Tocar(caso.fecha, p); !errors.Is(err, caso.error) {
				t.Fatalf("error: %v", err)
			}
			if copia.CaducaEn != ahora.Add(8*time.Hour) {
				t.Fatal("caducidad máxima desplazada")
			}
		})
	}
	for minuto := 20; minuto < 480; minuto += 20 {
		if err := s.Tocar(ahora.Add(time.Duration(minuto)*time.Minute), p); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Tocar(ahora.Add(8*time.Hour), p); !errors.Is(err, ErrSesionNoValida) {
		t.Fatal(err)
	}
	for _, invalido := range []string{"", "sesion", id + "=", id[:42] + "B"} {
		if _, err := HuellaIdentificador(invalido); !errors.Is(err, ErrSesionNoValida) {
			t.Fatalf("aceptado: %q", invalido)
		}
	}
}

func TestUsuarioDenegacionPorDefecto(t *testing.T) {
	u := Usuario{ID: "usuario-1", IdentidadDirectorio: IdentidadDirectorio{GUID: GUID{1}}, Estado: Activo}
	if estado, err := u.EstadoAcceso(); err != nil || estado != PendienteAlta {
		t.Fatalf("%s %v", estado, err)
	}
	u.Perfiles = []Perfil{Consulta, Administracion}
	if estado, err := u.EstadoAcceso(); err != nil || estado != Activo {
		t.Fatalf("%s %v", estado, err)
	}
	for _, estado := range []Estado{Baja, "desconocido", PendienteAlta} {
		u.Estado = estado
		if _, err := u.EstadoAcceso(); !errors.Is(err, ErrUsuarioNoDisponible) {
			t.Fatal(err)
		}
	}
	u.Estado = Activo
	for _, perfil := range []Perfil{"desconocido", ProfesionalETF} {
		u.Perfiles = []Perfil{perfil}
		if _, err := u.EstadoAcceso(); !errors.Is(err, ErrUsuarioNoDisponible) {
			t.Fatal(err)
		}
	}
	u.EquipoID = "equipo-4"
	if _, err := u.EstadoAcceso(); err != nil {
		t.Fatal(err)
	}
}

func TestPoliticasYCuenta(t *testing.T) {
	for _, p := range []PoliticaSesion{{}, {time.Minute, 0}, {time.Hour, time.Minute}} {
		if !errors.Is(p.Validar(), ErrConfiguracion) {
			t.Fatal("política aceptada", p)
		}
	}
	for _, p := range []PoliticaIntentos{{}, {time.Minute, 1, 1, 1}, {-1, 1, 1, 2}, {time.Minute, 0, 1, 2}} {
		if !errors.Is(p.Validar(), ErrConfiguracion) {
			t.Fatal("política aceptada", p)
		}
	}
	cuenta, err := NormalizarCuenta("  INES.MORENO  ")
	if err != nil || cuenta != "ines.moreno" {
		t.Fatalf("%q %v", cuenta, err)
	}
	for _, c := range []string{"", "ines@etf.interno", `ETF\ines`, "*)(uid=*)", "inés"} {
		if _, err := NormalizarCuenta(c); !errors.Is(err, ErrCredencialesInvalidas) {
			t.Fatalf("aceptada: %q", c)
		}
	}
}
