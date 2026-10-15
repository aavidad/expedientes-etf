package expediente

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrCargaCerrada           = errors.New("carga.plazo_cerrado")
	ErrDuplicadosSinConfirmar = errors.New("carga.duplicados_sin_confirmar")
	ErrCargaAgotaNumeracion   = errors.New("carga.agota_numeracion")
	ErrLoteConActividad       = errors.New("carga.lote_con_actividad")
	ErrControlCarga           = errors.New("carga.control_invalido")
)

type PlazoCarga struct {
	PermiteCarga   bool   `json:"abierta"`
	DiasDuplicados *int   `json:"dias_duplicados,omitempty"`
	Fecha          string `json:"fecha_cierre"`
	Version        int    `json:"version"`
}

func (p PlazoCarga) Abierta(ahora time.Time) bool {
	return p.Fecha != "" && FechaCivil(ahora) <= p.Fecha
}
func (p PlazoCarga) Validar() error {
	f, err := time.Parse("2006-01-02", p.Fecha)
	if err != nil || f.Year() < 1 || p.Version < 1 || p.DiasDuplicados != nil && (*p.DiasDuplicados < 0 || *p.DiasDuplicados > 366) {
		return ErrControlCarga
	}
	return nil
}

type AvisoCarga struct {
	Fila  int    `json:"fila"`
	Clave string `json:"clave"`
}
type LoteCarga struct {
	Referencia int64     `json:"referencia"`
	ID         string    `json:"id"`
	EquipoID   string    `json:"equipo_id"`
	Fecha      time.Time `json:"cargada_en"`
	Casos      int       `json:"casos"`
	Anulado    bool      `json:"anulado"`
}

// CoincidenciaHistorica compara identidades conocidas, nunca sexos ni campos vacíos.
// La ventana se aplica a la fecha de derivación de ambos expedientes.
func CoincidenciaHistorica(a, b CasoHistorico, dias int) bool {
	fa, ea := time.Parse("2006-01-02", a.FechaDerivacion)
	fb, eb := time.Parse("2006-01-02", b.FechaDerivacion)
	if ea != nil || eb != nil || fa.Sub(fb) > time.Duration(dias)*24*time.Hour || fb.Sub(fa) > time.Duration(dias)*24*time.Hour {
		return false
	}
	ka, kb := identidadesCarga(a), identidadesCarga(b)
	for k := range ka {
		if kb[k] {
			return true
		}
	}
	return false
}
func identidadesCarga(c CasoHistorico) map[string]bool {
	r := map[string]bool{}
	agregar := func(tipo, nombre string) {
		n := strings.ToLower(strings.Join(strings.Fields(nombre), " "))
		if len(strings.Fields(n)) > 1 {
			r[tipo+":"+n] = true
		}
	}
	for _, f := range c.Familia {
		if f.Adultos != nil {
			for _, a := range *f.Adultos {
				agregar("adulto", a.Nombre+" "+a.Apellidos)
			}
		}
	}
	for _, m := range c.Menores {
		if m.Menores != nil {
			for _, v := range *m.Menores {
				agregar("menor", v.Nombre+" "+v.Apellidos)
			}
		}
	}
	for _, f := range c.Datos.Extraccion.Campos {
		switch f.Codigo {
		case "familia.nombre":
			agregar("familia", f.Valor)
		case "sociodemografia.adulto.nombre":
			agregar("adulto", f.Valor)
		case "menores.persona.nombre":
			agregar("menor", f.Valor)
		}
	}
	return r
}
