// comprobar-auditoria verifica JSON local sin abrir conexiones ni modificarlo.
package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strconv"
	"syscall"

	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
)

type resultado struct {
	Clave    string `json:"clave"`
	Indice   *int   `json:"indice,omitempty"`
	EventoID uint64 `json:"evento_id,omitempty"`
}

func main() { os.Exit(ejecutar(os.Args[1:], os.Stdout)) }

// Uso: comprobar-auditoria fichero.json [id_final huella_final], o
// fichero.json id_anterior huella_anterior id_final huella_final para un tramo.
// Los puntos externos proceden de una copia independiente, nunca del JSON.
func ejecutar(args []string, salida io.Writer) int {
	var sello *auditoria.Punto
	anterior := auditoria.Inicio()
	if len(args) != 1 && len(args) != 3 && len(args) != 5 {
		return responder(salida, resultado{Clave: "auditoria.comprobador_uso"}, 2)
	}
	if len(args) >= 3 {
		id, err := strconv.ParseUint(args[len(args)-2], 10, 64)
		p := auditoria.Punto{ID: id, Huella: args[len(args)-1]}
		if err != nil || p.Validar() != nil {
			return responder(salida, resultado{Clave: "auditoria.sello_invalido"}, 2)
		}
		sello = &p
	}
	if len(args) == 5 {
		id, err := strconv.ParseUint(args[1], 10, 64)
		anterior = auditoria.Punto{ID: id, Huella: args[2]}
		if err != nil || anterior.Validar() != nil || anterior.ID > sello.ID {
			return responder(salida, resultado{Clave: "auditoria.ancla_invalida"}, 2)
		}
	}
	// O_NONBLOCK permite rechazar una FIFO sin esperar a un escritor.
	f, err := os.OpenFile(args[0], os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return responder(salida, resultado{Clave: "auditoria.fichero_invalido"}, 2)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return responder(salida, resultado{Clave: "auditoria.fichero_invalido"}, 2)
	}
	if info.Size() > auditoria.MaxBytesExportacion {
		return responder(salida, resultado{Clave: "auditoria.exportacion_demasiado_grande"}, 1)
	}
	contenido, err := io.ReadAll(io.LimitReader(f, auditoria.MaxBytesExportacion+1))
	if err != nil {
		return responder(salida, resultado{Clave: "auditoria.fichero_invalido"}, 2)
	}
	exportacion, err := auditoria.DecodificarExportacion(contenido)
	if err != nil {
		r := resultado{Clave: err.Error()}
		var roto *auditoria.ErrorVerificacion
		if errors.As(err, &roto) {
			r.Indice, r.EventoID = &roto.Indice, roto.EventoID
		}
		return responder(salida, r, 1)
	}
	if sello != nil && *sello != exportacion.Final {
		return responder(salida, resultado{Clave: "auditoria.sello_externo_no_coincide"}, 1)
	}
	if sello != nil && anterior != exportacion.Anterior {
		return responder(salida, resultado{Clave: "auditoria.ancla_externa_no_coincide"}, 1)
	}
	clave := "auditoria.cadena_coherente"
	if sello != nil {
		clave = "auditoria.sello_externo_coincide"
	}
	return responder(salida, resultado{Clave: clave}, 0)
}

func responder(salida io.Writer, r resultado, codigo int) int {
	if err := json.NewEncoder(salida).Encode(r); err != nil {
		return 2
	}
	return codigo
}
