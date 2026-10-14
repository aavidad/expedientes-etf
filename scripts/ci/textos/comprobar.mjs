// Análisis sintáctico de TypeScript; los textos de datos quedan fuera del código.
import ts from '../../../web/node_modules/typescript/lib/typescript.js';
import { readFileSync, readdirSync } from 'node:fs';
import { resolve } from 'node:path';

const raiz = process.argv[2] ? resolve(process.argv[2]) : resolve(import.meta.dirname, '../../..');
const recorrer = (dir) => readdirSync(dir, { withFileTypes: true }).flatMap((entrada) =>
  entrada.isDirectory() ? recorrer(`${dir}/${entrada.name}`) : [`${dir}/${entrada.name}`]);
const esClave = (valor) => /^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)+$/.test(valor);
const visibles = new Set(['aria-label', 'aria-description', 'title', 'placeholder', 'alt', 'label']);
let fallos = 0;
const fallo = (fichero, posicion, mensaje) => { console.error(`${fichero}:${posicion}: ${mensaje}`); fallos++; };
const usados = new Set();
for (const fichero of recorrer(`${raiz}/web/src`).filter((ruta) => /\.(tsx?|jsx?)$/.test(ruta) && !/\.(test|spec)\./.test(ruta))) {
  const contenido = readFileSync(fichero, 'utf8');
  const archivo = ts.createSourceFile(fichero, contenido, ts.ScriptTarget.Latest, true);
  const definiciones = new Map();
  function registrar(nodo) {
    if (ts.isVariableDeclaration(nodo) && ts.isIdentifier(nodo.name) && nodo.initializer) definiciones.set(nodo.name.text, nodo.initializer);
    ts.forEachChild(nodo, registrar);
  }
  registrar(archivo);
  function visible(expr, vistos = new Set()) {
    if (!expr) return;
    const linea = archivo.getLineAndCharacterOfPosition(expr.pos).line + 1;
    if (ts.isStringLiteralLike(expr) && expr.text.trim()) fallo(fichero, linea, 'Literal usado como texto visible');
    else if (ts.isTemplateExpression(expr) && (expr.head.text.trim() || expr.templateSpans.some((parte) => parte.literal.text.trim()))) fallo(fichero, linea, 'Plantilla con texto visible');
    else if (ts.isIdentifier(expr) && !vistos.has(expr.text)) {
      vistos.add(expr.text); visible(definiciones.get(expr.text), vistos);
    } else if (ts.isConditionalExpression(expr)) {
      visible(expr.whenTrue, vistos); visible(expr.whenFalse, vistos);
    } else if (ts.isBinaryExpression(expr)) {
      if (expr.operatorToken.kind !== ts.SyntaxKind.AmpersandAmpersandToken) visible(expr.left, vistos);
      visible(expr.right, vistos);
    } else if (ts.isParenthesizedExpression(expr)) visible(expr.expression, vistos);
  }
  function revisar(nodo) {
    if (ts.isJsxExpression(nodo) && nodo.expression && (!ts.isJsxAttribute(nodo.parent) || visibles.has(nodo.parent.name.getText(archivo)))) visible(nodo.expression);
    if (ts.isBinaryExpression(nodo) && nodo.operatorToken.kind === ts.SyntaxKind.EqualsToken && ['document.title'].includes(nodo.left.getText(archivo))) visible(nodo.right);
    const linea = archivo.getLineAndCharacterOfPosition(nodo.pos).line + 1;
    if (ts.isJsxText(nodo) && nodo.text.trim()) fallo(fichero, linea, 'Literal JSX visible');
    if (ts.isStringLiteralLike(nodo)) {
      const valor = nodo.text;
      let padre = nodo.parent;
      if ((ts.isCallExpression(padre) || ts.isNewExpression(padre)) && ['t', 'Error'].includes(padre.expression.getText(archivo))) {
        if (!esClave(valor)) fallo(fichero, linea, 'Se requiere una clave de texto');
        else usados.add(valor);
      }
      if (ts.isElementAccessExpression(padre) && /textos|traduccion/.test(padre.expression.getText(archivo)) && esClave(valor)) usados.add(valor);
      while (padre && !ts.isJsxAttribute(padre) && !ts.isJsxElement(padre) && !ts.isSourceFile(padre)) padre = padre.parent;
      if (padre && ts.isJsxAttribute(padre) && visibles.has(padre.name.getText(archivo))) fallo(fichero, linea, 'Literal en atributo visible');
      // Detecta también mensajes asignados fuera de JSX; excluye MIME y fragmentos técnicos.
      const referenciaARIA = padre && ts.isJsxAttribute(padre) && ['aria-describedby', 'aria-labelledby'].includes(padre.name.getText(archivo));
      if (!referenciaARIA && /\p{L}/u.test(valor) && /\s/.test(valor) && !/^(?:application\/|text\/|--|[.#[]|meta\[)/.test(valor)) {
        fallo(fichero, linea, 'Texto escrito en TypeScript');
      }
    }
    ts.forEachChild(nodo, revisar);
  }
  revisar(archivo);
}
const idiomas = readdirSync(`${raiz}/idiomas`, { withFileTypes: true }).filter((e) => e.isDirectory()).map((e) => e.name);
const cargar = (codigo, carpeta) => {
  const datos = {};
  for (const fichero of recorrer(`${raiz}/idiomas/${codigo}/${carpeta}`).filter((ruta) => ruta.endsWith('.json'))) {
    for (const [clave, valor] of Object.entries(JSON.parse(readFileSync(fichero, 'utf8')))) {
      if (clave in datos || typeof valor !== 'string' || !valor.trim()) fallo(fichero, 1, `Clave repetida o valor inválido: ${clave}`);
      datos[clave] = valor;
    }
  }
  return datos;
};
const base = cargar('es', 'textos');
for (const codigo of idiomas) {
  const manifiesto = JSON.parse(readFileSync(`${raiz}/idiomas/${codigo}/idioma.json`, 'utf8'));
  if (manifiesto.codigo !== codigo || !manifiesto.nombre || !manifiesto.version) fallo(codigo, 1, 'Manifiesto inválido');
  for (const carpeta of ['textos', 'catalogos']) {
    const esperadas = Object.keys(cargar('es', carpeta)).sort();
    const actuales = Object.keys(cargar(codigo, carpeta)).sort();
    if (JSON.stringify(esperadas) !== JSON.stringify(actuales)) fallo(codigo, 1, `Claves diferentes en ${carpeta}`);
  }
  readdirSync(`${raiz}/idiomas/${codigo}/plantillas`);
}
// El comprobador Go envía las claves usadas mediante JSON en stdin.
let entrada = '';
for await (const parte of process.stdin) entrada += parte;
for (const clave of JSON.parse(entrada || '[]')) usados.add(clave);
for (const clave of usados) if (!(clave in base)) fallo('idiomas/es', 1, `Clave usada inexistente: ${clave}`);
if (fallos) process.exitCode = 1;
else console.log(`Textos: ${idiomas.length} idiomas, ${usados.size} claves usadas; sin literales visibles.`);
