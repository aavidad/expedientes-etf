package pdf

import dslipak "github.com/dslipak/pdf"

// validarEstructura comprueba el árbol antes de Page, que confía en Count.
// Los límites del trabajador cubren además referencias cíclicas y streams.
func validarEstructura(r *dslipak.Reader, limites Limites) error {
	raiz := r.Trailer().Key("Root")
	if raiz.Key("Type").Name() != "Catalog" {
		return ErrCorrupto
	}
	restantes := limites.Objetos
	var visitar func(dslipak.Value, int) (int, error)
	visitar = func(v dslipak.Value, profundidad int) (int, error) {
		restantes--
		if profundidad > 32 || restantes < 0 {
			return 0, ErrRecursos
		}
		switch v.Key("Type").Name() {
		case "Page":
			contenido := v.Key("Contents")
			switch contenido.Kind() {
			case dslipak.Null, dslipak.Stream:
			case dslipak.Array:
				if contenido.Len() > limites.Objetos {
					return 0, ErrObjetos
				}
				for i := 0; i < contenido.Len(); i++ {
					if contenido.Index(i).Kind() != dslipak.Stream {
						return 0, ErrCorrupto
					}
				}
			default:
				return 0, ErrCorrupto
			}
			return 1, nil
		case "Pages":
			count := v.Key("Count")
			if count.Kind() != dslipak.Integer || count.Int64() < 1 {
				return 0, ErrCorrupto
			}
			if count.Int64() > int64(limites.Paginas) {
				return 0, ErrPaginas
			}
			kids := v.Key("Kids")
			if kids.Kind() != dslipak.Array || kids.Len() == 0 {
				return 0, ErrCorrupto
			}
			if kids.Len() > limites.Objetos {
				return 0, ErrObjetos
			}
			total := 0
			for i := 0; i < kids.Len(); i++ {
				n, err := visitar(kids.Index(i), profundidad+1)
				if err != nil {
					return 0, err
				}
				total += n
				if total > limites.Paginas {
					return 0, ErrPaginas
				}
			}
			if int64(total) != count.Int64() {
				return 0, ErrCorrupto
			}
			return total, nil
		default:
			return 0, ErrCorrupto
		}
	}
	_, err := visitar(raiz.Key("Pages"), 0)
	return err
}
