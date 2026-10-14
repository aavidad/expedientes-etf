#!/usr/bin/env bash
# Ejecución local aislada: escritura efímera con cuota y sin red.
set -euo pipefail
zona=$(cd -- "$(dirname -- "$0")" && pwd)
repo=$(cd -- "$zona/../../../.." && pwd)
scratch="$zona/.trabajo"
mkdir -p "$scratch"
printf 'sandbox:x:%s:%s::/scratch/home:/bin/sh\n' "$(id -u)" "$(id -g)" > "$scratch/passwd"
printf '127.0.0.1 localhost\n::1 localhost\n' > "$scratch/hosts"
modulos=$(go env GOMODCACHE)
cadena=$(go env GOROOT)
ulimit -t 60 -v 4194304 -u 4096 -f 524288 -c 0
exec bwrap --unshare-all --die-with-parent --new-session \
  --ro-bind /usr /usr --ro-bind /bin /bin --ro-bind /lib /lib --ro-bind /lib64 /lib64 \
  --ro-bind /etc/fonts /etc/fonts --ro-bind /etc/libreoffice /etc/libreoffice \
  --ro-bind /etc/ld.so.cache /etc/ld.so.cache \
  --ro-bind "$scratch/passwd" /etc/passwd --ro-bind "$scratch/hosts" /etc/hosts \
  --proc /proc --dev /dev --ro-bind "$repo" /repo \
  --tmpfs /repo/internal/adaptadores/anexos/pdf/.trabajo \
  --remount-ro /repo/internal/adaptadores/anexos/pdf/.trabajo \
  --ro-bind "$modulos" /deps --ro-bind "$cadena" /toolchain \
  --size 536870912 --tmpfs /scratch --size 268435456 --tmpfs /tmp \
  --clearenv --setenv PATH /toolchain/bin:/usr/bin:/bin \
  --setenv HOME /scratch/home --setenv TMPDIR /tmp --setenv LANG C.UTF-8 \
  --setenv GOCACHE /scratch/cache --setenv GOMODCACHE /deps \
  --setenv GOPROXY off --setenv GOTOOLCHAIN local --setenv CGO_ENABLED 0 \
  --setenv SAL_USE_VCLPLUGIN svp \
  --chdir /repo/internal/adaptadores/anexos/pdf \
  timeout 60 sh -c 'mkdir -p /scratch/home /scratch/cache; exec "$@"' sandbox "$@"
