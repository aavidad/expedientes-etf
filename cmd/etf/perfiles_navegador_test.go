//go:build navegador_integracion

package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math/big"
	"net"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros"
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/ldap"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	catalogoPostgres "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	paquetePostgres "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/paquete"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo"
	aplicacion "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/paquete"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"testing"
)

func TestNavegadorPerfiles(t *testing.T) {
	b, ctx := baseIntegracion(t)
	directorio, _, claveDirectorio := directorioNavegador(t)
	usuarios, err := postgres.NuevosUsuariosConAdministrador(b, "marta.lopez")
	if err != nil {
		t.Fatal(err)
	}
	identidad, err := directorio.Autenticar(ctx, "marta.lopez", claveDirectorio)
	if err != nil {
		t.Fatal(err)
	}
	admin, err := usuarios.RegistrarAcceso(ctx, identidad, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	administracion := aplicacion.NuevaAdministracionUsuarios(&postgres.Perfiles{}, b, postgres.NuevoRegistroAuditoria(b), relojNavegador{})
	if !admin.Permite(dominio.Administracion) {
		// Reutiliza un administrador sintético ya concedido por el recorrido de
		// integración. La cuenta nueva recibe sus permisos por el caso de uso real.
		id := sqlPerfiles(t, `SELECT u.id FROM identidad.usuario u JOIN identidad.usuario_perfil p ON p.usuario_id=u.id WHERE u.estado='activo' AND p.perfil='administracion' AND p.desde<=clock_timestamp() AND (p.hasta IS NULL OR p.hasta>clock_timestamp()) LIMIT 1;`)
		var respaldo dominio.Usuario
		err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: id}, func(ctx context.Context) error { var e error; respaldo, e = usuarios.Obtener(ctx, id); return e })
		if err != nil {
			t.Fatal(err)
		}
		lista, err := administracion.Listar(ctx, actorPerfiles(respaldo))
		if err != nil {
			t.Fatal(err)
		}
		var version int64
		for _, u := range lista {
			if u.ID == admin.ID {
				version = u.Version
			}
		}
		_, err = administracion.Cambiar(ctx, actorPerfiles(respaldo), admin.ID, dominio.CambioPerfiles{Version: version, Estado: dominio.Activo, Perfiles: []dominio.Concesion{{Perfil: dominio.Administracion, Desde: time.Now().UTC()}}})
		if err != nil {
			t.Fatal(err)
		}
	}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: admin.ID}, func(ctx context.Context) error { var e error; admin, e = usuarios.Obtener(ctx, admin.ID); return e })
	if err != nil {
		t.Fatal(err)
	}
	lista, err := administracion.Listar(ctx, actorPerfiles(admin))
	if err != nil {
		t.Fatal(err)
	}
	var versionAdmin int64
	for _, u := range lista {
		if u.ID == admin.ID {
			versionAdmin = u.Version
		}
	}
	// El segundo administrador concede la combinación; nunca el destinatario.
	var guidRespaldo dominio.GUID
	_, _ = rand.Read(guidRespaldo[:])
	respaldo, fallo := usuarios.RegistrarAcceso(ctx, dominio.IdentidadDirectorio{GUID: guidRespaldo, CuentaAD: "sergio.campos", NombreMostrado: "Sergio Campos Martín"}, time.Now().UTC())
	if fallo != nil {
		t.Fatal(fallo)
	}
	pendientes, fallo := administracion.Listar(ctx, actorPerfiles(admin))
	if fallo != nil {
		t.Fatal(fallo)
	}
	var versionRespaldo int64
	for _, u := range pendientes {
		if u.ID == respaldo.ID {
			versionRespaldo = u.Version
		}
	}
	if _, fallo = administracion.Cambiar(ctx, actorPerfiles(admin), respaldo.ID, dominio.CambioPerfiles{Version: versionRespaldo, Estado: dominio.Activo, Perfiles: []dominio.Concesion{{Perfil: dominio.Administracion, Desde: time.Now().UTC()}}}); fallo != nil {
		t.Fatal(fallo)
	}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: respaldo.ID}, func(ctx context.Context) error {
		var e error
		respaldo, e = usuarios.Obtener(ctx, respaldo.ID)
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = administracion.Cambiar(ctx, actorPerfiles(respaldo), admin.ID, dominio.CambioPerfiles{Version: versionAdmin, Estado: dominio.Activo, Perfiles: []dominio.Concesion{{Perfil: dominio.Coordinacion, Desde: time.Now().UTC()}, {Perfil: dominio.Administracion, Desde: time.Now().UTC()}}})
	if err != nil {
		t.Fatal(err)
	}
	limitador, _ := aplicacion.NuevoLimitador(dominio.PoliticaIntentos{Ventana: time.Minute, PorUsuario: 100, PorIP: 200, MaxClaves: 1000})
	servicio, err := aplicacion.NuevoServicio(directorio, usuarios, postgres.NuevasSesiones(b), relojNavegador{}, azarNavegador{}, dominio.PoliticaSesion{Inactividad: 30 * time.Minute, Maxima: time.Hour}, limitador, b)
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	disco := ficheros.NuevosPaquetes(filepath.Join(root, "idiomas"), filepath.Join(root, "temas"))
	servidor := httptest.NewUnstartedServer(nil)
	servidor.Config.ErrorLog = log.New(io.Discard, "", 0)
	servidor.StartTLS()
	defer servidor.Close()
	clave := make([]byte, 32)
	_, _ = rand.Read(clave)
	servidor.Config.Handler = adaptadorHTTP.Nuevo(paquete.NuevaConsulta(disco, disco), os.DirFS(filepath.Join(root, "web/dist")), "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), b.Comprobar, adaptadorHTTP.Opciones{
		SMI:       adaptadorHTTP.ConfiguracionSMI{Servicio: catalogo.NuevoSMI(&catalogoPostgres.Repositorio{}, b, postgres.NuevoRegistroAuditoriaTransaccional(b), time.Now)},
		Sesion:    adaptadorHTTP.ConfiguracionSesion{Servicio: servicio, Origen: servidor.URL, ClaveCSRF: clave},
		Usuarios:  adaptadorHTTP.ConfiguracionUsuarios{Servicio: administracion},
		Catalogos: adaptadorHTTP.ConfiguracionCatalogos{Consulta: catalogo.NuevaConsulta(&catalogoPostgres.Repositorio{}, b), Edicion: catalogo.NuevaEdicion(&catalogoPostgres.Repositorio{}, paquete.NuevaPublicacion(disco, &paquetePostgres.Referencias{}), b, postgres.NuevoRegistroAuditoriaTransaccional(b), time.Now)},
	})
	limite, cancelar := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelar()
	evidencia := os.Getenv("ETF_EVIDENCIA_RUTA")
	if evidencia == "" {
		evidencia = t.TempDir()
	}
	cmd := exec.CommandContext(limite, "node", filepath.Join(root, "web/tests/perfiles.navegador.mjs"))
	cmd.Env = append(os.Environ(), "ETF_PRUEBA_URL="+servidor.URL, "ETF_PRUEBA_CLAVE="+claveDirectorio, "ETF_EVIDENCIA_RUTA="+evidencia)
	salida, err := cmd.CombinedOutput()
	t.Log(string(salida))
	if err != nil {
		t.Fatal(err)
	}
}

