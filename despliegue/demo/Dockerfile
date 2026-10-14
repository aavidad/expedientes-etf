# Los binarios se compilan en local con las versiones y sumas del repositorio.
FROM postgres:18.4-alpine@sha256:9a8afca54e7861fd90fab5fdf4c42477a6b1cb7d293595148e674e0a3181de15 AS migrar
COPY .local/bin/migrar /usr/local/bin/migrar
COPY .local/migraciones /opt/etf/migraciones
ENV ETF_MIGRACIONES_RUTA=/opt/etf/migraciones
USER 70:70
ENTRYPOINT ["/usr/local/bin/migrar"]

FROM alpine:3.22@sha256:5291449c3df73caf6ed85e649dec1b9e818b39a5d8c871e97afc13e9cd5e8fa8 AS app
WORKDIR /opt/etf
COPY .local/bin/etf /usr/local/bin/etf
COPY --chown=10001:10001 .local/datos/catalogos /opt/etf/datos/catalogos
COPY --chown=10001:10001 .local/idiomas /datos/idiomas
COPY --chown=10001:10001 .local/temas /datos/temas
USER 10001:10001
HEALTHCHECK --interval=2s --timeout=3s --start-period=5s --retries=30 \
 CMD wget -q -O /dev/null http://127.0.0.1:8080/salud || exit 1
ENTRYPOINT ["/usr/local/bin/etf"]
