package lectura

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/anexos/docx"
	"io"
	"path"
	"strings"

	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente"
)

type Validador struct{}

var _ expediente.ValidadorFichero = Validador{}

func (Validador) Validar(ctx context.Context, nombre string, b []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if len(b) > dominio.MaxBytes {
		return "", dominio.ErrTamano
	}
	switch strings.ToLower(path.Ext(nombre)) {
	case ".pdf":
		if len(b) < 100 || !bytes.HasPrefix(b, []byte("%PDF-1.")) || !bytes.HasSuffix(bytes.TrimSpace(b), []byte("%%EOF")) {
			return "", dominio.ErrFormato
		}
		return "pdf", nil
	case ".docx":
		if err := validarDOCX(ctx, b); err != nil {
			return "", err
		}
		return "docx", nil
	case ".docm":
		return "", errors.New("anexo.docx_macros")
	default:
		return "", dominio.ErrFormato
	}
}

func validarDOCX(ctx context.Context, b []byte) error {
	if len(b) < 4 || !bytes.Equal(b[:4], []byte{'P', 'K', 3, 4}) {
		return dominio.ErrFormato
	}
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return errors.New("anexo.docx_zip_invalido")
	}
	if len(z.File) > 256 {
		return errors.New("anexo.docx_limite_zip")
	}
	var total uint64
	vistos := map[string]bool{}
	documento, tipo := false, false
	for _, f := range z.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		n := f.Name
		bajo := strings.ToLower(n)
		if strings.Contains(bajo, "vbaproject") {
			return errors.New("anexo.docx_macros")
		}
		if n != path.Clean(n) || strings.HasPrefix(n, "/") || strings.HasPrefix(n, "../") || strings.Contains(n, "\\") || strings.ContainsRune(n, 0) || vistos[n] || !f.Mode().IsRegular() {
			return errors.New("anexo.docx_zip_invalido")
		}
		vistos[n] = true
		if !partePermitida(n) {
			return dominio.ErrFormato
		}
		const maxExpandido = 32 << 20 // Coincide con la cota más estricta del lector DOCX.
		if f.UncompressedSize64 > maxExpandido || total > maxExpandido-f.UncompressedSize64 || (f.UncompressedSize64 > 1<<20 && (f.CompressedSize64 == 0 || f.UncompressedSize64/f.CompressedSize64 > 200)) {
			return errors.New("anexo.docx_limite_descompresion")
		}
		total += f.UncompressedSize64
		r, err := f.Open()
		if err != nil {
			return errors.New("anexo.docx_zip_invalido")
		}
		datos, fallo := io.ReadAll(io.LimitReader(r, int64(f.UncompressedSize64)+1))
		cierre := r.Close()
		if fallo != nil || cierre != nil || uint64(len(datos)) != f.UncompressedSize64 {
			return errors.New("anexo.docx_zip_invalido")
		}
		if path.Ext(n) == ".xml" || path.Ext(n) == ".rels" {
			if err := docx.RevisarMarcado(datos); err != nil {
				return err
			}
			d := xml.NewDecoder(bytes.NewReader(datos))
			profundidad, nodos := 0, 0
			for {
				t, err := d.Token()
				if err == io.EOF {
					break
				}
				if err != nil {
					return errors.New("anexo.docx_xml_invalido")
				}
				nodos++
				if nodos > 200000 {
					return errors.New("anexo.docx_limite_xml")
				}
				switch v := t.(type) {
				case xml.Directive:
					return errors.New("anexo.docx_xml_inseguro")
				case xml.StartElement:
					profundidad++
					if profundidad > 100 {
						return errors.New("anexo.docx_limite_xml")
					}
					if n == "word/document.xml" && v.Name.Local == "document" && v.Name.Space == "http://schemas.openxmlformats.org/wordprocessingml/2006/main" {
						documento = true
					}
					for _, a := range v.Attr {
						if n == "[Content_Types].xml" && a.Name.Local == "ContentType" {
							if strings.Contains(strings.ToLower(a.Value), "macroenabled") {
								return errors.New("anexo.docx_macros")
							}
							if a.Value == "application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml" {
								tipo = true
							}
						}
					}
				case xml.EndElement:
					profundidad--
				}
			}
		}
		clear(datos)
	}
	if !documento || !tipo {
		return dominio.ErrFormato
	}
	return nil
}

func partePermitida(n string) bool {
	if n == "[Content_Types].xml" || n == "_rels/.rels" {
		return true
	}
	if !strings.HasPrefix(n, "word/") && !strings.HasPrefix(n, "docProps/") {
		return false
	}
	if strings.Contains(n, "/embeddings/") {
		return false
	}
	switch strings.ToLower(path.Ext(n)) {
	case ".xml", ".rels", ".png", ".jpg", ".jpeg", ".wmf", ".emf", ".gif", ".bmp", ".tiff":
		return true
	}
	return false
}
