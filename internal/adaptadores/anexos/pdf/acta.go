package pdf

import (
	dslipak "github.com/dslipak/pdf"
	"strings"
	"time"
)

// fechaFirma lee la fecha declarada en campos de firma. No verifica confianza,
// integridad ni sellos de tiempo. Varias fechas distintas exigen entrada manual.
func fechaFirma(r *dslipak.Reader) string {
	fecha := ""
	ambiguo := false
	restantes := 100
	var visitar func(dslipak.Value, int)
	visitar = func(v dslipak.Value, nivel int) {
		restantes--
		if nivel > 8 || restantes < 0 {
			ambiguo = true
			return
		}
		firma := v.Key("V")
		if v.Key("FT").Name() == "Sig" && firma.Key("Type").Name() == "Sig" && firma.Key("ByteRange").Len() == 4 && firma.Key("Contents").Kind() == dslipak.String {
			valor := firma.Key("M").RawString()
			if strings.HasPrefix(valor, "D:") && len(valor) >= 10 {
				if f, err := time.Parse("20060102", valor[2:10]); err == nil {
					dia := f.Format("2006-01-02")
					if fecha != "" && fecha != dia {
						ambiguo = true
					}
					fecha = dia
				}
			}
		}
		hijos := v.Key("Kids")
		if hijos.Len() > restantes {
			ambiguo = true
			return
		}
		for i := 0; i < hijos.Len(); i++ {
			visitar(hijos.Index(i), nivel+1)
		}
	}
	campos := r.Trailer().Key("Root").Key("AcroForm").Key("Fields")
	if campos.Len() > restantes {
		return ""
	}
	for i := 0; i < campos.Len(); i++ {
		visitar(campos.Index(i), 0)
	}
	if ambiguo {
		return ""
	}
	return fecha
}
