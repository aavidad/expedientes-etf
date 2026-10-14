package ubicacion_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/aavidad/expedientes-etf/internal/dominio/expediente/ubicacion"
)

// Datos de datos/catalogos/municipios.json y nombres resueltos mediante
// clave_texto en idiomas/es/catalogos/ubicaciones.json. El dominio recibe valores.
func catalogoPrueba() []ubicacion.Municipio {
	return []ubicacion.Municipio{
		{Codigo: "18013", Nombre: "Alhama de Granada", AmbitoDiputacion: true},
		{Codigo: "18017", Nombre: "Almuñécar"},
		{Codigo: "18021", Nombre: "Armilla"},
		{Codigo: "18022", Nombre: "Atarfe"},
		{Codigo: "18023", Nombre: "Baza"},
		{Codigo: "18024", Nombre: "Beas de Granada", AmbitoDiputacion: true},
		{Codigo: "18025", Nombre: "Beas de Guadix", AmbitoDiputacion: true},
		{Codigo: "18027", Nombre: "Benalúa", AmbitoDiputacion: true},
		{Codigo: "18028", Nombre: "Benalúa de las Villas", AmbitoDiputacion: true},
		{Codigo: "18045", Nombre: "Castilléjar", AmbitoDiputacion: true},
		{Codigo: "18049", Nombre: "Cogollos de Guadix", AmbitoDiputacion: true},
		{Codigo: "18050", Nombre: "Cogollos de la Vega", AmbitoDiputacion: true},
		{Codigo: "18056", Nombre: "Cúllar", AmbitoDiputacion: true},
		{Codigo: "18057", Nombre: "Cúllar Vega", AmbitoDiputacion: true},
		{Codigo: "18062", Nombre: "Churriana de la Vega", Variantes: []string{"churriana"}, AmbitoDiputacion: true},
		{Codigo: "18064", Nombre: "Dehesas de Guadix", AmbitoDiputacion: true},
		{Codigo: "18065", Nombre: "Dehesas Viejas", AmbitoDiputacion: true},
		{Codigo: "18077", Nombre: "Fornes", Variantes: []string{"e l a fornes"}, AmbitoDiputacion: true},
		{Codigo: "18087", Nombre: "Granada"},
		{Codigo: "18089", Nombre: "Guadix"},
		{Codigo: "18094", Nombre: "Güéjar Sierra", AmbitoDiputacion: true},
		{Codigo: "18099", Nombre: "Huétor de Santillán", Variantes: []string{"huetor santillan"}, AmbitoDiputacion: true},
		{Codigo: "18100", Nombre: "Huétor Tájar", AmbitoDiputacion: true},
		{Codigo: "18101", Nombre: "Huétor Vega", AmbitoDiputacion: true},
		{Codigo: "18122", Nombre: "Loja"},
		{Codigo: "18127", Nombre: "Maracena"},
		{Codigo: "18140", Nombre: "Motril"},
		{Codigo: "18164", Nombre: "Puebla de Don Fadrique", Variantes: []string{"puebla de don fabrique"}, AmbitoDiputacion: true},
		{Codigo: "18174", Nombre: "Santa Cruz del Comercio", AmbitoDiputacion: true},
		{Codigo: "18175", Nombre: "Santa Fe", AmbitoDiputacion: true},
		{Codigo: "18188", Nombre: "Villanueva Mesía", Variantes: []string{"vva mesia"}, AmbitoDiputacion: true},
		{Codigo: "18193", Nombre: "La Zubia", Variantes: []string{"zubia la"}},
		{Codigo: "18905", Nombre: "Las Gabias", Variantes: []string{"gabias las"}},
		{Codigo: "18914", Nombre: "Valderrubio", AmbitoDiputacion: true},
		{Codigo: "18915", Nombre: "Domingo Pérez de Granada", AmbitoDiputacion: true},
	}
}

