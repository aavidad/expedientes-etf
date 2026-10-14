#!/usr/bin/env bash
# Instalación aislada; no altera Python ni el kit de referencia.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/../.."
destino_voz=$(python3 -B -c 'import sys; sys.path.insert(0, "scripts/recorridos"); from herramientas import destino; print(destino() / ".herramientas/voz")')
python3 -m venv "$destino_voz"
"$destino_voz/bin/python" -m pip install 'edge-tts @ git+https://github.com/rany2/edge-tts.git@0b1c544d020b01338ae491d7f4547729afdc39d3'
"$destino_voz/bin/python" -c 'from importlib.metadata import distribution; import json; d=json.loads(distribution("edge-tts").read_text("direct_url.json")); assert d["vcs_info"]["commit_id"] == "0b1c544d020b01338ae491d7f4547729afdc39d3"'
