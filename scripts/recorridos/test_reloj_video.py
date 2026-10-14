"""La compilación de formación se retira aunque fallen voz o inicialización."""
import io
import json
import os
from contextlib import redirect_stdout
from pathlib import Path
import tempfile
import unittest
from unittest.mock import Mock, patch

import reloj_video
import videos
from guion import leer


class RestauracionTest(unittest.TestCase):
    def test_restaurar_si_la_preparacion_de_voz_falla_o_se_interrumpe(self):
        origen = videos.RAIZ / 'docs/recorridos/guiones/11-ciclo-completo.md'
        guion = leer(origen)
        c = guion['config']
        for fallo in (RuntimeError('voz interrumpida'), KeyboardInterrupt()):
            with self.subTest(fallo=type(fallo).__name__), tempfile.TemporaryDirectory() as temporal:
                raiz = Path(temporal)
                guiones = raiz / 'docs/recorridos/guiones'
                guiones.mkdir(parents=True)
                (guiones / origen.name).write_text(origen.read_text())
                local = raiz / 'despliegue/demo/.local'
                local.mkdir(parents=True)
                (local / 'puerto').write_text('18507')
                (local / 'usuarios.json').write_text(json.dumps({c[f'{p}_demo']: 'clave-inventada' for p in c['perfiles']}))
                salida = raiz / 'videos'
                salida.mkdir()
                with patch.dict(os.environ, {}, clear=True), patch('sys.argv', ['videos.py', '--recorrido', '11-ciclo-completo']), patch.object(videos, 'RAIZ', raiz), patch.object(videos, 'destino', return_value=salida), patch.object(videos, 'ffmpeg', return_value='ffmpeg'), patch.object(videos.subprocess, 'run', return_value=Mock(returncode=0)), patch.object(videos, 'ejecutar') as ejecutar, patch.object(videos, 'preparar', side_effect=fallo):
                    with self.assertRaises(type(fallo)):
                        videos.main()
                    acciones = [llamada.args[0][-1] for llamada in ejecutar.call_args_list]
                    self.assertEqual(acciones, ['iniciar', 'restaurar'])

    def test_restaurar_si_falla_la_compilacion_overlay(self):
        d = Mock()
        d.LOCAL = Path('/demo-inventada')
        with patch.object(reloj_video, 'overlay', return_value=Path('overlay.json')), patch.object(reloj_video.subprocess, 'run', side_effect=KeyboardInterrupt()), patch.object(reloj_video, 'restaurar') as restaurar, redirect_stdout(io.StringIO()):
            with self.assertRaises(KeyboardInterrupt):
                reloj_video.iniciar(d)
            restaurar.assert_called_once_with(d)


if __name__ == '__main__':
    unittest.main()
