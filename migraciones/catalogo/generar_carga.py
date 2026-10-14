#!/usr/bin/env python3
"""Genera la carga inicial desde los JSON. Una migración aplicada no se regenera.

Sin argumentos escribe SQL a stdout; --comprobar valida la instantánea inicial.
Los cambios posteriores requieren reservar y crear otra migración.
"""
import hashlib
import json
import pathlib
import sys

RAIZ = pathlib.Path(__file__).resolve().parents[2]
ACTOR = '00000000-0000-4000-8000-000000000012'


def literal(valor):
    if valor is None:
        return 'NULL'
    if isinstance(valor, bool):
        return 'true' if valor else 'false'
    if isinstance(valor, int):
        return str(valor)
    return "'" + str(valor).replace("'", "''") + "'"


def generar():
    salida = ['-- Instantánea inicial generada desde datos/catalogos/*.json.\n'
              '-- Las variantes y etiquetas se resuelven en el paquete de idioma.']
    datos = {}
    for nombre in ['equipos', 'centros', 'municipios', 'listas']:
        contenido = (RAIZ / 'datos/catalogos' / (nombre + '.json')).read_bytes()
        datos[nombre] = json.loads(contenido)['opciones' if nombre == 'listas' else nombre]
        salida.append('-- fuente ' + nombre + '.json sha256 ' + hashlib.sha256(contenido).hexdigest())
    salida.append("""DO $$ BEGIN
    IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
       OR to_regclass('catalogo.lista_opcion') IS NULL
       OR EXISTS (SELECT 1 FROM catalogo.version) THEN
        RAISE EXCEPTION 'migracion.estado_previo';
    END IF;
END $$;""")
    for tabla, nombre in [('equipo', 'equipos'), ('centro_ss', 'centros')]:
        filas = [ '(' + ','.join(literal(r[c]) for c in ['codigo','clave_texto','activo_desde']) + ')' for r in datos[nombre]]
        salida.append('INSERT INTO catalogo.' + tabla + '(codigo,clave_texto,activo_desde) VALUES\n' + ',\n'.join(filas) + ';')
    # Primero municipios matriz; después las ELA, con su referencia ya existente.
    for tipo in ['municipio', 'ela']:
        filas = []
        for r in datos['municipios']:
            if r['tipo'] != tipo:
                continue
            matriz = 'NULL' if r['matriz_codigo'] is None else '(SELECT id FROM catalogo.municipio WHERE ine=' + literal(r['matriz_codigo']) + ')'
            filas.append('(' + ','.join([literal(r['codigo']),literal(r['clave_texto']),literal(tipo),matriz,literal(r['ambito_diputacion'])]) + ')')
        salida.append('INSERT INTO catalogo.municipio(ine,clave_texto,tipo,matriz_id,ambito_diputacion) VALUES\n' + ',\n'.join(filas) + ';')
    salida.append("INSERT INTO catalogo.version(ambito,numero,vigente_desde,motivo,creada_por) VALUES\n('asignacion_municipios',1,'2026-01-01','catalogo.carga_inicial','" + ACTOR + "'),\n('listas',1,'2026-01-01','catalogo.carga_inicial','" + ACTOR + "');")
    filas = ['(' + ','.join(literal(r[c]) for c in ['codigo','centro_codigo','equipo_codigo']) + ')' for r in datos['municipios']]
    salida.append("INSERT INTO catalogo.municipio_asignacion(version_id,municipio_id,centro_ss_id,equipo_id)\nSELECT v.id,m.id,c.id,e.id FROM (VALUES\n" + ',\n'.join(filas) + "\n) AS fuente(ine,centro,equipo)\nJOIN catalogo.municipio m ON m.ine=fuente.ine\nLEFT JOIN catalogo.centro_ss c ON c.codigo=fuente.centro\nLEFT JOIN catalogo.equipo e ON e.codigo=fuente.equipo\nCROSS JOIN catalogo.version v WHERE v.ambito='asignacion_municipios' AND v.numero=1;")
    filas = ['(' + ','.join(literal(r[c]) for c in ['lista','codigo','orden','admite_texto','activa']) + ')' for r in datos['listas']]
    salida.append("INSERT INTO catalogo.lista_opcion(version_id,lista,codigo,orden,admite_texto,activa)\nSELECT v.id,fuente.* FROM (VALUES\n" + ',\n'.join(filas) + "\n) AS fuente(lista,codigo,orden,admite_texto,activa)\nCROSS JOIN catalogo.version v WHERE v.ambito='listas' AND v.numero=1;")
    salida.append("SELECT catalogo.publicar(id,'" + ACTOR + "'::uuid) FROM catalogo.version ORDER BY ambito;")
    return '\n\n'.join(salida) + '\n'


if __name__ == '__main__':
    contenido = generar()
    if sys.argv[1:] == ['--comprobar']:
        assert (RAIZ / 'migraciones/catalogo/0002_carga_inicial.sql').read_text() == contenido, 'catalogo.fuente_modificada'
        opciones = json.loads((RAIZ / 'datos/catalogos/listas.json').read_text())['opciones']
        claves = {r['lista'] + '.' + r['codigo'] for r in opciones}
        assert len(claves) == len(opciones), 'catalogo.opcion_duplicada'
        assert all(set(r) == {'lista','codigo','orden','admite_texto','activa'} for r in opciones), 'catalogo.campos_invalidos'
        for idioma in ['es','en']:
            etiquetas = json.loads((RAIZ / 'idiomas' / idioma / 'catalogos/listas.json').read_text())
            assert set(etiquetas) == claves, 'catalogo.etiquetas_incompletas'
            assert all(isinstance(v,str) and v.strip() for v in etiquetas.values()), 'catalogo.etiqueta_invalida'
        print('CARGA-OK instantánea coincide con los cuatro JSON; etiquetas es/en completas')
    elif not sys.argv[1:]:
        print(contenido, end='')
    else:
        raise SystemExit('catalogo.argumentos_invalidos')
