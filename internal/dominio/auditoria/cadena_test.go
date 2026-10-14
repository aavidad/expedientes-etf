package auditoria_test

import (
	"encoding/hex"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
)

func TestCambioDeCadaBitDeLasHuellas(t *testing.T) {
	base := cadena(t, 3)
	for posicion := range base {
		for bit := 0; bit < 256; bit++ {
			for _, previa := range []bool{false, true} {
				e := copiar(base)
				huella := &e[posicion].Huella
				if previa {
					huella = &e[posicion].HuellaAnterior
				}
				b, err := hex.DecodeString(*huella)
				if err != nil {
					t.Fatal(err)
				}
				b[bit/8] ^= 1 << uint(bit%8)
				*huella = hex.EncodeToString(b)
				err = auditoria.Verificar(e, auditoria.Inicio(), base[2].Punto())
				var roto *auditoria.ErrorVerificacion
				if !errors.As(err, &roto) || roto.Indice != posicion {
					t.Fatalf("posición %d, bit %d, previa %t: %v", posicion, bit, previa, err)
				}
			}
		}
	}
}

func TestInsercionDeUnEventoNuevoEnUnSalto(t *testing.T) {
	uno, err := auditoria.Encadenar(10, datos(), auditoria.Inicio())
	if err != nil {
		t.Fatal(err)
	}
	dos, err := auditoria.Encadenar(20, datos(), uno.Punto())
	if err != nil {
		t.Fatal(err)
	}
	nuevo, err := auditoria.Encadenar(15, datos(), uno.Punto())
	if err != nil {
		t.Fatal(err)
	}
	var roto *auditoria.ErrorVerificacion
	err = auditoria.Verificar([]auditoria.Evento{uno, nuevo, dos}, auditoria.Inicio(), dos.Punto())
	if !errors.As(err, &roto) || roto.Indice != 2 || roto.Clave != "auditoria.enlace_invalido" {
		t.Fatal(err)
	}
}

func datos() auditoria.Datos {
	return auditoria.Datos{
		OcurridoEn: time.Date(2026, 10, 13, 8, 42, 11, 123456000, time.UTC),
		UsuarioID:  "51fa2864-7e56-4e34-a106-98ad46b5c201",
		Perfiles:   []string{"coordinacion", "administracion"},
		Accion:     "expediente.leer_ficha", Entidad: "expediente",
		EntidadID:    "ad8e5132-3b49-4fe4-926c-5bd8acd02817",
		ExpedienteID: "ad8e5132-3b49-4fe4-926c-5bd8acd02817", Resultado: "ok",
	}
}

func cadena(t *testing.T, cantidad int) []auditoria.Evento {
	t.Helper()
	eventos := make([]auditoria.Evento, cantidad)
	anterior := auditoria.Inicio()
	for i := range eventos {
		d := datos()
		d.OcurridoEn = d.OcurridoEn.Add(time.Duration(i) * time.Second)
		e, err := auditoria.Encadenar(uint64(i+1), d, anterior)
		if err != nil {
			t.Fatal(err)
		}
		eventos[i], anterior = e, e.Punto()
	}
	return eventos
}

func copiar(eventos []auditoria.Evento) []auditoria.Evento {
	copia := append([]auditoria.Evento{}, eventos...)
	for i := range copia {
		copia[i].Perfiles = append([]string{}, copia[i].Perfiles...)
	}
	return copia
}

func TestCadenaYTramos(t *testing.T) {
	e := cadena(t, 4)
	for _, tramo := range []struct {
		eventos         []auditoria.Evento
		anterior, final auditoria.Punto
	}{
		{e, auditoria.Inicio(), e[3].Punto()},
		{e[1:3], e[0].Punto(), e[2].Punto()},
		{e[:1], auditoria.Inicio(), e[0].Punto()},
		{nil, auditoria.Inicio(), auditoria.Inicio()},
		{nil, e[1].Punto(), e[1].Punto()},
	} {
		if err := auditoria.Verificar(tramo.eventos, tramo.anterior, tramo.final); err != nil {
			t.Fatal(err)
		}
	}
	// Una reserva fallida puede dejar saltos sin romper los enlaces.
	siguiente, err := auditoria.Encadenar(9, datos(), e[3].Punto())
	if err != nil || auditoria.Verificar([]auditoria.Evento{siguiente}, e[3].Punto(), siguiente.Punto()) != nil {
		t.Fatalf("salto de ID: %v", err)
	}
}

