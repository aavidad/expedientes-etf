#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/../.."
gitleaks git --redact --no-banner --no-color --log-opts='--all' .

# Incluir cambios sin commit y nuevos ficheros; excluir .git, secretos locales,
# dependencias y compilaciones ignoradas. Tar conserva enlaces sin seguirlos.
mkdir -p bin
copia=$(mktemp -d bin/ci-secretos.XXXXXX)
trap 'rm -rf -- "$copia"' EXIT
git ls-files -z --cached --others --exclude-standard |
    tar --null --verbatim-files-from -T - -cf - |
    tar -xf - -C "$copia"
gitleaks dir --redact --no-banner --no-color "$copia"
