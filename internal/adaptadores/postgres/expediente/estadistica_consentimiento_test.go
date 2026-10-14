package expediente

import (
	"testing"
	"time"
)

func TestConsentimientoReaperturaMismoDia(t *testing.T) {
	dia := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	episodios := []episodioLectura{
		{ID: "nuevo", Reapertura: "anterior", Apertura: fechaLectura{dia}, Creado: dia.Add(12 * time.Hour)},
		{ID: "anterior", Apertura: fechaLectura{dia}, Creado: dia.Add(8 * time.Hour)},
	}
	hitos := []hitoLectura{{EpisodioID: "anterior", Tipo: "consentimiento", Fecha: fechaLectura{dia}, Confirmado: dia.Add(9 * time.Hour)}}
	conocido := dia.Add(24 * time.Hour)
	historia := historiaConsentimiento(episodios, hitos)
	dato, _, err := historia.ValorEn(dia, conocido)
	if err != nil || dato.Conocido {
		t.Fatalf("la reapertura sin firma debe quedar sin dato: %+v, %v", dato, err)
	}
	hitos = append(hitos, hitoLectura{EpisodioID: "nuevo", Tipo: "consentimiento", Fecha: fechaLectura{dia}, Confirmado: dia.Add(13 * time.Hour)})
	historia = historiaConsentimiento(episodios, hitos)
	dato, _, err = historia.ValorEn(dia, conocido)
	if err != nil || !dato.Conocido || !dato.Valor {
		t.Fatalf("la firma del episodio nuevo debe contar: %+v, %v", dato, err)
	}
}
