package ficheros

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/xml"
	"image/jpeg"
	"io"
	"io/fs"
	"maps"
	"path"
	"strings"

	dominio "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/paquete"
)

var _ salida.ValidadorInstalacion = (*Paquetes)(nil)

const maxZIPPaquete = 20 << 20

// presupuestoZIP se comparte con las plantillas internas: un DOCX anidado
// no puede reiniciar los límites de descompresión ni el número de entradas.
type presupuestoZIP struct{ bytes, entradas uint64 }

func rutaZIPValida(nombre string) bool {
	return fs.ValidPath(nombre) && !strings.ContainsAny(nombre, "\\:\x00") &&
		!strings.HasPrefix(nombre, ".") && !strings.Contains(nombre, "/.")
}

func abrirZIP(ctx context.Context, datos []byte, presupuesto *presupuestoZIP, docx bool) (map[string][]byte, error) {
	z, err := zip.NewReader(bytes.NewReader(datos), int64(len(datos)))
	if err != nil {
		return nil, dominio.ErrPaqueteInvalido
	}
	archivos := map[string][]byte{}
	vistas := map[string]bool{}
	for _, f := range z.File {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		presupuesto.entradas++
		nombre := strings.TrimSuffix(f.Name, "/")
		rutaValida := rutaZIPValida(nombre) || (docx && nombre == "_rels/.rels")
		if presupuesto.entradas > 1000 || !rutaValida || vistas[nombre] ||
			f.Mode()&fs.ModeSymlink != 0 || (!f.Mode().IsRegular() && !f.Mode().IsDir()) ||
			f.UncompressedSize64 > (100<<20)-presupuesto.bytes {
			return nil, dominio.ErrPaqueteInvalido
		}
		vistas[nombre] = true
		if f.FileInfo().IsDir() {
			if f.UncompressedSize64 != 0 {
				return nil, dominio.ErrPaqueteInvalido
			}
			archivos[nombre+"/"] = nil
			continue
		}
		r, err := f.Open()
		if err != nil {
			return nil, dominio.ErrPaqueteInvalido
		}
		b, err := io.ReadAll(io.LimitReader(r, int64((100<<20)-presupuesto.bytes)+1))
		errCierre := r.Close()
		if err != nil || errCierre != nil {
			return nil, dominio.ErrPaqueteInvalido
		}
		presupuesto.bytes += uint64(len(b))
		if presupuesto.bytes > 100<<20 {
			return nil, dominio.ErrPaqueteInvalido
		}
		archivos[nombre] = b
	}
	// Un archivo no puede ser también el padre de otro archivo o directorio.
	for nombre := range archivos {
		for padre := path.Dir(strings.TrimSuffix(nombre, "/")); padre != "."; padre = path.Dir(padre) {
			if _, existe := archivos[padre]; existe {
				return nil, dominio.ErrPaqueteInvalido
			}
		}
	}
	return archivos, nil
}

