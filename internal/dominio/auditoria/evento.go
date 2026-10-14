package auditoria

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math"
	"strings"
	"time"
)

const HuellaInicial = "0000000000000000000000000000000000000000000000000000000000000000"

// Datos admite identificadores internos y códigos, nunca contenido de una ficha.
// El actor procede del contexto de confianza del servidor; sistema identifica
// tareas automáticas e intentos de acceso sin usuario interno conocido.
type Datos struct {
	OcurridoEn   time.Time `json:"ocurrido_en"`
	UsuarioID    string    `json:"usuario_id"`
	Perfiles     []string  `json:"perfiles"`
	Accion       string    `json:"accion"`
	Entidad      string    `json:"entidad"`
	EntidadID    string    `json:"entidad_id"`
	ExpedienteID string    `json:"expediente_id"`
	Resultado    string    `json:"resultado"`
}

type Evento struct {
	ID uint64 `json:"id"`
	Datos
	HuellaAnterior string `json:"huella_anterior"`
	Huella         string `json:"huella"`
}

// Punto también representa un sello guardado fuera de la cadena. El punto
// inicial tiene ID cero y 32 bytes cero. Los demás tienen ID positivo.
type Punto struct {
	ID     uint64 `json:"id"`
	Huella string `json:"huella"`
}

func Inicio() Punto { return Punto{Huella: HuellaInicial} }

func (e Evento) Punto() Punto { return Punto{ID: e.ID, Huella: e.Huella} }

func (p Punto) Validar() error {
	if p.ID > math.MaxInt64 || !huellaValida(p.Huella) ||
		(p.ID == 0) != (p.Huella == HuellaInicial) {
		return errors.New("auditoria.punto_invalido")
	}
	return nil
}

func (d Datos) Validar() error {
	if d.OcurridoEn.IsZero() || d.OcurridoEn.UTC().Year() < 1 || d.OcurridoEn.UTC().Year() > 9999 ||
		d.OcurridoEn.Nanosecond()%1000 != 0 {
		return errors.New("auditoria.fecha_invalida")
	}
	if d.UsuarioID != "sistema" && !identificadorValido(d.UsuarioID) {
		return errors.New("auditoria.usuario_invalido")
	}
	if !codigoValido(d.Accion) || !codigoValido(d.Entidad) {
		return errors.New("auditoria.codigo_invalido")
	}
	if (d.EntidadID != "" && !identificadorValido(d.EntidadID)) ||
		(d.ExpedienteID != "" && !identificadorValido(d.ExpedienteID)) {
		return errors.New("auditoria.identificador_invalido")
	}
	if d.Resultado != "ok" && d.Resultado != "denegado" && d.Resultado != "error" {
		return errors.New("auditoria.resultado_invalido")
	}
	if len(d.Perfiles) > 16 {
		return errors.New("auditoria.perfiles_invalidos")
	}
	for i, perfil := range d.Perfiles {
		if !codigoValido(perfil) {
			return errors.New("auditoria.perfiles_invalidos")
		}
		for _, previo := range d.Perfiles[:i] {
			if perfil == previo {
				return errors.New("auditoria.perfiles_invalidos")
			}
		}
	}
	return nil
}

// Encadenar recibe hora e ID: no consulta reloj ni almacenamiento. El ID debe
// crecer, pero puede tener saltos por reservas o transacciones revertidas.
func Encadenar(id uint64, datos Datos, anterior Punto) (Evento, error) {
	if err := anterior.Validar(); err != nil {
		return Evento{}, err
	}
	if id <= anterior.ID || id > math.MaxInt64 {
		return Evento{}, errors.New("auditoria.orden_invalido")
	}
	if err := datos.Validar(); err != nil {
		return Evento{}, err
	}
	datos.OcurridoEn = datos.OcurridoEn.UTC()
	datos.Perfiles = append([]string{}, datos.Perfiles...)
	e := Evento{ID: id, Datos: datos, HuellaAnterior: anterior.Huella}
	e.Huella = calcularHuella(e)
	return e, nil
}

// Forma canónica v1: huella previa (32 bytes), etiqueta UTF-8 con longitud,
// ID uint64 big endian y cadenas UTF-8 con longitud uint64 big endian.
// Orden: etiqueta, ID, fecha UTC con seis decimales, usuario, número de
// perfiles uint64, perfiles en su orden, acción, entidad, entidad_id,
// expediente_id, resultado. No incluye la huella propia.
func calcularHuella(e Evento) string {
	previa, _ := hex.DecodeString(e.HuellaAnterior) // validada antes de llamar
	contenido := append([]byte{}, previa...)
	texto := func(valor string) {
		contenido = binary.BigEndian.AppendUint64(contenido, uint64(len(valor)))
		contenido = append(contenido, valor...)
	}
	texto("etf.auditoria.v1")
	contenido = binary.BigEndian.AppendUint64(contenido, e.ID)
	texto(e.OcurridoEn.UTC().Format("2006-01-02T15:04:05.000000Z"))
	texto(e.UsuarioID)
	contenido = binary.BigEndian.AppendUint64(contenido, uint64(len(e.Perfiles)))
	for _, perfil := range e.Perfiles {
		texto(perfil)
	}
	texto(e.Accion)
	texto(e.Entidad)
	texto(e.EntidadID)
	texto(e.ExpedienteID)
	texto(e.Resultado)
	huella := sha256.Sum256(contenido)
	return hex.EncodeToString(huella[:])
}

func huellaValida(valor string) bool {
	return len(valor) == 64 && hexMinusculo(valor)
}

func hexMinusculo(valor string) bool {
	for _, c := range valor {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func identificadorValido(valor string) bool {
	if len(valor) != 36 || valor == "00000000-0000-0000-0000-000000000000" {
		return false
	}
	for i, c := range valor {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else if !hexMinusculo(string(c)) {
			return false
		}
	}
	return true
}

func codigoValido(valor string) bool {
	if len(valor) == 0 || len(valor) > 128 || valor[0] < 'a' || valor[0] > 'z' ||
		strings.Contains(valor, "..") || strings.HasSuffix(valor, ".") {
		return false
	}
	for _, c := range valor {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == '.') {
			return false
		}
	}
	return true
}
