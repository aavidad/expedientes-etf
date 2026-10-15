import { pantallas as voz } from './voz/pantallas';
import { pantallas as restauracion } from './restauracion/pantallas';
import { pantallas as anexo6, enlazarAnexo6 } from './anexo6/pantallas';
import { pantallas as auditoria } from './auditoria/pantallas';
import { pantallas as ubicaciones } from './ubicaciones/pantallas';
import { pantallas as conservacion } from './conservacion/pantallas';
import { pantallas as carga } from './carga/pantallas';
import { pantallas as salidas } from './salidas/pantallas';
import { pantallas as desgloseMemoria } from './memdes/pantallas';
import { pantallas as memoria } from './memoria/pantallas';
import { pantallas as plantillas } from './plantillas/pantallas';
import { pantallas as decision } from './decision/pantallas';
import { pantallas as bandeja } from './bandeja/pantallas';
import { pantallas as anexo } from './anexo/pantallas';
import { pantallas as actividad } from './actividad/pantallas';
import { pantallas as smi } from './smi/pantallas';
import { pantallas as spaf } from './spaf/pantallas';
import { pantallas as usuarios } from './usuarios/pantallas';
import { pantallas as catalogo } from './catalogo/pantallas';
import { pantallas as paquete } from './paquete/pantallas';
import { pantallas as equipo } from './equipo/pantallas';
import { pantallas as traslado } from './traslado/pantallas';
import { pantallas as fi1 } from './fi1/pantallas';
import { validarPantallas, type Pantalla } from './registro-pantallas';

// El orden de composición conserva la navegación publicada por el servidor.
export const pantallas: readonly Pantalla[] = enlazarAnexo6([...restauracion, ...anexo6, ...bandeja, ...salidas, ...smi, ...usuarios, ...catalogo, ...ubicaciones, ...paquete, ...memoria, ...desgloseMemoria, ...actividad, ...spaf, ...anexo, ...decision, ...plantillas, ...equipo, ...fi1, ...traslado, ...carga, ...conservacion, ...auditoria, ...voz]);

validarPantallas(pantallas);
