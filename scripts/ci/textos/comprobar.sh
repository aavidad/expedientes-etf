#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/../../.."
node --test scripts/ci/textos/comprobar.test.mjs
go run ./scripts/ci/textos/comprobar.go | node scripts/ci/textos/comprobar.mjs
