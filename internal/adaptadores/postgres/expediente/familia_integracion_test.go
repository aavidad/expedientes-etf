package expediente_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	nombres "github.com/aavidad/expedientes-etf/internal/adaptadores/expediente/ficheros"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	smiCatalogo "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/catalogo"
	adaptador "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	aplicacion "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"github.com/jackc/pgx/v5/pgconn"
	"strings"
	"sync"
	"testing"
)

func prepararFamilia(t *testing.T) (*postgres.Base, context.Context, aplicacion.Dependencias, dominio.Actor, string) {
	t.Helper()
	b, ctx, deps, a, p := prepararAceptacion(t)
	exp := subir(t, ctx, deps, a)
	if _, err := aplicacion.Nuevo(deps).Aceptar(ctx, a, exp.ID, "alta-"+exp.ID, conMenores(p, exp)); err != nil {
		t.Fatal(err)
	}
	a.Perfiles = []string{"profesional_etf"}
	a.EquipoID = p.EquipoID
	deps.Familias = &adaptador.Repositorio{}
	deps.SMI = &smiCatalogo.Repositorio{}
	deps.CatalogosFamilia = nombres.CatalogosFamilia{Listas: "../../../../idiomas/es/catalogos/listas.json", Familia: "../../../../idiomas/es/catalogos/familia.json", Parametros: "../../../../datos/catalogos/familia.json"}
	return b, ctx, deps, a, exp.ID
}
func texto(s string) *string { return &s }
func datosFamilia() dominio.Familia {
	return dominio.Familia{Direccion: "Calle Olivo 12", Telefono: "958123456", Composicion: texto("BIPARENTAL"), Vivienda: texto("ALQUILER"), Economia: texto("SIN_DIFICULTADES"), Tramo: texto("DE_100_A_150")}
}
func TestFamiliaVersionConcurrenteIdempotenciaYAuditoria(t *testing.T) {
	b, ctx, deps, a, id := prepararFamilia(t)
	s := aplicacion.Nuevo(deps)
	f := datosFamilia()
	p := dominio.CambioFamilia{Version: 2, Desde: "2026-02-03", Familia: &f}
	var wg sync.WaitGroup
	errs := make([]error, 2)
	inicio := make(chan struct{})
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-inicio
			_, errs[i] = s.EditarFamilia(ctx, a, id, fmt.Sprintf("guardar-%d", i), p)
		}(i)
	}
	close(inicio)
	wg.Wait()
	ganadora := -1
	conflictos := 0
	for i, e := range errs {
		if e == nil {
			ganadora = i
		} else {
			var c dominio.Conflicto
			if !errors.As(e, &c) || c.VersionActual != 3 {
				t.Fatal(e)
			}
			conflictos++
		}
	}
	if ganadora < 0 || conflictos != 1 {
		t.Fatal(errs)
	}
	r, e := s.EditarFamilia(ctx, a, id, fmt.Sprintf("guardar-%d", ganadora), p)
	if e != nil || r.Version != 3 {
		t.Fatal(r, e)
	}
	p.Familia = &dominio.Familia{}
	if _, e = s.EditarFamilia(ctx, a, id, fmt.Sprintf("guardar-%d", ganadora), p); !errors.Is(e, dominio.ErrIdempotencia) {
		t.Fatal(e)
	}
	ficha, e := s.ConsultarFamilia(ctx, a, id)
	if e != nil || ficha.Version != 3 || ficha.Familia.Direccion != f.Direccion {
		t.Fatal(ficha, e)
	}
	// Si falla auditoría, tampoco persisten contacto, historia, versión ni recibo.
	deps.RegistroTransaccional = registroFallido{}
	p.Version = 3
	p.Familia = &f
	if _, e = aplicacion.Nuevo(deps).EditarFamilia(ctx, a, id, "auditoria-fallida", p); e == nil {
		t.Fatal("auditoría obligatoria")
	}
	ficha, e = s.ConsultarFamilia(ctx, a, id)
	if e != nil || ficha.Version != 3 {
		t.Fatal(ficha, e)
	}
	reg := postgres.NuevoRegistroAuditoria(b)
	if roto, e := reg.Verificar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: []string{"administracion"}}, 1, 1000000); e != nil || roto != nil {
		t.Fatal(roto, e)
	}
	t.Log("PostgreSQL: edición simultánea → un éxito y un 409; repetición idéntica; reutilización 422; fallo auditoría revierte; cadena íntegra")
}
func TestFamiliaHistoriaAdultosContratoYRLS(t *testing.T) {
	b, ctx, deps, a, id := prepararFamilia(t)
	s := aplicacion.Nuevo(deps)
	f := datosFamilia()
	if _, e := s.EditarFamilia(ctx, a, id, "familia-febrero", dominio.CambioFamilia{Version: 2, Desde: "2026-02-03", Familia: &f}); e != nil {
		t.Fatal(e)
	}
	f.Composicion = texto("MONOPARENTAL_MADRE")
	if _, e := s.EditarFamilia(ctx, a, id, "familia-marzo", dominio.CambioFamilia{Version: 3, Desde: "2026-03-01", Familia: &f}); e != nil {
		t.Fatal(e)
	}
	f.Composicion = texto("EXTENSA")
	if _, e := s.EditarFamilia(ctx, a, id, "corregir-febrero", dominio.CambioFamilia{Version: 4, Desde: "2026-02-03", Familia: &f}); e != nil {
		t.Fatal(e)
	}
	historia, e := s.ConsultarHistoriaFamilia(ctx, a, id, "2026-02-28", 2026)
	if e != nil || historia.Familia.Composicion == nil || *historia.Familia.Composicion != "EXTENSA" || historia.VigenciaFamilia.Hasta == nil || *historia.VigenciaFamilia.Hasta != "2026-03-01" {
		t.Fatal(historia, e)
	}
	ficha, e := s.ConsultarFamilia(ctx, a, id)
	if e != nil || *ficha.Familia.Composicion != "MONOPARENTAL_MADRE" {
		t.Fatal(ficha, e)
	}
	adultos := []dominio.Adulto{{Orden: 1, Nombre: "Inés", Apellidos: "Moreno Castillo", Papel: "MADRE", Sexo: "M"}, {Orden: 2, Nombre: "Manuel", Apellidos: "Moreno Ruiz", Papel: "ABUELO", Sexo: "H"}}
	if _, e = s.EditarFamilia(ctx, a, id, "adultos-febrero", dominio.CambioFamilia{Version: 5, Desde: "2026-02-03", Adultos: &adultos}); e != nil {
		t.Fatal(e)
	}
	ficha, e = s.ConsultarFamilia(ctx, a, id)
	if e != nil || len(ficha.Adultos) != 2 || ficha.Adultos[0].ID == "" {
		t.Fatal(ficha, e)
	}
	adultoID := ficha.Adultos[0].ID
	conservar := ficha.Adultos[:1]
	if _, e = s.EditarFamilia(ctx, a, id, "adultos-marzo", dominio.CambioFamilia{Version: 6, Desde: "2026-03-01", Adultos: &conservar}); e != nil {
		t.Fatal(e)
	}
	historia, e = s.ConsultarHistoriaFamilia(ctx, a, id, "2026-02-28", 2026)
	if e != nil || len(historia.Adultos) != 2 {
		t.Fatal(historia, e)
	}
	bdatos, _ := json.Marshal(historia)
	if strings.Contains(string(bdatos), "Inés") || strings.Contains(string(bdatos), "Olivo") {
		t.Fatal("historia con datos personales de contacto")
	}
	e = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
		consulta, err := (adaptador.ConsultaAdultos{}).AdultosVigentes(ctx, id, "2026-03-01")
		if err != nil {
			return err
		}
		if len(consulta) != 1 || consulta[0].ID != adultoID || consulta[0].Papel != "MADRE" {
			return fmt.Errorf("contrato adultos inesperado")
		}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
	// Otro equipo no ve filas, tampoco por vistas ni por identificador conocido.
	otro := a
	e = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: []string{"coordinacion"}}, func(ctx context.Context) error {
		ej, _ := postgres.DesdeContexto(ctx)
		return ej.QueryRow(ctx, `SELECT id::text FROM catalogo.equipo WHERE id<>$1 LIMIT 1`, a.EquipoID).Scan(&otro.EquipoID)
	})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.ConsultarFamilia(ctx, otro, id); !errors.Is(e, dominio.ErrNoEncontrada) {
		t.Fatal(e)
	}
	if _, e = s.EditarFamilia(ctx, otro, id, "otro-equipo", dominio.CambioFamilia{Version: 7, Desde: "2026-03-01", Familia: &f}); !errors.Is(e, dominio.ErrNoEncontrada) {
		t.Fatal(e)
	}
	for _, actor := range []dominio.Actor{otro, {UsuarioID: a.UsuarioID, Perfiles: []string{"consulta"}}, {UsuarioID: a.UsuarioID, Perfiles: []string{"administracion"}}, {}} {
		e = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: actor.Perfiles, EquipoID: actor.EquipoID}, func(ctx context.Context) error {
			ej, _ := postgres.DesdeContexto(ctx)
			for _, tabla := range []string{"adulto", "familia_revision", "adultos_revision", "adulto_revision", "consulta_adulto", "consulta_familia_vigencia"} {
				var n int
				if err := ej.QueryRow(ctx, `SELECT count(*) FROM expediente.`+tabla+` WHERE expediente_id=$1`, id).Scan(&n); err != nil {
					return err
				}
				if n != 0 {
					return fmt.Errorf("RLS dejó ver %s", tabla)
				}
			}
			return nil
		})
		if e != nil {
			t.Fatal(e)
		}
	}
	// Se conserva la revisión sustituida, sin actualizar ni borrar historia.
	e = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
		ej, _ := postgres.DesdeContexto(ctx)
		var n int
		err := ej.QueryRow(ctx, `SELECT count(*) FROM expediente.familia_revision WHERE expediente_id=$1`, id).Scan(&n)
		if err != nil {
			return err
		}
		if n != 3 {
			return fmt.Errorf("revisiones perdidas: %d", n)
		}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
	// Las lecturas principales se pueden planificar con los índices del módulo.
	e = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
		ej, _ := postgres.DesdeContexto(ctx)
		filas, err := ej.Query(ctx, `EXPLAIN SELECT valor FROM expediente.consulta_familia_vigencia WHERE expediente_id=$1 AND vigente_desde<=$2::date AND (vigente_hasta IS NULL OR $2::date<vigente_hasta)`, id, "2026-02-28")
		if err != nil {
			return err
		}
		defer filas.Close()
		for filas.Next() {
			var plan string
			if err = filas.Scan(&plan); err != nil {
				return err
			}
		}
		return filas.Err()
	})
	if e != nil {
		t.Fatal(e)
	}
	// La reincorporación conserva el ID, incluso después de limpiar su contacto.
	todos := ficha.Adultos
	if _, e = s.EditarFamilia(ctx, a, id, "reincorporar-adulto", dominio.CambioFamilia{Version: 7, Desde: "2026-04-01", Adultos: &todos}); e != nil {
		t.Fatal(e)
	}
	reincorporada, e := s.ConsultarFamilia(ctx, a, id)
	if e != nil || len(reincorporada.Adultos) != 2 || reincorporada.Adultos[1].ID != todos[1].ID {
		t.Fatal(reincorporada, e)
	}
	deps.RegistroTransaccional = registroFallido{}
	lectura, e := aplicacion.Nuevo(deps).ConsultarFamilia(ctx, a, id)
	if e == nil || lectura.ID != "" {
		t.Fatal("lectura sin auditoría", lectura, e)
	}
	t.Log("PostgreSQL: historia retroactiva sin PII, adulto estable y contrato con vigencia; RLS dos equipos/Consulta/Administración/sin contexto")
}

