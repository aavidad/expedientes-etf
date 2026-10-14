package catalogo

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrSMI = errors.New("catalogo.smi_invalido")

type SMI struct {
	Anio                   int    `json:"anio"`
	ImporteMensualCentimos *int64 `json:"importe_mensual_centimos"`
	ImporteAnualCentimos   *int64 `json:"importe_anual_centimos"`
	ImporteMensual         string `json:"importe_mensual"`
	ImporteAnual           string `json:"importe_anual"`
	Version                int    `json:"version"`
}

type CambioSMI struct {
	Anio                   int   `json:"anio"`
	ImporteMensualCentimos int64 `json:"importe_mensual_centimos"`
	VersionEsperada        int   `json:"version_esperada"`
}

func (c CambioSMI) Validar() error {
	if c.Anio < 2026 || c.Anio > 2040 || c.ImporteMensualCentimos <= 0 || c.VersionEsperada < 0 {
		return ErrSMI
	}
	return nil
}

// ParsearImporteSMI acepta euros con hasta dos decimales y evita redondeos binarios.
func ParsearImporteSMI(valor string) (int64, error) {
	valor = strings.TrimSpace(valor)
	partes := strings.Split(valor, ".")
	if len(partes) > 2 || partes[0] == "" || len(partes) == 2 && (len(partes[1]) == 0 || len(partes[1]) > 2) {
		return 0, ErrSMI
	}
	for _, parte := range partes {
		for _, caracter := range parte {
			if caracter < '0' || caracter > '9' {
				return 0, ErrSMI
			}
		}
	}
	euros, err := strconv.ParseInt(partes[0], 10, 64)
	if err != nil || euros > (1<<63-1)/100 {
		return 0, ErrSMI
	}
	centimos := int64(0)
	if len(partes) == 2 {
		centimos, _ = strconv.ParseInt(partes[1]+strings.Repeat("0", 2-len(partes[1])), 10, 64)
	}
	importe := euros*100 + centimos
	if importe <= 0 {
		return 0, ErrSMI
	}
	return importe, nil
}

func ImporteSMI(centimos *int64) string {
	if centimos == nil {
		return ""
	}
	return fmt.Sprintf("%d.%02d", *centimos/100, *centimos%100)
}
