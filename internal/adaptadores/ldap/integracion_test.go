//go:build ldap_integracion

package ldap

import (
	"context"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
)

// Se ejecuta expresamente con:
// go test -tags ldap_integracion -count=1 -v ./internal/adaptadores/ldap
// Sin Docker o si el servidor no arranca, falla: nunca da una omisión por válida.
func TestIntegracionLDAPS(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	docker := func(args ...string) string {
		t.Helper()
		out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
		if err != nil {
			t.Fatalf("docker %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	imagen := "etf-ldap-pruebas:f0.09a"
	docker("build", "--quiet", "-t", imagen, "testdata")
	ruta := t.TempDir()
	if err := os.Chmod(ruta, 0755); err != nil {
		t.Fatal(err)
	}
	var azar [24]byte
	if _, err := rand.Read(azar[:]); err != nil {
		t.Fatal(err)
	}
	ipContenedor := fmt.Sprintf("10.254.%d.2", azar[0])
	raices, certificado, clave := certificadoLocal(t, net.ParseIP(ipContenedor))
	contrasena := base64.RawURLEncoding.EncodeToString(azar[:])
	archivos := map[string][]byte{"servidor.pem": certificado, "servidor.key": clave}
	for _, nombre := range []string{"slapd.conf", "identidad.schema"} {
		contenido, err := os.ReadFile(filepath.Join("testdata", nombre))
		if err != nil {
			t.Fatal(err)
		}
		archivos[nombre] = contenido
	}
	archivos["datos.ldif"] = []byte(fmt.Sprintf(`dn: dc=etf,dc=interno
objectClass: top
objectClass: domain
dc: etf

dn: ou=personas,dc=etf,dc=interno
objectClass: organizationalUnit
ou: personas

dn: uid=ines.moreno,ou=personas,dc=etf,dc=interno
objectClass: inetOrgPerson
objectClass: identidadETF
uid: ines.moreno
cn: Inés Moreno Castillo
sn: Moreno Castillo
displayName: Inés Moreno Castillo
userPassword: %s
objectGUID:: MyIRAFVEd2aImaq7zN3u/w==

dn: uid=pablo.ruiz,ou=personas,dc=etf,dc=interno
objectClass: inetOrgPerson
uid: pablo.ruiz
cn: Pablo Ruiz Vílchez
sn: Ruiz Vílchez
userPassword: %s

`, contrasena, contrasena))
	for nombre, contenido := range archivos {
		if err := os.WriteFile(filepath.Join(ruta, nombre), contenido, 0644); err != nil {
			t.Fatal(err)
		}
	}
	// Docker monta sólo esta carpeta desechable y de sólo lectura; el servidor
	// no tiene red exterior, privilegios ni acceso al resto del repositorio.
	// Subred explícita: los pools predeterminados pueden estar agotados por
	// otros proyectos del equipo. No se modifica ni reutiliza ninguna red ajena.
	red := docker("network", "create", "--internal", "--subnet", fmt.Sprintf("10.254.%d.0/24", azar[0]),
		"etf-ldap-"+strings.ToLower(base64.RawURLEncoding.EncodeToString(azar[:6])))
	t.Cleanup(func() { _ = exec.Command("docker", "network", "rm", red).Run() })
	id := docker("run", "--detach", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges",
		"--memory=128m", "--cpus=1", "--pids-limit=32", "--network", red, "--ip", ipContenedor,
		"--tmpfs", "/tmp:rw,nosuid,nodev,size=32m,uid=65534,gid=65534",
		"--mount", "type=bind,src="+ruta+",dst=/fixture,readonly",
		imagen)
	t.Cleanup(func() { _ = exec.Command("docker", "rm", "-f", id).Run() })
	// El host accede sólo a la IP de este contenedor en su red aislada. No se
	// publica ningún puerto, ni se conecta a directorios o redes corporativas.
	direccion := net.JoinHostPort(ipContenedor, "1636")
	c := Configuracion{URL: "ldaps://" + direccion, BaseDN: "ou=personas,dc=etf,dc=interno", FormatoBind: "uid={usuario},ou=personas,dc=etf,dc=interno",
		BindDN: true, AtributoCuenta: "uid", Autoridades: raices, TiempoLimite: time.Second}
	d, err := Nuevo(c)
	if err != nil {
		t.Fatal(err)
	}
	limite := time.Now().Add(15 * time.Second)
	for {
		_, err = d.Autenticar(ctx, "ines.moreno", contrasena)
		if err == nil {
			break
		}
		if time.Now().After(limite) {
			t.Fatalf("LDAPS no arrancó: %v\n%s", err, docker("logs", id))
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Run("bind_lectura_guid_login", func(t *testing.T) {
		i, err := d.Autenticar(ctx, "INES.MORENO", contrasena)
		esperado := dominio.GUID{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
		if err != nil || i.GUID != esperado || i.CuentaAD != "ines.moreno" || i.NombreMostrado != "Inés Moreno Castillo" {
			t.Fatalf("identidad inesperada: %v", err)
		}
	})
	for _, caso := range []struct{ nombre, cuenta, clave string }{
		{"vacia", "ines.moreno", ""}, {"incorrecta", "ines.moreno", "otra-clave-local"},
		{"inexistente", "elena.salas", contrasena}, {"sin_guid", "pablo.ruiz", contrasena}, {"inyeccion", "*)(uid=*)", contrasena},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := d.Autenticar(ctx, caso.cuenta, caso.clave); !errors.Is(err, dominio.ErrCredencialesInvalidas) {
				t.Fatal(err)
			}
		})
	}
	t.Run("certificado_no_confiable", func(t *testing.T) {
		copia := c
		copia.Autoridades = x509.NewCertPool()
		otras, _, _ := certificadoLocal(t)
		copia.Autoridades = otras
		otro, err := Nuevo(copia)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := otro.Autenticar(ctx, "ines.moreno", contrasena); !errors.Is(err, dominio.ErrDirectorioNoDisponible) {
			t.Fatal(err)
		}
	})
	t.Run("directorio_caido", func(t *testing.T) {
		docker("stop", "--time", "1", id)
		if _, err := d.Autenticar(ctx, "ines.moreno", contrasena); !errors.Is(err, dominio.ErrDirectorioNoDisponible) {
			t.Fatal(err)
		}
	})
}
