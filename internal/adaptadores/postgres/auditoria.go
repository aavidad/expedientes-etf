package postgres

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	salida "github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

// RegistroAuditoria confirma lecturas en su propia transacción y ofrece un
// registro separado para escrituras que ya tienen una transacción de negocio.
type RegistroAuditoria struct{ base *Base }

var _ salida.Registro = (*RegistroAuditoria)(nil)

func NuevoRegistroAuditoria(base *Base) *RegistroAuditoria {
	return &RegistroAuditoria{base: base}
}

// RegistroAuditoriaTransaccional adapta el puerto de registro a los casos de
// uso que confirman el cambio y su auditoría en una transacción de negocio.
type RegistroAuditoriaTransaccional struct{ *RegistroAuditoria }

func NuevoRegistroAuditoriaTransaccional(base *Base) *RegistroAuditoriaTransaccional {
	return &RegistroAuditoriaTransaccional{NuevoRegistroAuditoria(base)}
}

func (r *RegistroAuditoriaTransaccional) Confirmar(ctx context.Context, datos auditoria.Datos) (auditoria.Evento, error) {
	return r.RegistrarEnTransaccion(ctx, datos)
}

func (r *RegistroAuditoria) Confirmar(ctx context.Context, datos auditoria.Datos) (auditoria.Evento, error) {
	if r == nil || r.base == nil {
		return auditoria.Evento{}, errors.New("auditoria.configuracion_invalida")
	}
	if err := datos.Validar(); err != nil {
		return auditoria.Evento{}, err
	}
	identidad := plataforma.Identidad{UsuarioID: datos.UsuarioID, Perfiles: datos.Perfiles}
	if datos.UsuarioID == "sistema" {
		if len(datos.Perfiles) != 0 {
			return auditoria.Evento{}, errors.New("auditoria.perfiles_invalidos")
		}
		identidad = plataforma.Identidad{}
	}
	var evento auditoria.Evento
	err := r.base.Ejecutar(ctx, identidad, func(ctx context.Context) error {
		var err error
		evento, err = r.RegistrarEnTransaccion(ctx, datos)
		return err
	})
	if err != nil {
		return auditoria.Evento{}, errorAuditoria(err)
	}
	return evento, nil
}

// RegistrarEnTransaccion nunca confirma ni abre otra transacción. El llamante
// confirma el cambio y el evento juntos mediante Base.Ejecutar. Los metadatos
// recibidos deben coincidir con el actor y perfiles del contexto autorizado.
func (r *RegistroAuditoria) RegistrarEnTransaccion(ctx context.Context, datos auditoria.Datos) (auditoria.Evento, error) {
	if err := datos.Validar(); err != nil {
		return auditoria.Evento{}, err
	}
	ejecutor, err := DesdeContexto(ctx)
	if err != nil {
		return auditoria.Evento{}, err
	}
	var evento auditoria.Evento
	err = ejecutor.QueryRow(ctx, `SELECT e.id, e.ocurrido_en,
		CASE WHEN e.actor_sistema THEN 'sistema' ELSE e.usuario_id::text END,
		e.perfiles, e.accion, e.entidad, coalesce(e.entidad_id::text, ''),
		coalesce(e.expediente_id::text, ''), e.resultado,
		encode(e.huella_anterior, 'hex'), encode(e.huella, 'hex')
		FROM auditoria.registrar($1, $2, $3, $4::uuid, $5::uuid, $6) e`,
		datos.OcurridoEn, datos.Accion, datos.Entidad, uuidOpcional(datos.EntidadID),
		uuidOpcional(datos.ExpedienteID), datos.Resultado).Scan(
		&evento.ID, &evento.OcurridoEn, &evento.UsuarioID, &evento.Perfiles,
		&evento.Accion, &evento.Entidad, &evento.EntidadID, &evento.ExpedienteID,
		&evento.Resultado, &evento.HuellaAnterior, &evento.Huella)
	if err != nil {
		return auditoria.Evento{}, errorAuditoria(err)
	}
	previo := auditoria.Inicio()
	if evento.HuellaAnterior != auditoria.HuellaInicial && evento.ID > 0 {
		previo = auditoria.Punto{ID: evento.ID - 1, Huella: evento.HuellaAnterior}
	}
	esperado, err := auditoria.Encadenar(evento.ID, datos, previo)
	if err != nil || esperado.Huella != evento.Huella {
		return auditoria.Evento{}, errors.New("auditoria.confirmacion_invalida")
	}
	return evento, nil
}

// Exportar retiene el documento en memoria hasta confirmar su apunte. La
// función SQL comprueba además el perfil; una transacción externa se rechaza.
func (r *RegistroAuditoria) Exportar(ctx context.Context, identidad plataforma.Identidad, desde, hasta uint64) (auditoria.Exportacion, error) {
	if err := tramoAuditoria(desde, hasta); err != nil {
		return auditoria.Exportacion{}, err
	}
	var exportacion auditoria.Exportacion
	err := r.operar(ctx, identidad, func(ctx context.Context, ejecutor Ejecutor) error {
		var documento []byte
		if err := ejecutor.QueryRow(ctx, "SELECT auditoria.exportar($1, $2)", int64(desde), int64(hasta)).Scan(&documento); err != nil { //etf:tecnico
			return err
		}
		var err error
		exportacion, err = auditoria.DecodificarExportacion(documento)
		return err
	})
	if err != nil {
		return auditoria.Exportacion{}, err
	}
	return exportacion, nil
}

