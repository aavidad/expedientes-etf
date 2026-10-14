package ficheros

import "encoding/binary"

// Directorio WOFF2: etiquetas conocidas, longitudes UIntBase128 y transformaciones.
// No ejecuta la fuente ni interpreta su código de instrucciones.
var etiquetasWOFF2 = []string{
	"cmap", "head", "hhea", "hmtx", "maxp", "name", "OS/2", "post", "cvt ", "fpgm", "glyf", "loca", "prep", "CFF ", "VORG", "EBDT", "EBLC", "gasp", "hdmx", "kern", "LTSH", "PCLT", "VDMX", "vhea", "vmtx", "BASE", "GDEF", "GPOS", "GSUB", "EBSC", "JSTF", "MATH", "CBDT", "CBLC", "COLR", "CPAL", "SVG ", "sbix", "acnt", "avar", "bdat", "bloc", "bsln", "cvar", "fdsc", "feat", "fmtx", "fvar", "gvar", "hsty", "just", "lcar", "mort", "morx", "opbd", "prop", "trak", "Zapf", "Silf", "Glat", "Gloc", "Feat", "Sill",
}

func leerBase128(datos []byte, pos *int) (uint32, bool) {
	var valor uint32
	for i := 0; i < 5; i++ {
		if *pos >= len(datos) {
			return 0, false
		}
		b := datos[*pos]
		*pos++
		if i == 0 && b == 0x80 || valor&0xfe000000 != 0 {
			return 0, false
		}
		valor = (valor << 7) | uint32(b&0x7f)
		if b&0x80 == 0 {
			return valor, true
		}
	}
	return 0, false
}

func woff2Valida(datos []byte) bool {
	if len(datos) < 48 || string(datos[:4]) != "wOF2" || binary.BigEndian.Uint32(datos[8:12]) != uint32(len(datos)) || binary.BigEndian.Uint16(datos[14:16]) != 0 {
		return false
	}
	flavor := binary.BigEndian.Uint32(datos[4:8])
	if flavor != 0x00010000 && flavor != 0x4f54544f {
		return false
	} // TrueType u OpenType CFF; no colecciones.
	numero := int(binary.BigEndian.Uint16(datos[12:14]))
	sfnt := uint64(binary.BigEndian.Uint32(datos[16:20]))
	comprimido := uint64(binary.BigEndian.Uint32(datos[20:24]))
	if numero < 1 || numero > 256 || sfnt > 100<<20 || comprimido == 0 {
		return false
	}
	pos := 48
	tamano := uint64(12 + 16*numero)
	expandido := uint64(0)
	tablas := map[string]uint32{}
	transformaciones := map[string]byte{}
	for range numero {
		if pos >= len(datos) {
			return false
		}
		flags := datos[pos]
		pos++
		indice := int(flags & 63)
		tag := ""
		if indice == 63 {
			if pos+4 > len(datos) {
				return false
			}
			tag = string(datos[pos : pos+4])
			pos += 4
		} else {
			tag = etiquetasWOFF2[indice]
		}
		if _, existe := tablas[tag]; existe {
			return false
		}
		original, ok := leerBase128(datos, &pos)
		if !ok || original == 0 {
			return false
		}
		version := flags >> 6
		transformada := false
		if tag == "glyf" || tag == "loca" {
			if version != 0 && version != 3 {
				return false
			}
			transformada = version == 0
		} else if tag == "hmtx" {
			if version > 1 {
				return false
			}
			transformada = version == 1
		} else if version != 0 {
			return false
		}
		longitud := original
		if transformada {
			longitud, ok = leerBase128(datos, &pos)
			if !ok || (tag == "loca" && longitud != 0) || (tag != "loca" && longitud == 0) {
				return false
			}
		}
		tablas[tag] = original
		transformaciones[tag] = version
		tamano += (uint64(original) + 3) &^ uint64(3)
		expandido += uint64(longitud)
		if tamano > 100<<20 || expandido > 100<<20 {
			return false
		}
	}
	if tamano != sfnt || expandido == 0 || comprimido > uint64(len(datos)-pos) {
		return false
	}
	for tag, minimo := range map[string]uint32{"head": 54, "hhea": 36, "maxp": 6, "hmtx": 4, "cmap": 4, "name": 6} {
		if tablas[tag] < minimo {
			return false
		}
	}
	if flavor == 0x00010000 {
		if tablas["glyf"] == 0 || tablas["loca"] == 0 || transformaciones["glyf"] != transformaciones["loca"] {
			return false
		}
	} else if tablas["CFF "] == 0 && tablas["CFF2"] == 0 {
		return false
	}
	final := uint64(pos) + comprimido
	// Metadatos y datos privados deben ocupar bloques separados y acotados.
	metaOffset := uint64(binary.BigEndian.Uint32(datos[28:32]))
	metaLen := uint64(binary.BigEndian.Uint32(datos[32:36]))
	metaOriginal := uint64(binary.BigEndian.Uint32(datos[36:40]))
	privadoOffset := uint64(binary.BigEndian.Uint32(datos[40:44]))
	privadoLen := uint64(binary.BigEndian.Uint32(datos[44:48]))
	for _, bloque := range [][2]uint64{{metaOffset, metaLen}, {privadoOffset, privadoLen}} {
		if bloque[0] == 0 {
			if bloque[1] != 0 {
				return false
			}
			continue
		}
		if bloque[1] == 0 || bloque[0] < final || bloque[0] > final+3 || bloque[0]+bloque[1] > uint64(len(datos)) {
			return false
		}
		for _, b := range datos[final:bloque[0]] {
			if b != 0 {
				return false
			}
		}
		final = bloque[0] + bloque[1]
	}
	if metaOffset == 0 && metaOriginal != 0 || metaOffset != 0 && (metaOriginal == 0 || metaOriginal > 100<<20) {
		return false
	}
	if final > uint64(len(datos)) || uint64(len(datos))-final > 3 {
		return false
	}
	for _, b := range datos[final:] {
		if b != 0 {
			return false
		}
	}
	return true
}
