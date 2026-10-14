FROM alpine:3.22@sha256:5291449c3df73caf6ed85e649dec1b9e818b39a5d8c871e97afc13e9cd5e8fa8
RUN apk add --no-cache openldap=2.6.8-r0 openldap-back-mdb=2.6.8-r0
USER 65534:65534
ENTRYPOINT ["/bin/sh", "-ec", "mkdir -p /tmp/datos; slapadd -f /fixture/slapd.conf -l /fixture/datos.ldif; exec slapd -d 0 -f /fixture/slapd.conf -h ldaps://0.0.0.0:1636/"]
