//go:build !linux

package pdf

import "github.com/aavidad/expedientes-etf/internal/dominio/expediente"

func limitarProceso() error { return expediente.ErrNoDisponible }
