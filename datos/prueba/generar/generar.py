#!/usr/bin/env python3
"""Genera expedientes sintéticos estables para los entornos no productivos."""

import argparse
import json
import random
from datetime import date, timedelta
from pathlib import Path

RAIZ = Path(__file__).resolve().parents[1]
SALIDA = RAIZ / "expedientes.json"
SEMILLA = 150826

NOMBRES = ["Lucía", "Mateo", "Carmen", "Hugo", "Elena", "Daniel", "Irene", "Pablo", "Nuria", "Adrián", "Marta", "Sergio", "Aitana", "Marcos", "Claudia", "Álvaro", "Rocío", "Mario", "Inés", "David", "Noelia", "Javier", "Lola", "Samuel"]
APELLIDOS = ["García", "Martín", "López", "Sánchez", "Ruiz", "Moreno", "Jiménez", "Díaz", "Alonso", "Romero", "Navarro", "Torres", "Molina", "Castro", "Ortega", "Delgado", "Vargas", "Serrano", "Cano", "Medina", "Cortés", "Reyes", "Santos", "Nieto"]
MUNICIPIOS = ["Granada", "Motril", "Almuñécar", "Loja", "Guadix", "Baza", "Órgiva", "Montefrío", "Alhama de Granada", "Huéscar", "Pinos Puente", "Salobreña", "Íllora", "La Zubia", "Maracena", "Ugíjar"]
EDUCACION = ["sin_estudios", "primaria", "secundaria", "bachillerato", "formacion_profesional", "universitaria"]
TRABAJO = ["empleo", "desempleo", "cuidados", "formacion", "incapacidad_temporal", "pension"]
FACTORES_FAMILIA = ["dificultad_economica", "conflicto_familiar", "aislamiento_social", "vivienda_inestable", "salud_adulta"]
FACTORES_MENOR = ["absentismo", "dificultad_emocional", "necesidad_apoyo_escolar", "conflicto_familiar", "falta_supervision"]
TIPOLOGIAS = ["negligencia", "maltrato_emocional", "dificultad_parental", "violencia_familiar", "riesgo_social"]
MOTIVOS_CIERRE = ["objetivos_alcanzados", "fin_intervencion", "traslado_residencia", "renuncia_familiar", "derivacion_a_otro_servicio"]

# Casos frontera identificables por la documentación que acompaña a los datos.
CASOS = {
    1: "consentimiento justo el día 30",
    2: "sin consentimiento al día 31",
    3: "cambio de composición familiar durante el año",
    4: "reapertura tras cierre",
    5: "traslado del equipo 1 al equipo 2",
    6: "familia con menores de Riesgo y Reunificación",
    7: "derivación no aceptada, sin numeración",
    8: "en lista de espera, sin numeración",
    9: "quince menores, límite superior de composición",
    10: "un menor cumple 18 años durante el año evaluado",
    11: "cierre sin consentimiento",
    12: "cambio de composición con alta y baja de menores",
    13: "cierre y reapertura en episodios distintos",
    14: "traslado del equipo 7 al equipo 8",
    15: "segunda familia mixta de Riesgo y Reunificación",
    16: "fecha de derivación al cierre del año 2025",
}


def persona(rng, indice, adulto=False):
    nombre = NOMBRES[(indice * 5 + (3 if adulto else 0)) % len(NOMBRES)]
    apellido = APELLIDOS[(indice * 3 + 2) % len(APELLIDOS)]
    segundo = APELLIDOS[(indice * 7 + 9) % len(APELLIDOS)]
    return nombre, f"{apellido} {segundo}"


def fecha_edad(anio, edad, rng):
    mes = rng.randint(1, 12)
    dia = rng.randint(1, 27)
    return date(anio - edad, mes, dia).isoformat()


