package main

import (
	"encoding/json"
	"os"
	"strconv"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
)

func ejecutar(args []string) int {
	salida := func(clave string) int {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]string{"clave": clave})
		return 1
	}
	if len(args) != 1 && len(args) != 5 {
		return salida("auditoria.uso_verificador")
	}
	previo, err := os.Lstat(args[0])
	if err != nil || !previo.Mode().IsRegular() {
		return salida("auditoria.fichero_invalido")
	}
	f, err := os.Open(args[0])
	if err != nil {
		return salida("auditoria.fichero_invalido")
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || !os.SameFile(previo, info) {
		return salida("auditoria.fichero_invalido")
	}
	m, err := ficheros.VerificarPaqueteAuditoria(f, info.Size())
	if err != nil {
		return salida(err.Error())
	}
	if len(args) == 5 {
		anterior, e1 := strconv.ParseUint(args[1], 10, 64)
		final, e2 := strconv.ParseUint(args[3], 10, 64)
		a := auditoria.Punto{ID: anterior, Huella: args[2]}
		z := auditoria.Punto{ID: final, Huella: args[4]}
		if e1 != nil || e2 != nil || a.Validar() != nil || z.Validar() != nil || a != m.Tramo.Anterior || z != m.Tramo.Final {
			return salida("auditoria.sello_no_coincide")
		}
	}
	_ = json.NewEncoder(os.Stdout).Encode(struct {
		Clave string                     `json:"clave"`
		Filas int                        `json:"filas"`
		Tramo auditoria.TramoExportacion `json:"tramo"`
	}{"auditoria.exportacion_verificada", m.Filas, m.Tramo})
	return 0
}
func main() { os.Exit(ejecutar(os.Args[1:])) }
