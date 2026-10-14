package expediente

import (
	"context"
	"encoding/json"
	"github.com/aavidad/expedientes-etf/internal/adaptadores/postgres"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"time"
)

//etf:tecnico
func cargarTrasladoSeguimiento(ctx context.Context, ficha *d.Derivacion) error {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `SELECT origen_id::text,destino_id::text,equipo_origen_id::text,equipo_destino_id::text FROM expediente.traslado WHERE origen_id=$1 OR destino_id=$1 ORDER BY trasladado_en DESC,(origen_id=$1) DESC,destino_id`, ficha.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var enlace d.EnlaceTraslado
		if err = rows.Scan(&enlace.OrigenID, &enlace.DestinoID, &enlace.EquipoOrigenID, &enlace.EquipoDestinoID); err != nil {
			rows.Close()
			return err
		}
		if ficha.Traslado == nil {
			ficha.Traslado = &enlace
		}
		ficha.Traslados = append(ficha.Traslados, enlace)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return err
	}
	ficha.Seguimiento, err = (&Repositorio{}).ObtenerSeguimiento(ctx, ficha.ID, false)
	return err
}

//etf:tecnico
func (*Repositorio) RepetirTraslado(ctx context.Context, u, c, r, h string, t time.Time) (d.ResultadoTraslado, bool, error) {
	return repeticion[d.ResultadoTraslado](ctx, u, c, r, h, t)
}

//etf:tecnico
func (*Repositorio) RecordarTraslado(ctx context.Context, u, c, r, h string, p d.ResultadoTraslado, t time.Time) error {
	return recordar(ctx, u, c, r, h, p, t)
}

// CrearTraslado copia datos, no permisos, números, consentimientos ni hitos.
//
//etf:tecnico
func (*Repositorio) CrearTraslado(ctx context.Context, origen, destino, lectura string, p d.PeticionTraslado, copia d.CopiaTraslado, u string, t time.Time) error {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	b, err := json.Marshal(copia)
	if err != nil {
		return err
	}
	defer clear(b)
	familia, err := json.Marshal(copia.Familia)
	if err != nil {
		return err
	}
	defer clear(familia)
	_, err = tx.Exec(ctx, `INSERT INTO expediente.expediente(id,fecha_recepcion,origen,traslado_origen_id,traslado_equipo_id,copia_traslado,servicio_derivante,familia)
 VALUES($1,$2,'traslado',$3,$4,$5,'OTRO_ETF',$6)`, destino, t, origen, p.EquipoDestinoID, b, familia)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO expediente.lectura_anexo(id,expediente_id,formato,tamano_bytes,huella_sha256,antivirus,antivirus_detalle,plantilla_detectada,campos_dudosos,leida_por,leida_en)
 SELECT $1,$2,formato,tamano_bytes,huella_sha256,antivirus,antivirus_detalle,plantilla_detectada,campos_dudosos,$3,$4
 FROM expediente.lectura_anexo WHERE expediente_id=$5`, lectura, destino, u, t, origen)
	if err != nil {
		return err
	}
	datos, err := json.Marshal(copia.Datos)
	if err != nil {
		return err
	}
	defer clear(datos)
	_, err = tx.Exec(ctx, `INSERT INTO expediente.derivacion_datos(expediente_id,esquema_version,contenido) VALUES($1,'1',$2)`, destino, datos)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO expediente.traslado(origen_id,destino_id,equipo_origen_id,equipo_destino_id,trasladado_por,trasladado_en)
 SELECT id,$2,equipo_id,$3,$4,$5 FROM expediente.expediente WHERE id=$1`, origen, destino, p.EquipoDestinoID, u, t)
	return err
}

