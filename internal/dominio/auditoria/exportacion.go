package auditoria

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

const (
	VersionExportacion  = "etf.auditoria.v1"
	MaxBytesExportacion = 64 * 1024 * 1024
)

// Exportacion es un tramo contiguo de la cadena, nunca una selección filtrada
// por usuario o expediente. Anterior queda fuera del tramo; Final está incluido.
type Exportacion struct {
	Version      string   `json:"version"`
	Algoritmo    string   `json:"algoritmo"`
	Canonizacion string   `json:"canonizacion"`
	Orden        string   `json:"orden"`
	Anterior     Punto    `json:"anterior"`
	Final        Punto    `json:"final"`
	Eventos      []Evento `json:"eventos"`
}

func NuevaExportacion(eventos []Evento, anterior, final Punto) (Exportacion, error) {
	e := Exportacion{
		Version: VersionExportacion, Algoritmo: "sha256",
		Canonizacion: "etf.auditoria.binaria.v1", Orden: "id_ascendente",
		Anterior: anterior, Final: final, Eventos: append([]Evento{}, eventos...),
	}
	for i := range e.Eventos {
		e.Eventos[i].Perfiles = append([]string{}, e.Eventos[i].Perfiles...)
		e.Eventos[i].OcurridoEn = e.Eventos[i].OcurridoEn.UTC()
	}
	return e, e.Verificar()
}

func (e Exportacion) Verificar() error {
	if e.Version != VersionExportacion || e.Algoritmo != "sha256" ||
		e.Canonizacion != "etf.auditoria.binaria.v1" || e.Orden != "id_ascendente" || e.Eventos == nil {
		return errors.New("auditoria.formato_invalido")
	}
	for _, evento := range e.Eventos {
		if evento.Perfiles == nil {
			return errors.New("auditoria.formato_invalido")
		}
	}
	return Verificar(e.Eventos, e.Anterior, e.Final)
}

func (e Exportacion) Codificar() ([]byte, error) {
	if err := e.Verificar(); err != nil {
		return nil, err
	}
	contenido, err := json.Marshal(e)
	if err != nil {
		return nil, errors.New("auditoria.formato_invalido")
	}
	if len(contenido) > MaxBytesExportacion {
		return nil, errors.New("auditoria.exportacion_demasiado_grande")
	}
	return contenido, nil
}

// DecodificarExportacion rechaza ambigüedades (claves duplicadas, null, claves
// desconocidas, más de un documento), sin devolver fragmentos del fichero.
func DecodificarExportacion(contenido []byte) (Exportacion, error) {
	if len(contenido) > MaxBytesExportacion {
		return Exportacion{}, errors.New("auditoria.exportacion_demasiado_grande")
	}
	if err := validarJSON(contenido); err != nil {
		return Exportacion{}, errors.New("auditoria.formato_invalido")
	}
	var e Exportacion
	d := json.NewDecoder(bytes.NewReader(contenido))
	d.DisallowUnknownFields()
	if err := d.Decode(&e); err != nil {
		return Exportacion{}, errors.New("auditoria.formato_invalido")
	}
	if err := e.Verificar(); err != nil {
		return Exportacion{}, err
	}
	return e, nil
}

func validarJSON(contenido []byte) error {
	d := json.NewDecoder(bytes.NewReader(contenido))
	d.UseNumber()
	var valor func(int) error
	valor = func(profundidad int) error {
		if profundidad > 16 {
			return errors.New("auditoria.formato_invalido")
		}
		token, err := d.Token()
		if err != nil || token == nil {
			return errors.New("auditoria.formato_invalido")
		}
		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '{':
				claves := make(map[string]bool)
				for d.More() {
					clave, err := d.Token()
					if err != nil {
						return err
					}
					nombre, ok := clave.(string)
					if !ok || claves[nombre] || strings.ToLower(nombre) != nombre {
						return errors.New("auditoria.formato_invalido")
					}
					claves[nombre] = true
					if err := valor(profundidad + 1); err != nil {
						return err
					}
				}
			case '[':
				for d.More() {
					if err := valor(profundidad + 1); err != nil {
						return err
					}
				}
			default:
				return errors.New("auditoria.formato_invalido")
			}
			_, err = d.Token()
			return err
		}
		return nil
	}
	if err := valor(0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return errors.New("auditoria.formato_invalido")
	}
	return nil
}
