package documento_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	word "github.com/aavidad/expedientes-etf/internal/adaptadores/docx/documento"
	ficheros "github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/documento"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/documento"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

type antivirusPlantilla struct {
	estado plataforma.EstadoAntivirus
	eicar  bool
}

func (a antivirusPlantilla) Revisar(_ context.Context, b []byte) (plataforma.EstadoAntivirus, error) {
	if a.eicar && string(b) == `X5O!P%@AP[4\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*` {
		return plataforma.AntivirusInfectado, nil
	}
	return a.estado, nil
}

type publicacionesMemoria struct {
	mu      sync.Mutex
	actual  d.ReferenciaPlantilla
	n       int
	eventos int
}

func (m *publicacionesMemoria) Ejecutar(ctx context.Context, _ plataforma.Identidad, fn func(context.Context) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return fn(ctx)
}
func (m *publicacionesMemoria) Actual(context.Context, string, string) (d.ReferenciaPlantilla, error) {
	return m.actual, nil
}
func (m *publicacionesMemoria) Publicar(_ context.Context, p d.Plantilla, v int64) (d.ReferenciaPlantilla, error) {
	if v != m.actual.Revision {
		return d.ReferenciaPlantilla{}, &d.ConflictoPlantilla{VersionActual: m.actual.Revision}
	}
	m.actual = d.ReferenciaPlantilla{Plantilla: p, Revision: v + 1}
	m.n++
	return m.actual, nil
}
func (m *publicacionesMemoria) Registrada(_ context.Context, p d.Plantilla) (bool, error) {
	return m.actual.Plantilla.ID == p.ID, nil
}
func (m *publicacionesMemoria) Confirmar(context.Context, auditoria.Datos) (auditoria.Evento, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventos++
	return auditoria.Evento{}, nil
}
func (*publicacionesMemoria) RegistrarEnTransaccion(context.Context, auditoria.Datos) (auditoria.Evento, error) {
	return auditoria.Evento{}, nil
}
func prepararPublicacion(t *testing.T, av plataforma.Antivirus) (*app.AdministracionPlantillas, *publicacionesMemoria, []byte, ficheros.Plantillas) {
	t.Helper()
	b, err := os.ReadFile("../../../idiomas/es/plantillas/entrada/aceptacion/v1.1.0.docx")
	if err != nil {
		t.Fatal(err)
	}
	m := &publicacionesMemoria{actual: d.ReferenciaPlantilla{Plantilla: d.Plantilla{ID: usuario, Tipo: d.ActaAceptacion, Idioma: "es", Version: "1.1.0", Ruta: "plantillas/entrada/aceptacion/v1.1.0.docx", Huella: d.Huella(b), RolFirmante: "aceptacion"}, Revision: 2}}
	lectura := ficheros.Plantillas{Raiz: "../../../idiomas", Paquetes: referencias{}}
	almacen := ficheros.Plantillas{Raiz: t.TempDir()}
	s := app.NuevaAdministracionPlantillas(app.DependenciasPlantillas{Repositorio: m, Almacen: almacen, Validador: word.ValidadorPlantillas{}, Antivirus: av, Idiomas: lectura, Ficheros: lectura, Transacciones: m, Registro: m, RegistroTransaccional: m, Reloj: reloj{}, Identificadores: &ids{}})
	return s, m, b, almacen
}
func TestPublicacionPlantillaTamanoAntivirusYPermiso(t *testing.T) {
	for _, c := range []struct {
		nombre   string
		av       plataforma.Antivirus
		b        []byte
		perfil   string
		esperado error
	}{
		{"mayor_10_MB", antivirusPlantilla{estado: plataforma.AntivirusLimpio}, make([]byte, d.MaximoPlantilla+1), "administracion", d.ErrTamanoPlantilla},
		{"EICAR", antivirusPlantilla{eicar: true}, []byte(`X5O!P%@AP[4\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*`), "administracion", d.ErrInfectada},
		{"sin_clamd", nil, []byte("PK"), "administracion", d.ErrAntivirus},
		{"clamd_caido", antivirusPlantilla{estado: plataforma.AntivirusNoDisponible}, []byte("PK"), "administracion", d.ErrAntivirus},
		{"coordinacion", nil, []byte("PK"), "coordinacion", d.ErrAcceso},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			s, m, _, _ := prepararPublicacion(t, c.av)
			_, err := s.Publicar(t.Context(), d.Actor{UsuarioID: usuario, Perfiles: []string{c.perfil}}, d.PublicarPlantilla{Tipo: d.ActaAceptacion, Idioma: "es", Version: "2.0.0", VersionPrevia: 2}, c.b)
			if !errors.Is(err, c.esperado) || m.n != 0 || m.eventos != 1 {
				t.Fatal(err, m.n, m.eventos)
			}
		})
	}
}
func TestPublicacionPlantillaCarreraDosActivaciones(t *testing.T) {
	s, m, b, almacen := prepararPublicacion(t, antivirusPlantilla{estado: plataforma.AntivirusLimpio})
	errores := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.Publicar(t.Context(), d.Actor{UsuarioID: usuario, Perfiles: []string{"administracion"}}, d.PublicarPlantilla{Tipo: d.ActaAceptacion, Idioma: "es", Version: "2.0.0", VersionPrevia: 2}, b)
			errores <- err
		}()
	}
	wg.Wait()
	close(errores)
	ok, conflictos := 0, 0
	for err := range errores {
		var c *d.ConflictoPlantilla
		if err == nil {
			ok++
		} else if errors.As(err, &c) && c.VersionActual == 3 {
			conflictos++
		} else {
			t.Fatal(err)
		}
	}
	if ok != 1 || conflictos != 1 || m.n != 1 {
		t.Fatal(ok, conflictos, m.n)
	}
	leida, err := almacen.Leer(t.Context(), d.IdiomaPlantilla{Ruta: "es"}, m.actual.Plantilla)
	if err != nil || d.Huella(leida) != d.Huella(b) {
		t.Fatal(err)
	}
	if err := almacen.Guardar(t.Context(), m.actual.Plantilla, b); err == nil {
		t.Fatal("sobrescritura permitida")
	}
}