// MaterializarCopiaTraslado crea identidades e historia del destino al aceptar.
// Los identificadores de adultos ya se renovaron al tomar la copia.
//
//etf:tecnico
func materializarCopiaTraslado(ctx context.Context, id, equipo, fecha, u string, version int, t time.Time) error {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	var b []byte
	if err = tx.QueryRow(ctx, `SELECT to_jsonb(e)->'copia_traslado' FROM expediente.expediente e WHERE id=$1`, id).Scan(&b); err != nil {
		return err
	}
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	defer clear(b)
	var c d.CopiaTraslado
	if err = json.Unmarshal(b, &c); err != nil {
		return err
	}
	familia, err := json.Marshal(c.Familia.Estadistica())
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO expediente.familia_revision(expediente_id,equipo_id,version,vigente_desde,valor,registrado_por,registrado_en)
 VALUES($1,$2,$3,$4::date,$5,$6,$7)`, id, equipo, version, fecha, familia, u, t)
	if err != nil {
		return err
	}
	adultos, err := json.Marshal(c.Adultos)
	if err != nil {
		return err
	}
	defer clear(adultos)
	_, err = tx.Exec(ctx, `INSERT INTO expediente.adulto(id,expediente_id,equipo_id,papel,sexo,datos)
 SELECT (j->>'id')::uuid,$1::uuid,$2::uuid,j->>'papel',j->>'sexo',j FROM jsonb_array_elements($3::jsonb) j`, id, equipo, adultos)
	if err != nil {
		return err
	}
	var revision string
	err = tx.QueryRow(ctx, `INSERT INTO expediente.adultos_revision(expediente_id,equipo_id,version,vigente_desde,registrado_por,registrado_en)
 VALUES($1,$2,$3,$4::date,$5,$6) RETURNING id::text`, id, equipo, version, fecha, u, t).Scan(&revision)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO expediente.adulto_revision(revision_id,adulto_id,expediente_id,equipo_id,orden,papel,sexo,fecha_nacimiento,edad_declarada,nivel_educativo,situacion_laboral,es_tutor)
 SELECT $1::uuid,(j->>'id')::uuid,$2::uuid,$3::uuid,(j->>'orden')::integer,j->>'papel',j->>'sexo',(j->>'fecha_nacimiento')::date,(j->>'edad_declarada')::integer,j->>'nivel_educativo',j->>'situacion_laboral',(j->>'es_tutor')::boolean
 FROM jsonb_array_elements($4::jsonb) j`, revision, id, equipo, adultos)
	return err
}

// La aceptación ordinaria conserva su contrato. En un traslado se insertan los
// menores de la copia vigente, con identidades y revisiones propias del destino.
//
//etf:tecnico
func insertarMenoresAceptacion(ctx context.Context, r d.ResultadoAceptacion, p d.Aceptacion, u string, t time.Time) error {
	tx, err := postgres.DesdeContexto(ctx)
	if err != nil {
		return err
	}
	var b []byte
	if err = tx.QueryRow(ctx, `SELECT to_jsonb(e)->'copia_traslado' FROM expediente.expediente e WHERE id=$1`, r.ID).Scan(&b); err != nil {
		return err
	}
	if len(b) == 0 || string(b) == "null" {
		sexos, err := json.Marshal(p.Menores)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO expediente.menor(expediente_id,equipo_id,orden,sexo)
 SELECT $1::uuid,$2::uuid,m.orden,m.sexo FROM jsonb_to_recordset($3::jsonb) AS m(orden integer,sexo text)`, r.ID, r.EquipoID, sexos)
		return err
	}
	defer clear(b)
	var copia d.CopiaTraslado
	if err = json.Unmarshal(b, &copia); err != nil {
		return err
	}
	if len(copia.Menores) != len(p.Menores) {
		return d.ErrSexo
	}
	for i := range copia.Menores {
		copia.Menores[i].Sexo = p.Menores[i].Sexo
	}
	menores, err := json.Marshal(copia.Menores)
	if err != nil {
		return err
	}
	defer clear(menores)
	_, err = tx.Exec(ctx, `INSERT INTO expediente.menor(id,expediente_id,equipo_id,orden,sexo,datos)
 SELECT (j->>'id')::uuid,$1::uuid,$2::uuid,(j->>'orden')::integer,j->>'sexo',j FROM jsonb_array_elements($3::jsonb) j`, r.ID, r.EquipoID, menores)
	if err != nil {
		return err
	}
	var revision string
	err = tx.QueryRow(ctx, `INSERT INTO expediente.menores_revision(expediente_id,equipo_id,version,vigente_desde,registrado_por,registrado_en)
 VALUES($1,$2,$3,$4::date,$5,$6) RETURNING id::text`, r.ID, r.EquipoID, r.Version, p.FechaActa, u, t).Scan(&revision)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO expediente.menor_revision(revision_id,menor_id,expediente_id,equipo_id,orden,sexo,fecha_nacimiento,nivel_escolarizacion,subprograma,centro_proteccion_codigo,propuesto_atencion_temprana)
 SELECT $1::uuid,m.id,$2::uuid,$3::uuid,m.orden,m.sexo,m.fecha_nacimiento,m.nivel_escolarizacion,m.subprograma,m.centro_proteccion_codigo,m.propuesto_atencion_temprana
 FROM jsonb_to_recordset($4::jsonb) AS m(id uuid,orden integer,sexo text,fecha_nacimiento date,nivel_escolarizacion text,subprograma text,centro_proteccion_codigo text,propuesto_atencion_temprana boolean)`, revision, r.ID, r.EquipoID, menores)
	return err
}
