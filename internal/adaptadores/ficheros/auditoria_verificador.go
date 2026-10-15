package ficheros

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"

	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
)

// VerificarPaqueteAuditoria no extrae archivos ni abre conexiones. Comprueba
// cadena, huellas de archivos y que el registro sea la selección completa.
func VerificarPaqueteAuditoria(r io.ReaderAt, tamano int64) (auditoria.ManifiestoExportacion, error) {
	fallo := errors.New("auditoria.formato_invalido")
	var vacio auditoria.ManifiestoExportacion
	if tamano < 0 || tamano > auditoria.MaxBytesExportacion {
		return vacio, fallo
	}
	z, err := zip.NewReader(r, tamano)
	if err != nil || len(z.File) != 4 {
		return vacio, fallo
	}
	archivos := map[string]*zip.File{}
	var total uint64
	for _, f := range z.File {
		if archivos[f.Name] != nil || !f.Mode().IsRegular() {
			return vacio, fallo
		}
		archivos[f.Name] = f
		total += f.UncompressedSize64
		if total > auditoria.MaxBytesExportacion+65536 {
			return vacio, fallo
		}
	}
	leer := func(nombre string, limite int64) ([]byte, error) {
		f := archivos[nombre]
		if f == nil || f.UncompressedSize64 > uint64(limite) {
			return nil, fallo
		}
		r, err := f.Open()
		if err != nil {
			return nil, fallo
		}
		defer r.Close()
		b, err := io.ReadAll(io.LimitReader(r, limite+1))
		if err != nil || int64(len(b)) > limite {
			return nil, fallo
		}
		return b, nil
	}
	b, err := leer("manifiesto.json", 8192)
	if err != nil {
		return vacio, err
	}
	m, err := auditoria.DecodificarManifiesto(b)
	if err != nil {
		return vacio, err
	}
	if archivos["cadena.json"] == nil || archivos["registro."+m.Formato] == nil || archivos["LEEME.txt"] == nil {
		return vacio, fallo
	}
	if _, err = leer("LEEME.txt", 32769); err != nil {
		return vacio, err
	}
	prueba, err := archivos["cadena.json"].Open()
	if err != nil {
		return vacio, fallo
	}
	defer prueba.Close()
	hp, esperado := sha256.New(), sha256.New()
	filas, eventos := 0, 0
	var tramo auditoria.TramoExportacion
	err = escribirRegistro(esperado, m.Formato, func(seleccionar func(auditoria.Evento) error) error {
		var err error
		tramo, eventos, err = auditoria.RecorrerExportacion(io.TeeReader(prueba, hp), func(e auditoria.Evento) error {
			if m.Filtro.Incluye(e) {
				filas++
				return seleccionar(e)
			}
			return nil
		})
		return err
	})
	if err != nil {
		return vacio, err
	}
	registro, err := archivos["registro."+m.Formato].Open()
	if err != nil {
		return vacio, fallo
	}
	defer registro.Close()
	hr := sha256.New()
	n, err := io.Copy(hr, io.LimitReader(registro, auditoria.MaxBytesExportacion+1))
	if err != nil || n > auditoria.MaxBytesExportacion {
		return vacio, fallo
	}
	if tramo != m.Tramo || eventos != m.EventosCadena || filas != m.Filas || hex.EncodeToString(hp.Sum(nil)) != m.HuellaCadena || hex.EncodeToString(hr.Sum(nil)) != m.HuellaRegistro || hex.EncodeToString(esperado.Sum(nil)) != m.HuellaRegistro {
		return vacio, errors.New("auditoria.cadena_alterada")
	}
	return m, nil
}
