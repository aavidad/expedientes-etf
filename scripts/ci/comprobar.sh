#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/../.."


for herramienta in git go npm python3 gitleaks tar docker; do
    command -v "$herramienta" >/dev/null || {
        printf 'Falta la herramienta requerida: %s\n' "$herramienta" >&2
        exit 1
    }
done

printf '\n== Formato Go ==\n'
sin_formato=$(git ls-files -z --cached --others --exclude-standard -- '*.go' | xargs -0 -r gofmt -l)
if [[ -n "$sin_formato" ]]; then
    printf '%s\n' "$sin_formato" >&2
    exit 1
fi

printf '\n== Catálogos de ubicación ==\n'
python3 -B scripts/ci/catalogos/comprobar.py
python3 -B migraciones/catalogo/generar_carga.py --comprobar

printf '\n== Índice de documentos de producto ==\n'
python3 -B scripts/ci/indice/generar.py --comprobar

printf '\n== Enlaces Markdown ==\n'
python3 -B -m unittest discover -s scripts/ci -p 'test_*.py'
python3 -B scripts/ci/enlaces.py

printf '\n== Guiones y montaje de recorridos ==\n'
python3 -B scripts/recorridos/videos.py --validar

printf '\n== Secretos: historial y ficheros locales ==\n'
./scripts/ci/secretos.sh

python3 -B scripts/voz/construir.py
python3 -B -m unittest discover -s scripts/voz -p 'test_*.py'
# Sólo silencio local: nunca se invoca edge-tts desde CI.
make voz-simulada

printf '\n== Frontend: npm ci, tsc, pruebas y build ==\n'
(
    cd web
    npm ci --ignore-scripts --no-audit --no-fund
    ./node_modules/.bin/tsc --noEmit
    npm test
    npm run build
)

python3 -B -m unittest discover -s scripts/recorridos -p 'test_*.py'
python3 -B -m unittest discover -s despliegue/demo -p 'test_*.py'
python3 -B -m unittest discover -s despliegue/windows -p 'test_*.py'
python3 -B -m unittest discover -s scripts/pwa1 -p 'test_*.py'

printf '\n== Textos e idiomas ==\n'
./scripts/ci/textos/comprobar.sh

printf '\n== Temas ==\n'
./scripts/ci/temas/comprobar.sh

printf '\n== Go: vet, pruebas (incluida arquitectura) y build ==\n'
go vet ./...
go test ./...
mkdir -p bin
go build -trimpath -o bin/etf ./cmd/etf
go build -trimpath -o bin/migrar ./cmd/migrar
printf '\n== Arranque PostgreSQL: servidor provisional y TCP autenticado ==\n'
python3 -B scripts/ci/ensayar-arranque-postgres.py
printf '\n== Identidad SEG2: autocambios, RLS y auditoría ==\n'
python3 -B migraciones/identidad/ensayar.py
printf '\n== Catálogo: PostgreSQL 18, versiones, publicación y permisos ==\n'
python3 -B migraciones/catalogo/ensayar.py
printf '\n== Expediente inicial: PostgreSQL 18, actualización y ClamAV ==\n'
python3 -B migraciones/expediente/ensayar.py
printf '\n== Aceptación: PostgreSQL 18, migraciones, concurrencia y RLS ==\n'
python3 -B migraciones/expediente/ensayar_aceptacion.py
printf '\nDocumentos de entrada: PostgreSQL 18, emisiones, registro y RLS ==\n'
python3 -B migraciones/documento/ensayar.py
printf '\n== Actividad: PostgreSQL 18, comisiones, informes, recuentos y RLS ==\n'
python3 -B migraciones/actividad/ensayar.py
printf '\n== Memoria: PostgreSQL 18, narrativas, Word, idempotencia y RLS ==\n'
python3 -B migraciones/estadistica/ensayar.py
printf '\n== Evaluación: PostgreSQL 18, respuestas, HTTP y RLS ==\n'
python3 -B migraciones/evaluacion/ensayar.py
python3 -B migraciones/expediente/ensayar_bandeja.py
python3 -B migraciones/expediente/ensayar_conservacion.py
python3 -B despliegue/demo/ensayar_incorporacion.py
printf '\n== Entrada, alta y permisos: Chrome del sistema y PostgreSQL ==\n'
python3 -B scripts/recorridos/entrada_permisos.py
printf '\n== Carga de 2026: alta, hoja y recuperación con Chrome ==\n'
# Cada bloque independiente dispone de su propia ventana de admisión por IP.
reiniciar_demo() {
python3 -B - <<'PYTHON'
import importlib.util
from pathlib import Path
ruta = Path("despliegue/demo/demo.py").resolve()
spec = importlib.util.spec_from_file_location("demo_ci_carga", ruta)
demo = importlib.util.module_from_spec(spec)
spec.loader.exec_module(demo)
demo.compose("restart", "--no-deps", "app")
demo.compose("up", "--detach", "--no-deps", "--wait", "app")
PYTHON
}
reiniciar_demo
puerto_carga=$(cat despliegue/demo/.local/puerto)
node scripts/recorridos/seg3.mjs "http://127.0.0.1:${puerto_carga}"
reiniciar_demo
node scripts/recorridos/seg2.mjs "http://127.0.0.1:${puerto_carga}"
node scripts/recorridos/carga.mjs "http://127.0.0.1:${puerto_carga}"
node scripts/recorridos/carga-actas.mjs "http://127.0.0.1:${puerto_carga}"
reiniciar_demo
node scripts/recorridos/resp4.mjs "http://127.0.0.1:${puerto_carga}"
printf '\n== Restauración: revisión obligatoria, PostgreSQL/RLS y Chrome ==\n'
python3 -B scripts/ensayar-restauracion.py
reiniciar_demo
node scripts/recorridos/ubicaciones.mjs "http://127.0.0.1:${puerto_carga}"
# Separa las suites para respetar la ventana de admisión por IP.
python3 -B -c 'import time; time.sleep(60)'
node scripts/recorridos/anexo6.mjs "http://127.0.0.1:${puerto_carga}"
python3 -B -c 'import time; time.sleep(60)'
printf '\n== Desglose de Memoria: suma, permisos y Chrome ==\n'
node scripts/recorridos/memoria-desglose.mjs "http://127.0.0.1:${puerto_carga}"
reiniciar_demo
node scripts/recorridos/voz.mjs "http://127.0.0.1:${puerto_carga}"
printf '\nCI mínima completada.\n'
