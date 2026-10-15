.PHONY: voz voz-simulada curso ci test vet web build comprobar textos temas videos video demo demo-parar demo-comprobar demo-recorrido demo-limpiar recorridos paquete-windows instalador-windows

paquete-windows:
	python3 -B scripts/voz/construir.py
	python3 -B despliegue/windows/paquete.py

# Sólo el instalador .exe, a partir del ZIP ya construido de la misma VERSION.
instalador-windows:
	python3 -B despliegue/windows/paquete.py --solo-instalador

demo:
	python3 -B despliegue/demo/demo.py iniciar

demo-parar:
	python3 -B despliegue/demo/demo.py parar

demo-comprobar:
	python3 -B despliegue/demo/demo.py comprobar

demo-recorrido:
	python3 -B despliegue/demo/demo.py recorrer

demo-limpiar:
	python3 -B despliegue/demo/demo.py limpiar

recorridos:
	@estado=0; \
	for recorrido in entrada_permisos ficha_integrada salidas_spaf memoria_integrada accesibilidad; do \
		python3 -B scripts/recorridos/$$recorrido.py || estado=1; \
	done; \
	exit $$estado

test:
	go test ./...
	cd web && npm test

vet:
	go vet ./...

web:
	python3 -B scripts/voz/construir.py
	cd web && npm ci --ignore-scripts && npm run build

build: web
	mkdir -p bin
	go build -trimpath -o bin/etf ./cmd/etf
	go build -trimpath -o bin/migrar ./cmd/migrar

comprobar: build vet test

ci:
	python3 -B datos/prueba/generar/generar.py --check
	python3 -B datos/prueba/generar/comprobar.py
	./scripts/ci/comprobar.sh

textos:
	./scripts/ci/textos/comprobar.sh

temas:
	./scripts/ci/temas/comprobar.sh

videos:
	python3 -B scripts/recorridos/videos.py $(foreach recorrido,$(RECORRIDOS),--recorrido "$(recorrido)")

video:
	@test -n "$(RECORRIDO)" || { echo "Indica RECORRIDO=01-entrada" >&2; exit 1; }
	python3 -B scripts/recorridos/videos.py --recorrido "$(RECORRIDO)"

curso:
	python3 -B curso/construir.py

voz:
	python3 -B scripts/voz/construir.py --audios

voz-simulada:
	python3 -B scripts/voz/construir.py --audios --simulado --destino bin/voz-simulada
