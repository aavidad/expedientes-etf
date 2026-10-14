package expediente_test

import (
	"context"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	"github.com/jackc/pgx/v5/pgconn"
	"os"
	"testing"

	aplicacion "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

func TestRESP4ConfirmacionODTYPDFFirmado(t *testing.T) {
	for _, caso := range []struct{ nombre, formato string }{{"anexo3.odt", "odt"}, {"anexo4-firmado.pdf", "pdf"}} {
		t.Run(caso.nombre, func(t *testing.T) {
			base, ctx, deps, actor := preparar(t)
			contenido, err := os.ReadFile("../../anexos/odt/testdata/" + caso.nombre)
			if err != nil {
				t.Fatal(err)
			}
			servicio := aplicacion.Nuevo(deps)
			leida, err := servicio.Subir(ctx, actor, entrada.Subida{Nombre: caso.nombre, Contenido: contenido})
			if err != nil {
				t.Fatal(err)
			}
			opciones := leida.Datos.OpcionesUbicacion
			peticion := entrada.Confirmacion{LecturaID: leida.Lectura.ID, Tipo: leida.Datos.Extraccion.Tipo, Campos: leida.Datos.Extraccion.Campos, Ubicacion: dominio.UbicacionRevisada{MunicipioID: opciones.Municipios[0].ID, CentroID: opciones.Centros[0].ID, EquipoID: opciones.Equipos[0].ID}}
			clave := "resp4-" + leida.Lectura.ID
			recibo, err := servicio.ConfirmarDerivacion(ctx, actor, clave, peticion)
			if err != nil {
				t.Fatal(err)
			}
			repetido, err := servicio.ConfirmarDerivacion(ctx, actor, clave, peticion)
			if err != nil || repetido != recibo {
				t.Fatal(repetido, err)
			}
			guardada, err := servicio.Consultar(ctx, actor, recibo.ID)
			if err != nil {
				t.Fatal(err)
			}
			if guardada.Lectura.Formato != caso.formato || guardada.Estado != dominio.DerivacionRecibida || guardada.Numero != nil {
				t.Fatal(guardada.Lectura.Formato, guardada.Estado)
			}
			id := plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: actor.Perfiles}
			err = base.Ejecutar(ctx, id, func(txctx context.Context) error {
				tx, e := postgres.DesdeContexto(txctx)
				if e != nil {
					return e
				}
				var lecturas, eventos int
				if e = tx.QueryRow(txctx, `SELECT count(*) FROM expediente.lectura_anexo WHERE id=$1 AND formato=$2`, leida.Lectura.ID, caso.formato).Scan(&lecturas); e != nil {
					return e
				}
				if e = tx.QueryRow(txctx, `SELECT count(*) FROM auditoria.historial_expediente($1) WHERE accion='expediente.confirmar_derivacion'`, recibo.ID).Scan(&eventos); e != nil {
					return e
				}
				if lecturas != 1 || eventos != 1 {
					t.Fatalf("lecturas=%d eventos=%d", lecturas, eventos)
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			// Un valor fuera de la lista sigue rechazado por PostgreSQL.
			err = base.Ejecutar(ctx, id, func(txctx context.Context) error {
				tx, e := postgres.DesdeContexto(txctx)
				if e != nil {
					return e
				}
				_, e = tx.Exec(txctx, `INSERT INTO expediente.lectura_anexo (id,expediente_id,formato,tamano_bytes,huella_sha256,antivirus,antivirus_detalle,plantilla_detectada,campos_dudosos,leida_por,leida_en) SELECT gen_random_uuid(),expediente_id,'html',tamano_bytes,huella_sha256,antivirus,antivirus_detalle,plantilla_detectada,campos_dudosos,leida_por,leida_en FROM expediente.lectura_anexo WHERE id=$1`, leida.Lectura.ID)
				return e
			})
			var pgerr *pgconn.PgError
			if !errors.As(err, &pgerr) || pgerr.Code != "23514" || pgerr.ConstraintName != "lectura_anexo_formato_check" {
				t.Fatal("formato ajeno no rechazado por CHECK", err)
			}
			t.Logf("%s: formato=%s persistido; un recibo y un evento; repetición idempotente; formato ajeno rechazado", caso.nombre, caso.formato)
		})
	}
}
