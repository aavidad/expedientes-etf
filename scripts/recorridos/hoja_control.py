"""Extrae fotogramas de introducción, voz de cada rótulo y cierre."""
import json
from pathlib import Path
from herramientas import ejecutar, RAIZ


def generar(dir, guion, plan, ffmpeg):
    dir = Path(dir)
    puntos = [{"id": "intro", "segundo": 1, "texto": guion["intro"]["Cartela"], "voz": guion["intro"]["Locución"]}]
    for p, paso in zip(plan["plan"], guion["pasos"], strict=True):
        puntos.append({"id": paso["id"], "segundo": (p["sale_ini"] + p["sale_fin"]) / 2,
                       "texto": paso["Rótulo"], "voz": paso["Locución"]})
    puntos.append({"id": "cierre", "segundo": plan["inicio_cierre"] + 1,
                   "texto": guion["cierre"]["Cartela"], "voz": guion["cierre"]["Locución"]})
    for i, p in enumerate(puntos):
        p["imagen"] = f"control-{i:02}.png"
        ejecutar([ffmpeg, "-v", "error", "-y", "-ss", f"{p['segundo']:.3f}", "-i", dir / "video.mp4",
                  "-frames:v", "1", dir / p["imagen"]])
    (dir / "control.json").write_text(json.dumps(puntos, ensure_ascii=False, indent=2), encoding="utf-8")
    ejecutar(["node", RAIZ / "scripts/recorridos/cartelas.mjs", dir, "control"])
