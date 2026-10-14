package postgres

import (
	"context"
	"errors"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

var (
	ErrConfiguracion = errors.New("postgres.configuracion_invalida")
	ErrNoDisponible  = errors.New("postgres.no_disponible")
	ErrIdentidad     = errors.New("postgres.identidad_invalida")
	ErrTransaccion   = errors.New("postgres.transaccion_no_disponible")
	ErrAnidada       = errors.New("postgres.transaccion_anidada")
)

// Base mantiene el pool privado: los repositorios sólo obtienen el ejecutor
// asociado al contexto de una operación, nunca una conexión sin contexto RLS.
type Base struct{ pool *pgxpool.Pool }

var _ plataforma.Transacciones = (*Base)(nil)

// Nueva usa únicamente las opciones ETF_BD_* del arranque. No recibe ni lee
// secretos del migrador. La construcción del pool no exige una base disponible.
// Sin configuración válida devuelve una base desactivada y una clave de error;
// el proceso puede seguir atendiendo salud y recursos que no usan persistencia.
// Composición debe retirar PG* antes de arrancar goroutines: pgx no permite
// desactivar su lectura de PGSERVICE/PGSERVICEFILE desde ParseConfig.
func Nueva(ctx context.Context, leer func(string) string) (*Base, error) {
	b := &Base{}
	cfg, err := configuracion(leer)
	if err != nil {
		return b, err
	}
	b.pool, err = pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return b, ErrNoDisponible
	}
	return b, nil
}

// NuevaCalculoMemoria conecta exclusivamente con la credencial técnica.
// Nunca reutiliza la contraseña de etf_app ni acepta el rol del navegador.
func NuevaCalculoMemoria(ctx context.Context, leer func(string) string) (*Base, error) {
	secreto := leer("ETF_CALCULO_MEMORIA_CONTRASENA")
	if secreto == "" {
		return &Base{}, ErrConfiguracion
	}
	cfg, err := configuracion(func(k string) string {
		if k == "ETF_BD_USUARIO" {
			return "etf_app"
		}
		if k == "ETF_BD_CONTRASENA" {
			return secreto
		}
		return leer(k)
	})
	if err != nil {
		return &Base{}, err
	}
	cfg.ConnConfig.User = "etf_calculo_memoria"
	cfg.MaxConns = 2
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		var permitido bool
		err := conn.QueryRow(ctx, `SELECT current_user='etf_calculo_memoria' AND session_user='etf_calculo_memoria' AND NOT rolsuper AND NOT rolbypassrls AND NOT rolcreaterole AND NOT rolcreatedb AND NOT rolreplication AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=r.oid) FROM pg_catalog.pg_roles r WHERE rolname=current_user`).Scan(&permitido) //etf:tecnico
		if err != nil || !permitido {
			return ErrNoDisponible
		}
		return nil
	}
	b := &Base{}
	b.pool, err = pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return &Base{}, ErrNoDisponible
	}
	return b, nil
}

func configuracion(leer func(string) string) (*pgxpool.Config, error) {
	valor := func(clave, reserva string) string {
		if v := leer(clave); v != "" {
			return v
		}
		return reserva
	}
	if valor("ETF_BD_USUARIO", "etf_app") != "etf_app" || leer("ETF_BD_CONTRASENA") == "" {
		return nil, ErrConfiguracion
	}
	puerto, err := strconv.Atoi(valor("ETF_BD_PUERTO", "5432"))
	if err != nil || puerto < 1 || puerto > 65535 {
		return nil, ErrConfiguracion
	}
	maximo, err := strconv.ParseInt(valor("ETF_BD_MAX_CONEXIONES", "4"), 10, 32)
	if err != nil || maximo < 1 || maximo > 64 {
		return nil, ErrConfiguracion
	}
	host := valor("ETF_BD_HOST", "127.0.0.1")
	if strings.ContainsAny(host, ",/\\ \t\r\n") {
		return nil, ErrConfiguracion
	}
	u := url.URL{Scheme: "postgres", Host: net.JoinHostPort(host, strconv.Itoa(puerto)),
		User: url.UserPassword("etf_app", leer("ETF_BD_CONTRASENA")),
		Path: "/" + valor("ETF_BD_NOMBRE", "etf")}
	// Compose usa una red interna sin TLS. Una instalación con TLS puede exigir
	// verify-full y su CA. No se acepta prefer/allow, que permiten bajar a claro.
	ssl := valor("ETF_BD_SSLMODE", "disable")
	if ssl != "disable" && ssl != "verify-full" {
		return nil, ErrConfiguracion
	}
	q := url.Values{"sslmode": {ssl}, "connect_timeout": {"2"},
		"target_session_attrs": {"any"}, "sslsni": {"1"}, "sslnegotiation": {"postgres"},
		"sslcert": {""}, "sslkey": {""}, "sslrootcert": {leer("ETF_BD_SSLROOTCERT")}}
	u.RawQuery = q.Encode()
	cfg, err := pgxpool.ParseConfig(u.String())
	if err != nil {
		return nil, ErrConfiguracion
	}
	// El entorno libpq (PGOPTIONS, etc.) no puede aportar roles ni contexto RLS.
	cfg.ConnConfig.RuntimeParams = map[string]string{"application_name": "etf"}
	cfg.MaxConns, cfg.MinConns, cfg.MinIdleConns = int32(maximo), 0, 0
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		var permitido bool
		err := conn.QueryRow(ctx, `SELECT current_user = 'etf_app' AND session_user = 'etf_app'
			AND NOT rolsuper AND NOT rolbypassrls FROM pg_catalog.pg_roles WHERE rolname = current_user`).Scan(&permitido)
		if err != nil || !permitido {
			return ErrNoDisponible
		}
		return nil
	}
	return cfg, nil
}