func TestFamiliaImpideAdultoDeOtroExpediente(t *testing.T) {
	b, ctx, deps, a, id := prepararFamilia(t)
	_, ctx2, deps2, a2, id2 := prepararFamilia(t)
	adultos := []dominio.Adulto{{Orden: 1, Nombre: "Teresa", Apellidos: "López Rivas", Papel: "MADRE", Sexo: "M"}}
	servicio := aplicacion.Nuevo(deps2)
	if _, err := servicio.EditarFamilia(ctx2, a2, id2, "adulto-origen", dominio.CambioFamilia{Version: 2, Desde: "2026-02-03", Adultos: &adultos}); err != nil {
		t.Fatal(err)
	}
	ficha, err := servicio.ConsultarFamilia(ctx2, a2, id2)
	if err != nil {
		t.Fatal(err)
	}
	ajenos := ficha.Adultos
	if _, err = aplicacion.Nuevo(deps).EditarFamilia(ctx, a, id, "adulto-ajeno", dominio.CambioFamilia{Version: 2, Desde: "2026-02-03", Adultos: &ajenos}); !errors.Is(err, dominio.ErrFamilia) {
		t.Fatal(err)
	}
	// Incluso con ambos expedientes visibles al equipo, la FK compuesta bloquea
	// una revisión cuyo adulto pertenece al otro expediente.
	adultos[0].ID = ""
	if _, err = aplicacion.Nuevo(deps).EditarFamilia(ctx, a, id, "adulto-destino", dominio.CambioFamilia{Version: 2, Desde: "2026-02-03", Adultos: &adultos}); err != nil {
		t.Fatal(err)
	}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
		e, _ := postgres.DesdeContexto(ctx)
		_, err := e.Exec(ctx, `INSERT INTO expediente.adulto_revision(revision_id,adulto_id,expediente_id,equipo_id,orden,papel,sexo)
   SELECT id,$2::uuid,expediente_id,equipo_id,2,'MADRE','M' FROM expediente.adultos_revision WHERE expediente_id=$1`, id, ajenos[0].ID)
		return err
	})
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || pg.Code != "23503" {
		t.Fatal("FK compuesta", err)
	}
	// La historia es sólo de adición incluso para el equipo que la creó.
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
		e, _ := postgres.DesdeContexto(ctx)
		_, err := e.Exec(ctx, `DELETE FROM expediente.adultos_revision WHERE expediente_id=$1`, id)
		return err
	})
	if !errors.As(err, &pg) || pg.Code != "42501" {
		t.Fatal("historia inmutable", err)
	}
	t.Log("PostgreSQL: adulto ajeno rechazado por aplicación; FK compuesta 23503 aun en mismo equipo; borrado de historia 42501")
}

