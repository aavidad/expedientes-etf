#!/usr/bin/env bash
# Pruebas y análisis locales sin red ni escritura en el repositorio.
set -euo pipefail
zona=$(cd -- "$(dirname -- "$0")" && pwd)
repo=$(cd -- "$zona/../../../.." && pwd)
modulos=$(go env GOMODCACHE)
cadena=$(go env GOROOT)
ulimit -t 120 -v 6291456 -u 4096 -f 524288 -c 0
exec bwrap --unshare-all --die-with-parent --new-session \
  --ro-bind /usr /usr --ro-bind /bin /bin --ro-bind /lib /lib --ro-bind /lib64 /lib64 \
  --ro-bind /etc/ld.so.cache /etc/ld.so.cache \
  --ro-bind /etc/ssl/certs /etc/ssl/certs \
  --ro-bind "$repo" /repo --ro-bind "$modulos" /deps --ro-bind "$cadena" /toolchain \
  --size 536870912 --tmpfs /scratch --size 268435456 --tmpfs /tmp \
  --proc /proc --dev /dev --clearenv \
  --setenv PATH /toolchain/bin:/usr/bin:/bin \
  --setenv HOME /scratch/home --setenv TMPDIR /tmp --setenv LANG C.UTF-8 \
  --setenv GOCACHE /scratch/cache --setenv GOMODCACHE /deps \
  --setenv GOPROXY off --setenv GOTOOLCHAIN local --setenv CGO_ENABLED 0 \
  --setenv SEMGREP_SEND_METRICS off \
  --setenv GOMAXPROCS 4 --chdir /repo \
  timeout 120 sh -c 'mkdir -p /scratch/home /scratch/cache; exec "$@"' sandbox "$@"
