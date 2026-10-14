package identidad

import (
	"testing"
	"time"
)

func TestPerfilesCombinadosAdscripcionYVigencia(t *testing.T) {
	desde := time.Now().UTC()
	hasta := desde.Add(time.Hour)
	c := CambioPerfiles{Version: 1, Estado: Activo, Perfiles: []Concesion{{Coordinacion, desde, nil}, {Administracion, desde, nil}, {ProfesionalETF, desde, &hasta}}, Equipo: &Adscripcion{"equipo-uno", "ts", desde, &hasta}}
	if err := c.Validar(); err != nil {
		t.Fatal(err)
	}
	for _, mutar := range []func(*CambioPerfiles){func(c *CambioPerfiles) { c.Equipo = nil }, func(c *CambioPerfiles) { c.Equipo.Puesto = "otro" }, func(c *CambioPerfiles) { c.Equipo.Hasta = nil }, func(c *CambioPerfiles) { c.Perfiles = append(c.Perfiles, c.Perfiles[0]) }, func(c *CambioPerfiles) { c.Estado = Baja }, func(c *CambioPerfiles) { c.Perfiles[0].Hasta = &desde }} {
		copia := c
		copia.Perfiles = append([]Concesion(nil), c.Perfiles...)
		e := *c.Equipo
		copia.Equipo = &e
		mutar(&copia)
		if err := copia.Validar(); err != ErrPerfilesInvalidos {
			t.Fatal("configuración inválida admitida", err)
		}
	}
	if err := (CambioPerfiles{Version: 1, Estado: Baja}).Validar(); err != nil {
		t.Fatal(err)
	}
	u := Usuario{Estado: Activo, Perfiles: []Perfil{Administracion, Consulta}}
	if !u.Permite(Administracion) || u.Permite(ProfesionalETF) || u.Permite(Coordinacion) {
		t.Fatal("escalada vertical")
	}
	u.Estado = Baja
	if u.Permite(Administracion) {
		t.Fatal("baja autorizada")
	}
}