func TestFamiliaTramoConSMIRealYLecturaPorEquipo(t *testing.T) {
	b, ctx, deps, a, id := prepararFamilia(t)
	// Coordinación registra el SMI inventado: 10 000,00 al mes, 140 000,00 al año.
	err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: []string{"coordinacion"}}, func(ctx context.Context) error {
		e, err := postgres.DesdeContexto(ctx)
		if err != nil {
			return err
		}
		var version int
		return e.QueryRow(ctx, `SELECT catalogo.guardar_smi(2026::smallint,1000000::bigint,0)`).Scan(&version)
	})
	if err != nil {
		t.Fatal(err)
	}
	ingreso := int64(7000000) // 70 000,00: exactamente el 50 % del SMI anual.
	familia := datosFamilia()
	familia.Ingresos = &ingreso
	familia.Tramo = nil
	if _, err = aplicacion.Nuevo(deps).EditarFamilia(ctx, a, id, "ingresos-con-smi", dominio.CambioFamilia{Version: 2, Desde: "2026-02-03", Familia: &familia}); err != nil {
		t.Fatal(err)
	}
	historia, err := aplicacion.Nuevo(deps).ConsultarHistoriaFamilia(ctx, a, id, "2026-02-03", 2026)
	if err != nil || historia.TramoCalculado == nil || *historia.TramoCalculado != "DE_50_A_100" || historia.VersionSMI == nil || *historia.VersionSMI != 1 {
		t.Fatal(historia, err)
	}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
		e, err := postgres.DesdeContexto(ctx)
		if err != nil {
			return err
		}
		var n int
		if err = e.QueryRow(ctx, `SELECT count(*) FROM catalogo.smi WHERE anio=2026`).Scan(&n); err != nil {
			return err
		}
		if n != 1 {
			return fmt.Errorf("SMI no visible para el equipo: %d", n)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, perfil := range []string{"administracion", "consulta", "profesional_etf"} {
		sinEquipo := plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: []string{perfil}}
		err = b.Ejecutar(ctx, sinEquipo, func(ctx context.Context) error {
			e, err := postgres.DesdeContexto(ctx)
			if err != nil {
				return err
			}
			var n int
			if err = e.QueryRow(ctx, `SELECT count(*) FROM catalogo.smi WHERE anio=2026`).Scan(&n); err != nil {
				return err
			}
			if n != 0 {
				return fmt.Errorf("SMI visible sin equipo o perfil: %s", perfil)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, func(ctx context.Context) error {
		e, err := postgres.DesdeContexto(ctx)
		if err != nil {
			return err
		}
		_, err = e.Exec(ctx, `SELECT catalogo.guardar_smi(2026::smallint,1000000::bigint,1)`)
		return err
	})
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || pg.Code != "42501" {
		t.Fatal("escritura SMI no denegada al equipo", err)
	}
	t.Log("PostgreSQL: equipo lee SMI, calcula tramo al 50 % con 14 pagas; Consulta/Administración/sin equipo no ven SMI; sólo Coordinación escribe")
}

func TestFamiliaYPropuestaCierreCompartenVersion(t *testing.T) {
	b, ctx, deps, actor, id := prepararFamilia(t)
	deps.Propuestas = &adaptador.Repositorio{}
	deps.MotivosCierre = &smiCatalogo.Repositorio{}
	servicio := aplicacion.Nuevo(deps)
	familia := datosFamilia()
	if _, err := servicio.EditarFamilia(ctx, actor, id, "integracion-familia", dominio.CambioFamilia{Version: 2, Desde: "2026-02-03", Familia: &familia}); err != nil {
		t.Fatal(err)
	}
	final := dominio.Finalizacion{Version: 3, Motivo: "SIN_CONSENTIMIENTO", SinConsentimiento: true}
	if r, err := servicio.GuardarFinalizacion(ctx, actor, id, "integracion-finalizacion", final); err != nil || r.Version != 4 {
		t.Fatal(r, err)
	}
	propuesta, err := servicio.ProponerCierre(ctx, actor, id, "integracion-propuesta", dominio.VersionCierre{Version: 4})
	if err != nil || propuesta.Version != 5 || propuesta.Estado != "pendiente" {
		t.Fatal(propuesta, err)
	}
	_, err = servicio.EditarFamilia(ctx, actor, id, "integracion-familia-obsoleta", dominio.CambioFamilia{Version: 4, Desde: "2026-02-03", Familia: &familia})
	var conflicto dominio.Conflicto
	if !errors.As(err, &conflicto) || conflicto.VersionActual != 5 {
		t.Fatal("versión compartida", err)
	}
	familia.Direccion = "Calle Almendro 24"
	if r, err := servicio.EditarFamilia(ctx, actor, id, "integracion-familia-pendiente", dominio.CambioFamilia{Version: 5, Desde: "2026-02-03", Familia: &familia}); err != nil || r.Version != 6 {
		t.Fatal(r, err)
	}
	final.Version = 6
	if _, err := servicio.GuardarFinalizacion(ctx, actor, id, "integracion-finalizacion-bloqueada", final); !errors.Is(err, dominio.ErrEstado) {
		t.Fatal("finalización pendiente editable", err)
	}
	if r, err := servicio.RetirarPropuestaCierre(ctx, actor, id, "integracion-retirada", dominio.VersionCierre{Version: 6}); err != nil || r.Version != 7 || r.Estado != "retirada" {
		t.Fatal(r, err)
	}
	ficha, err := servicio.ConsultarFamilia(ctx, actor, id)
	if err != nil || ficha.Version != 7 || ficha.Familia.Direccion != familia.Direccion {
		t.Fatal(ficha, err)
	}
	coord := actor
	coord.Perfiles = []string{"coordinacion"}
	recibida := subir(t, ctx, deps, coord)
	for _, perfiles := range [][]string{{"coordinacion"}, {"coordinacion", "profesional_etf"}} {
		err = b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: perfiles, EquipoID: actor.EquipoID}, func(ctx context.Context) error {
			e, err := postgres.DesdeContexto(ctx)
			if err != nil {
				return err
			}
			_, err = e.Exec(ctx, `UPDATE expediente.expediente SET familia='{"direccion":"Calle Laurel 18"}'::jsonb,
    version=version+1,modificado_por=$2,modificado_en=now() WHERE id=$1`, recibida.ID, actor.UsuarioID)
			return err
		})
		var pg *pgconn.PgError
		if !errors.As(err, &pg) || pg.Code != "42501" || pg.Message != "error.estado_no_permite" {
			t.Fatal("familia fuera de estado activo", perfiles, err)
		}
	}
	t.Log("PostgreSQL: familia y propuesta conservan versión común; 409 entre capacidades; familia editable y finalización bloqueada con propuesta pendiente; retirada sin pérdida de familia")
}

