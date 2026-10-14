package expediente_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/anexos/lectura"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/clamav"
	nombres "github.com/aavidad/expedientes-etf/internal/adaptadores/expediente/ficheros"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/expediente/memoria"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	catalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	adaptador "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	aplicacion "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

func TestMain(m *testing.M) {
	if len(os.Args) == 2 && os.Args[1] == "--lector-anexo-pdf" {
		if err := lectura.EjecutarPDF(os.Stdin, os.Stdout); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

type reloj struct{}

func (reloj) Ahora() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }

type ids struct{}

func (ids) Nuevo() (string, error) {
	var b [16]byte
	_, err := rand.Read(b[:])
	b[6] = b[6]&15 | 64
	b[8] = b[8]&63 | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:]), err
}

type antivirusCaido struct{}

func (antivirusCaido) Revisar(context.Context, []byte) (plataforma.EstadoAntivirus, error) {
	return plataforma.AntivirusNoDisponible, errors.New("antivirus.no_disponible")
}

type registroFallido struct{}

func (registroFallido) RegistrarEnTransaccion(context.Context, auditoria.Datos) (auditoria.Evento, error) {
	return auditoria.Evento{}, errors.New("auditoria.no_disponible")
}

func preparar(t *testing.T) (*postgres.Base, context.Context, aplicacion.Dependencias, entrada.Actor) {
	t.Helper()
	if os.Getenv("ETF_POSTGRES_INTEGRACION") != "1" {
		t.Skip("ETF_POSTGRES_INTEGRACION=1")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 40*time.Second)
	t.Cleanup(cancelar)
	b, err := postgres.Nueva(ctx, os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(b.Cerrar)
	var guid identidad.GUID
	_, _ = rand.Read(guid[:])
	u, err := postgres.NuevosUsuarios(b).RegistrarAcceso(ctx, identidad.IdentidadDirectorio{GUID: guid, CuentaAD: fmt.Sprintf("ines.%x", guid[:4]), NombreMostrado: "Inés Moreno Castillo"}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	a := entrada.Actor{UsuarioID: u.ID, Perfiles: []string{"coordinacion"}}
	r := postgres.NuevoRegistroAuditoria(b)
	d := aplicacion.Dependencias{Borradores: memoria.NuevosBorradores(time.Minute, 32), Confirmaciones: &adaptador.Repositorio{}, Repositorio: &adaptador.Repositorio{}, Transacciones: b, Registro: r, RegistroTransaccional: r, Validador: lectura.Validador{}, Antivirus: antivirusCaido{}, Lector: lectura.NuevoLector(), Catalogo: &catalogo.Repositorio{}, NombresMunicipios: nombres.NuevosMunicipios("../../../../idiomas", "es"), Reloj: reloj{}, Identificadores: ids{}}
	return b, ctx, d, a
}

func subir(t *testing.T, ctx context.Context, d aplicacion.Dependencias, a entrada.Actor) dominio.Derivacion {
	t.Helper()
	b, err := os.ReadFile("../../anexos/docx/testdata/anexo3.docx")
	if err != nil {
		t.Fatal(err)
	}
	r, err := aplicacion.Nuevo(d).Subir(ctx, a, entrada.Subida{Nombre: "Derivacion.docx", Contenido: b})
	if err != nil {
		t.Fatal(err)
	}

	return confirmarLectura(t, ctx, d, a, r)
}

func confirmarLectura(t *testing.T, ctx context.Context, d aplicacion.Dependencias, a entrada.Actor, r dominio.Derivacion) dominio.Derivacion {
	t.Helper()
	if r.ID != "" || r.Estado != "" {
		t.Fatal("análisis crea derivación")
	}
	opciones := r.Datos.OpcionesUbicacion
	ubicacion := dominio.UbicacionRevisada{MunicipioID: opciones.Municipios[0].ID, CentroID: opciones.Centros[0].ID, EquipoID: opciones.Equipos[0].ID}
	if len(r.Datos.PropuestaMunicipio.Candidatos) == 1 {
		c := r.Datos.PropuestaMunicipio.Candidatos[0]
		ubicacion.MunicipioID = c.MunicipioID
		if c.CentroID != nil {
			ubicacion.CentroID = *c.CentroID
		}
		if c.EquipoID != nil {
			ubicacion.EquipoID = *c.EquipoID
		}
	}
	recibo, err := aplicacion.Nuevo(d).ConfirmarDerivacion(ctx, a, r.Lectura.ID, entrada.Confirmacion{LecturaID: r.Lectura.ID, Tipo: r.Datos.Extraccion.Tipo, Campos: r.Datos.Extraccion.Campos, Ubicacion: ubicacion})
	if err != nil {
		t.Fatal(err)
	}
	guardada, err := aplicacion.Nuevo(d).Consultar(ctx, a, recibo.ID)
	if err != nil {
		t.Fatal(err)
	}
	return guardada
}

func TestIntegracionGuardarRecuperarRLSYAuditoria(t *testing.T) {
	b, ctx, deps, a := preparar(t)
	d := subir(t, ctx, deps, a)
	leida, err := aplicacion.Nuevo(deps).Consultar(ctx, a, d.ID)
	if err != nil || !reflect.DeepEqual(d, leida) {
		t.Fatal("recuperación distinta", err)
	}
	if d.Datos.PropuestaMunicipio.Estado == 0 {
		t.Fatal("no se propuso municipio con catálogo real")
	}
	for _, p := range []string{"", "administracion", "consulta", "profesional_etf"} {
		i := plataforma.Identidad{UsuarioID: a.UsuarioID}
		if p != "" {
			i.Perfiles = []string{p}
		}
		if err := b.Ejecutar(ctx, i, func(ctx context.Context) error {
			for _, tabla := range []string{"expediente", "lectura_anexo", "derivacion_datos"} {
				e, err := postgres.DesdeContexto(ctx)
				if err != nil {
					return err
				}
				var n int
				if err := e.QueryRow(ctx, "SELECT count(*) FROM expediente."+tabla).Scan(&n); err != nil {
					return err
				}
				if n != 0 {
					t.Fatalf("perfil %q ve %s", p, tabla)
				}
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
		e, err := postgres.DesdeContexto(ctx)
		if err != nil {
			return err
		}
		var plan []byte
		return e.QueryRow(ctx, `EXPLAIN (FORMAT JSON) SELECT id FROM expediente.expediente WHERE id=$1`, d.ID).Scan(&plan)
	}); err != nil {
		t.Fatal(err)
	}
	// La exportación comprueba la cadena completa y permite contar acciones
	// sin obtener ni registrar contenido personal de la derivación.
	r := postgres.NuevoRegistroAuditoria(b)
	exportacion, err := r.Exportar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: []string{"administracion"}}, 1, 1000000)
	if err != nil {
		t.Fatal(err)
	}
	acciones := map[string]int{}
	for _, evento := range exportacion.Eventos {
		if evento.ExpedienteID == d.ID {
			acciones[evento.Accion]++
		}
	}
	for _, accion := range []string{"expediente.confirmar_derivacion", "expediente.leer_derivacion"} {
		esperado := 1
		if accion == "expediente.leer_derivacion" {
			esperado = 2
		}
		if acciones[accion] != esperado {
			t.Fatal(accion, acciones)
		}
	}
}

func TestIntegracionFalloAuditoriaRevierteTresTablas(t *testing.T) {
	b, ctx, deps, a := preparar(t)
	deps.RegistroTransaccional = registroFallido{}
	contenido, err := os.ReadFile("../../anexos/docx/testdata/anexo3.docx")
	if err != nil {
		t.Fatal(err)
	}
	antes := [3]int{}
	contar := func(destino *[3]int) error {
		return b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles}, func(ctx context.Context) error {
			e, err := postgres.DesdeContexto(ctx)
			if err != nil {
				return err
			}
			for i, tabla := range []string{"expediente", "lectura_anexo", "derivacion_datos"} {
				if err := e.QueryRow(ctx, "SELECT count(*) FROM expediente."+tabla).Scan(&destino[i]); err != nil {
					return err
				}
			}
			return nil
		})
	}
	if err := contar(&antes); err != nil {
		t.Fatal(err)
	}
	d, err := aplicacion.Nuevo(deps).Subir(ctx, a, entrada.Subida{Nombre: "Derivacion.docx", Contenido: contenido})
	if err == nil || d.ID != "" {
		t.Fatal("escritura sin auditoría")
	}
	despues := [3]int{}
	if err := contar(&despues); err != nil {
		t.Fatal(err)
	}
	if antes != despues {
		t.Fatal("escritura parcial", antes, despues)
	}
}

