package postgres

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

func entornoPrueba(valores map[string]string) func(string) string {
	return func(clave string) string { return valores[clave] }
}

func TestConfiguracionNoHeredaCredencialesNiOpcionesLibpq(t *testing.T) {
	t.Setenv("PGUSER", "etf_migrador")
	t.Setenv("PGPASSWORD", "secreto-ajeno")
	t.Setenv("PGDATABASE", "otra-base")
	t.Setenv("PGHOST", "otro-servidor")
	t.Setenv("PGPORT", "15432")
	t.Setenv("PGOPTIONS", "-c role=etf_migrador -c app.perfiles=coordinacion")
	t.Setenv("PGSERVICE", "")
	t.Setenv("PGSSLMODE", "verify-full")
	t.Setenv("PGTARGETSESSIONATTRS", "standby")
	t.Setenv("PGSSLNEGOTIATION", "direct")
	t.Setenv("PGSSLSNI", "0")
	cfg, err := configuracion(entornoPrueba(map[string]string{"ETF_BD_CONTRASENA": "clave-con@/'?&"}))
	if err != nil {
		t.Fatal(err)
	}
	c := cfg.ConnConfig
	if c.User != "etf_app" || c.Password != "clave-con@/'?&" || c.Database != "etf" || c.Host != "127.0.0.1" || c.Port != 5432 || c.TLSConfig != nil || len(c.Fallbacks) != 0 || c.ValidateConnect != nil || c.SSLNegotiation != "postgres" {
		t.Fatal("configuración heredada o alterada")
	}
	if len(c.RuntimeParams) != 1 || c.RuntimeParams["application_name"] != "etf" || cfg.MaxConns != 4 || cfg.MinConns != 0 || cfg.MinIdleConns != 0 {
		t.Fatal("opciones ajenas o conexiones anticipadas")
	}
}

func TestConfiguracionRechazaRolesAjenosYValoresInvalidos(t *testing.T) {
	for _, par := range [][2]string{
		{"ETF_BD_USUARIO", "etf_migrador"}, {"ETF_BD_USUARIO", "etf_propietario"},
		{"ETF_BD_USUARIO", "etf_copia"}, {"ETF_BD_CONTRASENA", ""},
		{"ETF_BD_PUERTO", "0"}, {"ETF_BD_PUERTO", "65536"}, {"ETF_BD_PUERTO", "5432 role=etf_migrador"},
		{"ETF_BD_MAX_CONEXIONES", "0"}, {"ETF_BD_MAX_CONEXIONES", "65"},
		{"ETF_BD_HOST", "localhost,otro"}, {"ETF_BD_SSLMODE", "prefer"},
	} {
		t.Run(par[0]+par[1], func(t *testing.T) {
			valores := map[string]string{"ETF_BD_CONTRASENA": "secreto-inventado"}
			valores[par[0]] = par[1]
			base, err := Nueva(context.Background(), entornoPrueba(valores))
			defer base.Cerrar()
			if !errors.Is(err, ErrConfiguracion) || !errors.Is(base.Comprobar(context.Background()), ErrNoDisponible) || strings.Contains(err.Error(), "secreto-inventado") {
				t.Fatal("configuración inválida aceptada o expuesta")
			}
		})
	}
}

func TestBaseAusenteNoImpideConstruirAdaptador(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_, puerto, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	base, err := Nueva(context.Background(), entornoPrueba(map[string]string{
		"ETF_BD_HOST": "127.0.0.1", "ETF_BD_PUERTO": puerto, "ETF_BD_CONTRASENA": "clave-inventada",
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer base.Cerrar()
	if !errors.Is(base.Comprobar(context.Background()), ErrNoDisponible) {
		t.Fatal("salud incorrecta")
	}
	ejecutada := false
	err = base.Ejecutar(context.Background(), plataforma.Identidad{}, func(context.Context) error { ejecutada = true; return nil })
	if !errors.Is(err, ErrNoDisponible) || ejecutada {
		t.Fatal("se ejecutó sin base")
	}
}

func TestIdentidadIncompletaOConSeparadoresSeRechaza(t *testing.T) {
	usuario := "a66cfd72-7f02-4b1e-b934-5da9d245bc33"
	for _, identidad := range []plataforma.Identidad{
		{UsuarioID: "valor-invalido"}, {Perfiles: []string{"coordinacion"}},
		{UsuarioID: usuario, EquipoID: "{otro-equipo}"},
		{UsuarioID: usuario, Perfiles: []string{"equipo,coordinacion"}},
		{UsuarioID: usuario, Perfiles: []string{""}},
	} {
		if _, _, _, err := contextoRLS(identidad); !errors.Is(err, ErrIdentidad) {
			t.Fatal("identidad aceptada")
		}
	}
	if _, err := DesdeContexto(context.Background()); !errors.Is(err, ErrTransaccion) {
		t.Fatal("ejecutor fuera de transacción")
	}
}