func TestParesConflictivosYErratas(t *testing.T) {
	casos := []struct{ texto, codigo string }{
		{"Cúllar", "18056"},
		{"Cúllar Vega", "18057"},
		{"Beas de Granada", "18024"},
		{"Beas de Guadix", "18025"},
		{"Cogollos de la Vega", "18050"},
		{"Cogollos de Guadix", "18049"},
		{"Huétor Santillán", "18099"},
		{"Huétor de Santillán", "18099"},
		{"Huétor Vega", "18101"},
		{"Huétor Tájar", "18100"},
		{"Dehesas de Guadix", "18064"},
		{"Dehesas Viejas", "18065"},
		{"Benalúa", "18027"},
		{"Benalúa de las Villas", "18028"},
		{"Churriana", "18062"},
		{"Churriana de la Vega", "18062"},
		{"Alhama de Granada", "18013"},
		{"Santa Fe", "18175"},
		{"Santa Cruz del Comercio", "18174"},
		{"E.L.A. Fornes", "18077"},
		{"Fornes", "18077"},
		{"cullar vega", "18057"},
		{"Cástilléjar", "18045"},
		{"Gúejar Sierra", "18094"},
		{"Vva. Mesía", "18188"},
		{"Valderrubio", "18914"},
		{"Puebla de Don Fabrique", "18164"},
		{"DOMINGO PÉREZ DE GRANADA", "18915"},
		{"CÚLLAR---VEGA", "18057"},
		{"Huétor\n\tTájar", "18100"},
		{"Gu\u0308e\u0301jar Sierra", "18094"},
		{"Benalu\u0301a", "18027"},
	}
	for _, caso := range casos {
		t.Run(caso.texto, func(t *testing.T) {
			// El prefijo multibyte verifica que las posiciones sirven para subrayar
			// el texto original, sin utilizar los índices del texto normalizado.
			direccion := "C/ Álamo, 12 · " + caso.texto + ", 2.º"
			propuesta := ubicacion.ProponerMunicipio(direccion, "", catalogoConClaves())
			comprobar(t, propuesta, ubicacion.CandidatoUnico, []ubicacion.Candidato{
				candidato(caso.codigo, caso.texto, direccion, ubicacion.CampoDireccion, false),
			})
		})
	}
}

