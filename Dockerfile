FROM node:22.14.0-alpine3.21@sha256:9bef0ef1e268f60627da9ba7d7605e8831d5b56ad07487d24d1aa386336d1944 AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci --ignore-scripts --no-audit --no-fund
COPY web/index.html web/tsconfig.json web/vite.config.ts ./
COPY web/src ./src
RUN npm run build

FROM golang:1.26.9-alpine3.23@sha256:6750308561953d451210ed1a8086d36c8c8bdcbf672d38d9d0d424b8a7a0216c AS go
WORKDIR /src
ENV CGO_ENABLED=0 GOTOOLCHAIN=local
COPY go.mod go.sum ./
RUN go mod download && go mod verify
ENV GOPROXY=off
COPY cmd ./cmd
COPY internal ./internal
COPY temas/reserva.go ./temas/reserva.go
COPY temas/expedientes-etf ./temas/expedientes-etf
COPY web/embed.go ./web/embed.go
COPY --from=web /src/web/dist ./web/dist
RUN go build -trimpath -ldflags='-s -w' -o /out/etf ./cmd/etf && \
    go build -trimpath -ldflags='-s -w' -o /out/migrar ./cmd/migrar

FROM postgres:18.4-alpine@sha256:9a8afca54e7861fd90fab5fdf4c42477a6b1cb7d293595148e674e0a3181de15 AS migrar
COPY --from=go /out/migrar /usr/local/bin/migrar
COPY migraciones /opt/etf/migraciones
ENV ETF_MIGRACIONES_RUTA=/opt/etf/migraciones
USER 70:70
WORKDIR /opt/etf
ENTRYPOINT ["/usr/local/bin/migrar"]

FROM alpine:3.22@sha256:5291449c3df73caf6ed85e649dec1b9e818b39a5d8c871e97afc13e9cd5e8fa8 AS app
RUN addgroup -g 10001 etf && adduser -D -u 10001 -G etf etf
WORKDIR /opt/etf
COPY --from=go /out/etf /usr/local/bin/etf
COPY idiomas ./reserva/idiomas
COPY datos/catalogos/memoria ./datos/catalogos/memoria
COPY datos/catalogos/familia.json ./datos/catalogos/familia.json
COPY datos/catalogos/conservacion.json ./datos/catalogos/conservacion.json
COPY temas ./reserva/temas
COPY despliegue/base/arrancar-app.sh /usr/local/bin/arrancar-app
ENV ETF_HTTP_DIRECCION=0.0.0.0:8080 ETF_IDIOMAS_RUTA=/datos/idiomas ETF_TEMAS_RUTA=/datos/temas
USER 10001:10001
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=3 \
 CMD wget -q -O /dev/null http://127.0.0.1:8080/salud || exit 1
ENTRYPOINT ["/usr/local/bin/arrancar-app"]
