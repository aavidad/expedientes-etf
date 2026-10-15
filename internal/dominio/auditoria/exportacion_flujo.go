package auditoria

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

// RecorrerExportacion comprueba el formato v1 con memoria proporcional a un
// evento. Los metadatos preceden a eventos; el archivo generado respeta ese orden.
func RecorrerExportacion(r io.Reader, consumir func(Evento) error) (TramoExportacion, int, error) {
	d := json.NewDecoder(io.LimitReader(r, MaxBytesExportacion+1))
	d.DisallowUnknownFields()
	invalido := errors.New("auditoria.formato_invalido")
	t, err := d.Token()
	if err != nil || t != json.Delim('{') {
		return TramoExportacion{}, 0, invalido
	}
	claves := map[string]bool{}
	var e Exportacion
	cantidad := 0
	for d.More() {
		token, err := d.Token()
		if err != nil {
			return TramoExportacion{}, 0, invalido
		}
		clave, ok := token.(string)
		if !ok || claves[clave] {
			return TramoExportacion{}, 0, invalido
		}
		claves[clave] = true
		switch clave {
		case "version":
			err = d.Decode(&e.Version)
		case "algoritmo":
			err = d.Decode(&e.Algoritmo)
		case "canonizacion":
			err = d.Decode(&e.Canonizacion)
		case "orden":
			err = d.Decode(&e.Orden)
		case "anterior", "final":
			var raw json.RawMessage
			err = d.Decode(&raw)
			if err == nil {
				err = validarJSON(raw)
			}
			if err == nil {
				if clave == "anterior" {
					pd := json.NewDecoder(bytes.NewReader(raw))
					pd.DisallowUnknownFields()
					err = pd.Decode(&e.Anterior)
				} else {
					pd := json.NewDecoder(bytes.NewReader(raw))
					pd.DisallowUnknownFields()
					err = pd.Decode(&e.Final)
				}
			}
		case "eventos":
			if len(claves) != 7 || e.Version != VersionExportacion || e.Algoritmo != "sha256" || e.Canonizacion != "etf.auditoria.binaria.v1" || e.Orden != "id_ascendente" || e.Anterior.Validar() != nil || e.Final.Validar() != nil {
				return TramoExportacion{}, 0, invalido
			}
			token, err = d.Token()
			if err != nil || token != json.Delim('[') {
				return TramoExportacion{}, 0, invalido
			}
			previo := e.Anterior
			for d.More() {
				var raw json.RawMessage
				if err = d.Decode(&raw); err != nil || len(raw) > 32<<10 {
					return TramoExportacion{}, 0, invalido
				}
				if err = validarJSON(raw); err != nil {
					return TramoExportacion{}, 0, invalido
				}
				var evento Evento
				eventoDecoder := json.NewDecoder(bytes.NewReader(raw))
				eventoDecoder.DisallowUnknownFields()
				if err = eventoDecoder.Decode(&evento); err != nil || evento.Perfiles == nil {
					return TramoExportacion{}, 0, invalido
				}
				if err = Verificar([]Evento{evento}, previo, evento.Punto()); err != nil {
					return TramoExportacion{}, 0, err
				}
				if evento.ID > e.Final.ID {
					return TramoExportacion{}, 0, invalido
				}
				previo = evento.Punto()
				cantidad++
				if cantidad > MaxEventosExportacion {
					return TramoExportacion{}, 0, errors.New("auditoria.exportacion_demasiado_grande")
				}
				if consumir != nil {
					if err = consumir(evento); err != nil {
						return TramoExportacion{}, 0, err
					}
				}
			}
			if _, err = d.Token(); err != nil {
				return TramoExportacion{}, 0, invalido
			}
			if previo != e.Final {
				return TramoExportacion{}, 0, errors.New("auditoria.sello_no_coincide")
			}
		default:
			return TramoExportacion{}, 0, invalido
		}
		if err != nil {
			return TramoExportacion{}, 0, invalido
		}
	}
	if len(claves) != 7 {
		return TramoExportacion{}, 0, invalido
	}
	if _, err = d.Token(); err != nil {
		return TramoExportacion{}, 0, invalido
	}
	if _, err = d.Token(); err != io.EOF || d.InputOffset() > MaxBytesExportacion {
		return TramoExportacion{}, 0, invalido
	}
	return TramoExportacion{Anterior: e.Anterior, Final: e.Final}, cantidad, nil
}

func DecodificarManifiesto(contenido []byte) (ManifiestoExportacion, error) {
	var m ManifiestoExportacion
	if len(contenido) > 8192 || validarJSON(contenido) != nil {
		return m, errors.New("auditoria.formato_invalido")
	}
	d := json.NewDecoder(bytes.NewReader(contenido))
	d.DisallowUnknownFields()
	if err := d.Decode(&m); err != nil {
		return m, errors.New("auditoria.formato_invalido")
	}
	if m.Version != "etf.auditoria.paquete.v1" || (m.Formato != "csv" && m.Formato != "json") || m.Filtro.Validar() != nil || m.Tramo.Anterior.Validar() != nil || m.Tramo.Final.Validar() != nil || m.Filas < 0 || m.EventosCadena < 0 || m.EventosCadena > MaxEventosExportacion || m.Filas > m.EventosCadena || !huellaValida(m.HuellaRegistro) || !huellaValida(m.HuellaCadena) {
		return m, errors.New("auditoria.formato_invalido")
	}
	return m, nil
}
