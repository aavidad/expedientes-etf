package identidad

import (
	"context"
	"errors"
	"testing"
	"time"

	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	auditoria "github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
)

type registroRechazos struct {
	salida.RegistroTransaccional
	datos []auditoria.Datos
	fallo error
}

func (r *registroRechazos) Confirmar(_ context.Context, d auditoria.Datos) (auditoria.Evento, error) {
	r.datos = append(r.datos, d)
	return auditoria.Evento{}, r.fallo
}

func TestCambioPropioSeDeniegaAntesDePersistirYSeAudita(t *testing.T) {
	for _, fallaAuditoria := range []bool{false, true} {
		t.Run(map[bool]string{false: "auditado", true: "fallo_auditoria"}[fallaAuditoria], func(t *testing.T) {
			r := &registroRechazos{}
			if fallaAuditoria {
				r.fallo = errors.New("prueba.auditoria")
			}
			s := NuevaAdministracionUsuarios(nil, nil, r, &relojMemoria{ahora: time.Now()})
			a := entrada.Acceso{Estado: dominio.Activo, Usuario: dominio.Usuario{ID: "persona", Estado: dominio.Activo, Perfiles: []dominio.Perfil{dominio.Administracion}}}
			u, err := s.Cambiar(context.Background(), a, a.Usuario.ID, entrada.CambioPerfiles{})
			esperado := dominio.ErrCambioPropioDenegado
			if fallaAuditoria {
				esperado = dominio.ErrPersistencia
			}
			if !errors.Is(err, esperado) || u.ID != "" {
				t.Fatalf("resultado: %v %v", u, err)
			}
			if len(r.datos) != 1 {
				t.Fatalf("apuntes: %d", len(r.datos))
			}
			d := r.datos[0]
			if d.UsuarioID != a.Usuario.ID || d.EntidadID != a.Usuario.ID || d.Resultado != "denegado" || d.Accion != "identidad.perfiles_cambiar" {
				t.Fatalf("auditoría: %+v", d)
			}
		})
	}
}
