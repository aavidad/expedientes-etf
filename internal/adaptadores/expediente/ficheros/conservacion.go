package ficheros

import (
	"context"
	"encoding/json"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"os"
)

type Conservacion struct{ Ruta string }

func (c Conservacion) Anios(context.Context) (int, error) {
	b, err := os.ReadFile(c.Ruta)
	if err != nil {
		return 0, d.ErrNoDisponible
	}
	var dato struct {
		Anios int `json:"anios"`
	}
	if json.Unmarshal(b, &dato) != nil || dato.Anios < 1 || dato.Anios > 200 {
		return 0, d.ErrNoDisponible
	}
	return dato.Anios, nil
}