func (p *Paquetes) Validar(ctx context.Context, tipo string, datos []byte) (dominio.ContenidoInstalacion, error) {
	vacio := dominio.ContenidoInstalacion{}
	if len(datos) > maxZIPPaquete || (tipo != dominio.TipoTema && tipo != dominio.TipoIdioma) {
		return vacio, dominio.ErrPaqueteInvalido
	}
	presupuesto := &presupuestoZIP{}
	archivos, err := abrirZIP(ctx, datos, presupuesto, false)
	if err != nil {
		return vacio, err
	}
	contenido := dominio.ContenidoInstalacion{Tipo: tipo, Archivos: archivos}
	if tipo == dominio.TipoTema {
		var tema dominio.Tema
		var tokens map[string]string
		if decodificarJSON(archivos["tema.json"], &tema) != nil || !tema.Valido() ||
			decodificarJSON(archivos["tokens.json"], &tokens) != nil || len(tokens) == 0 {
			return vacio, dominio.ErrPaqueteInvalido
		}
		base, err := p.Tokens(ctx, dominio.TemaPredeterminado)
		if err != nil {
			return vacio, err
		}
		base = maps.Clone(base)
		maps.Copy(base, tokens)
		if !dominio.TokensValidos(base, tema.Tipo) {
			return vacio, dominio.ErrPaqueteInvalido
		}
		for _, ruta := range tema.Fuentes {
			if _, existe := archivos[ruta]; !existe {
				return vacio, dominio.ErrPaqueteInvalido
			}
		}
		contenido.Codigo, contenido.Version = tema.Identificador, tema.Version
	} else {
		var idioma dominio.Idioma
		if decodificarJSON(archivos["idioma.json"], &idioma) != nil || !idioma.Valido() {
			return vacio, dominio.ErrPaqueteInvalido
		}
		contenido.Codigo, contenido.Version = idioma.Codigo, idioma.Version
	}
	claves := map[string]bool{}
	valores := map[string]string{}
	tamanoTextos := 0
	textos := 0
	for nombre, b := range archivos {
		if err := ctx.Err(); err != nil {
			return vacio, err
		}
		if strings.HasSuffix(nombre, "/") {
			dir := strings.TrimSuffix(nombre, "/")
			permitido := tipo == dominio.TipoTema && (dir == "fuentes" || dir == "iconos") ||
				tipo == dominio.TipoIdioma && (dir == "textos" || dir == "catalogos" || dir == "plantillas" || strings.HasPrefix(dir, "textos/") || strings.HasPrefix(dir, "catalogos/"))
			if !permitido {
				return vacio, dominio.ErrPaqueteInvalido
			}
			continue
		}
		extension := path.Ext(nombre)
		if tipo == dominio.TipoTema {
			if nombre == "tema.json" || nombre == "tokens.json" {
				if len(b) > 1<<20 {
					return vacio, dominio.ErrPaqueteInvalido
				}
				continue
			}
			if extension != ".png" && extension != ".woff2" {
				return vacio, dominio.ErrPaqueteInvalido
			}
			if strings.Contains(nombre, "/") && path.Dir(nombre) != "iconos" && path.Dir(nombre) != "fuentes" {
				return vacio, dominio.ErrPaqueteInvalido
			}
			if _, err := validarRecurso(b, extension); err != nil {
				return vacio, dominio.ErrPaqueteInvalido
			}
		} else {
			if nombre == "idioma.json" {
				if len(b) > 1<<20 {
					return vacio, dominio.ErrPaqueteInvalido
				}
				continue
			}
			if extension == ".json" && (strings.HasPrefix(nombre, "textos/") || strings.HasPrefix(nombre, "catalogos/")) {
				var objeto any
				if len(b) > 1<<20 || decodificarJSON(b, &objeto) != nil {
					return vacio, dominio.ErrPaqueteInvalido
				}
				if strings.HasPrefix(nombre, "textos/") {
					var parte map[string]string
					if decodificarJSON(b, &parte) != nil || parte == nil {
						return vacio, dominio.ErrPaqueteInvalido
					}
					for clave := range parte {
						if !claveTexto.MatchString(clave) || claves[clave] {
							return vacio, dominio.ErrPaqueteInvalido
						}
						claves[clave] = true
						valores[clave] = parte[clave]
						tamanoTextos += len(clave) + len(parte[clave])
					}
					textos++
					if textos > 64 || tamanoTextos > 2<<20 {
						return vacio, dominio.ErrPaqueteInvalido
					}
				}
				continue
			}
			if extension == ".docx" && path.Dir(nombre) == "plantillas" && len(b) <= 10<<20 {
				if validarDOCX(ctx, b, presupuesto) != nil {
					return vacio, dominio.ErrPaqueteInvalido
				}
				continue
			}
			return vacio, dominio.ErrPaqueteInvalido
		}
	}
	if tipo == dominio.TipoIdioma && textos == 0 {
		return vacio, dominio.ErrPaqueteInvalido
	}
	if tipo == dominio.TipoIdioma && contenido.Codigo == dominio.IdiomaPredeterminado {
		for _, clave := range []string{"comun.titulo", "comun.error_carga", "comun.reintentar"} {
			if strings.TrimSpace(valores[clave]) == "" {
				return vacio, dominio.ErrPaqueteInvalido
			}
		}
	}
	huella := sha256.Sum256(datos)
	contenido.Huella = hex.EncodeToString(huella[:])
	return contenido, nil
}

