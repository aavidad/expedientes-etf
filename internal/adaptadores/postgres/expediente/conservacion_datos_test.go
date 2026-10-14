package expediente_test

import (
	"context"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	evRepo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/evaluacion"
	expRepo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	evApp "github.com/aavidad/expedientes-etf/internal/aplicacion/evaluacion"
	expApp "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	ev "github.com/aavidad/expedientes-etf/internal/dominio/evaluacion"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"testing"
	"time"
)

func datosConservacionOtrosModulos(t *testing.T, b *postgres.Base, ctx context.Context, deps expApp.Dependencias, coord, equipo d.Actor, id string) string {
	t.Helper()
	ficha, err := expApp.Nuevo(deps).ConsultarFamilia(ctx, equipo, id)
	if err != nil || len(ficha.Adultos) == 0 {
		t.Fatal(ficha, err)
	}
	s := &evApp.Servicio{Repo: &evRepo.Repositorio{}, Catalogo: catalogoEvaluacion{}, Familias: &expRepo.Repositorio{}, Adultos: expRepo.ConsultaAdultos{}, Tx: b, Auditoria: postgres.NuevoRegistroAuditoriaTransaccional(b), Ahora: time.Now}
	a := ev.Actor{UsuarioID: equipo.UsuarioID, EquipoID: equipo.EquipoID, Perfiles: equipo.Perfiles}
	insID, _ := ids{}.Nuevo()
	escalaID, _ := ids{}.Nuevo()
	sujeto := ev.Sujeto{Tipo: "figura_parental", AdultoID: ficha.Adultos[0].ID}
	fecha := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	valor := 1.0
	instrumento := ev.InstrumentoAplicado{ID: insID, ExpedienteID: id, EquipoID: a.EquipoID, Codigo: "OTROS", Fecha: fecha, Sujeto: &sujeto, Momento: "inicial", TextoOtros: "Teresa López Rivas acude con su hija Lucía."}
	if _, err = s.Guardar(ctx, a, id, ev.Cambio{Instrumento: &instrumento, Clave: "conservar-instrumento"}); err != nil {
		t.Fatal(err)
	}
	escala := ev.Aplicacion{ID: escalaID, ExpedienteID: id, EquipoID: a.EquipoID, Instrumento: "PRUEBA", Momento: "inicial", Sujeto: sujeto, Fecha: &fecha, VersionEscalaID: "11111111-1111-4111-8111-111111111111", DescripcionCualitativa: "Teresa López Rivas indica su teléfono 600 845 217.", Puntuaciones: []ev.Puntuacion{{Item: "item01", ValorNumerico: &valor}}}
	if _, err = s.Guardar(ctx, a, id, ev.Cambio{Aplicacion: &escala, Clave: "conservar-escala"}); err != nil {
		t.Fatal(err)
	}
	emisionID, _ := ids{}.Nuevo()
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: coord.UsuarioID, Perfiles: coord.Perfiles}, func(ctx context.Context) error {
		tx, _ := postgres.DesdeContexto(ctx)
		tag, err := tx.Exec(ctx, `INSERT INTO documento.emision(id,tipo_documento,expediente_id,equipo_id,plantilla_id,idioma,fecha_documento,generada_por,generada_en,huella_sha256,huella_datos,versiones_datos,version_motor,ruta_idioma,version_idioma,aviso_clave)
  SELECT $1,p.tipo_documento,$2,$3,p.id,p.idioma,DATE '2026-10-01',$4,clock_timestamp(),decode(repeat('a',64),'hex'),decode(repeat('b',64),'hex'),'{"expediente":"1"}','motor_docx_1','idiomas/es','1.0.0',''
  FROM documento.referencia_activa r JOIN documento.plantilla p ON p.id=r.plantilla_id WHERE r.tipo_documento='entrada/aceptacion/oficio_sscc' AND r.idioma='es'`, emisionID, id, equipo.EquipoID, coord.UsuarioID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			t.Fatal("falta plantilla de oficio")
		}
		if _, err = tx.Exec(ctx, `INSERT INTO documento.registro_salida(emision_id,adaptador,estado,version) VALUES($1,'manual','pendiente',1)`, emisionID); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE documento.registro_salida SET numero_registro='2026/Teresa López Rivas',referencia_externa='Teresa López Rivas - Lucía',fecha_registro=DATE '2026-10-01',anotado_por=$2,anotado_en=clock_timestamp(),estado='anotado',version=2 WHERE emision_id=$1`, emisionID, coord.UsuarioID); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO documento.registro_peticion(usuario_id,clave,emision_id,ruta,huella_peticion,respuesta,creada_en) VALUES($1,'conservar-registro',$2::uuid,'/api/documentos/emisiones/'||$2::text||'/registro-salida',decode(repeat('c',64),'hex'),'{"numero_registro":"Teresa López Rivas"}',clock_timestamp())`, coord.UsuarioID, emisionID); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO documento.peticion_idempotente(usuario_id,clave,emision_id,ruta,huella_peticion,respuesta,creada_en) VALUES($1,'conservar-emision',$2::uuid,'/api/documentos/emisiones',decode(repeat('d',64),'hex'),jsonb_build_object('emision_id',$2::text),clock_timestamp())`, coord.UsuarioID, emisionID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return emisionID
}
