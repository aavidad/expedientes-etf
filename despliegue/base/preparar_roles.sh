#!/bin/sh
set -eu
# Nunca activar xtrace: los secretos sólo pasan por el entorno y variables psql.
: "${ETF_MIGRADOR_CONTRASENA:?}" "${ETF_APP_CONTRASENA:?}" "${ETF_COPIA_CONTRASENA:?}"
export ETF_CALCULO_MEMORIA_CONTRASENA="${ETF_CALCULO_MEMORIA_CONTRASENA:-}"
# Sólo el socket privado; el instalador nunca recupera una contraseña TCP.
psql -X -q --no-password --set=ON_ERROR_STOP=1 --host=/var/run/postgresql \
    --username=etf_instalador --dbname="${POSTGRES_DB:-etf}" <<'SQL'
\getenv migrador ETF_MIGRADOR_CONTRASENA
\getenv app ETF_APP_CONTRASENA
\getenv copia ETF_COPIA_CONTRASENA
\getenv calculo ETF_CALCULO_MEMORIA_CONTRASENA
BEGIN;
SET LOCAL lock_timeout = '30s';
SET LOCAL statement_timeout = '60s';
SELECT pg_advisory_xact_lock(741260, 1);
SELECT format('CREATE ROLE %I NOLOGIN', nombre)
FROM (VALUES ('etf_propietario'), ('etf_migrador'), ('etf_app'), ('etf_copia'), ('etf_calculo_memoria')) AS roles(nombre)
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = nombre) \gexec
-- Una pertenencia inesperada no se corrige concediendo más privilegios.
DO $$ BEGIN
 IF EXISTS (
   SELECT 1 FROM pg_auth_members m
   JOIN pg_roles miembro ON miembro.oid = m.member
   JOIN pg_roles concedido ON concedido.oid = m.roleid
   WHERE miembro.rolname IN ('etf_propietario', 'etf_migrador', 'etf_app', 'etf_copia', 'etf_calculo_memoria')
     AND NOT (miembro.rolname = 'etf_migrador' AND concedido.rolname = 'etf_propietario' AND NOT m.admin_option)
 ) THEN RAISE EXCEPTION 'postgres.configuracion_invalida'; END IF;
END $$;
ALTER ROLE etf_propietario NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD NULL;
ALTER ROLE etf_migrador LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'migrador';
ALTER ROLE etf_app LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'app';
ALTER ROLE etf_copia LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION BYPASSRLS PASSWORD :'copia';
ALTER ROLE etf_calculo_memoria NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD NULL;
SELECT length(:'calculo') > 0 AS habilitar_calculo \gset
\if :habilitar_calculo
ALTER ROLE etf_calculo_memoria LOGIN PASSWORD :'calculo';
\endif
GRANT etf_propietario TO etf_migrador WITH INHERIT FALSE, SET TRUE;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SELECT format('REVOKE ALL ON DATABASE %I FROM PUBLIC', current_database()) \gexec
SELECT format('ALTER DATABASE %I OWNER TO etf_propietario', current_database()) \gexec
SELECT format('GRANT CONNECT ON DATABASE %I TO etf_migrador, etf_app, etf_copia, etf_calculo_memoria', current_database()) \gexec
ALTER DEFAULT PRIVILEGES FOR ROLE etf_propietario REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;
ALTER ROLE etf_instalador PASSWORD NULL;
COMMIT;
SQL
