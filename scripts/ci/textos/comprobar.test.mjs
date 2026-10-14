import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, mkdirSync, cpSync, writeFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { resolve } from 'node:path';
import { spawnSync } from 'node:child_process';

const repo = resolve(import.meta.dirname, '../../..');
function comprobar(tsx, go = '', claves = [], alterar) {
  const base = mkdtempSync(`${tmpdir()}/etf-textos-`);
  try {
    mkdirSync(`${base}/web/src`, { recursive: true });
    cpSync(`${repo}/idiomas`, `${base}/idiomas`, { recursive: true });
    if (alterar) alterar(base);
    writeFileSync(`${base}/web/src/pantalla.tsx`, tsx);
    writeFileSync(`${base}/pantalla.go`, go || 'package pantalla');
    const rgo = spawnSync('go', ['run', `${repo}/scripts/ci/textos/comprobar.go`, base], { encoding: 'utf8', cwd: repo });
    const rts = spawnSync('node', [`${repo}/scripts/ci/textos/comprobar.mjs`, base], { input: JSON.stringify([...JSON.parse(rgo.stdout || '[]'), ...claves]), encoding: 'utf8' });
    return { ok: rgo.status === 0 && rts.status === 0, errores: rgo.stderr + rts.stderr };
  } finally { rmSync(base, { recursive: true, force: true }); }
}

test('rechaza literales JSX, atributos, expresiones y variables visibles', () => {
  for (const fuente of ['const a=<p>Aceptar</p>;', 'const a=<input aria-label="Aceptar"/>;', "const a=<p>{'Aceptar'}</p>;", "const etiqueta='Aceptar'; const a=<p>{etiqueta}</p>;"]) {
    assert.equal(comprobar(fuente).ok, false, fuente);
  }
});
test('rechaza errores Go visibles y claves inexistentes', () => {
  assert.equal(comprobar('', 'package pantalla\nimport "errors"\nvar err=errors.New("Aceptar")').ok, false);
  assert.equal(comprobar("const a=<p>{t('comun.inexistente')}</p>;").ok, false);
  assert.equal(comprobar('', 'package pantalla\nvar x=textos["comun.inexistente"]').ok, false);
});
test('admite claves existentes y cadenas técnicas', () => {
  assert.equal(comprobar("const a=<p>{t('comun.titulo')}</p>; const ruta='/api/idiomas';").ok, true);
});

test('rechaza idiomas con claves diferentes', () => {
 assert.equal(comprobar('', '', [], (base) => writeFileSync(`${base}/idiomas/en/textos/comun.json`, '{}')).ok, false);
});

test('rechaza una palabra en escrituras HTTP y campos visibles Go', () => {
 for (const fuente of [
  'package pantalla\nimport "fmt"\nfunc pintar(w any){fmt.Fprint(w,"Aceptar")}',
  'package pantalla\nfunc pintar(w any){w.Write([]byte("Aceptar"))}',
  'package pantalla\nfunc pintar(w any){texto:="Aceptar";w.Write([]byte(texto))}',
  'package pantalla\nvar x=Respuesta{Mensaje:"Aceptar"}',
 ]) assert.equal(comprobar('',fuente).ok,false,fuente);
});

test('admite la marca técnica exacta en una línea o función', () => {
  for (const fuente of [
    'package pantalla\nfunc guion(w any){w.WriteString("SET ROLE etf_propietario;") //etf:tecnico\n}',
    'package pantalla\nfunc guion(w any){sql:="SET ROLE etf_propietario;" //etf:tecnico\nw.WriteString(sql)}',
    'package pantalla\nfunc guion(w any){w.WriteString(`SET ROLE\netf_propietario;`) //etf:tecnico\n}',
    'package pantalla\n//etf:tecnico\nfunc guion(w any){w.WriteString("SET ROLE etf_propietario;")}',
    'package pantalla\nfunc guion(w any){ //etf:tecnico\nw.WriteString("SET ROLE etf_propietario;")}',
    'package pantalla\n//etf:tecnico\nfunc guion(w any){sql:="SET ROLE etf_propietario;";w.WriteString(sql)}',
  ]) {
    assert.equal(comprobar('', fuente).ok, true, fuente);
    assert.equal(comprobar('', fuente.replace('//etf:tecnico', '')).ok, false, fuente);
  }
});

test('la marca técnica no alcanza otras líneas, funciones ni paquetes', () => {
  for (const fuente of [
    '//etf:tecnico\npackage pantalla\nfunc pintar(w any){w.WriteString("Aceptar")}',
    'package pantalla\nfunc pintar(w any){w.WriteString("</") //etf:tecnico\nw.WriteString("Aceptar")}',
    'package pantalla\n//etf:tecnico\nfunc guion(w any){w.WriteString("</")}\nfunc pintar(w any){w.WriteString("Aceptar")}',
    'package docx\nfunc pintar(w any){w.WriteString("Aceptar")}',
    'package migracion\nfunc pintar(w any){w.WriteString("Aceptar")}',
    'package pantalla\nfunc pintar(w any){w.WriteString("Aceptar")} //etf:tecnico-extra',
    'package pantalla\nfunc pintar(w any){w.WriteString("Aceptar")} /*etf:tecnico*/',
    'package pantalla\nconst marca="//etf:tecnico"\nfunc pintar(w any){w.WriteString("Aceptar")}',
  ]) assert.equal(comprobar('', fuente).ok, false, fuente);
});

test('la marca técnica conserva la validación de errores y claves usadas', () => {
  for (const fuente of [
    'package pantalla\nimport "errors"\n//etf:tecnico\nfunc guion() error{return errors.New("Aceptar")}',
    'package pantalla\nimport "errors"\nfunc guion() error{return errors.New("Aceptar")} //etf:tecnico',
    'package pantalla\n//etf:tecnico\nfunc guion(){_ = textos["comun.inexistente"]}',
  ]) assert.equal(comprobar('', fuente).ok, false, fuente);
});
