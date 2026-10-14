#!/usr/bin/env python3
"""Genera docs/producto/INDICE.md: cada apartado de los documentos de producto con su rango de líneas.
Sirve para que los encargos citen el trozo exacto y nadie lea documentos enteros."""
import pathlib, re, sys
base = pathlib.Path(__file__).resolve().parents[3]
docs = ["PRD", "TRD", "FLUJO_APP", "DISENO_UI_UX", "ESQUEMA_BACKEND", "PLAN_IMPLEMENTACION"]
out = ["# Índice de los documentos de producto", "",
       "Generado por `scripts/ci/indice/generar.py`; no se edita a mano. Cada apartado con sus líneas,",
       "para leer sólo el trozo necesario (`sed -n 'INICIO,FINp' fichero`).", ""]
for d in docs:
    p = base / "docs" / "producto" / f"{d}.md"
    lineas = p.read_text(encoding="utf-8").splitlines()
    cab = [(i + 1, l) for i, l in enumerate(lineas) if re.match(r"^#{2,3} ", l)]
    out += [f"## {d}.md ({len(lineas)} líneas)", "", "| Apartado | Líneas |", "| --- | --- |"]
    for n, (ini, l) in enumerate(cab):
        fin = cab[n + 1][0] - 1 if n + 1 < len(cab) else len(lineas)
        out.append(f"| {l.lstrip('#').strip().replace('|', '/')} | {ini}-{fin} |")
    out.append("")
nuevo = "\n".join(out)
destino = base / "docs" / "producto" / "INDICE.md"
if len(sys.argv) > 1 and sys.argv[1] == "--comprobar":
    if not destino.exists() or destino.read_text(encoding="utf-8") != nuevo:
        print("INDICE.md desactualizado: ejecuta python3 scripts/ci/indice/generar.py"); sys.exit(1)
    print("INDICE_OK"); sys.exit(0)
destino.write_text(nuevo, encoding="utf-8"); print("INDICE.md generado")