// Verificar devuelve nil si la cadena es coherente, o el primer ID roto.
func (r *RegistroAuditoria) Verificar(ctx context.Context, identidad plataforma.Identidad, desde, hasta uint64) (*uint64, error) {
	if err := tramoAuditoria(desde, hasta); err != nil {
		return nil, err
	}
	var roto *int64
	err := r.operar(ctx, identidad, func(ctx context.Context, ejecutor Ejecutor) error {
		if err := ejecutor.QueryRow(ctx, "SELECT auditoria.verificar($1, $2)", int64(desde), int64(hasta)).Scan(&roto); err != nil { //etf:tecnico
			return err
		}
		resultado := "ok"
		if roto != nil {
			resultado = "error"
		}
		_, err := ejecutor.Exec(ctx, `SELECT auditoria.registrar(clock_timestamp(),
			'auditoria.verificar', 'auditoria', NULL, NULL, $1)`, resultado)
		return err
	})
	if err != nil || roto == nil {
		return nil, err
	}
	id := uint64(*roto)
	return &id, nil
}

// SellarDiario devuelve un punto confirmado para custodiarlo en el registro
// técnico y la copia independiente. Aquí no se escriben esos sistemas externos.
func (r *RegistroAuditoria) SellarDiario(ctx context.Context, identidad plataforma.Identidad, dia time.Time) (auditoria.Punto, error) {
	var punto auditoria.Punto
	err := r.operar(ctx, identidad, func(ctx context.Context, ejecutor Ejecutor) error {
		if err := ejecutor.QueryRow(ctx, `SELECT s.ultimo_evento_id, encode(s.ultima_huella, 'hex')
			FROM auditoria.sellar_diario($1::date) s`, dia.UTC().Format("2006-01-02")).Scan(&punto.ID, &punto.Huella); err != nil {
			return err
		}
		if err := punto.Validar(); err != nil {
			return err
		}
		_, err := ejecutor.Exec(ctx, `SELECT auditoria.registrar(clock_timestamp(),
			'auditoria.sellar_diario', 'auditoria', NULL, NULL, 'ok')`)
		return err
	})
	if err != nil {
		return auditoria.Punto{}, err
	}
	return punto, nil
}

func (r *RegistroAuditoria) operar(ctx context.Context, identidad plataforma.Identidad, operacion func(context.Context, Ejecutor) error) error {
	if r == nil || r.base == nil {
		return errors.New("auditoria.configuracion_invalida")
	}
	err := r.base.Ejecutar(ctx, identidad, func(ctx context.Context) error {
		ejecutor, err := DesdeContexto(ctx)
		if err != nil {
			return err
		}
		return operacion(ctx, ejecutor)
	})
	return errorAuditoria(err)
}

func tramoAuditoria(desde, hasta uint64) error {
	if desde == 0 || hasta < desde || hasta > math.MaxInt64 {
		return errors.New("auditoria.tramo_invalido")
	}
	return nil
}

func uuidOpcional(id string) any {
	if id == "" {
		return nil
	}
	return id
}

// Nunca se propagan diagnósticos SQL, valores ni nombres de objetos al usuario.
func errorAuditoria(err error) error {
	if err == nil {
		return nil
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Message {
		case "auditoria.usuario_invalido", "auditoria.perfiles_invalidos", "auditoria.fecha_invalida",
			"auditoria.codigo_invalido", "auditoria.resultado_invalido", "auditoria.identificador_invalido",
			"auditoria.permiso_denegado", "auditoria.tramo_invalido", "auditoria.cadena_alterada",
			"auditoria.dia_invalido", "auditoria.exportacion_demasiado_grande":
			return errors.New(pg.Message)
		}
		return errors.New("auditoria.persistencia_fallida")
	}
	// Base y dominio ya devuelven claves; conservamos cancelación y anidamiento.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, ErrNoDisponible) || errors.Is(err, ErrIdentidad) ||
		errors.Is(err, ErrTransaccion) || errors.Is(err, ErrAnidada) {
		return err
	}
	switch err.Error() {
	case "auditoria.formato_invalido", "auditoria.exportacion_demasiado_grande", "auditoria.confirmacion_invalida",
		"auditoria.orden_invalido", "auditoria.enlace_invalido", "auditoria.huella_invalida",
		"auditoria.ancla_invalida", "auditoria.sello_invalido", "auditoria.sello_no_coincide",
		"auditoria.fecha_invalida", "auditoria.usuario_invalido", "auditoria.perfiles_invalidos",
		"auditoria.codigo_invalido", "auditoria.identificador_invalido", "auditoria.resultado_invalido", "auditoria.punto_invalido":
		return err
	}
	return errors.New("auditoria.persistencia_fallida")
}