// Comprobar reintenta en cada llamada y limita la espera, también si el pool
// está ocupado. Nunca devuelve diagnósticos SQL, credenciales ni direcciones.
func (b *Base) Comprobar(ctx context.Context) error {
	if b.pool == nil {
		return ErrNoDisponible
	}
	ctx, cancelar := context.WithTimeout(ctx, 2*time.Second)
	defer cancelar()
	if err := b.pool.Ping(ctx); err != nil {
		return ErrNoDisponible
	}
	return nil
}

func (b *Base) Cerrar() {
	if b.pool != nil {
		b.pool.Close()
	}
}

type claveTransaccion struct{}

// Ejecutor es sólo para los adaptadores PostgreSQL de los módulos. Omite
// Commit/Rollback: el único propietario del ciclo de vida es Ejecutar.
type Ejecutor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// DesdeContexto falla fuera de una operación; no hay reserva al pool.
func DesdeContexto(ctx context.Context) (Ejecutor, error) {
	tx, ok := ctx.Value(claveTransaccion{}).(pgx.Tx)
	if !ok {
		return nil, ErrTransaccion
	}
	return tx, nil
}

func (b *Base) Ejecutar(ctx context.Context, identidad plataforma.Identidad, operacion func(context.Context) error) (err error) {
	if ctx.Value(claveTransaccion{}) != nil {
		return ErrAnidada
	}
	usuario, perfiles, equipos, err := contextoRLS(identidad)
	if err != nil {
		return err
	}
	if operacion == nil {
		return ErrTransaccion
	}
	if b.pool == nil {
		return ErrNoDisponible
	}
	tx, err := b.pool.Begin(ctx)
	if err != nil {
		return ErrNoDisponible
	}
	defer func() {
		// La cancelación de la petición no impide devolver la conexión limpia.
		cierre, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancelar()
		if fallo := tx.Rollback(cierre); fallo != nil && !errors.Is(fallo, pgx.ErrTxClosed) {
			err = errors.Join(err, ErrNoDisponible)
		}
	}()
	if _, err := tx.Exec(ctx, `SELECT set_config('app.usuario_id', $1, true),
		set_config('app.perfiles', $2, true), set_config('app.equipos', $3, true)`, usuario, perfiles, equipos); err != nil {
		return ErrNoDisponible
	}
	if err := operacion(context.WithValue(ctx, claveTransaccion{}, tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return ErrNoDisponible
	}
	return nil
}

func contextoRLS(i plataforma.Identidad) (string, string, string, error) {
	uuid := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	codigo := regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
	if i.UsuarioID == "" {
		if len(i.Perfiles) != 0 || i.EquipoID != "" {
			return "", "", "", ErrIdentidad
		}
		return "", "", "", nil
	}
	if !uuid.MatchString(i.UsuarioID) || (i.EquipoID != "" && !uuid.MatchString(i.EquipoID)) {
		return "", "", "", ErrIdentidad
	}
	for _, perfil := range i.Perfiles {
		if !codigo.MatchString(perfil) {
			return "", "", "", ErrIdentidad
		}
	}
	equipos := "{}"
	if i.EquipoID != "" {
		equipos = "{" + i.EquipoID + "}"
	}
	return i.UsuarioID, strings.Join(i.Perfiles, ","), equipos, nil
}
