package entrada

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"time"
)

type Inicio struct {
	Cuenta     string
	Contrasena string
	IP         dominio.IP
}

// Acceso contiene permisos recién leídos del servidor, nunca del cliente.
type Acceso struct {
	Usuario  dominio.Usuario
	Estado   dominio.Estado
	CaducaEn time.Time
	MaximaEn time.Time
}

type SesionAbierta struct {
	Identificador string
	Acceso        Acceso
}

type Sesiones interface {
	// AdmitirPeticion aplica el límite de transporte antes del caso de uso.
	AdmitirPeticion(context.Context) error
	Entrar(context.Context, Inicio) (SesionAbierta, error)
	Consultar(context.Context, string) (Acceso, error)
	// RegistrarRechazo registra peticiones rechazadas por validación HTTP, sin autenticar.
	RegistrarRechazo(context.Context, string, string) error
	Salir(context.Context, string) error
}

// El adaptador HTTP obtiene el origen de RemoteAddr, nunca de cabeceras o JSON.
type claveOrigen struct{}

func ConOrigen(ctx context.Context, ip dominio.IP) context.Context {
	return context.WithValue(ctx, claveOrigen{}, ip)
}
func Origen(ctx context.Context) dominio.IP {
	ip, _ := ctx.Value(claveOrigen{}).(dominio.IP)
	return ip
}

type claveConsultaPasiva struct{}

// ConConsultaPasiva comprueba vigencia sin prolongar la inactividad.
func ConConsultaPasiva(ctx context.Context) context.Context {
	return context.WithValue(ctx, claveConsultaPasiva{}, true)
}
func ConsultaPasiva(ctx context.Context) bool {
	v, _ := ctx.Value(claveConsultaPasiva{}).(bool)
	return v
}
