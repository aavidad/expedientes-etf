package expediente_test

import (
	aplicacion "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"os"
	"sync"
	"testing"
)

func TestIntegracionConfirmacionConcurrenteSinDuplicados(t *testing.T) {
	_, ctx, deps, a := preparar(t)
	s := aplicacion.Nuevo(deps)
	b, err := os.ReadFile("../../anexos/docx/testdata/anexo3.docx")
	if err != nil {
		t.Fatal(err)
	}
	d, err := s.Subir(ctx, a, entrada.Subida{Nombre: "Derivacion.docx", Contenido: b})
	if err != nil {
		t.Fatal(err)
	}
	if d.ID != "" || d.Estado != "" {
		t.Fatal("análisis creó derivación")
	}
	o := d.Datos.OpcionesUbicacion
	p := entrada.Confirmacion{LecturaID: d.Lectura.ID, Tipo: d.Datos.Extraccion.Tipo, Campos: d.Datos.Extraccion.Campos, Ubicacion: dominio.UbicacionRevisada{MunicipioID: o.Municipios[0].ID, CentroID: o.Centros[0].ID, EquipoID: o.Equipos[0].ID}}
	clave := "confirmar-" + d.Lectura.ID
	recibos := make([]dominio.ReciboDerivacion, 2)
	fallos := make([]error, 2)
	var espera sync.WaitGroup
	for i := range recibos {
		espera.Add(1)
		go func(i int) { defer espera.Done(); recibos[i], fallos[i] = s.ConfirmarDerivacion(ctx, a, clave, p) }(i)
	}
	espera.Wait()
	if fallos[0] != nil || fallos[1] != nil || recibos[0] != recibos[1] {
		t.Fatal(recibos, fallos)
	}
	guardada, err := s.Consultar(ctx, a, recibos[0].ID)
	if err != nil || guardada.Numero != nil || guardada.Estado != dominio.DerivacionRecibida || guardada.Datos.UbicacionConfirmada == nil {
		t.Fatal(guardada, err)
	}
	p.Campos[0].Valor = "Centro de Servicios Sociales de Montefrío"
	if _, err := s.ConfirmarDerivacion(ctx, a, clave, p); err != dominio.ErrIdempotencia {
		t.Fatal("reutilización de clave no rechazada", err)
	}
	t.Log("POST de confirmación concurrente: mismo id, sin número; corrección incompatible rechazada")
}