func TestIntegracionClamAVRealEICARLimpioYCaido(t *testing.T) {
	_, ctx, deps, a := preparar(t)
	direccion := os.Getenv("ETF_PRUEBA_CLAMAV_DIRECCION")
	if direccion == "" {
		t.Skip("ETF_PRUEBA_CLAMAV_DIRECCION")
	}
	av, err := clamav.Nuevo(clamav.Configuracion{Direccion: direccion, TiempoLimite: 5 * time.Second, MaximoBytes: dominio.MaxBytes})
	if err != nil {
		t.Fatal(err)
	}
	deps.Antivirus = av
	d := subir(t, ctx, deps, a)
	if d.Lectura.Antivirus != "limpio" {
		t.Fatal(d.Lectura)
	}
	contenido, err := os.ReadFile("../../anexos/docx/testdata/anexo3.docx")
	if err != nil {
		t.Fatal(err)
	}
	z, err := zip.NewReader(bytes.NewReader(contenido), int64(len(contenido)))
	if err != nil {
		t.Fatal(err)
	}
	var infectado bytes.Buffer
	w := zip.NewWriter(&infectado)
	for _, f := range z.File {
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		destino, err := w.Create(f.Name)
		if err != nil {
			t.Fatal(err)
		}
		_, fallo := io.Copy(destino, r)
		cierre := r.Close()
		if fallo != nil || cierre != nil {
			t.Fatal(fallo, cierre)
		}
	}
	firma, err := w.Create("word/media/image_eicar.png")
	if err != nil {
		t.Fatal(err)
	}
	eicar := "X5O!P%@AP[4\\PZX54(P^)7CC)7}$" + "EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*"
	if _, err := firma.Write([]byte(eicar)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	rechazada, err := aplicacion.Nuevo(deps).Subir(ctx, a, entrada.Subida{Nombre: "Derivacion.docx", Contenido: infectado.Bytes()})
	if !errors.Is(err, dominio.ErrInfectado) || rechazada.ID != "" {
		t.Fatal("EICAR no rechazado", err)
	}
	caido, err := clamav.Nuevo(clamav.Configuracion{Direccion: "127.0.0.1:1", TiempoLimite: time.Second, MaximoBytes: dominio.MaxBytes})
	if err != nil {
		t.Fatal(err)
	}
	deps.Antivirus = caido
	d = subir(t, ctx, deps, a)
	if d.Lectura.Antivirus != "no_revisado" {
		t.Fatal(d.Lectura)
	}
	if err := aplicacion.Nuevo(deps).Rechazar(ctx, entrada.Actor{}, dominio.ErrAcceso.Error()); err != nil {
		t.Fatal("rechazo anónimo sin auditoría", err)
	}
}
