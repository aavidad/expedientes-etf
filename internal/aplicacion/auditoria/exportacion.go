package auditoria

import (
	"context"
	"errors"
	"fmt"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/auditoria/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"time"
)

type Exportador struct {
	lectura  salida.LecturaExportacion
	tx       plataforma.Transacciones
	paquetes salida.Empaquetador
	registro *Registro
	ahora    func() time.Time
}

func NuevoExportador(l salida.LecturaExportacion, tx plataforma.Transacciones, p salida.Empaquetador, r salida.Registro, ahora func() time.Time) *Exportador {
	return &Exportador{l, tx, p, NuevoRegistro(r), ahora}
}

func (s *Exportador) Exportar(ctx context.Context, a dominio.ActorExportacion, f dominio.FiltroExportacion, formato, instrucciones string) (entrada.DescargaAuditoria, error) {
	if err := a.Autorizar(); err != nil {
		return entrada.DescargaAuditoria{}, err
	}
	fallar := func(causa error) (entrada.DescargaAuditoria, error) {
		if err := s.Rechazar(ctx, a); err != nil {
			return entrada.DescargaAuditoria{}, errors.New("auditoria.no_disponible")
		}
		return entrada.DescargaAuditoria{}, causa
	}

	if err := f.Validar(); err != nil {
		return fallar(err)
	}
	if formato != "csv" && formato != "json" {
		return fallar(errors.New("auditoria.formato_invalido"))
	}
	ctx, cancelar := context.WithTimeout(ctx, 30*time.Second)
	defer cancelar()
	var archivo salida.ArchivoExportacion
	var tramo dominio.TramoExportacion
	err := s.tx.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		var err error
		tramo, err = s.lectura.Delimitar(ctx, f)
		if err != nil {
			return err
		}
		archivo, err = s.paquetes.Preparar(ctx, tramo, f, formato, instrucciones, func(consumir func(dominio.Evento) error) error {
			previo := tramo.Anterior
			cantidad := 0
			for previo.ID < tramo.Final.ID {
				filas, err := s.lectura.Pagina(ctx, tramo, previo.ID)
				if err != nil {
					return err
				}
				if len(filas) == 0 {
					return errors.New("auditoria.cadena_alterada")
				}
				for _, e := range filas {
					cantidad++
					if cantidad > dominio.MaxEventosExportacion {
						return errors.New("auditoria.exportacion_demasiado_grande")
					}
					if err := dominio.Verificar([]dominio.Evento{e}, previo, e.Punto()); err != nil || e.ID > tramo.Final.ID {
						return errors.New("auditoria.cadena_alterada")
					}
					if err := consumir(e); err != nil {
						return err
					}
					previo = e.Punto()
				}
			}
			if previo != tramo.Final {
				return errors.New("auditoria.cadena_alterada")
			}
			return ctx.Err()
		})
		return err
	})
	if err != nil {
		if archivo != nil {
			_ = archivo.Close()
		}
		return fallar(err)
	}
	// El registro confirma otra transacción antes de que HTTP reciba el archivo.
	_, err = s.registro.RegistrarAntesDeEntregar(ctx, dominio.Datos{OcurridoEn: s.ahora().UTC().Truncate(time.Microsecond), UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, Accion: fmt.Sprintf("auditoria.exportar_%s.filas_%d", formato, archivo.Filas()), Entidad: "auditoria", Resultado: "ok"}, func(context.Context) error { return nil })
	if err != nil {
		_ = archivo.Close()
		return fallar(err)
	}
	return entrada.DescargaAuditoria{Contenido: archivo, Tamano: archivo.Tamano(), Filas: archivo.Filas(), Final: tramo.Final}, nil
}

// Rechazar confirma el fallo fuera de cualquier transacción de lectura revertida.
func (s *Exportador) Rechazar(ctx context.Context, a dominio.ActorExportacion) error {
	if err := a.Autorizar(); err != nil {
		return err
	}
	cierre, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancelar()
	_, err := s.registro.RegistrarAntesDeEntregar(cierre, dominio.Datos{OcurridoEn: s.ahora().UTC().Truncate(time.Microsecond), UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, Accion: "auditoria.exportar", Entidad: "auditoria", Resultado: "error"}, func(context.Context) error { return nil })
	return err
}
