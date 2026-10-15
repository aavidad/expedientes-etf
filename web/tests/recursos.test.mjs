import { readFileSync, readdirSync } from 'node:fs';
import assert from 'node:assert/strict';
import { test } from 'node:test';
import ts from 'typescript';

const leer = (ruta) => readFileSync(new URL(ruta, import.meta.url), 'utf8');
const directorioTextos = new URL('../../idiomas/es/textos/', import.meta.url);
const textos = Object.assign({}, ...readdirSync(directorioTextos, { recursive: true }).filter((nombre) => nombre.endsWith('.json')).map((nombre) => JSON.parse(readFileSync(new URL(nombre, directorioTextos), 'utf8'))));
const tokens = JSON.parse(leer('../../temas/expedientes-etf/tokens.json'));

test('todas las variables CSS usadas existen en el tema', () => {
  for (const tema of ['expedientes-etf', 'alto-contraste']) {
    const variables = new Set(Object.keys(JSON.parse(leer(`../../temas/${tema}/tokens.json`))).map((clave) => `--${clave.replaceAll('.', '-')}`));
    for (const [, variable] of (leer('../src/estructura.css') + leer('../src/administracion.css') + leer('../src/catalogo/presentacion.css') + leer('../src/actividad/actividad.css')).matchAll(/var\((--[\w-]+)\)/g)) assert.ok(variables.has(variable), `${tema}: ${variable}`);
  }
  assert.equal(tokens['color.borde-control'], '#8a8577');
});

test('los textos propios de pantalla salen del idioma', () => {
  for (const fichero of ['main.tsx', 'sesion.tsx', 'usuarios.tsx', 'usuarios/pantalla.tsx', 'armazon.tsx', 'cambios.tsx', 'administracion.tsx', 'catalogo/administracion.tsx', 'AdministracionPaquetes.tsx', 'smi.tsx', 'actividad/Actividad.tsx']) {
    const fuente = leer(`../src/${fichero}`);
    const archivo = ts.createSourceFile('main.tsx', fuente, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
    const revisar = (nodo) => {
      if (ts.isJsxText(nodo)) assert.equal(nodo.text.trim(), '');
      if (ts.isJsxAttribute(nodo) && ['aria-label', 'title', 'placeholder'].includes(nodo.name.getText(archivo))) {
        assert.ok(!nodo.initializer || !ts.isStringLiteral(nodo.initializer));
      }
      ts.forEachChild(nodo, revisar);
    };
    revisar(archivo);
    for (const [, clave] of fuente.matchAll(/(?:textos\[|new Error\(|\bt\()'([^']+)'/g)) {
      assert.equal(typeof textos[clave], 'string', clave);
    }
  }
  assert.ok(textos['comun.error_carga']);
  assert.ok(textos['comun.reintentar']);
});

test('la estructura no contiene colores, fuentes ni dimensiones de tema', () => {
  const css = leer('../src/estructura.css') + leer('../src/administracion.css') + leer('../src/actividad/actividad.css');
  assert.doesNotMatch(css, /#[0-9a-f]{3,8}\b|\b(?:rgb|hsl)a?\(|\d(?:px|rem|em)\b|Public Sans|Source Serif/i);
  assert.doesNotMatch(css, /@import|url\(/);
});

test('identidad tiene las mismas claves y textos no vacíos en ambos idiomas', () => {
  const es = JSON.parse(leer('../../idiomas/es/textos/identidad.json'));
  const en = JSON.parse(leer('../../idiomas/en/textos/identidad.json'));
  assert.deepEqual(Object.keys(es).sort(), Object.keys(en).sort());
  for (const catalogo of [es, en]) for (const [clave, texto] of Object.entries(catalogo)) {
    assert.match(clave, /^identidad\.[a-z_]+$/);
    assert.equal(typeof texto, 'string');
    assert.ok(texto.trim());
  }
});
