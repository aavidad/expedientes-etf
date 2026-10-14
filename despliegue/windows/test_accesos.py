"""Ensaya límites de la instalación y la retirada de archivos en PowerShell."""
import json
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

RAIZ = Path(__file__).resolve().parents[2]
PWSH = shutil.which("pwsh") or str(Path.home() / ".local/bin/pwsh")


@unittest.skipUnless(Path(PWSH).is_file(), "PowerShell no está instalado en este puesto")
class LimitesInstalacion(unittest.TestCase):
    def ejecutar(self, codigo):
        with tempfile.TemporaryDirectory(prefix="win1-test-") as temporal:
            raiz = Path(temporal)
            paquete = raiz / "fuente"
            paquete.mkdir()
            for nombre in ("comun.ps1", "accesos.ps1"):
                shutil.copyfile(RAIZ / "despliegue/windows" / nombre, paquete / nombre)
            shutil.copyfile(RAIZ / "idiomas/es/textos/windows.json", paquete / "mensajes.json")
            script = raiz / "ensayo.ps1"
            script.write_text("""param([string]$Raiz)
. (Join-Path $Raiz 'fuente/comun.ps1')
. (Join-Path $Raiz 'fuente/accesos.ps1')
$Puerto = 18381
$Motor = 'Podman'
""" + codigo, encoding="utf-8-sig")
            resultado = subprocess.run([PWSH, "-NoProfile", "-File", str(script), str(raiz)],
                                       capture_output=True, text=True, timeout=30)
            self.assertEqual(resultado.returncode, 0, resultado.stdout + resultado.stderr)

    def test_no_adopta_carpeta_ajena(self):
        self.ejecutar("""
$ajeno = Join-Path $Raiz 'datos'
New-Item -ItemType Directory $ajeno | Out-Null
[IO.File]::WriteAllText((Join-Path $ajeno 'notas.txt'), 'Documento que se conserva')
$rechaza = $false
try { Preparar-Directorio $ajeno } catch { $rechaza = $true }
if (-not $rechaza -or -not (Test-Path (Join-Path $ajeno 'notas.txt'))) { throw 'No protege la carpeta ajena' }
if (Test-Path (Join-Path $ajeno 'instalacion.json')) { throw 'Adopta la carpeta ajena' }
""")

    def test_retira_registrados_y_conserva_anadidos(self):
        self.ejecutar("""
Preparar-Directorio (Join-Path $Raiz 'privado')
$cache = Ruta-PaqueteInstalado
New-Item -ItemType Directory $cache | Out-Null
[IO.File]::WriteAllText((Join-Path $cache 'abrir.ps1'), 'Contenido de la aplicación')
[IO.File]::WriteAllText((Join-Path $cache 'nota-personal.txt'), 'Contenido que se conserva')
[IO.File]::WriteAllText((Join-Path $cache 'SUMAS.json'), '{"abrir.ps1":"registro"}')
[IO.File]::WriteAllText((Join-Path $script:Privado 'notas.txt'), 'Contenido que se conserva')
Eliminar-ArchivosInstalados
if (Test-Path (Join-Path $cache 'abrir.ps1')) { throw 'No retira el archivo registrado' }
if (-not (Test-Path (Join-Path $cache 'nota-personal.txt')) -or -not (Test-Path (Join-Path $script:Privado 'notas.txt'))) { throw 'Borra archivos añadidos' }
Preparar-Directorio (Join-Path $Raiz 'privado')
if (-not (Test-Path (Join-Path $script:Privado 'notas.txt'))) { throw 'Impide reinstalar conservando notas' }
[IO.File]::WriteAllText((Join-Path $script:Paquete 'abrir.ps1'), 'Nueva instalación')
[IO.File]::WriteAllText((Join-Path $script:Paquete 'SUMAS.json'), '{"abrir.ps1":"registro"}')
Guardar-PaqueteInstalado
if (-not (Test-Path (Join-Path (Ruta-PaqueteInstalado) 'abrir.ps1')) -or -not (Get-ChildItem $script:Privado -Recurse -Filter 'nota-personal.txt')) { throw 'No reinstala conservando la nota de la copia' }
""")

    def test_cierra_auxiliares_y_respeta_perfiles_ajenos(self):
        self.ejecutar("""
Preparar-Directorio (Join-Path $Raiz 'privado')
$perfil = Join-Path $script:Privado 'perfil-navegador'
$script:cerrados = [Collections.Generic.List[int]]::new()
function Get-CimInstance {
    [CmdletBinding()] param([string]$ClassName, [string]$Filter)
    @(
      [pscustomobject]@{ProcessId=1; ParentProcessId=0; CommandLine=('msedge --user-data-dir="' + $perfil + '"')},
      [pscustomobject]@{ProcessId=2; ParentProcessId=1; CommandLine='msedge --type=gpu-process'},
      [pscustomobject]@{ProcessId=3; ParentProcessId=1; CommandLine='chrome --user-data-dir="/perfil-ajeno"'},
      [pscustomobject]@{ProcessId=4; ParentProcessId=3; CommandLine='chrome --type=renderer'},
      [pscustomobject]@{ProcessId=5; ParentProcessId=1; CommandLine=$null},
      [pscustomobject]@{ProcessId=6; ParentProcessId=0; CommandLine='chrome --user-data-dir="/perfil-ajeno"'}
    )
}
function Stop-Process { [CmdletBinding()] param([int]$Id, [switch]$Force); $script:cerrados.Add($Id) }
function Wait-Process { [CmdletBinding()] param([int]$Id, [int]$Timeout) }
Cerrar-NavegadorPropio
if (($script:cerrados | Sort-Object) -join ',' -ne '1,2') { throw 'Cierra un perfil ajeno o deja abierto un auxiliar propio' }
""")

    def test_rechaza_ruta_fuera_del_paquete(self):
        self.ejecutar("""
Preparar-Directorio (Join-Path $Raiz 'privado')
$cache = Ruta-PaqueteInstalado
New-Item -ItemType Directory $cache | Out-Null
$otro = Join-Path $script:Privado 'datos.txt'
[IO.File]::WriteAllText($otro, 'Datos conservados')
[IO.File]::WriteAllText((Join-Path $cache 'SUMAS.json'), '{"../datos.txt":"registro"}')
$rechaza = $false
try { Eliminar-ArchivosInstalados } catch { $rechaza = $true }
if (-not $rechaza -or -not (Test-Path $otro)) { throw 'Se sale del paquete registrado' }
""")

    def test_copia_estable_no_incorpora_archivos_ajenos(self):
        self.ejecutar("""
Preparar-Directorio (Join-Path $Raiz 'privado')
[IO.File]::WriteAllText((Join-Path $script:Paquete 'nota-personal.txt'), 'Documento ajeno')
[IO.File]::WriteAllText((Join-Path $script:Paquete 'abrir.ps1'), 'Aplicación')
[IO.File]::WriteAllText((Join-Path $script:Paquete 'SUMAS.json'), '{"abrir.ps1":"registro"}')
Guardar-PaqueteInstalado
$cache = Ruta-PaqueteInstalado
if (-not (Test-Path (Join-Path $cache 'abrir.ps1')) -or (Test-Path (Join-Path $cache 'nota-personal.txt'))) { throw 'Copia archivos ajenos' }
""")


if __name__ == "__main__":
    unittest.main()
