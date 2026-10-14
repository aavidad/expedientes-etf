package expediente

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	ErrCargaHistorica  = errors.New("expediente.carga_historica_invalida")
	ErrNumeroHistorico = errors.New("expediente.numero_historico_invalido")
	ErrNumeroDuplicado = errors.New("expediente.numero_historico_duplicado")
)

// NumeroHistorico conserva el identificador original; el vacío significa desconocido.
type NumeroHistorico struct {
	Numero string `json:"numero"`
}

func (n NumeroHistorico) Validar(equipo string, anio int) (int, error) {
	partes := strings.Split(n.Numero, "-")
	if !regexp.MustCompile(`^[0-9]{2}-[0-9]{2}-[0-9]{2}$`).MatchString(n.Numero) || partes[0] != equipo || partes[2] != strconv.Itoa(anio%100 + 100)[1:] {
		return 0, ErrNumeroHistorico
	}
	correlativo, _ := strconv.Atoi(partes[1])
	if correlativo == 0 {
		return 0, ErrNumeroHistorico
	}
	return correlativo, nil
}

type EpisodioHistorico struct {
	Apertura         string `json:"apertura_fecha"`
	Cierre           string `json:"cierre_fecha,omitempty"`
	MotivoCierre     string `json:"motivo_cierre,omitempty"`
	MotivoActaTexto  string `json:"motivo_acta_texto,omitempty"`
	MotivoReapertura string `json:"motivo_reapertura,omitempty"`
}
type CasoHistorico struct {
	NumeroHistorico
	FechaDerivacion string              `json:"fecha_derivacion"`
	Estado          string              `json:"estado"`
	Datos           DatosDerivacion     `json:"derivacion_datos"`
	Aceptacion      *Aceptacion         `json:"aceptacion,omitempty"`
	NoAceptacion    *NoAceptacion       `json:"no_aceptacion,omitempty"`
	Episodios       []EpisodioHistorico `json:"episodios"`
	Familia         []CambioFamilia     `json:"familia"`
	Menores         []CambioMenores     `json:"menores"`
}
type CargaHistorica struct {
	FuenteHoja *FuenteHoja     `json:"-"`
	EquipoID   string          `json:"equipo_id"`
	Casos      []CasoHistorico `json:"casos"`
}
type ResultadoCargaHistorica struct {
	LoteID      string   `json:"lote_id"`
	Expedientes []string `json:"expedientes"`
	Marca       string   `json:"marca"`
}
type ValidacionCargaHistorica struct {
	LoteID string `json:"lote_id"`
	Casos  int    `json:"casos"`
}

