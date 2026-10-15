package documento

import (
	expediente "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"testing"
	"time"
)

func TestAnexo6FichaVigenteTraduceYNoResucitaDatosBorrados(t *testing.T) {
	ingresos := int64(128001)
	papel, estudios, nivel, fecha := "MADRE", "UNIVERSITARIOS", "PRIMARIA", "2016-05-07"
	adultos := []expediente.Adulto{{Nombre: "Clara", Apellidos: "Ruiz Vega", Papel: papel, Educacion: &estudios}}
	f := &expediente.FichaParaAnexo6{Familia: &expediente.Familia{Ingresos: &ingresos}, Adultos: &adultos, Menores: []expediente.Menor{{Nombre: "Marina", Apellidos: "Ruiz Vega", Escolarizacion: &nivel, Nacimiento: &fecha, CentroEducativo: "Colegio del Olivar"}}}
	c := map[string]string{"direccion": "Dirección anterior", "estructura": "BIPARENTAL"}
	filas := map[string][]map[string]string{"contactos": {{"nombre": "Anterior", "telefono": "600123456"}}, "adultos": {{"nombre": "Anterior"}}, "menores": {{"nombre": "Menor anterior"}, {"nombre": "Menor retirado"}}}
	nombres := map[string]string{"papel_parental.MADRE": "Madre", "nivel_educativo.UNIVERSITARIOS": "Estudios universitarios", "nivel_escolarizacion.PRIMARIA": "Educación primaria"}
	rellenarFichaAnexo6(f, c, filas, nombres, time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC), ",")
	if c["ingresos"] != "1280,01" || c["direccion"] != "" || c["estructura"] != "" || len(filas["contactos"]) != 0 || len(filas["menores"]) != 1 || filas["menores"][0]["nombre"] != "Marina Ruiz Vega" || filas["menores"][0]["nacimiento"] != "07/05/2016" || filas["menores"][0]["escolarizacion"] != "Educación primaria" || filas["adultos"][0]["papel"] != "Madre" {
		t.Fatalf("ficha vigente: %v %v", c, filas)
	}
}

func TestAnexo6FallbackSóloEnMenorSinFichaPorSuOrden(t *testing.T) {
	filas := map[string][]map[string]string{"menores": {{"nombre": "Primero"}, {"nombre": "Segundo"}}}
	f := &expediente.FichaParaAnexo6{Menores: []expediente.Menor{{Orden: 2, DatosDesdeAnexo: true}, {Orden: 1}}}
	rellenarFichaAnexo6(f, map[string]string{}, filas, map[string]string{}, time.Now(), ",")
	if filas["menores"][0]["nombre"] != "Segundo" || filas["menores"][1]["nombre"] != "" {
		t.Fatal(filas)
	}
}

func TestAnexo6AdultosGuardadosSinFamiliaSustituyenContactos(t *testing.T) {
	adultos := []expediente.Adulto{{Nombre: "Clara", Apellidos: "Ruiz Vega", Telefono: "600987654"}}
	filas := map[string][]map[string]string{"contactos": {{"nombre": "Anterior", "telefono": "600123456"}}}
	f := &expediente.FichaParaAnexo6{Adultos: &adultos}
	rellenarFichaAnexo6(f, map[string]string{}, filas, map[string]string{}, time.Now(), ",")
	if len(filas["contactos"]) != 1 || filas["contactos"][0]["nombre"] != "Clara" || filas["contactos"][0]["telefono"] != "600987654" {
		t.Fatal(filas)
	}
	adultos = nil
	rellenarFichaAnexo6(f, map[string]string{}, filas, map[string]string{}, time.Now(), ",")
	if len(filas["contactos"]) != 0 {
		t.Fatal("contacto retirado recuperado", filas)
	}
}
