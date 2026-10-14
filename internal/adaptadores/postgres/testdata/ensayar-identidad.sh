#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/../../../.."
# Entorno desechable, sin datos reales ni contraseñas persistentes.
ruta=$(mktemp -d)
nombre="etf-f009b-${RANDOM}-${RANDOM}"
red="${nombre}-red"
limpiar() {
    docker rm -f "$nombre" >/dev/null 2>&1 || true
    docker network rm "$red" >/dev/null 2>&1 || true
    rm -rf "$ruta"
}
trap limpiar EXIT
umask 077
secreto=$(openssl rand -hex 24)
cat > "$ruta/postgres.env" <<ENV
POSTGRES_USER=etf_instalador
POSTGRES_DB=etf
POSTGRES_PASSWORD=$secreto
ETF_MIGRADOR_CONTRASENA=$secreto
ETF_APP_CONTRASENA=$secreto
ETF_COPIA_CONTRASENA=$secreto
ENV
cat > "$ruta/migrador.env" <<ENV
ETF_BD_HOST=$nombre
ETF_BD_PUERTO=5432
ETF_BD_NOMBRE=etf
ETF_MIGRADOR_CONTRASENA=$secreto
ETF_MIGRACIONES_RUTA=/migraciones
ENV
docker network create --internal --subnet "10.253.$((RANDOM % 240 + 10)).0/24" "$red" >/dev/null
docker run --detach --name "$nombre" --network "$red" \
    --read-only --user postgres --cap-drop=ALL --security-opt=no-new-privileges \
    --tmpfs /var/lib/postgresql:rw,uid=999,gid=999 --tmpfs /var/run/postgresql:rw,uid=999,gid=999 \
    --tmpfs /tmp:rw --env-file "$ruta/postgres.env" \
    --mount "type=bind,src=$PWD/despliegue/base/01_roles.sh,dst=/docker-entrypoint-initdb.d/01_roles.sh,readonly" \
    postgres:18 >/dev/null
for ((i=0;i<60;i++)); do
    if docker exec "$nombre" pg_isready -h 127.0.0.1 -U etf_instalador -d etf >/dev/null 2>&1; then break; fi
    sleep 0.25
done
docker exec "$nombre" pg_isready -h 127.0.0.1 -U etf_instalador -d etf >/dev/null
mkdir -p "$ruta/anterior" "$ruta/conjunto"
cp -R migraciones/. "$ruta/conjunto/"
# Compatibilidad con ramas anteriores a F0.08b. Se incorpora sólo su migración en esta
# carpeta desechable, sin fusionar ni modificar otro módulo del worktree.
if [[ ! -f "$ruta/conjunto/auditoria/manifiesto.json" ]]; then
    auditoria_ref=$(git rev-parse "${ETF_AUDITORIA_REF:-main}")
    git archive "$auditoria_ref" migraciones/auditoria | tar -x -C "$ruta"
    cp -R "$ruta/migraciones/auditoria" "$ruta/conjunto/"
    printf 'DEPENDENCIA-F0.08b %s\n' "$auditoria_ref"
fi
chmod -R a+rX "$ruta/conjunto"
cp -R "$ruta/conjunto/." "$ruta/anterior/"
rm -f "$ruta/anterior/identidad/0002_perfiles.sql"
python3 - "$ruta/anterior/identidad/manifiesto.json" <<'PY'
import json,sys
p=sys.argv[1]
v=json.load(open(p));open(p,'w').write(json.dumps([m for m in v if m['numero']!='0002']))
PY
chmod 755 "$ruta"
chmod -R a+rX "$ruta/anterior"
# El migrador del proyecto necesita psql; la imagen local de F0.03 lo incluye.
migrar() {
    docker run --rm --network "$red" --read-only --cap-drop=ALL --security-opt=no-new-privileges \
        --tmpfs /tmp:rw --env-file "$ruta/migrador.env" \
        --mount "type=bind,src=$1,dst=/migraciones,readonly" etf-f003-migrar
}
migrar "$ruta/anterior"
migrar "$ruta/conjunto"
migrar "$ruta/conjunto"
# También se ensaya desde cero, no sólo desde la versión anterior.
reiniciar_base() {
    docker exec "$nombre" psql -X -U etf_instalador -d etf -v ON_ERROR_STOP=1 \
        -c 'DROP SCHEMA documento CASCADE; DROP SCHEMA expediente CASCADE; DROP SCHEMA paquete CASCADE; DROP SCHEMA identidad CASCADE; DROP SCHEMA catalogo CASCADE; DROP SCHEMA auditoria CASCADE; DROP SCHEMA plataforma CASCADE;' >/dev/null 2>&1
    migrar "$ruta/conjunto"
}
reiniciar_base
direccion=$(docker inspect --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$nombre")
export ETF_POSTGRES_INTEGRACION=1 ETF_BD_HOST="$direccion" ETF_BD_PUERTO=5432 ETF_BD_NOMBRE=etf
export ETF_POSTGRES_CONTENEDOR="$nombre"
export ETF_AUDITORIA_REF="${auditoria_ref:-${ETF_AUDITORIA_REF:-main}}"
export ETF_BD_CONTRASENA="$secreto" ETF_MIGRADOR_CONTRASENA="$secreto" ETF_BD_SSLMODE=disable
unset PGOPTIONS PGSERVICE PGSERVICEFILE PGPASSWORD PGUSER
# Las pruebas de base fuerzan una conexión; la de bloqueo usa otro pool.
# Los paquetes comparten esta base desechable; no se ejecutan sus preparaciones a la vez.
go test -p 1 -count=1 -v ./internal/adaptadores/postgres ./internal/adaptadores/postgres/catalogo ./internal/adaptadores/postgres/expediente
if [[ "${ETF_NAVEGADOR_INTEGRACION:-}" == 1 ]]; then
    # Las pruebas de paquetes publican referencias a ficheros temporales que ya han limpiado.
    # El navegador arranca con las referencias iniciales y los ficheros reales del worktree.
    reiniciar_base
    (cd web && npm run build)
    go test -tags navegador_integracion -count=1 -run TestNavegadorIdentidad -v ./internal/adaptadores/postgres
    go test -tags navegador_integracion -count=1 -run TestNavegadorPerfiles -v ./cmd/etf
fi
printf 'ENSAYO-OK\n'