func (p CargaHistorica) Validar(ahora time.Time) error {
	if !IDValido(p.EquipoID) || len(p.Casos) < 1 || len(p.Casos) > 100 {
		return ErrCargaHistorica
	}
	numeros := map[string]bool{}
	for _, c := range p.Casos {
		derivacion, err := time.Parse("2006-01-02", c.FechaDerivacion)
		if err != nil || derivacion.Year() < 1 || c.FechaDerivacion > FechaCivil(ahora) || c.FechaDerivacion > "2026-12-31" || c.Datos.Extraccion.Tipo != "anexo3" && c.Datos.Extraccion.Tipo != "anexo4" {
			return ErrCargaHistorica
		}
		if c.Numero != "" {
			if numeros[c.Numero] {
				return ErrNumeroDuplicado
			}
			numeros[c.Numero] = true
		}
		if c.Estado == NoAceptada {
			if c.NoAceptacion == nil || c.Aceptacion != nil || c.Numero != "" || len(c.Episodios)+len(c.Familia)+len(c.Menores) != 0 || c.NoAceptacion.Fecha < "2026-01-01" || c.NoAceptacion.Fecha > "2026-12-31" || c.NoAceptacion.Fecha < c.FechaDerivacion || c.NoAceptacion.Fecha > FechaCivil(ahora) {
				return ErrCargaHistorica
			}
			decision := *c.NoAceptacion
			decision.Version = 1
			if _, err := decision.Validar(Derivacion{Estado: DerivacionRecibida, Datos: c.Datos}); err != nil {
				return err
			}
			continue
		}
		if c.Estado != Aceptada && c.Estado != EnIntervencion && c.Estado != Cerrada || c.Aceptacion == nil || c.NoAceptacion != nil || c.Aceptacion.EquipoID != p.EquipoID || len(c.Episodios) < 1 || len(c.Episodios) > 100 {
			return ErrCargaHistorica
		}
		a := *c.Aceptacion
		a.Version = 1
		if err := a.Validar(Derivacion{Estado: DerivacionRecibida, Datos: c.Datos}); err != nil {
			return err
		}
		if a.FechaActa < c.FechaDerivacion || a.FechaActa > FechaCivil(ahora) || a.FechaActa > "2026-12-31" || c.Episodios[0].Apertura != a.FechaActa || c.Episodios[0].MotivoReapertura != "" {
			return ErrCargaHistorica
		}
		cubre2026 := false
		for i, e := range c.Episodios {
			f, err := time.Parse("2006-01-02", e.Apertura)
			if err != nil || f.Year() < 1 || e.Apertura > FechaCivil(ahora) || e.Apertura > "2026-12-31" {
				return ErrCargaHistorica
			}
			if i > 0 && (c.Episodios[i-1].Cierre == "" || e.Apertura < c.Episodios[i-1].Cierre || strings.TrimSpace(e.MotivoReapertura) == "" || len(e.MotivoReapertura) > 4000) {
				return ErrCargaHistorica
			}
			if e.Cierre != "" {
				_, err := time.Parse("2006-01-02", e.Cierre)
				if err != nil || e.Cierre < e.Apertura || e.Cierre > FechaCivil(ahora) || e.Cierre > "2026-12-31" || e.MotivoCierre == "" || len(e.MotivoActaTexto) > 4000 {
					return ErrCargaHistorica
				}
			} else if i != len(c.Episodios)-1 || e.MotivoCierre != "" || e.MotivoActaTexto != "" {
				return ErrCargaHistorica
			}
			if e.Apertura <= "2026-12-31" && (e.Cierre == "" || e.Cierre >= "2026-01-01") {
				cubre2026 = true
			}
		}
		if !cubre2026 || (c.Episodios[len(c.Episodios)-1].Cierre != "") != (c.Estado == Cerrada) {
			return ErrCargaHistorica
		}
		for _, cambio := range c.Familia {
			if cambio.Adultos != nil {
				ids := []string{}
				for _, adulto := range *cambio.Adultos {
					ids = append(ids, adulto.ID)
				}
				if !identidadesHistoricasValidas(ids) {
					return ErrFamilia
				}
			}
		}
		for _, cambio := range c.Menores {
			if cambio.Menores != nil {
				ids := []string{}
				for _, menor := range *cambio.Menores {
					ids = append(ids, menor.ID)
				}
				if !identidadesHistoricasValidas(ids) {
					return ErrMenores
				}
			}
		}
		for _, fechas := range [][]string{fechasFamilia(c.Familia, false), fechasFamilia(c.Familia, true), fechasMenores(c.Menores)} {
			previa := ""
			if len(fechas) > 100 {
				return ErrCargaHistorica
			}
			for _, f := range fechas {
				if f < a.FechaActa || f > "2026-12-31" || f > FechaCivil(ahora) || f <= previa {
					return ErrCargaHistorica
				}
				previa = f
			}
		}
	}
	return nil
}
func fechasFamilia(cs []CambioFamilia, adultos bool) []string {
	fs := []string{}
	for _, c := range cs {
		if (c.Adultos != nil) == adultos {
			fs = append(fs, c.Desde)
		}
	}
	return fs
}
func fechasMenores(cs []CambioMenores) []string {
	fs := make([]string, len(cs))
	for i, c := range cs {
		fs[i] = c.Desde
	}
	return fs
}

func identidadesHistoricasValidas(ids []string) bool {
	vistos := map[string]bool{}
	for _, id := range ids {
		if id == "" {
			continue
		}
		if !IDValido(id) || vistos[id] {
			return false
		}
		vistos[id] = true
	}
	return true
}