def crear_familia(indice, rng):
    equipo = (indice - 1) % 8 + 1
    ciclo = (indice - 1) // 8
    anio = 2025 if ciclo % 2 == 0 else 2026
    dia = 3 + (indice * 3) % 23
    derivacion = date(anio, 1 + ((indice - 1) % 11), dia)
    if indice == 16:
        anio = 2025
        derivacion = date(2025, 12, 31)
    municipio = MUNICIPIOS[(indice * 5 + equipo) % len(MUNICIPIOS)]
    estado = ["en_intervencion", "cerrada", "derivacion_recibida", "no_aceptada", "lista_espera", "aceptada", "en_intervencion", "cerrada"][ciclo]
    if indice == 7:
        estado = "no_aceptada"
    elif indice == 8:
        estado = "lista_espera"
    aceptacion = None
    consentimiento = None
    cierre = None
    hitos = [{"tipo": "derivacion_recibida", "fecha": derivacion.isoformat()}]
    numero = None
    if estado not in ("derivacion_recibida", "no_aceptada", "lista_espera"):
        aceptacion = derivacion + timedelta(days=4 + indice % 9)
        numero = f"{equipo:02d}-{(indice - 1) // 8 + 1:02d}-{str(aceptacion.year)[-2:]}"
        hitos.append({"tipo": "aceptada", "fecha": aceptacion.isoformat()})
    if aceptacion and estado not in ("aceptada",) and indice not in (2, 11):
        consentimiento = aceptacion + timedelta(days=30 if indice == 1 else 31 if indice == 2 else 2 + indice % 12)
        hitos.append({"tipo": "consentimiento", "fecha": consentimiento.isoformat()})
    if estado == "cerrada" and indice != 11:
        cierre = min((consentimiento or aceptacion or derivacion) + timedelta(days=90 + indice), date(2026, 12, 20))
        hitos.append({"tipo": "cerrada", "fecha": cierre.isoformat()})
    if indice == 1:
        consentimiento = aceptacion + timedelta(days=30)
        estado = "en_intervencion"
    if indice == 2:
        consentimiento = None
        estado = "aceptada"
    if indice in (4, 13):
        cierre = (aceptacion or derivacion) + timedelta(days=75)
        estado = "en_intervencion"
        hitos.extend([{"tipo": "cerrada", "fecha": cierre.isoformat()}, {"tipo": "reabierta", "fecha": (cierre + timedelta(days=20)).isoformat()}])
    if indice == 11:
        consentimiento = None
        estado = "cerrada"
        cierre = (aceptacion or derivacion) + timedelta(days=40)
        hitos.append({"tipo": "cerrada", "fecha": cierre.isoformat(), "motivo": "renuncia_familiar"})
    if indice == 5:
        hitos.append({"tipo": "traslado_equipo", "fecha": "2026-03-16", "equipo_origen": f"equipo-{equipo:02d}", "equipo_destino": f"equipo-{(equipo % 8) + 1:02d}"})
    if indice == 14:
        hitos.append({"tipo": "traslado_equipo", "fecha": "2026-05-12", "equipo_origen": f"equipo-{equipo:02d}", "equipo_destino": f"equipo-{(equipo % 8) + 1:02d}"})
    if indice in (3, 12):
        hitos.append({"tipo": "cambio_composicion", "fecha": f"{anio}-09-15", "detalle": "Se incorpora un menor y se actualiza la composición familiar."})
    adultos = []
    cantidad_adultos = 1 + indice % 3
    for orden in range(1, cantidad_adultos + 1):
        nombre, apellidos = persona(rng, indice * 4 + orden, adulto=True)
        adultos.append({
            "orden": orden,
            "nombre": nombre,
            "apellidos": apellidos,
            "papel": ["madre", "padre", "tutor_legal"][((indice + orden) % 3)],
            "sexo": "mujer" if (indice + orden) % 2 else "hombre",
            "edad": 23 + ((indice * 3 + orden * 7) % 43),
            "nivel_educativo": EDUCACION[(indice + orden) % len(EDUCACION)],
            "situacion_laboral": TRABAJO[(indice * 2 + orden) % len(TRABAJO)],
            "es_tutor": orden == 1,
        })

    cuenta_menores = 15 if indice == 9 else 1 + indice % 3
    menores = []
    for orden in range(1, cuenta_menores + 1):
        nombre, apellidos = persona(rng, indice * 6 + orden)
        edad = (indice + orden * 4) % 18
        nacimiento = fecha_edad(anio, edad, rng)
        if indice == 10 and orden == 1:
            nacimiento = f"{anio - 18}-06-18"
        programa = "Reunificación" if (indice == 6 and orden % 2 == 0) or (indice == 15 and orden == 1) else "Riesgo"
        menores.append({
            "orden": orden,
            "nombre": nombre,
            "apellidos": apellidos,
            "sexo": "mujer" if (indice + orden) % 2 else "hombre",
            "fecha_nacimiento": nacimiento,
            "nivel_escolarizacion": ["infantil", "primaria", "secundaria", "no_escolarizado"][orden % 4],
            "centro_educativo": f"Centro educativo {((indice + orden) % 8) + 1}",
            "subprograma": programa,
        })

    marcas = {
        "factores_familia": sorted(rng.sample(FACTORES_FAMILIA, 2)),
        "factores_menor": sorted(rng.sample(FACTORES_MENOR, 2)),
        "tipologias": sorted(rng.sample(TIPOLOGIAS, 2)),
    }
    equipo_actual = (equipo % 8) + 1 if indice in (5, 14) else equipo
    if indice in (6, 15):
        subprograma_familiar = "mixta"
    elif menores[0]["subprograma"] == "Reunificación":
        subprograma_familiar = "Reunificacion"
    else:
        subprograma_familiar = "Riesgo"
    familia = {
        "id": f"familia-{indice:02d}",
        "numero": numero,
        "equipo_id": f"equipo-{equipo_actual:02d}",
        "equipo_inicial_id": f"equipo-{equipo:02d}",
        "municipio": municipio,
        "anio_derivacion": derivacion.year,
        "estado": estado,
        "subprograma": subprograma_familiar,
        "fechas": {
            "derivacion": derivacion.isoformat(),
            "aceptacion": aceptacion.isoformat() if aceptacion else None,
            "consentimiento": consentimiento.isoformat() if consentimiento else None,
            "cierre": cierre.isoformat() if cierre else None,
            "lista_espera": derivacion.isoformat() if estado == "lista_espera" else None,
            "no_aceptacion": derivacion.isoformat() if estado == "no_aceptada" else None,
        },
        "motivo_cierre": ("renuncia_familiar" if indice == 11 else MOTIVOS_CIERRE[indice % len(MOTIVOS_CIERRE)]) if cierre else None,
        "adultos": adultos,
        "menores": menores,
        "ingresos_anuales_eur": 7200 + (indice * 1379) % 35600,
        "factores": marcas,
        "hitos": hitos,
        "casos_frontera": [CASOS[indice]] if indice in CASOS else [],
    }
    if estado == "no_aceptada":
        familia["motivo_no_aceptacion"] = ["objetivos_no_compartidos", "sin_cobertura", "derivacion_inadecuada"][indice % 3]
    if estado == "lista_espera":
        familia["posicion_lista_espera"] = 2 + indice % 11
    return familia


def generar():
    rng = random.Random(SEMILLA)
    familias = [crear_familia(i, rng) for i in range(1, 65)]
    return {
        "version": 1,
        "generado_con_semilla": SEMILLA,
        "nota": "Conjunto sintético. Identidades, contactos y circunstancias creados para desarrollo, CI y preproducción.",
        "familias": familias,
    }


def contenido():
    return json.dumps(generar(), ensure_ascii=False, indent=2) + "\n"


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--check", action="store_true", help="falla si el JSON no coincide con el generador")
    args = parser.parse_args()
    esperado = contenido()
    if args.check:
        if not SALIDA.exists() or SALIDA.read_text(encoding="utf-8") != esperado:
            raise SystemExit("expedientes.json no coincide con el generador; ejecuta generar.py")
        print("Datos sintéticos reproducibles: correctos (64 familias, semilla fija).")
    else:
        SALIDA.write_text(esperado, encoding="utf-8")
        print(f"Generadas 64 familias en {SALIDA.relative_to(RAIZ.parent.parent)}.")


if __name__ == "__main__":
    main()
