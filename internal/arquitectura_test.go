package internal_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const modulo = "github.com/aavidad/expedientes-etf/"

// Se analiza todo el código de producción, incluidas variantes de plataforma
// y etiquetas de build. Las pruebas pueden usar adaptadores y dobles libremente.
func TestArquitectura(t *testing.T) {
	for _, carpeta := range []string{"internal", "cmd", "web"} {
		err := filepath.WalkDir(filepath.Join("..", carpeta), func(ruta string, entrada fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entrada.IsDir() {
				if entrada.Name() == "node_modules" || entrada.Name() == "dist" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(ruta, ".go") || strings.HasSuffix(ruta, "_test.go") {
				return nil
			}
			archivo, err := parser.ParseFile(token.NewFileSet(), ruta, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			origen := filepath.ToSlash(strings.TrimPrefix(filepath.Dir(ruta), "../"))
			for _, imp := range archivo.Imports {
				destino, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					return err
				}
				if !importacionPermitida(origen, destino) {
					t.Errorf("%s importa %s: rompe TRD §2.1", ruta, destino)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func bajo(ruta, base string) bool { return ruta == base || strings.HasPrefix(ruta, base+"/") }

func puertoEntrada(ruta string) bool {
	partes := strings.Split(ruta, "/")
	return len(partes) >= 5 && partes[0] == "internal" && partes[1] == "aplicacion" && partes[3] == "puertos" && partes[4] == "entrada"
}

func moduloCapa(ruta string) string {
	partes := strings.Split(ruta, "/")
	if len(partes) >= 3 {
		return partes[2]
	}
	return ""
}

func estandarPura(destino string) bool {
	if destino == "C" || strings.Contains(strings.Split(destino, "/")[0], ".") {
		return false
	}
	for _, prohibido := range []string{"net", "database", "os", "syscall", "unsafe", "plugin", "io/fs", "io/ioutil", "path/filepath", "runtime/cgo", "log", "embed"} {
		if bajo(destino, prohibido) {
			return false
		}
	}
	return true
}

func importacionPermitida(origen, destino string) bool {
	interno := strings.HasPrefix(destino, modulo)
	destino = strings.TrimPrefix(destino, modulo)
	if bajo(origen, "cmd/etf") {
		return !bajo(destino, "cmd/migrar")
	}
	if bajo(origen, "cmd/migrar") {
		return !interno || bajo(destino, "internal/plataforma") || bajo(destino, "internal/adaptadores/postgres")
	}
	if bajo(origen, "cmd/verificar-exportacion") {
		return (!interno && destino != "C" && !strings.Contains(strings.Split(destino, "/")[0], ".") && !bajo(destino, "net") && !bajo(destino, "database")) || (interno && (bajo(destino, "internal/dominio/auditoria") || bajo(destino, "internal/adaptadores/ficheros")))
	}
	if bajo(origen, "cmd/comprobar-auditoria") {
		return (!interno && destino != "C" && !strings.Contains(strings.Split(destino, "/")[0], ".")) ||
			(interno && bajo(destino, "internal/dominio/auditoria"))
	}
	if bajo(origen, "web") || bajo(origen, "internal/plataforma") {
		return !interno && !strings.Contains(strings.Split(destino, "/")[0], ".")
	}
	if bajo(origen, "internal/dominio") {
		return (!interno && estandarPura(destino)) || (interno && bajo(destino, "internal/dominio/"+moduloCapa(origen)))
	}
	if puertoEntrada(origen) || bajo(origen, "internal/puertos/salida") {
		return (!interno && estandarPura(destino)) || (interno && bajo(destino, "internal/dominio"))
	}
	if bajo(origen, "internal/aplicacion") {
		return (!interno && estandarPura(destino)) || (interno && (bajo(destino, "internal/dominio") || bajo(destino, "internal/puertos/salida") ||
			puertoEntrada(destino) || bajo(destino, "internal/aplicacion/"+moduloCapa(origen))))
	}
	if bajo(origen, "internal/adaptadores/http") {
		if !interno {
			// HTTP no puede saltarse los puertos leyendo disco o tablas directamente.
			return !strings.Contains(strings.Split(destino, "/")[0], ".") && destino != "C" &&
				!bajo(destino, "database") && !bajo(destino, "os") && !bajo(destino, "syscall") && !bajo(destino, "io/ioutil")
		}
		return puertoEntrada(destino) || bajo(destino, "internal/adaptadores/http")
	}
	if bajo(origen, "internal/adaptadores") {
		return !interno || bajo(destino, "internal/dominio") || bajo(destino, "internal/puertos/salida") || bajo(destino, "internal/adaptadores/"+moduloCapa(origen))
	}
	return false // Cualquier capa nueva requiere decidir y documentar sus reglas.
}

func TestReglasRechazanDependenciasProhibidas(t *testing.T) {
	casos := [][2]string{
		{"internal/dominio/expediente", "net/http"},
		{"internal/dominio/expediente", "database/sql"},
		{"internal/dominio/expediente", "os"},
		{"internal/dominio/expediente", "os/exec"},
		{"internal/dominio/expediente", "github.com/jackc/pgx/v5"},
		{"internal/dominio/expediente", "github.com/go-ldap/ldap/v3"},
		{"internal/dominio/expediente", modulo + "internal/aplicacion/expediente"},
		{"internal/dominio/expediente", modulo + "internal/adaptadores/postgres"},
		{"internal/dominio/expediente", modulo + "internal/puertos/salida/expediente"},
		{"internal/aplicacion/expediente/puertos/entrada", modulo + "internal/aplicacion/expediente"},
		{"internal/puertos/salida/expediente", modulo + "internal/adaptadores/postgres"},
		{"internal/puertos/salida/expediente", modulo + "internal/aplicacion/expediente"},
		{"internal/aplicacion/expediente", modulo + "internal/adaptadores/postgres"},
		{"internal/aplicacion/expediente", modulo + "internal/aplicacion/catalogo"},
		{"internal/adaptadores/http", modulo + "internal/aplicacion/paquete"},
		{"internal/adaptadores/http", modulo + "internal/adaptadores/ficheros"},
		{"internal/adaptadores/http", modulo + "internal/dominio/expediente"},
		{"internal/adaptadores/http", "database/sql"},
		{"internal/adaptadores/http", "os"},
		{"internal/adaptadores/http", "github.com/jackc/pgx/v5"},
		{"internal/adaptadores/http", "github.com/go-ldap/ldap/v3"},
		{"internal/adaptadores/postgres", modulo + "internal/adaptadores/ldap"},
		{"internal/adaptadores/postgres", modulo + "internal/aplicacion/expediente"},
		{"cmd/etf", modulo + "cmd/migrar"},
		{"cmd/migrar", modulo + "internal/aplicacion/expediente"},
		{"cmd/comprobar-auditoria", modulo + "internal/aplicacion/auditoria"},
		{"cmd/comprobar-auditoria", modulo + "internal/adaptadores/postgres"},
		{"cmd/comprobar-auditoria", modulo + "internal/dominio/identidad"},
		{"cmd/comprobar-auditoria", "github.com/jackc/pgx/v5"},
		{"cmd/comprobar-auditoria", "C"},
		{"cmd/verificar-exportacion", modulo + "internal/adaptadores/postgres"},
		{"cmd/verificar-exportacion", modulo + "internal/aplicacion/auditoria"},
		{"cmd/verificar-exportacion", "net/http"},
	}
	for _, caso := range casos {
		if importacionPermitida(caso[0], caso[1]) {
			t.Errorf("dependencia prohibida aceptada: %v", caso)
		}
	}
}

func TestReglasPermitenPuertosYDominio(t *testing.T) {
	casos := [][2]string{
		{"internal/dominio/expediente", "time"},
		{"internal/aplicacion/expediente/puertos/entrada", modulo + "internal/dominio/expediente"},
		{"internal/puertos/salida/paquete", "context"},
		{"internal/aplicacion/expediente", modulo + "internal/aplicacion/catalogo/puertos/entrada"},
		{"internal/aplicacion/expediente", modulo + "internal/puertos/salida/expediente"},
		{"internal/adaptadores/http", modulo + "internal/aplicacion/paquete/puertos/entrada"},
		{"internal/adaptadores/postgres", modulo + "internal/puertos/salida/expediente"},
		{"cmd/etf", modulo + "internal/adaptadores/http"},
		{"cmd/comprobar-auditoria", "os"},
		{"cmd/comprobar-auditoria", "encoding/json"},
		{"cmd/comprobar-auditoria", modulo + "internal/dominio/auditoria"},
	}
	for _, caso := range casos {
		if !importacionPermitida(caso[0], caso[1]) {
			t.Errorf("dependencia válida rechazada: %v", caso)
		}
	}
}
