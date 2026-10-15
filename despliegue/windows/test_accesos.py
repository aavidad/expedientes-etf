"""Ensaya límites de la instalación y la retirada de archivos en PowerShell."""
import json
import re
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

RAIZ = Path(__file__).resolve().parents[2]
WINDOWS = RAIZ / "despliegue/windows"
# Órdenes NSIS que lanzan otro programa.
ORDEN_EXEC = re.compile(r"^\s*(Exec|ExecWait|ExecShell|ExecShellWait|nsExec::\w+)\s", re.MULTILINE)
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
""")

    def test_accesos_solo_mediante_el_lanzador(self):
        # Los scripts no abren, crean ni borran accesos: ejecutan el lanzador
        # «/accesos crear|quitar» y traducen su código de salida.
        self.ejecutar(r"""
Preparar-Directorio (Join-Path $Raiz 'privado')
$script:EsWindows = $true
$script:llamadas = [Collections.Generic.List[string]]::new()
$script:codigo = 0
function Buscar-Navegador { return 'navegador' }
function Guardar-PaqueteInstalado { }
function Write-Host { param($Object) }
function Start-Process {
    param([string]$FilePath, [string[]]$ArgumentList, [switch]$Wait, [switch]$PassThru)
    if (-not $Wait -or -not $PassThru) { throw 'No espera al lanzador' }
    $script:llamadas.Add((Split-Path -Leaf $FilePath) + ' ' + ($ArgumentList -join ' '))
    return [pscustomobject]@{ ExitCode = $script:codigo }
}
Crear-Accesos
if (($script:llamadas -join ';') -ne 'Expedientes ETF.exe /accesos crear') { throw ('Crear: ' + ($script:llamadas -join ';')) }
if (-not (Test-Path (Join-Path $script:Privado 'inicio.json'))) { throw 'No guarda el motor para el acceso' }
$script:codigo = 2
$mensaje = $null
try { Crear-Accesos } catch { $mensaje = $_.Exception.Message }
if ($mensaje -ne (Texto 'acceso_ocupado')) { throw ('Ocupado: ' + $mensaje) }
$script:codigo = 1
$mensaje = $null
try { Crear-Accesos } catch { $mensaje = $_.Exception.Message }
if (-not $mensaje -or $mensaje -ne (Texto 'accesos_error')) { throw ('Fallo: ' + $mensaje) }
# Sin lanzador en la copia estable no se ejecuta nada.
$script:llamadas.Clear()
$script:codigo = 0
Eliminar-Accesos
if ($script:llamadas.Count) { throw 'Ejecuta un lanzador que no existe' }
New-Item -ItemType Directory (Ruta-PaqueteInstalado) -Force | Out-Null
[IO.File]::WriteAllText((Ruta-Lanzador), 'lanzador')
Eliminar-Accesos
if (($script:llamadas -join ';') -ne 'Expedientes ETF.exe /accesos quitar') { throw ('Quitar: ' + ($script:llamadas -join ';')) }
""")

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

class AccesosEnNsis(unittest.TestCase):
    """Los .lnk sólo los tocan el lanzador, el instalador y su desinstalador."""
    NSH = (WINDOWS / "accesos.nsh").read_text(encoding="utf-8")
    LANZADOR = (WINDOWS / "lanzador.nsi").read_text(encoding="utf-8")
    INSTALADOR = (WINDOWS / "instalador.nsi").read_text(encoding="utf-8")

    def test_ningun_script_toca_accesos_directos(self):
        # Algunos antivirus matan cmd o PowerShell si abren un .lnk y después
        # crean ejecutables en AppData y lanzan otro intérprete.
        scripts = sorted(WINDOWS.glob("*.ps1")) + sorted(WINDOWS.glob("*.cmd"))
        self.assertTrue(scripts)
        for ruta in scripts:
            codigo = ruta.read_text(encoding="utf-8-sig").lower()
            for prohibido in (".lnk", "createshortcut", "iwshshortcut", "shell.application"):
                self.assertNotIn(prohibido, codigo, f"{ruta.name}: {prohibido}")
            # WScript.Shell sólo para el aviso de error de abrir.ps1 (Popup,
            # en proceso, sin ficheros); nunca para accesos directos.
            if "wscript.shell" in codigo:
                self.assertEqual(ruta.name, "abrir.ps1")
                self.assertEqual(codigo.count("wscript.shell"), 1)
                self.assertIn("$aviso.popup(", codigo)

    def test_nsis_crea_migra_y_borra_los_accesos(self):
        for carpeta in ("$DESKTOP", "$SMPROGRAMS"):
            acceso = f'"{carpeta}\\${{ACCESO_NOMBRE}}.lnk"'
            self.assertIn(f'CreateShortCut {acceso} "$R3\\${{LANZADOR_EXE}}" "" "$R3\\aplicacion.ico" 0 SW_SHOWNORMAL "" "${{ACCESO_DESCRIPCION}}"', self.NSH)
            self.assertIn(f"Delete {acceso}", self.NSH)
        # Sólo se sustituyen o borran los propios, también los antiguos a PowerShell.
        self.assertIn('${If} $R1 == "$R3\\${LANZADOR_EXE}"', self.NSH)
        self.assertIn('-File "$R3\\abrir.ps1"', self.NSH)
        self.assertIn('-WindowStyle Hidden -File "$R3\\abrir.ps1"', self.NSH)
        self.assertFalse(re.search(ORDEN_EXEC, self.NSH))

    def test_lanzador_acepta_accesos_crear_y_quitar(self):
        self.assertIn('!include "${__FILEDIR__}/accesos.nsh"', self.LANZADOR)
        crear = self.LANZADOR.index('${ElseIf} $0 == "/accesos crear"')
        quitar = self.LANZADOR.index('${ElseIf} $0 == "/accesos quitar"')
        self.assertLess(self.LANZADOR.index('${If} $0 == ""'), self.LANZADOR.index("Exec "))
        self.assertLess(self.LANZADOR.index("Exec "), crear)
        self.assertIn("Call ComprobarAccesos", self.LANZADOR[crear:quitar])
        self.assertIn("SetErrorLevel 2", self.LANZADOR[crear:quitar])
        self.assertIn("Call CrearAccesos", self.LANZADOR[crear:quitar])
        self.assertIn("Call QuitarAccesos", self.LANZADOR[quitar:])
        # Con /accesos no se lanza ningún intérprete.
        self.assertEqual(len(re.findall(ORDEN_EXEC, self.LANZADOR)), 1)

    def test_instalador_comprueba_antes_y_desinstalador_borra_al_final(self):
        seccion = self.INSTALADOR[self.INSTALADOR.index("Section\n"):self.INSTALADOR.index('Section "Uninstall"')]
        self.assertLess(seccion.index("Call ComprobarAccesos"), seccion.index("File /r"))
        self.assertIn("$(instalador_acceso_ocupado)", seccion)
        self.assertLess(seccion.index("Call CrearAccesos"), seccion.index("Exec '"))
        retirada = self.INSTALADOR[self.INSTALADOR.index('Section "Uninstall"'):]
        # PowerShell termina antes de tocar los accesos, y no se lanza después.
        self.assertLess(retirada.index("desinstalar.ps1"), retirada.index("Call un.QuitarAccesos"))
        self.assertEqual(len(re.findall(ORDEN_EXEC, retirada)), 1)
        self.assertEqual(retirada.rstrip().splitlines()[-2].strip(), "Call un.QuitarAccesos")


if __name__ == "__main__":
    unittest.main()
