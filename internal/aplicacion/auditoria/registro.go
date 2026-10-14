package auditoria

import (
	"context"
	"errors"

	"github.com/aavidad/expedientes-etf/internal/aplicacion/auditoria/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
)

type Registro struct {
	registro salida.Registro
}

var _ entrada.Registro = (*Registro)(nil)

func NuevoRegistro(registro salida.Registro) *Registro {
	return &Registro{registro: registro}
}

// RegistrarAntesDeEntregar recibe metadatos obtenidos del contexto autorizado
// y una entrega de datos ya reunidos en memoria. La entrega, incluido su primer
// byte, sólo se ejecuta tras Confirmar. No concede permisos de exportación:
// el caso de uso llamante debe autorizarlos antes de reunir los datos.
func (r *Registro) RegistrarAntesDeEntregar(ctx context.Context, datos dominio.Datos, entregar func(context.Context) error) (dominio.Evento, error) {
	if r == nil || r.registro == nil || entregar == nil {
		return dominio.Evento{}, errors.New("auditoria.configuracion_invalida")
	}
	if err := ctx.Err(); err != nil {
		return dominio.Evento{}, &fallo{clave: "auditoria.cancelada", causa: err}
	}
	if err := datos.Validar(); err != nil {
		return dominio.Evento{}, err
	}
	// Separa las colecciones mutables del llamante y de las que recibe el puerto.
	datos.Perfiles = append([]string{}, datos.Perfiles...)
	entrada := datos
	entrada.Perfiles = append([]string{}, datos.Perfiles...)
	evento, err := r.registro.Confirmar(ctx, entrada)
	if err != nil {
		return dominio.Evento{}, &fallo{clave: "auditoria.registro_fallido", causa: err}
	}
	// Se conoce la huella anterior, pero su ID pertenece al adaptador. Basta
	// un ID inferior para comprobar este evento aislado y los datos enviados.
	anterior := dominio.Inicio()
	if evento.HuellaAnterior != dominio.HuellaInicial && evento.ID > 0 {
		anterior = dominio.Punto{ID: evento.ID - 1, Huella: evento.HuellaAnterior}
	}
	esperado, err := dominio.Encadenar(evento.ID, datos, anterior)
	if err != nil || esperado.Huella != evento.Huella ||
		dominio.Verificar([]dominio.Evento{evento}, anterior, evento.Punto()) != nil {
		return dominio.Evento{}, errors.New("auditoria.confirmacion_invalida")
	}
	if err := ctx.Err(); err != nil {
		return evento, &fallo{clave: "auditoria.cancelada", causa: err}
	}
	if err := entregar(ctx); err != nil {
		return evento, &fallo{clave: "auditoria.entrega_fallida", causa: err}
	}
	return evento, nil
}

// Error conserva la causa para errors.Is/As sin incluirla en la clave visible.
type fallo struct {
	clave string
	causa error
}

func (f *fallo) Error() string { return f.clave }
func (f *fallo) Unwrap() error { return f.causa }
