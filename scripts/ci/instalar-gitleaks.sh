#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/../.."
# Herramienta del runner ubuntu-24.04 (Linux x64), no instalador de la app.
mkdir -p bin/ci-gitleaks
archivo=bin/ci-gitleaks/gitleaks.tar.gz
trap 'rm -f -- "$archivo"' EXIT
curl --fail --silent --show-error --location --proto '=https' --proto-redir '=https' \
    https://github.com/gitleaks/gitleaks/releases/download/v8.30.1/gitleaks_8.30.1_linux_x64.tar.gz \
    --output "$archivo"
printf '%s  %s\n' \
    551f6fc83ea457d62a0d98237cbad105af8d557003051f41f3e7ca7b3f2470eb "$archivo" |
    sha256sum --check --strict
tar -xzf "$archivo" -C bin/ci-gitleaks gitleaks
chmod +x bin/ci-gitleaks/gitleaks