func TestAlteracionDeCadaCampo(t *testing.T) {
	base := cadena(t, 4)
	cambios := map[string]func(*auditoria.Evento){
		"id":                func(e *auditoria.Evento) { e.ID += 100 },
		"fecha":             func(e *auditoria.Evento) { e.OcurridoEn = e.OcurridoEn.Add(time.Microsecond) },
		"usuario":           func(e *auditoria.Evento) { e.UsuarioID = "26e4ca8f-1fd4-4942-b523-01f65924b946" },
		"perfil":            func(e *auditoria.Evento) { e.Perfiles[0] = "consulta" },
		"orden_perfiles":    func(e *auditoria.Evento) { e.Perfiles[0], e.Perfiles[1] = e.Perfiles[1], e.Perfiles[0] },
		"accion":            func(e *auditoria.Evento) { e.Accion = "expediente.listar" },
		"entidad":           func(e *auditoria.Evento) { e.Entidad = "documento" },
		"entidad_id":        func(e *auditoria.Evento) { e.EntidadID = "" },
		"expediente_id":     func(e *auditoria.Evento) { e.ExpedienteID = "" },
		"resultado":         func(e *auditoria.Evento) { e.Resultado = "denegado" },
		"huella_previa":     func(e *auditoria.Evento) { e.HuellaAnterior = strings.Repeat("a", 64) },
		"huella":            func(e *auditoria.Evento) { e.Huella = strings.Repeat("b", 64) },
		"huella_malformada": func(e *auditoria.Evento) { e.Huella = "no_es_una_huella" },
	}
	for nombre, cambiar := range cambios {
		for posicion := range base {
			t.Run(nombre+"/"+string(rune('0'+posicion)), func(t *testing.T) {
				e := copiar(base)
				cambiar(&e[posicion])
				err := auditoria.Verificar(e, auditoria.Inicio(), base[3].Punto())
				var roto *auditoria.ErrorVerificacion
				if !errors.As(err, &roto) || roto.Indice != posicion {
					t.Fatalf("primer fallo en %d: %v", posicion, err)
				}
			})
		}
	}
}

func TestBorradoInsercionEInversionEnTodaPosicion(t *testing.T) {
	base := cadena(t, 5)
	for i := range base {
		t.Run("borrar/"+string(rune('0'+i)), func(t *testing.T) {
			e := append(copiar(base[:i]), base[i+1:]...)
			err := auditoria.Verificar(e, auditoria.Inicio(), base[4].Punto())
			var roto *auditoria.ErrorVerificacion
			if !errors.As(err, &roto) || roto.Indice != i {
				t.Fatalf("borrado %d: %v", i, err)
			}
		})
		t.Run("duplicar/"+string(rune('0'+i)), func(t *testing.T) {
			e := append(copiar(base[:i+1]), append([]auditoria.Evento{base[i]}, base[i+1:]...)...)
			if err := auditoria.Verificar(e, auditoria.Inicio(), base[4].Punto()); err == nil {
				t.Fatal("duplicado admitido")
			}
		})
		if i+1 < len(base) {
			t.Run("invertir/"+string(rune('0'+i)), func(t *testing.T) {
				e := copiar(base)
				e[i], e[i+1] = e[i+1], e[i]
				if err := auditoria.Verificar(e, auditoria.Inicio(), base[4].Punto()); err == nil {
					t.Fatal("inversión admitida")
				}
			})
		}
	}
	if err := auditoria.Verificar(nil, auditoria.Inicio(), base[4].Punto()); err == nil {
		t.Fatal("cadena entera borrada")
	}
	// Un evento añadido legítimamente después del límite también se rechaza.
	if err := auditoria.Verificar(base, auditoria.Inicio(), base[3].Punto()); err == nil {
		t.Fatal("evento fuera del tramo")
	}
}

