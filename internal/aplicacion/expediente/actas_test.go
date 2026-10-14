package expediente

import (
	"context"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"testing"
)

type validadorActaPrueba struct{}

func (validadorActaPrueba) Validar(context.Context, string, []byte) (string, error) {
	return "pdf", nil
}

type lectorActaPrueba struct {
	llamadas int
	tipo     string
}

func (l *lectorActaPrueba) LeerActa(context.Context, string, []byte) (d.Extraccion, error) {
	l.llamadas++
	return d.Extraccion{Tipo: l.tipo, Campos: []d.Campo{{Codigo: "numero", Valor: "02-71-22", Dudoso: true}}}, nil
}

func TestActasPermisosAntivirusYAuditoria(t *testing.T) {
	const equipo = "ab60479d-b778-407f-a83e-012345678902"
	const otro = "ab60479d-b778-407f-a83e-012345678903"
	for _, tt := range []struct {
		nombre, perfil, equipoActor string
		av                          plataforma.EstadoAntivirus
		falloAV                     bool
		auditoria                   bool
		tipo                        string
		esperado                    error
	}{
		{"coordinacion", "coordinacion", "", plataforma.AntivirusLimpio, false, false, "aceptacion", nil},
		{"equipo propio", "profesional_etf", equipo, plataforma.AntivirusLimpio, false, false, "aceptacion", nil},
		{"equipo ajeno", "profesional_etf", otro, plataforma.AntivirusLimpio, false, false, "aceptacion", d.ErrAcceso},
		{"consulta", "consulta", "", plataforma.AntivirusLimpio, false, false, "aceptacion", d.ErrAcceso},
		{"administracion", "administracion", "", plataforma.AntivirusLimpio, false, false, "aceptacion", d.ErrAcceso},
		{"infectado", "coordinacion", "", plataforma.AntivirusInfectado, false, false, "aceptacion", d.ErrInfectado},
		{"caida antivirus", "coordinacion", "", plataforma.AntivirusNoDisponible, true, false, "aceptacion", nil},
		{"fallo auditoria", "coordinacion", "", plataforma.AntivirusLimpio, false, true, "aceptacion", d.ErrNoDisponible},
		{"tipo equivocado", "coordinacion", "", plataforma.AntivirusLimpio, false, false, "cierre", entrada.ErrActa},
		{"escaneado", "coordinacion", "", plataforma.AntivirusLimpio, false, false, "", nil},
	} {
		t.Run(tt.nombre, func(t *testing.T) {
			e := &entorno{antivirus: tt.av, falloAuditoria: tt.auditoria}
			if tt.falloAV {
				e.falloAV = errors.New("caida")
			}
			l := &lectorActaPrueba{tipo: tt.tipo}
			s := ServicioActas{Validador: validadorActaPrueba{}, Antivirus: e, Lector: l, Registro: e, Reloj: e}
			a := d.Actor{UsuarioID: "ab60479d-b778-407f-a83e-012345678901", EquipoID: tt.equipoActor, Perfiles: []string{tt.perfil}}
			r, err := s.LeerActa(context.Background(), a, equipo, "aceptacion", entrada.Subida{Nombre: "acta.pdf", Contenido: []byte("inventado")})
			if !errors.Is(err, tt.esperado) {
				t.Fatal(err, tt.esperado)
			}
			if err != nil && len(r.Contenido.Campos) != 0 {
				t.Fatal("rechazo entrega datos")
			}
			if tt.esperado == d.ErrAcceso && (e.llamadasAV != 0 || l.llamadas != 0) {
				t.Fatal("lee sin permiso")
			}
			if tt.esperado == d.ErrInfectado && l.llamadas != 0 {
				t.Fatal("lee fichero infectado")
			}
			if tt.falloAV && (r.Antivirus != "no_revisado" || len(e.eventos) != 2 || e.eventos[0].Accion != "expediente.acta_carga_no_revisada") {
				t.Fatal("caída sin aviso auditado", r, e.eventos)
			}
			if err == nil && e.eventos[len(e.eventos)-1].Accion != "expediente.leer_acta_carga" {
				t.Fatal("datos sin auditoria")
			}
			if len(e.datos) != 0 || e.ids != 0 {
				t.Fatal("lectura persiste borrador")
			}
		})
	}
}
