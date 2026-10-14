package main

import (
	"crypto/rand"
	"fmt"
	"os"
	"time"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/anexos/lectura"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/clamav"
	nombres "github.com/aavidad/expedientes-etf/internal/adaptadores/expediente/ficheros"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/expediente/memoria"
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	catalogoPostgres "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	expedientePostgres "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
)

type identificadoresDerivacion struct{}

func (identificadoresDerivacion) Nuevo() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

func configurarDerivaciones(base *postgres.Base, idiomasRuta, idioma string, leer func(string) string) (adaptadorHTTP.ConfiguracionDerivaciones, error) {
	direccion := leer("ETF_CLAMAV_DIRECCION")
	if direccion == "" {
		direccion = "clamav:3310"
	}
	antivirus, err := clamav.Nuevo(clamav.Configuracion{Direccion: direccion, TiempoLimite: 3 * time.Second, MaximoBytes: 10_000_000})
	if err != nil {
		return adaptadorHTTP.ConfiguracionDerivaciones{}, err
	}
	registro := registroAvisosMemoria(postgres.NuevoRegistroAuditoria(base))
	return adaptadorHTTP.ConfiguracionDerivaciones{Servicio: expediente.Nuevo(expediente.Dependencias{
		Borradores: memoria.NuevosBorradores(30*time.Minute, 32), Confirmaciones: &expedientePostgres.Repositorio{}, Etiquetas: nombres.NuevosMunicipios(idiomasRuta, idioma),
		Repositorio: &expedientePostgres.Repositorio{}, Transacciones: base, Registro: registro, RegistroTransaccional: registro,
		Validador: lectura.Validador{}, Antivirus: antivirus, Lector: lectura.NuevoLector(), Catalogo: &catalogoPostgres.Repositorio{},
		NombresMunicipios: nombres.NuevosMunicipios(idiomasRuta, idioma), Reloj: relojIdentidad{}, Identificadores: identificadoresDerivacion{},
	})}, nil
}

func moduloDerivaciones(d dependenciasModulos) adaptadorHTTP.ModuloDerivaciones {
	cfg, err := configurarDerivaciones(d.base, d.cfg.IdiomasRuta, d.cfg.Idioma, os.Getenv)
	if err != nil {
		d.registro.Warn("expediente.no_disponible")
	}
	return adaptadorHTTP.ModuloDerivaciones{Configuracion: cfg}
}
