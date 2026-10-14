package catalogo

import "testing"

func TestParsearImporteSMI(t *testing.T) {
	for _, caso := range []struct {
		valor  string
		valido bool
	}{{"950", true}, {"950.5", true}, {"950.50", true}, {"0", false}, {"-1", false}, {"950.501", false}, {"1,25", false}} {
		importe, err := ParsearImporteSMI(caso.valor)
		if (err == nil) != caso.valido {
			t.Errorf("ParsearImporteSMI(%q) = %d, %v", caso.valor, importe, err)
		}
	}
}
