#!/bin/sh
set -eu
# Operación previa a migrar una instalación existente. Secretos sólo por entorno.
: "${ETF_CALCULO_MEMORIA_CONTRASENA:?}"
export ETF_CALCULO_MEMORIA_CONTRASENA
psql -X --no-password --set=ON_ERROR_STOP=1 --username="${POSTGRES_USER:-etf_instalador}" --dbname="${POSTGRES_DB:-etf}" <<'SQL'
\getenv calculo ETF_CALCULO_MEMORIA_CONTRASENA
BEGIN;
SELECT 'CREATE ROLE etf_calculo_memoria NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS'
WHERE NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='etf_calculo_memoria') \gexec
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='etf_calculo_memoria' AND (rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR rolreplication))
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member=(SELECT oid FROM pg_roles WHERE rolname='etf_calculo_memoria')) THEN
 RAISE EXCEPTION 'postgres.configuracion_invalida'; END IF;
END $$;
ALTER ROLE etf_calculo_memoria LOGIN PASSWORD :'calculo';
SELECT format('GRANT CONNECT ON DATABASE %I TO etf_calculo_memoria',current_database()) \gexec
COMMIT;
SQL