func TestFamiliaYConsentimientoConservanVersion(t *testing.T) {
	b, ctx, deps, actor, id := prepararFamilia(t)
	deps.Hitos = &adaptador.Repositorio{}
	servicio := aplicacion.Nuevo(deps)
	familia := datosFamilia()
	if _, err := servicio.EditarFamilia(ctx, actor, id, "familia-antes-firma", dominio.CambioFamilia{Version: 2, Desde: "2026-02-03", Familia: &familia}); err != nil {
		t.Fatal(err)
	}
	err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: actor.Perfiles, EquipoID: actor.EquipoID}, func(ctx context.Context) error {
		e, err := postgres.DesdeContexto(ctx)
		if err != nil {
			return err
		}
		_, err = e.Exec(ctx, `UPDATE expediente.expediente SET estado='en_intervencion',version=version+1,modificado_por=$2,modificado_en=now() WHERE id=$1`, id, actor.UsuarioID)
		return err
	})
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || pg.Code != "42501" {
		t.Fatal("intervención sin consentimiento", err)
	}
	hito, err := servicio.ConfirmarHito(ctx, actor, id, "firma-tras-familia", dominio.ConfirmacionHito{Tipo: "consentimiento", Fecha: "2026-02-05", Version: 3})
	if err != nil || hito.Version != 4 || hito.Estado != dominio.EnIntervencion {
		t.Fatal(hito, err)
	}
	familia.Direccion = "Calle Encina 16"
	if _, err := servicio.EditarFamilia(ctx, actor, id, "familia-version-antes-firma", dominio.CambioFamilia{Version: 3, Desde: "2026-02-05", Familia: &familia}); err == nil {
		t.Fatal("versión de familia obsoleta aceptada")
	}
	if r, err := servicio.EditarFamilia(ctx, actor, id, "familia-despues-firma", dominio.CambioFamilia{Version: 4, Desde: "2026-02-05", Familia: &familia}); err != nil || r.Version != 5 {
		t.Fatal(r, err)
	}
	ficha, err := servicio.ConsultarFamilia(ctx, actor, id)
	if err != nil || ficha.Version != 5 || ficha.Estado != dominio.EnIntervencion || ficha.Familia.Direccion != familia.Direccion {
		t.Fatal(ficha, err)
	}
	t.Log("PostgreSQL: familia, firma e intervención comparten versión; sin consentimiento no cambia estado; familia permanece editable tras confirmar el hito")
}
