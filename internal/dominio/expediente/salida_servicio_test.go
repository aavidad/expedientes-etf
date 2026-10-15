package expediente

import "testing"

func TestSalidaServicioPermisos(t *testing.T) {
	usuario := "11111111-1111-4111-8111-111111111111"
	equipo := "22222222-2222-4222-8222-222222222222"
	ajeno := "33333333-3333-4333-8333-333333333333"
	for _, caso := range []struct {
		perfil, equipo, destino string
		permitido               bool
	}{
		{"coordinacion", "", ajeno, true},
		{"profesional_etf", equipo, equipo, true},
		{"profesional_etf", equipo, ajeno, false},
		{"profesional_etf", "", equipo, false},
		{"consulta", equipo, equipo, false},
		{"administracion", equipo, equipo, false},
	} {
		t.Run(caso.perfil+caso.destino+caso.equipo, func(t *testing.T) {
			err := (Actor{UsuarioID: usuario, Perfiles: []string{caso.perfil}, EquipoID: caso.equipo}).AutorizarSalida(caso.destino)
			if (err == nil) != caso.permitido {
				t.Fatalf("permiso: %v", err)
			}
		})
	}
}

func TestSalidaServicioValidacion(t *testing.T) {
	p := NuevaSalidaServicio{Tipo: "otro_servicio", Version: 1, Servicio: "SALUD", Motivo: "Coordinación de la atención familiar"}
	if err := p.Validar(); err != nil {
		t.Fatal(err)
	}
	for _, cambiar := range []func(*NuevaSalidaServicio){
		func(p *NuevaSalidaServicio) { p.Version = 0 },
		func(p *NuevaSalidaServicio) { p.Tipo = "spm" },
		func(p *NuevaSalidaServicio) { p.Servicio = "Salud" },
		func(p *NuevaSalidaServicio) { p.Motivo = " " },
	} {
		invalida := p
		cambiar(&invalida)
		if invalida.Validar() == nil {
			t.Fatal("acepta entrada inválida")
		}
	}
	h := ConfirmarSalidaServicio{Tipo: "derivacion_otro_servicio", SalidaID: "11111111-1111-4111-8111-111111111111", Version: 2, VersionSalida: 1, Fecha: "2026-10-13"}
	if err := h.Validar(); err != nil {
		t.Fatal(err)
	}
	h.Fecha = "2026-02-30"
	if h.Validar() == nil {
		t.Fatal("acepta fecha inexistente")
	}
	if SalidaPermitida("cerrada") || SalidaPermitida("no_aceptada") || !SalidaPermitida("aceptada") {
		t.Fatal("estados de salida")
	}
}

func TestSalidaServicioDestinoEscritoODeCatalogoExcluyentes(t *testing.T) {
	p := NuevaSalidaServicio{Tipo: "otro_servicio", Version: 1, ServicioTexto: "Centro de atención sanitaria", Motivo: "Coordinación sanitaria familiar"}
	if err := p.Validar(); err != nil {
		t.Fatal(err)
	}
	p.Servicio = "SALUD"
	if p.Validar() == nil {
		t.Fatal("destino mixto admitido")
	}
	p.Servicio = ""
	p.ServicioTexto = "  "
	if p.Validar() == nil {
		t.Fatal("destino vacío admitido")
	}
	p.ServicioTexto = string(make([]byte, 255))
	if p.Validar() == nil {
		t.Fatal("destino inválido admitido")
	}
}
