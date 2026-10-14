package expediente

import (
	"context"
	"encoding/json"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"strings"
	"testing"
	"time"
)

type historicosResumen struct {
	entrada.CargasHistoricas
	fallo error
}

func (h historicosResumen) RechazarCargaHistorica(context.Context, entrada.Actor) error {
	return h.fallo
}

type lectorResumen struct{}

func (lectorResumen) LeerHoja(context.Context, string, []byte) ([]d.FilaHoja, error) {
	return []d.FilaHoja{{Numero: 3, Caso: d.CasoHistorico{NumeroHistorico: d.NumeroHistorico{Numero: "dato-personal"}, Estado: "estado-inventado", Datos: d.DatosDerivacion{Extraccion: d.Extraccion{Campos: []d.Campo{{Codigo: "menores.persona.nombre", Valor: "Clara Villena Robles"}}}}}, Incidencias: []d.IncidenciaHoja{{Fila: 3, Clave: "expediente.carga_historica_invalida"}}}}, nil
}

type antivirusResumen struct{}

func (antivirusResumen) Revisar(context.Context, []byte) (plataforma.EstadoAntivirus, error) {
	return plataforma.AntivirusLimpio, nil
}

type relojResumen struct{}

func (relojResumen) Ahora() time.Time { return time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC) }
func TestVistaPreviaHojaNoEntregaNombresNiValoresInvalidos(t *testing.T) {
	s := ServicioHoja{Historicos: historicosResumen{}, Lector: lectorResumen{}, Antivirus: antivirusResumen{}, Reloj: relojResumen{}}
	a := d.Actor{UsuarioID: "11111111-1111-4111-8111-111111111111", Perfiles: []string{"coordinacion"}}
	u := entrada.SubidaHoja{EquipoID: "22222222-2222-4222-8222-222222222222", Formato: "csv", Contenido: []byte("datos")}
	r, err := s.ValidarHoja(context.Background(), a, u)
	if err != nil || r.Valida || r.LoteID != "" || len(r.Filas) != 1 || r.Filas[0].Fila != 3 || r.Filas[0].Iniciales != "C. V. R." || r.Filas[0].Numero != "" || r.Filas[0].Estado != "" {
		t.Fatal(r, err)
	}
	b, _ := json.Marshal(r)
	for _, prohibido := range []string{"Clara", "Villena", "Robles", "dato-personal", "estado-inventado"} {
		if strings.Contains(string(b), prohibido) {
			t.Fatal("dato personal en vista previa")
		}
	}
	s.Historicos = historicosResumen{fallo: errors.New("prueba.auditoria")}
	r, err = s.ValidarHoja(context.Background(), a, u)
	if err == nil || len(r.Filas) != 0 {
		t.Fatal("vista previa entregada sin confirmar auditoría")
	}
}
