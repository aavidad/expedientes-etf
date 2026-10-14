import { execFileSync } from 'node:child_process';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';

export function contratoServidor() {
  const raiz = fileURLToPath(new URL('../../', import.meta.url));
  const salida = execFileSync('go', ['test', '-run', '^TestContratoSPA$', '-count=1', '-v', './internal/adaptadores/http'], { cwd: raiz, encoding: 'utf8', timeout: 180000 });
  const datos = salida.match(/INT5_SPA=(.*)/)?.[1];
  assert.ok(datos, salida);
  return JSON.parse(datos);
}
