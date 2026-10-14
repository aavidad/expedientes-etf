#!/bin/sh
set -eu
if [ ! -r "${ETF_IDIOMAS_RUTA:-/datos/idiomas}/es/textos/comun.json" ]; then
    export ETF_IDIOMAS_RUTA=/opt/etf/reserva/idiomas
    echo 'idiomas.reserva'
fi
if [ ! -r "${ETF_TEMAS_RUTA:-/datos/temas}/expedientes-etf/tokens.json" ]; then
    export ETF_TEMAS_RUTA=/opt/etf/reserva/temas
    echo 'temas.reserva'
fi
exec /usr/local/bin/etf
