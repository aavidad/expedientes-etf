#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/../../.."
python3 -B -m unittest discover -s scripts/ci/temas -p 'test_*.py'
python3 -B scripts/ci/temas/comprobar.py
go test ./internal/dominio/paquete ./internal/adaptadores/ficheros ./internal/aplicacion/paquete
