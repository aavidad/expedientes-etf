FROM alpine:3.22@sha256:5291449c3df73caf6ed85e649dec1b9e818b39a5d8c871e97afc13e9cd5e8fa8 AS preparar
RUN apk add --no-cache python3=3.12.15-r0 curl=8.14.1-r3 openssl=3.5.9-r0 tzdata=2026e-r0
WORKDIR /opt/etf
COPY despliegue/demo ./despliegue/demo
COPY internal/adaptadores/ldap/testdata ./internal/adaptadores/ldap/testdata
COPY internal/adaptadores/anexos/docx/testdata ./internal/adaptadores/anexos/docx/testdata
COPY idiomas ./idiomas
COPY temas ./temas
COPY despliegue/windows/preparar.py /opt/etf/preparar.py
COPY despliegue/windows/archivos.py /opt/etf/archivos.py
ENV PYTHONDONTWRITEBYTECODE=1 TMPDIR=/tmp
ENTRYPOINT ["python3", "-B", "/opt/etf/preparar.py"]

FROM postgres:18.4-alpine@sha256:9a8afca54e7861fd90fab5fdf4c42477a6b1cb7d293595148e674e0a3181de15 AS postgres
COPY despliegue/base/01_roles.sh /docker-entrypoint-initdb.d/01_roles.sh
COPY despliegue/base/preparar_roles.sh /opt/etf/preparar_roles.sh
COPY despliegue/base/habilitar_calculo_memoria.sh /opt/etf/habilitar_calculo_memoria.sh

FROM clamav/clamav:1.4.3@sha256:75fb5fd95fcbe1d7e6d240c369c1572b686ee2c95949d1042b5148de8eddebb4 AS clamav
COPY despliegue/base/clamd.conf /etc/clamav/clamd.conf

FROM nginx:1.27-alpine@sha256:65645c7bb6a0661892a8b03b89d0743208a18dd2f3f17a54ef4b76fb8e2f2a10 AS entrada
COPY despliegue/demo/nginx.conf /etc/nginx/nginx.conf
