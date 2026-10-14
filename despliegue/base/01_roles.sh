#!/bin/sh
set -eu
# Nunca activar xtrace: los secretos sólo pasan por el entorno y variables psql.
export ETF_CALCULO_MEMORIA_CONTRASENA="${ETF_CALCULO_MEMORIA_CONTRASENA:-}"
psql -X --no-password --set=ON_ERROR_STOP=1 --username="$POSTGRES_USER" --dbname="$POSTGRES_DB" <<'SQL'
\getenv migrador ETF_MIGRADOR_CONTRASENA
\getenv app ETF_APP_CONTRASENA
\getenv copia ETF_COPIA_CONTRASENA
\getenv calculo ETF_CALCULO_MEMORIA_CONTRASENA
BEGIN;
CREATE ROLE etf_propietario NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE etf_migrador LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'migrador';
CREATE ROLE etf_app LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'app';
CREATE ROLE etf_copia LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION BYPASSRLS PASSWORD :'copia';
CREATE ROLE etf_calculo_memoria NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
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
-- Retirar el secreto de instalación: el operador sólo administra por socket local.
ALTER ROLE etf_instalador PASSWORD NULL;
COMMIT;
SQL