func TestPrioridadAmbiguedadYPalabrasCompletas(t *testing.T) {
	casos := []struct {
		nombre, direccion, valoracion string
		estado                        ubicacion.Estado
		codigos, fragmentos           []string
		campo                         ubicacion.Campo
		fuera                         []bool
	}{
		{"sólo valoración", "C/ Olivo, 8", "Reside en Huétor Vega desde septiembre.", ubicacion.CandidatoUnico, []string{"18101"}, []string{"Huétor Vega"}, ubicacion.CampoValoracionDiagnostico, []bool{false}},
		{"prioridad dirección", "Santa Fe", "Reside en Cúllar Vega.", ubicacion.CandidatoUnico, []string{"18175"}, []string{"Santa Fe"}, ubicacion.CampoDireccion, []bool{false}},
		{"dirección ambigua", "Santa Fe; domicilio anterior en Cúllar Vega", "Reside en Santa Fe.", ubicacion.Ambiguo, []string{"18175", "18057"}, []string{"Santa Fe", "Cúllar Vega"}, ubicacion.CampoDireccion, []bool{false, false}},
		{"larga y corta independientes", "Cúllar Vega; anteriormente Cúllar", "", ubicacion.Ambiguo, []string{"18057", "18056"}, []string{"Cúllar Vega", "Cúllar"}, ubicacion.CampoDireccion, []bool{false, false}},
		{"granada independiente", "Alhama de Granada; antes en Granada", "", ubicacion.Ambiguo, []string{"18013", "18087"}, []string{"Alhama de Granada", "Granada"}, ubicacion.CampoDireccion, []bool{false, true}},
		{"valoración ambigua", "C/ Acacia, 5", "De Beas de Guadix a Santa Fe", ubicacion.Ambiguo, []string{"18025", "18175"}, []string{"Beas de Guadix", "Santa Fe"}, ubicacion.CampoValoracionDiagnostico, []bool{false, false}},
		{"mismo municipio repetido", "Churriana; Churriana de la Vega; Churriana de la Vega", "", ubicacion.CandidatoUnico, []string{"18062"}, []string{"Churriana de la Vega"}, ubicacion.CampoDireccion, []bool{false}},
		{"larga repetida", "Cúllar Vega; Cúllar Vega", "", ubicacion.CandidatoUnico, []string{"18057"}, []string{"Cúllar Vega"}, ubicacion.CampoDireccion, []bool{false}},
		{"sufijo", "Cullarvega y Benalúanos", "", ubicacion.SinCandidatos, nil, nil, 0, nil},
		{"prefijo", "preCúllar y protoSanta Fe", "", ubicacion.SinCandidatos, nil, nil, 0, nil},
		{"dígitos adyacentes", "Cúllar2 y 2Santa Fe", "", ubicacion.SinCandidatos, nil, nil, 0, nil},
		{"vacío", "", "", ubicacion.SinCandidatos, nil, nil, 0, nil},
		{"signos", "...---", "\n\t", ubicacion.SinCandidatos, nil, nil, 0, nil},
		{"sin coincidencia", "C/ Encina, 17", "Pendiente de confirmar residencia", ubicacion.SinCandidatos, nil, nil, 0, nil},
		{"alias alhama ausente del catálogo", "Alhama", "", ubicacion.SinCandidatos, nil, nil, 0, nil},
		{"alias domingo ausente del catálogo", "Domingo Pérez (ELA)", "", ubicacion.SinCandidatos, nil, nil, 0, nil},
		{"fuera en dirección conserva prioridad", "Motril", "Reside en Santa Fe", ubicacion.CandidatoUnico, []string{"18140"}, []string{"Motril"}, ubicacion.CampoDireccion, []bool{true}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			propuesta := ubicacion.ProponerMunicipio(caso.direccion, caso.valoracion, catalogoConClaves())
			var esperados []ubicacion.Candidato
			texto := caso.direccion
			if caso.campo == ubicacion.CampoValoracionDiagnostico {
				texto = caso.valoracion
			}
			for i, codigo := range caso.codigos {
				inicioBusqueda := 0
				if i > 0 {
					inicioBusqueda = esperados[i-1].Fin
				}
				esperado := candidato(codigo, caso.fragmentos[i], texto[inicioBusqueda:], caso.campo, caso.fuera[i])
				esperado.Inicio += inicioBusqueda
				esperado.Fin += inicioBusqueda
				esperados = append(esperados, esperado)
			}
			comprobar(t, propuesta, caso.estado, esperados)
		})
	}
}

func TestMunicipiosFueraDelAmbito(t *testing.T) {
	casos := []struct{ texto, codigo string }{
		{"Almuñécar", "18017"}, {"Armilla", "18021"}, {"Atarfe", "18022"},
		{"Baza", "18023"}, {"Granada", "18087"}, {"Guadix", "18089"},
		{"Loja", "18122"}, {"Maracena", "18127"}, {"Motril", "18140"},
		{"La Zubia", "18193"}, {"Las Gabias", "18905"},
	}
	for _, caso := range casos {
		t.Run(caso.texto, func(t *testing.T) {
			propuesta := ubicacion.ProponerMunicipio(caso.texto, "", catalogoConClaves())
			comprobar(t, propuesta, ubicacion.CandidatoUnico, []ubicacion.Candidato{
				candidato(caso.codigo, caso.texto, caso.texto, ubicacion.CampoDireccion, true),
			})
		})
	}
}

