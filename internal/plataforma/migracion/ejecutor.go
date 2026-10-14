package migracion

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Configuracion no acepta un rol variable: siempre conecta como etf_migrador.
type Configuracion struct{ Host, Puerto, Base, Contrasena string }

// Ejecutar usa una única sesión, bloqueo asesor de sesión y una transacción por
// migración. No expone SQL, errores del servidor ni contraseñas en los registros.
func Ejecutar(ctx context.Context, c Configuracion, plan []Migracion, registro io.Writer) error {
	if c.Host == "" || c.Puerto == "" || c.Base == "" || c.Contrasena == "" || len(plan) == 0 {
		return errors.New("migracion.configuracion")
	}
	comando := exec.CommandContext(ctx, "psql", "-X", "-q", "--no-password", "-v", "ON_ERROR_STOP=1", "-f", "-")
	// Entorno cerrado: ignora PGOPTIONS, .psqlrc, servicios y roles del operador.
	comando.Env = []string{"PATH=/usr/local/bin:/usr/bin:/bin", "HOME=/nonexistent", "PGHOST=" + c.Host, "PGPORT=" + c.Puerto, "PGDATABASE=" + c.Base, "PGUSER=etf_migrador", "PGPASSWORD=" + c.Contrasena, "PGCONNECT_TIMEOUT=10", "PGSSLMODE=disable", "LC_ALL=C"}
	comando.Stdin = strings.NewReader(guion(plan))
	var salida, fallo bytes.Buffer
	comando.Stdout = &salida
	comando.Stderr = &fallo
	if err := comando.Run(); err != nil {
		// Sólo claves propias de SQL; nunca el diagnóstico crudo de PostgreSQL.
		for _, clave := range []string{"migracion.rol", "migracion.huella", "migracion.historial", "migracion.retroceso", "migracion.dependencia", "migracion.estado_previo"} {
			if strings.Contains(fallo.String(), clave) {
				return errors.New(clave)
			}
		}
		return errors.New("migracion.postgres")
	}
	// stdout sólo contiene las claves creadas por el guion, sin filas ni SQL.
	for _, m := range plan {
		for _, estado := range []string{"aplicada", "existente"} {
			clave := "migracion." + estado + "." + m.Modulo + "." + m.Numero //etf:tecnico
			if strings.Contains(salida.String(), clave+"\n") {
				fmt.Fprintln(registro, clave)
			}
		}
	}
	return nil
}

func literal(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// El guion SQL/psql se envía a stdin; no contiene textos de pantalla.
//
//etf:tecnico
func guion(plan []Migracion) string {
	var s strings.Builder
	s.WriteString("\\o /dev/null\nSET standard_conforming_strings = on;\nSET lock_timeout = '10s';\nSET statement_timeout = '60s';\nDO $$ BEGIN IF session_user <> 'etf_migrador' THEN RAISE EXCEPTION 'migracion.rol'; END IF; END $$;\nSELECT pg_advisory_lock(697003);\nSET ROLE etf_propietario;\nSELECT to_regclass('plataforma.migracion_aplicada') IS NOT NULL AS control_existe \\gset\n\\if :control_existe\n")
	// Valida todo el historial ANTES de aplicar ninguna migración pendiente.
	s.WriteString("DO $control$ BEGIN\nIF NOT EXISTS (SELECT 1 FROM plataforma.migracion_aplicada WHERE modulo='plataforma' AND numero='0001') THEN RAISE EXCEPTION 'migracion.historial'; END IF;\nIF EXISTS (SELECT 1 FROM plataforma.migracion_aplicada WHERE (modulo, numero, nombre, huella_sha256) NOT IN (VALUES\n")
	for i, m := range plan {
		if i > 0 {
			s.WriteString(",\n")
		}
		fmt.Fprintf(&s, "(%s,%s,%s,%s)", literal(m.Modulo), literal(m.Numero), literal(m.Nombre), literal(m.Huella))
	}
	s.WriteString(")) THEN RAISE EXCEPTION 'migracion.huella'; END IF;\n")
	for _, m := range plan {
		fmt.Fprintf(&s, "IF NOT EXISTS (SELECT 1 FROM plataforma.migracion_aplicada WHERE modulo=%s AND numero=%s) AND EXISTS (SELECT 1 FROM plataforma.migracion_aplicada WHERE modulo=%s AND numero>%s) THEN RAISE EXCEPTION 'migracion.retroceso'; END IF;\n", literal(m.Modulo), literal(m.Numero), literal(m.Modulo), literal(m.Numero))
		for _, d := range m.Dependencias {
			fmt.Fprintf(&s, "IF EXISTS (SELECT 1 FROM plataforma.migracion_aplicada WHERE modulo=%s AND numero=%s) AND NOT EXISTS (SELECT 1 FROM plataforma.migracion_aplicada WHERE modulo=%s AND numero=%s) THEN RAISE EXCEPTION 'migracion.dependencia'; END IF;\n", literal(m.Modulo), literal(m.Numero), literal(d.Modulo), literal(d.Numero))
		}
	}
	s.WriteString("END $control$;\n\\endif\n")
	for i, m := range plan {
		if i == 0 {
			s.WriteString("\\if :control_existe\n\\set pendiente false\n\\else\n\\set pendiente true\n\\endif\n")
		} else {
			fmt.Fprintf(&s, "SELECT NOT EXISTS (SELECT 1 FROM plataforma.migracion_aplicada WHERE modulo=%s AND numero=%s) AS pendiente \\gset\n", literal(m.Modulo), literal(m.Numero))
		}
		s.WriteString("\\if :pendiente\nBEGIN;\nSET LOCAL lock_timeout = '10s';\nSET LOCAL statement_timeout = '60s';\n")
		for _, d := range m.Dependencias {
			fmt.Fprintf(&s, "DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM plataforma.migracion_aplicada WHERE modulo=%s AND numero=%s) THEN RAISE EXCEPTION 'migracion.dependencia'; END IF; END $$;\n", literal(d.Modulo), literal(d.Numero))
		}
		s.WriteString(m.SQL)
		s.WriteString("\n")
		fmt.Fprintf(&s, "INSERT INTO plataforma.migracion_aplicada(modulo,numero,nombre,huella_sha256,aplicada_por) VALUES (%s,%s,%s,%s,session_user);\nCOMMIT;\n\\echo migracion.aplicada.%s.%s\n\\else\n\\echo migracion.existente.%s.%s\n\\endif\n", literal(m.Modulo), literal(m.Numero), literal(m.Nombre), literal(m.Huella), m.Modulo, m.Numero, m.Modulo, m.Numero)
	}
	s.WriteString("SELECT pg_advisory_unlock(697003);\n")
	return s.String()
}
