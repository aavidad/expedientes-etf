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


    def test_accesos_abren_el_lanzador_y_migran_los_anteriores(self):
        self.ejecutar("""
$env:SystemRoot = Join-Path $Raiz 'Windows'
Preparar-Directorio (Join-Path $Raiz 'privado')
$lanzador = Join-Path (Ruta-PaqueteInstalado) 'Expedientes ETF.exe'
if ((Ruta-Lanzador) -ne $lanzador) { throw 'El lanzador no está en la copia estable' }
if (-not (Es-AccesoPropio ([pscustomobject]@{TargetPath=$lanzador; Arguments=''}))) { throw 'No reconoce el acceso al lanzador' }
if (Es-AccesoPropio ([pscustomobject]@{TargetPath=$lanzador; Arguments='-otro'})) { throw 'Acepta argumentos ajenos' }
$powershell = Join-Path $env:SystemRoot 'System32\\WindowsPowerShell\\v1.0\\powershell.exe'
$abrir = Join-Path (Ruta-PaqueteInstalado) 'abrir.ps1'
$anterior = [pscustomobject]@{TargetPath=$powershell; Arguments=('-NoLogo -NoProfile -ExecutionPolicy Bypass -File "' + $abrir + '"')}
if (-not (Es-AccesoPropio $anterior)) { throw 'No migra el acceso anterior a PowerShell' }
$ajeno = [pscustomobject]@{TargetPath=$powershell; Arguments='-File "C:\\otro.ps1"'}
if (Es-AccesoPropio $ajeno) { throw 'Adopta un acceso ajeno' }
""")

    def test_crear_accesos_apunta_al_lanzador_sin_script(self):
        codigo = (RAIZ / "despliegue/windows/accesos.ps1").read_text(encoding="utf-8-sig")
        crear = codigo[codigo.index("function Crear-Accesos"):codigo.index("function Eliminar-Accesos")]
        self.assertIn("$enlace.TargetPath = Ruta-Lanzador", crear)
        self.assertIn("$enlace.Arguments = ''", crear)
        self.assertNotIn("powershell", crear.lower())


    PODMAN_SIMULADO = """
$script:Maquina = 'podman-machine-default'
$motor = Join-Path $Raiz 'podman'
New-Item -ItemType Directory $motor | Out-Null
$script:Ejecutable = Join-Path $motor 'podman.exe'
[IO.File]::WriteAllText((Join-Path $motor 'win-sshproxy.exe'), 'simulado')
$identidad = Join-Path $Raiz 'machine'
[IO.File]::WriteAllText($identidad, 'clave simulada')
$env:XDG_DATA_HOME = Join-Path $Raiz 'datos'
$conexion = [pscustomobject]@{Name='podman-machine-default-root'; URI='ssh://root@127.0.0.1:49708/run/podman/podman.sock'; Identity=$identidad; Default=$true}
$script:ordenes = [Collections.Generic.List[string]]::new()
$script:proxy = $null
$script:mensajes = [Collections.Generic.List[string]]::new()
function Ejecutar-Motor {
    param([string[]]$Argumentos, [string]$Entrada = '')
    $script:ordenes.Add($Argumentos -join ' ')
    if (($Argumentos -join ' ') -eq 'machine list --format json') { return '[{"Name":"podman-machine-default","VMType":"wsl","Running":true,"Starting":false}]' }
    if ($Argumentos[1] -eq 'inspect') { return '[{"Name":"podman-machine-default","ConnectionInfo":{"PodmanPipe":{"Path":"\\\\\\\\.\\\\pipe\\\\podman-machine-default"}}}]' }
    return ''
}
function Lanzar-Proxy([string]$Ejecutable, [string[]]$Argumentos) { $script:proxy = @($Ejecutable) + $Argumentos }
function Start-Sleep { [CmdletBinding()] param([int]$Seconds) }
function Write-Host { param($Object) $script:mensajes.Add([string]$Object) }
"""

    def test_tuberia_presente_no_reinicia(self):
        self.ejecutar(self.PODMAN_SIMULADO + """
function Probar-Tuberia([string]$Ruta) { if ($Ruta -ne '\\\\.\\pipe\\podman-machine-default') { throw ('Tubería inesperada ' + $Ruta) }; return $true }
Asegurar-TuberiaPodman $conexion
if (($script:ordenes -join ';') -ne 'machine inspect podman-machine-default' -or $script:proxy) { throw ('Reinicia sin necesidad: ' + ($script:ordenes -join ';')) }
""")

    def test_tuberia_ausente_reintenta_y_lanza_proxy(self):
        self.ejecutar(self.PODMAN_SIMULADO + """
function Probar-Tuberia([string]$Ruta) { return [bool]$script:proxy }
Asegurar-TuberiaPodman $conexion
$esperadas = 'machine inspect podman-machine-default;machine list --format json;machine stop podman-machine-default;machine start podman-machine-default'
if (($script:ordenes -join ';') -ne $esperadas) { throw ('Orden de reintento inesperado: ' + ($script:ordenes -join ';')) }
if (-not $script:proxy) { throw 'No lanza win-sshproxy' }
if ((Split-Path -Leaf $script:proxy[0]) -ne 'win-sshproxy.exe') { throw 'Ejecutable de proxy inesperado' }
$argumentos = $script:proxy[1..5]
if ($argumentos[0] -ne 'podman-machine-default' -or $argumentos[2] -ne 'npipe:////./pipe/podman-machine-default' -or
    $argumentos[3] -ne $conexion.URI -or $argumentos[4] -ne $identidad) { throw ('Argumentos del proxy: ' + ($argumentos -join ' ')) }
if (-not $argumentos[1].StartsWith($env:XDG_DATA_HOME)) { throw 'Directorio de estado fuera de los datos de Podman' }
if ($script:mensajes -notcontains (Texto 'tuberia_reintento') -or $script:mensajes -notcontains (Texto 'tuberia_proxy')) { throw 'No avisa del reintento' }
""")

    def test_reintento_basta_sin_proxy(self):
        self.ejecutar(self.PODMAN_SIMULADO + """
function Probar-Tuberia([string]$Ruta) { return ($script:ordenes -contains 'machine start podman-machine-default') }
Asegurar-TuberiaPodman $conexion
if ($script:proxy) { throw 'Lanza el proxy aunque el reinicio bastaba' }
""")

    def test_tuberia_imposible_da_mensaje_claro(self):
        self.ejecutar(self.PODMAN_SIMULADO + """
function Probar-Tuberia([string]$Ruta) { return $false }
$mensaje = $null
try { Asegurar-TuberiaPodman $conexion } catch { $mensaje = $_.Exception.Message }
if ($mensaje -ne (Texto 'tuberia') -or -not $mensaje) { throw ('Mensaje inesperado: ' + $mensaje) }
if (-not $script:proxy) { throw 'No intenta el proxy antes de rendirse' }
""")

    def test_espera_a_que_el_antivirus_analice(self):
        self.ejecutar("""
$script:intentos = 0
$script:mensajes = [Collections.Generic.List[string]]::new()
function Componer {
    param([string[]]$Argumentos, [string]$Entrada = '')
    if (($Argumentos -join ' ') -ne 'exec -T clamav timeout 20 clamdscan --no-summary -') { throw 'Orden inesperada' }
    $script:intentos++
    if ($script:intentos -lt 3) { throw 'clamd todavía carga' }
    return "stream: OK`n"
}
function Start-Sleep { [CmdletBinding()] param([int]$Seconds) }
function Write-Host { param($Object) $script:mensajes.Add([string]$Object) }
Esperar-Antivirus
if ($script:intentos -ne 3) { throw ('Intentos: ' + $script:intentos) }
if ($script:mensajes -notcontains (Texto 'antivirus_espera')) { throw 'No avisa de la espera' }
function Componer { param([string[]]$Argumentos, [string]$Entrada = '') return 'stream: ERROR' }
$mensaje = $null
try { Esperar-Antivirus -Intentos 4 } catch { $mensaje = $_.Exception.Message }
if (-not $mensaje -or $mensaje -ne (Texto 'antivirus')) { throw ('Mensaje inesperado: ' + $mensaje) }
""")

    def test_arrancar_espera_al_antivirus(self):
        codigo = (RAIZ / "despliegue/windows/comun.ps1").read_text(encoding="utf-8-sig")
        arranque = codigo[codigo.index("function Arrancar"):]
        self.assertLess(arranque.index("'clamav', 'ldap', 'app', 'entrada-demo'"), arranque.index("Esperar-Antivirus"))
        self.assertLess(arranque.index("Esperar-Antivirus"), arranque.index("'herramientas', 'sembrar'"))

if __name__ == "__main__":
    unittest.main()