func TestCatalogoRecibido(t *testing.T) {
	t.Run("sin catálogo", func(t *testing.T) {
		comprobar(t, ubicacion.ProponerMunicipio("Santa Fe", "Cúllar", nil), ubicacion.SinCandidatos, nil)
	})
	t.Run("variantes inyectadas", func(t *testing.T) {
		municipios := catalogoConClaves()
		for i := range municipios {
			switch municipios[i].Codigo {
			case "18013":
				municipios[i].Variantes = []string{"Alhama"}
			case "18915":
				municipios[i].Variantes = []string{"Domingo Pérez (ELA)"}
			}
		}
		for _, caso := range []struct{ texto, codigo string }{
			{"Alhama", "18013"}, {"Alhama de Granada", "18013"}, {"Domingo Pérez (ELA)", "18915"},
		} {
			comprobar(t, ubicacion.ProponerMunicipio(caso.texto, "", municipios), ubicacion.CandidatoUnico,
				[]ubicacion.Candidato{candidato(caso.codigo, strings.TrimSuffix(caso.texto, ")"), caso.texto, ubicacion.CampoDireccion, false)})
		}
	})
	t.Run("variante compartida no elige por orden", func(t *testing.T) {
		municipios := []ubicacion.Municipio{
			{Codigo: "18057", ClaveTexto: "municipio.18057", Variantes: []string{"vega"}, AmbitoDiputacion: true},
			{Codigo: "18101", ClaveTexto: "municipio.18101", Variantes: []string{"vega"}, AmbitoDiputacion: true},
			{Codigo: "vacío", Variantes: []string{"", "---"}},
		}
		comprobar(t, ubicacion.ProponerMunicipio("Vega", "", municipios), ubicacion.Ambiguo,
			[]ubicacion.Candidato{candidato("18057", "Vega", "Vega", ubicacion.CampoDireccion, false), candidato("18101", "Vega", "Vega", ubicacion.CampoDireccion, false)})
	})
	t.Run("ámbito procede del catálogo", func(t *testing.T) {
		municipios := []ubicacion.Municipio{{Codigo: "18087", ClaveTexto: "municipio.18087", Nombre: "Granada", AmbitoDiputacion: true}}
		comprobar(t, ubicacion.ProponerMunicipio("Granada", "", municipios), ubicacion.CandidatoUnico,
			[]ubicacion.Candidato{candidato("18087", "Granada", "Granada", ubicacion.CampoDireccion, false)})
	})
	t.Run("no modifica catálogo y resultado estable", func(t *testing.T) {
		municipios := catalogoConClaves()
		antes := catalogoConClaves()
		propuesta := ubicacion.ProponerMunicipio("Cúllar Vega y Benalúa", "", municipios)
		if !reflect.DeepEqual(municipios, antes) {
			t.Fatal("Se ha modificado el catálogo recibido")
		}
		for i, j := 0, len(municipios)-1; i < j; i, j = i+1, j-1 {
			municipios[i], municipios[j] = municipios[j], municipios[i]
		}
		if inversa := ubicacion.ProponerMunicipio("Cúllar Vega y Benalúa", "", municipios); !reflect.DeepEqual(propuesta, inversa) {
			t.Fatalf("El orden del catálogo cambia el resultado: %+v / %+v", propuesta, inversa)
		}
	})
}

func catalogoConClaves() []ubicacion.Municipio {
	municipios := catalogoPrueba()
	for i := range municipios {
		municipios[i].ClaveTexto = "municipio." + municipios[i].Codigo
	}
	return municipios
}

func candidato(codigo, fragmento, texto string, campo ubicacion.Campo, fuera bool) ubicacion.Candidato {
	inicio := strings.Index(texto, fragmento)
	return ubicacion.Candidato{Codigo: codigo, ClaveTexto: "municipio." + codigo, Campo: campo,
		FueraAmbitoDiputacion: fuera, Fragmento: fragmento, Inicio: inicio, Fin: inicio + len(fragmento)}
}

func comprobar(t *testing.T, propuesta ubicacion.Propuesta, estado ubicacion.Estado, candidatos []ubicacion.Candidato) {
	t.Helper()
	esperada := ubicacion.Propuesta{Estado: estado, Candidatos: candidatos}
	if !reflect.DeepEqual(propuesta, esperada) {
		t.Fatalf("Propuesta recibida: %+v; esperada: %+v", propuesta, esperada)
	}
}