func validarDOCX(ctx context.Context, datos []byte, presupuesto *presupuestoZIP) error {
	archivos, err := abrirZIP(ctx, datos, presupuesto, true)
	if err != nil {
		return err
	}
	for _, nombre := range []string{"[Content_Types].xml", "_rels/.rels", "word/document.xml"} {
		if len(archivos[nombre]) == 0 {
			return dominio.ErrPaqueteInvalido
		}
	}
	const nsTipos = "http://schemas.openxmlformats.org/package/2006/content-types"
	const nsRelaciones = "http://schemas.openxmlformats.org/package/2006/relationships"
	tipoDocumento, relacionDocumento := false, false
	for nombre, b := range archivos {
		if strings.HasSuffix(nombre, "/") {
			continue
		}
		if strings.Contains(strings.ToLower(nombre), "vba") || strings.Contains(nombre, "embeddings/") {
			return dominio.ErrPaqueteInvalido
		}
		switch path.Ext(nombre) {
		case ".xml", ".rels":
			d := xml.NewDecoder(bytes.NewReader(b))
			profundidad, raices := 0, 0
			for {
				token, err := d.Token()
				if err == io.EOF {
					break
				}
				if err != nil {
					return dominio.ErrPaqueteInvalido
				}
				if _, esDirectiva := token.(xml.Directive); esDirectiva {
					return dominio.ErrPaqueteInvalido
				}
				if elemento, ok := token.(xml.StartElement); ok {
					if profundidad == 0 {
						raices++
						if raices > 1 || (nombre == "word/document.xml" && elemento.Name.Local != "document") ||
							(nombre == "[Content_Types].xml" && elemento.Name.Local != "Types") ||
							(path.Ext(nombre) == ".rels" && elemento.Name.Local != "Relationships") {
							return dominio.ErrPaqueteInvalido
						}
						if nombre == "[Content_Types].xml" && elemento.Name.Space != nsTipos ||
							path.Ext(nombre) == ".rels" && elemento.Name.Space != nsRelaciones ||
							nombre == "word/document.xml" && elemento.Name.Space != "http://schemas.openxmlformats.org/wordprocessingml/2006/main" && elemento.Name.Space != "http://purl.oclc.org/ooxml/wordprocessingml/main" {
							return dominio.ErrPaqueteInvalido
						}
					}
					profundidad++
					atributos := map[string]string{}
					atributosVistos := map[xml.Name]bool{}
					for _, atributo := range elemento.Attr {
						if atributosVistos[atributo.Name] {
							return dominio.ErrPaqueteInvalido
						}
						atributosVistos[atributo.Name] = true
						if atributo.Name.Space != "" {
							continue
						}
						atributos[atributo.Name.Local] = atributo.Value
						if atributo.Name.Local == "TargetMode" && atributo.Value != "Internal" ||
							atributo.Name.Local == "ContentType" && strings.Contains(strings.ToLower(atributo.Value), "macro") ||
							atributo.Name.Local == "Target" && strings.ContainsAny(atributo.Value, "\\:") {
							return dominio.ErrPaqueteInvalido
						}
					}
					if nombre == "[Content_Types].xml" && profundidad == 2 && elemento.Name == (xml.Name{Space: nsTipos, Local: "Override"}) && atributos["PartName"] == "/word/document.xml" {
						if tipoDocumento || atributos["ContentType"] != "application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml" {
							return dominio.ErrPaqueteInvalido
						}
						tipoDocumento = true
					}
					if nombre == "_rels/.rels" && profundidad == 2 && elemento.Name == (xml.Name{Space: nsRelaciones, Local: "Relationship"}) &&
						(atributos["Type"] == "http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" || atributos["Type"] == "http://purl.oclc.org/ooxml/officeDocument/relationships/officeDocument") {
						if relacionDocumento || (atributos["Target"] != "word/document.xml" && atributos["Target"] != "/word/document.xml") || atributos["Id"] == "" {
							return dominio.ErrPaqueteInvalido
						}
						relacionDocumento = true
					}
				}
				if _, ok := token.(xml.EndElement); ok {
					profundidad--
				}
				if contenido, ok := token.(xml.CharData); ok && profundidad == 0 && len(bytes.TrimSpace(contenido)) > 0 {
					return dominio.ErrPaqueteInvalido
				}
			}
			if raices != 1 || profundidad != 0 {
				return dominio.ErrPaqueteInvalido
			}
		case ".png":
			if _, err := validarRecurso(b, ".png"); err != nil {
				return dominio.ErrPaqueteInvalido
			}
		case ".jpg", ".jpeg":
			cfg, err := jpeg.DecodeConfig(bytes.NewReader(b))
			if err != nil || cfg.Width > 4096 || cfg.Height > 4096 || cfg.Width*cfg.Height > 16<<20 {
				return dominio.ErrPaqueteInvalido
			}
		case ".wmf":
			if !strings.HasPrefix(nombre, "word/media/") || len(b) < 18 ||
				(binary.LittleEndian.Uint32(b[:4]) != 0x9ac6cdd7 &&
					!(binary.LittleEndian.Uint16(b[:2]) <= 2 && binary.LittleEndian.Uint16(b[2:4]) == 9)) {
				return dominio.ErrPaqueteInvalido
			}
		case ".emf":
			if !strings.HasPrefix(nombre, "word/media/") || len(b) < 88 || binary.LittleEndian.Uint32(b[:4]) != 1 || string(b[40:44]) != " EMF" {
				return dominio.ErrPaqueteInvalido
			}
		default:
			return dominio.ErrPaqueteInvalido
		}
	}
	if !tipoDocumento || !relacionDocumento {
		return dominio.ErrPaqueteInvalido
	}
	return nil
}