type relojNavegador struct{}

func (relojNavegador) Ahora() time.Time { return time.Now().UTC() }

type azarNavegador struct{}

func (azarNavegador) Llenar(b []byte) error { _, err := rand.Read(b); return err }

func directorioNavegador(t *testing.T) (*ldap.Directorio, string, string) {
	t.Helper()
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	docker := func(args ...string) string {
		t.Helper()
		salida, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
		if err != nil {
			t.Fatalf("docker: %v %s", err, salida)
		}
		return strings.TrimSpace(string(salida))
	}
	var azar [24]byte
	_, _ = rand.Read(azar[:])
	nombre := fmt.Sprintf("etf-f009b-ldap-%x", azar[:6])
	ip := fmt.Sprintf("10.252.%d.2", azar[0])
	red := nombre + "-red"
	docker("network", "create", "--internal", "--subnet", fmt.Sprintf("10.252.%d.0/24", azar[0]), red)
	t.Cleanup(func() { _ = exec.Command("docker", "network", "rm", red).Run() })
	ruta := t.TempDir()
	if err := os.Chmod(ruta, 0755); err != nil {
		t.Fatal(err)
	}
	privada, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	cert := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "directorio.etf.interno"}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), IPAddresses: []net.IP{net.ParseIP(ip)}, IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, cert, cert, &privada.PublicKey, privada)
	if err != nil {
		t.Fatal(err)
	}
	certificado := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	contrasena := base64.RawURLEncoding.EncodeToString(azar[:])
	privadaDER, err := x509.MarshalPKCS8PrivateKey(privada)
	if err != nil {
		t.Fatal(err)
	}
	archivos := map[string][]byte{"servidor.pem": certificado, "servidor.key": pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privadaDER})}
	for _, nombre := range []string{"slapd.conf", "identidad.schema"} {
		contenido, err := os.ReadFile(filepath.Join("../../internal/adaptadores/ldap/testdata", nombre))
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

dn: uid=marta.lopez,ou=personas,dc=etf,dc=interno
objectClass: inetOrgPerson
objectClass: identidadETF
uid: marta.lopez
cn: Marta López Rivas
sn: López Rivas
displayName: Marta López Rivas
userPassword: %s
objectGUID:: MyIRAFVEd2aImaq7zN3u/g==
`, contrasena, contrasena))
	for nombre, contenido := range archivos {
		if err := os.WriteFile(filepath.Join(ruta, nombre), contenido, 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(filepath.Join(ruta, nombre), 0644); err != nil {
			t.Fatal(err)
		}
	}
	id := docker("run", "--detach", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--memory=128m", "--cpus=1", "--pids-limit=32", "--network", red, "--ip", ip, "--tmpfs", "/tmp:rw,nosuid,nodev,size=32m,uid=65534,gid=65534", "--mount", "type=bind,src="+ruta+",dst=/fixture,readonly", "etf-ldap-pruebas:f0.09a")
	t.Cleanup(func() { _ = exec.Command("docker", "rm", "-f", id).Run() })
	ca := x509.NewCertPool()
	ca.AppendCertsFromPEM(certificado)
	directorio, err := ldap.Nuevo(ldap.Configuracion{URL: "ldaps://" + net.JoinHostPort(ip, "1636"), BaseDN: "ou=personas,dc=etf,dc=interno", FormatoBind: "uid={usuario},ou=personas,dc=etf,dc=interno", BindDN: true, AtributoCuenta: "uid", Autoridades: ca, TiempoLimite: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	for inicio := time.Now(); ; time.Sleep(100 * time.Millisecond) {
		if _, err := directorio.Autenticar(ctx, "ines.moreno", contrasena); err == nil {
			break
		}
		if time.Since(inicio) > 10*time.Second {
			t.Fatalf("LDAPS no disponible; estado=%s", docker("inspect", "--format", "{{.State.Status}} {{.State.ExitCode}}", id))
		}
	}
	return directorio, id, contrasena
}

func sqlPerfiles(t *testing.T, s string) string {
	t.Helper()
	c := os.Getenv("ETF_POSTGRES_CONTENEDOR")
	if c == "" {
		t.Skip("ETF_POSTGRES_CONTENEDOR")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", c, "psql", "-X", "-qAt", "-U", "etf_instalador", "-d", "etf", "-v", "ON_ERROR_STOP=1")
	cmd.Stdin = strings.NewReader(s)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("SQL local: %v %s", err, out)
	}
	return strings.TrimSpace(string(out))
}
func actorPerfiles(u dominio.Usuario) entrada.Acceso {
	return entrada.Acceso{Usuario: u, Estado: dominio.Activo}
}

func baseIntegracion(t *testing.T) (*postgres.Base, context.Context) {
	t.Helper()
	if os.Getenv("ETF_POSTGRES_INTEGRACION") != "1" {
		t.Skip("ETF_POSTGRES_INTEGRACION=1")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 90*time.Second)
	t.Cleanup(cancelar)
	b, err := postgres.Nueva(ctx, os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(b.Cerrar)
	if err = b.Comprobar(ctx); err != nil {
		t.Fatal(err)
	}
	return b, ctx
}
