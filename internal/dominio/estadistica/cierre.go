package estadistica

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"time"
)

const (
	ErrActoMemoria         Error = "estadistica.memoria_acto_invalido"
	ErrIdempotenciaMemoria Error = "estadistica.memoria_idempotencia"
	ErrRevisionMemoria     Error = "estadistica.memoria_revision_inexistente"
	ErrBorradorMemoria     Error = "estadistica.memoria_borrador"
	VersionMotorMemoria          = "F4.06.1"
)

type EstadoMemoria struct {
	FiltroMemoria
	ID             string `json:"id"`
	Estado         string `json:"estado"`
	Version        int    `json:"version"`
	RevisionActual int    `json:"revision_actual"`
}
type ActoMemoria struct {
	Version int    `json:"version"`
	Motivo  string `json:"motivo,omitempty"`
}

func (p ActoMemoria) Validar(clave string, reabrir bool) error {
	if p.Version < 0 || !textoNarrativo(clave, 128, 1) || strings.TrimSpace(clave) == "" {
		return ErrActoMemoria
	}
	if reabrir {
		if strings.TrimSpace(p.Motivo) == "" || !textoNarrativo(p.Motivo, 4000, 25) {
			return ErrActoMemoria
		}
	} else if p.Motivo != "" {
		return ErrActoMemoria
	}
	return nil
}
func AutorizarActoMemoria(a ActorMemoria, f FiltroMemoria, reabrir bool) (FiltroMemoria, error) {
	if reabrir && !slices.Contains(a.Perfiles, "coordinacion") {
		return f, ErrPermisoMemoria
	}
	return AutorizarMemoria(a, f)
}

type RevisionMemoria struct {
	ID         string         `json:"id"`
	MemoriaID  string         `json:"memoria_id"`
	Revision   int            `json:"revision"`
	Informe    InformeMemoria `json:"informe"`
	Huella     string         `json:"huella"`
	Motor      string         `json:"motor"`
	CerradaPor string         `json:"cerrada_por"`
	CerradaEn  time.Time      `json:"cerrada_en"`
}

func HuellaMemoria(informe InformeMemoria) (string, error) {
	b, err := json.Marshal(struct {
		Motor   string
		Informe InformeMemoria
	}{VersionMotorMemoria, informe})
	if err != nil {
		return "", err
	}
	defer clear(b)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

type ResultadoActoMemoria struct {
	EstadoMemoria
	RevisionID string `json:"revision_id,omitempty"`
	Huella     string `json:"huella,omitempty"`
	Repetida   bool   `json:"repetida"`
}
type DiscrepanciaMemoria struct {
	ID           string     `json:"id"`
	MemoriaID    string     `json:"memoria_id"`
	Revision     int        `json:"revision"`
	EquipoID     string     `json:"equipo_id"`
	ExpedienteID string     `json:"expediente_id,omitempty"`
	EventoID     uint64     `json:"evento_id,omitempty"`
	Apartados    []Apartado `json:"apartados"`
	DetectadaEn  time.Time  `json:"detectada_en"`
}

type DiferenciaApartadoMemoria struct {
	Codigo  Apartado        `json:"codigo"`
	Cerrado ApartadoMemoria `json:"cerrado"`
	Actual  ApartadoMemoria `json:"actual"`
}
type DiferenciaMemoria struct {
	FiltroMemoria
	Revision  int                         `json:"revision"`
	Apartados []DiferenciaApartadoMemoria `json:"apartados"`
}
type DiscrepanciasMemoria struct {
	Avisos      []DiscrepanciaMemoria `json:"avisos"`
	Diferencias []DiferenciaMemoria   `json:"diferencias"`
}

// BaseEquipoMemoria es una proyección numérica; nunca contiene narrativas.
// Permite detectar correcciones propias que afectan también a la provincial.
type BaseEquipoMemoria struct {
	MemoriaID string
	Revision  int
	Informe   InformeMemoria
}

// ApartadosDiferentes compara cifras y avisos, sin fechas de lectura ni huellas
// de fuentes. Una corrección que no altera ningún resultado no produce aviso.
func ApartadosDiferentes(a, b InformeMemoria) []Apartado {
	anterior := map[Apartado]ApartadoMemoria{}
	for _, v := range a.Apartados {
		anterior[v.Codigo] = v
	}
	diferentes := []Apartado{}
	for _, v := range b.Apartados {
		if !reflect.DeepEqual(anterior[v.Codigo], v) || a.HuellasComposicion[v.Codigo] != b.HuellasComposicion[v.Codigo] {
			diferentes = append(diferentes, v.Codigo)
		}
		delete(anterior, v.Codigo)
	}
	for k := range anterior {
		diferentes = append(diferentes, k)
	}
	slices.Sort(diferentes)
	return diferentes
}