func TestSelloDetectaRecalculo(t *testing.T) {
	base := cadena(t, 4)
	for _, operacion := range []string{"alterar", "borrar", "insertar", "invertir"} {
		t.Run(operacion, func(t *testing.T) {
			e := copiar(base)
			switch operacion {
			case "alterar":
				e[1].Accion = "expediente.listar"
			case "borrar":
				e = append(e[:1], e[2:]...)
			case "insertar":
				e = append(e[:1], append([]auditoria.Evento{base[0]}, e[1:]...)...)
			case "invertir":
				e[1], e[2] = e[2], e[1]
			}
			previo := auditoria.Inicio()
			for i := range e {
				var err error
				e[i], err = auditoria.Encadenar(uint64(i+1), e[i].Datos, previo)
				if err != nil {
					t.Fatal(err)
				}
				previo = e[i].Punto()
			}
			if err := auditoria.Verificar(e, auditoria.Inicio(), base[3].Punto()); err == nil {
				t.Fatal("recalculado admitido con sello original")
			}
			// Un sello recalculado en el mismo fichero sólo demuestra coherencia.
			if err := auditoria.Verificar(e, auditoria.Inicio(), previo); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRechazaValoresFueraDelContrato(t *testing.T) {
	cambios := map[string]func(*auditoria.Datos){
		"fecha_cero":         func(d *auditoria.Datos) { d.OcurridoEn = time.Time{} },
		"precision":          func(d *auditoria.Datos) { d.OcurridoEn = d.OcurridoEn.Add(time.Nanosecond) },
		"usuario_nombre":     func(d *auditoria.Datos) { d.UsuarioID = "ines.moreno" },
		"usuario_vacio":      func(d *auditoria.Datos) { d.UsuarioID = "" },
		"usuario_cero":       func(d *auditoria.Datos) { d.UsuarioID = "00000000-0000-0000-0000-000000000000" },
		"uuid_formato":       func(d *auditoria.Datos) { d.EntidadID = "ad8e5132_3b49-4fe4-926c-5bd8acd02817" },
		"uuid_mayusculas":    func(d *auditoria.Datos) { d.ExpedienteID = strings.ToUpper(d.ExpedienteID) },
		"texto_libre":        func(d *auditoria.Datos) { d.Accion = "Abre la ficha de la familia" },
		"accion_vacia":       func(d *auditoria.Datos) { d.Accion = "" },
		"codigo_largo":       func(d *auditoria.Datos) { d.Entidad = strings.Repeat("a", 129) },
		"codigo_inicial":     func(d *auditoria.Datos) { d.Entidad = "1expediente" },
		"codigo_puntos":      func(d *auditoria.Datos) { d.Entidad = "expediente..ficha" },
		"codigo_final":       func(d *auditoria.Datos) { d.Entidad = "expediente." },
		"resultado":          func(d *auditoria.Datos) { d.Resultado = "con nombres" },
		"perfil_duplicado":   func(d *auditoria.Datos) { d.Perfiles = []string{"consulta", "consulta"} },
		"perfil_nombre":      func(d *auditoria.Datos) { d.Perfiles = []string{"Inés Moreno"} },
		"perfiles_excesivos": func(d *auditoria.Datos) { d.Perfiles = make([]string, 17) },
		"fecha_utc_fuera":    func(d *auditoria.Datos) { d.OcurridoEn = time.Date(1, 1, 1, 0, 0, 0, 0, time.FixedZone("zona", 3600)) },
	}
	for nombre, cambiar := range cambios {
		t.Run(nombre, func(t *testing.T) {
			d := datos()
			cambiar(&d)
			if _, err := auditoria.Encadenar(1, d, auditoria.Inicio()); err == nil {
				t.Fatal("valor inválido admitido")
			}
		})
	}
	for _, id := range []uint64{0, math.MaxUint64} {
		if _, err := auditoria.Encadenar(id, datos(), auditoria.Inicio()); err == nil {
			t.Fatal("ID inválido")
		}
	}
	base := cadena(t, 2)
	if _, err := auditoria.Encadenar(2, datos(), base[1].Punto()); err == nil {
		t.Fatal("ID repetido")
	}
	for _, punto := range []auditoria.Punto{{}, {ID: 0, Huella: strings.Repeat("a", 64)}, {ID: 1, Huella: auditoria.HuellaInicial}, {ID: 1, Huella: strings.Repeat("A", 64)}} {
		if _, err := auditoria.Encadenar(2, datos(), punto); err == nil {
			t.Fatal("punto inválido")
		}
		if auditoria.Verificar(nil, punto, auditoria.Inicio()) == nil {
			t.Fatal("ancla inválida")
		}
		if auditoria.Verificar(nil, auditoria.Inicio(), punto) == nil {
			t.Fatal("sello inválido")
		}
	}
}

func TestCanonicalizacionYCopia(t *testing.T) {
	d := datos()
	e, err := auditoria.Encadenar(1, d, auditoria.Inicio())
	if err != nil {
		t.Fatal(err)
	}
	d.OcurridoEn = d.OcurridoEn.In(time.FixedZone("zona", 2*3600))
	otro, err := auditoria.Encadenar(1, d, auditoria.Inicio())
	if err != nil || otro.Huella != e.Huella {
		t.Fatal("zona cambió el instante firmado")
	}
	d.Perfiles[0] = "consulta"
	if e.Perfiles[0] != "coordinacion" {
		t.Fatal("alias del llamante")
	}
	d = datos()
	d.UsuarioID = "sistema"
	d.Perfiles = nil
	d.EntidadID = ""
	d.ExpedienteID = ""
	if _, err := auditoria.Encadenar(1, d, auditoria.Inicio()); err != nil {
		t.Fatal(err)
	}
}

func TestVectorCanonicoIndependiente(t *testing.T) {
	// Calculado aparte con Python hashlib.sha256 y struct.pack('>Q', ...),
	// sobre el contrato binario documentado, sin llamar a código Go.
	e, err := auditoria.Encadenar(1, datos(), auditoria.Inicio())
	if err != nil {
		t.Fatal(err)
	}
	const esperado = "8fe3ef13875a0b5fbe10c10987cbde71df7cf1e261d79912dc4ba53bab99a2c6"
	if e.Huella != esperado {
		t.Fatalf("vector: %s", e.Huella)
	}
}
