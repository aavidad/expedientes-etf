"""Simula en PowerShell cada requisito ausente y comprueba qué instala instalar.cmd."""
import hashlib
import json
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

RAIZ = Path(__file__).resolve().parents[2]
PWSH = shutil.which("pwsh") or str(Path.home() / ".local/bin/pwsh")
MSI = "requisitos/podman-installer-windows-amd64.msi"
COMPOSE = "requisitos/docker-compose.exe"

# Sustituye lo que en Windows toca el sistema y anota cada acción.
SIMULACION = r"""
Preparar-Directorio (Join-Path $Raiz 'privado')
$script:EsWindows = $true
$env:SystemRoot = Join-Path $Raiz 'windows'
$env:LOCALAPPDATA = Join-Path $Raiz 'local'
$env:ProgramFiles = Join-Path $Raiz 'programas'
[void](New-Item -ItemType Directory -Force (Join-Path $env:SystemRoot 'System32'))
[IO.File]::WriteAllText((Join-Path $env:SystemRoot 'System32\wsl.exe'), 'simulado')
$script:acciones = [Collections.Generic.List[string]]::new()
$script:mensajes = [Collections.Generic.List[string]]::new()
$script:wslListo = $true
$script:wslTrasInstalar = $false
$script:codigoElevado = 0
$script:rechazaUac = $false
$script:codigoMsi = 0
$script:maquinas = '[{"Name":"podman-machine-default","VMType":"wsl","Running":true}]'
$script:composeListo = $true
$script:arranque = 'arranque-1'
$script:reanudacion = $null
function Comprobar-Equipo { $script:acciones.Add('equipo') }
function Leer-CodigoProducto([string]$Msi) { return '{11111111-2222-3333-4444-555555555555}' }
function Instalado { return (Leer-Instalado) }
function Probar-Wsl { return $script:wslListo }
function Arranque-Sistema { return $script:arranque }
function Ejecutar-Elevado([string]$Programa, [string]$Argumentos) {
    if ($script:rechazaUac) { throw (Texto 'uac_rechazado') }
    $script:acciones.Add('elevado ' + (Split-Path -Leaf $Programa) + ' ' + $Argumentos)
    $script:wslListo = $script:wslTrasInstalar
    return $script:codigoElevado
}
function Ejecutar-Programa([string]$Programa, [string]$Argumentos, [int]$Segundos = 1800) {
    $script:acciones.Add((Split-Path -Leaf $Programa) + ' ' + $Argumentos)
    if ($script:codigoMsi -eq 0) {
        $carpeta = Join-Path $env:LOCALAPPDATA 'Programs\Podman'
        [void](New-Item -ItemType Directory -Force $carpeta)
        [IO.File]::WriteAllText((Join-Path $carpeta 'podman.exe'), 'simulado')
    }
    return $script:codigoMsi
}
function Guardar-Reanudacion([string]$Orden) { $script:reanudacion = $Orden }
function Quitar-Reanudacion { $script:acciones.Add('quitar-reanudacion') }
function Ejecutar-Motor {
    param([string[]]$Argumentos, [string]$Entrada = '')
    $orden = $Argumentos -join ' '
    if ($orden -eq 'compose version') {
        if (-not $script:composeListo -and -not $env:PODMAN_COMPOSE_PROVIDER) { throw 'sin proveedor' }
        return 'v5'
    }
    $script:acciones.Add('podman ' + $orden)
    if ($orden -eq 'machine init --rootful --cpus 2 --memory 4096') {
        if ($env:CONTAINERS_MACHINE_PROVIDER -ne 'wsl') { throw 'proveedor no fijado' }
        $script:maquinas = '[{"Name":"vieja","VMType":"hyperv","Running":false},{"Name":"podman-machine-default","VMType":"wsl","Running":false}]'
    }
    if ($orden -eq 'machine list --format json') { return $script:maquinas }
    return ''
}
function Write-Host { param($Object) $script:mensajes.Add([string]$Object) }
function Get-Command { [CmdletBinding()] param([string]$Name, [string]$CommandType) }
function Debe([bool]$Condicion, [string]$Mensaje) { if (-not $Condicion) { throw ($Mensaje + ' | ' + ($script:acciones -join '; ')) } }
function Podman-Instalado {
    $carpeta = Join-Path $env:LOCALAPPDATA 'Programs\Podman'
    [void](New-Item -ItemType Directory -Force $carpeta)
    [IO.File]::WriteAllText((Join-Path $carpeta 'podman.exe'), 'simulado')
}
"""


@unittest.skipUnless(Path(PWSH).is_file(), "PowerShell no está instalado en este puesto")
class InstalarRequisitos(unittest.TestCase):
    def ejecutar(self, codigo, sumas_falsas=()):
        with tempfile.TemporaryDirectory(prefix="wreq-test-") as temporal:
            raiz = Path(temporal)
            paquete = raiz / "fuente"
            (paquete / "requisitos").mkdir(parents=True)
            for nombre in ("comun.ps1", "instalar.ps1"):
                shutil.copyfile(RAIZ / "despliegue/windows" / nombre, paquete / nombre)
            shutil.copyfile(RAIZ / "idiomas/es/textos/windows.json", paquete / "mensajes.json")
            sumas = {}
            for nombre in (MSI, COMPOSE):
                contenido = ("instalador " + nombre).encode()
                (paquete / nombre).write_bytes(contenido)
                sumas[nombre] = hashlib.sha256(contenido).hexdigest()
            for nombre in sumas_falsas:
                sumas[nombre] = "0" * 64
            (paquete / "SUMAS.json").write_text(json.dumps(sumas), encoding="utf-8")
            script = raiz / "ensayo.ps1"
            script.write_text("param([string]$Raiz)\n. (Join-Path $Raiz 'fuente/comun.ps1')\n$Puerto = 18381\n"
                              + SIMULACION + codigo, encoding="utf-8-sig")
            resultado = subprocess.run([PWSH, "-NoProfile", "-File", str(script), str(raiz)],
                                       capture_output=True, text=True, timeout=60)
            self.assertEqual(resultado.returncode, 0, resultado.stdout + resultado.stderr)

    def test_todo_presente_no_instala_nada(self):
        self.ejecutar(r"""
Podman-Instalado
Debe (-not (Instalar-Requisitos)) 'Pide reiniciar sin motivo'
Debe (($script:acciones -join ';') -eq 'equipo;quitar-reanudacion;podman machine list --format json') 'Acciones de más'
Debe (-not @((Instalado).PSObject.Properties).Count) 'Anota como propio lo que ya estaba'
Debe ($env:PATH.StartsWith((Join-Path $env:LOCALAPPDATA 'Programs\Podman'))) 'No añade Podman al PATH del proceso'
""")

    def test_sin_wsl_pide_permiso_y_programa_reinicio(self):
        self.ejecutar(r"""
$script:wslListo = $false
Debe (Instalar-Requisitos) 'No pide reiniciar'
Debe ($script:acciones -contains 'elevado wsl.exe --install --no-distribution') 'No instala WSL con permiso'
Debe (-not ($script:acciones -match 'msiexec|machine')) 'Sigue antes de reiniciar'
Debe ((Instalado).wsl -eq $true) 'No anota que instaló WSL'
Debe ($script:mensajes -contains (Texto 'wsl_instalando') -and $script:mensajes -contains (Texto 'reiniciar')) 'No avisa'
Debe ($script:reanudacion -like '*powershell.exe" * -NoExit -File "*instalar.ps1" -DirectorioPrivado "*privado" -Puerto 18381') ('Reanudación: ' + $script:reanudacion)
Debe ([IO.File]::ReadAllText((Join-Path $script:Privado 'wsl-pendiente.txt')).Trim() -eq 'arranque-1') 'Sin marca de arranque'
# Antes de reiniciar no vuelve a instalar; después, si sigue sin WSL, da un error claro.
$script:acciones.Clear()
Debe (Instalar-Requisitos) 'Olvida el reinicio pendiente'
Debe (-not ($script:acciones -match 'elevado')) 'Reinstala antes de reiniciar'
$script:arranque = 'arranque-2'
$mensaje = $null
try { [void](Instalar-Requisitos) } catch { $mensaje = $_.Exception.Message }
Debe ($mensaje -eq (Texto 'wsl_sin_activar')) ('Mensaje tras reiniciar: ' + $mensaje)
# Ya activo tras reiniciar: borra la marca y sigue con Podman.
$script:wslListo = $true
Podman-Instalado
Debe (-not (Instalar-Requisitos)) 'Pide reiniciar con WSL listo'
Debe (-not (Test-Path (Join-Path $script:Privado 'wsl-pendiente.txt'))) 'Conserva la marca'
""")

    def test_uac_rechazado_da_error_claro(self):
        self.ejecutar(r"""
$script:wslListo = $false
$script:rechazaUac = $true
$mensaje = $null
try { [void](Instalar-Requisitos) } catch { $mensaje = $_.Exception.Message }
Debe ($mensaje -eq (Texto 'uac_rechazado')) ('Mensaje: ' + $mensaje)
Debe (-not $script:reanudacion) 'Programa reinicio sin WSL'
Debe (-not (Test-Path (Join-Path $script:Privado 'wsl-pendiente.txt'))) 'Deja marca sin instalar'
""")

    def test_wsl_sin_reinicio_continua_y_fallo_de_wsl(self):
        self.ejecutar(r"""
$script:wslListo = $false
$script:wslTrasInstalar = $true
Podman-Instalado
Debe (-not (Instalar-Requisitos)) 'Pide reiniciar aunque WSL ya funciona'
Debe (-not $script:reanudacion) 'Programa reanudación innecesaria'
$script:wslListo = $false
$script:wslTrasInstalar = $false
$script:codigoElevado = 1
$mensaje = $null
try { [void](Instalar-Requisitos) } catch { $mensaje = $_.Exception.Message }
Debe ($mensaje -eq ((Texto 'wsl_fallo') -f 1)) ('Mensaje: ' + $mensaje)
""")

    def test_sin_podman_instala_el_msi_por_usuario(self):
        self.ejecutar(r"""
Debe (-not (Instalar-Requisitos)) 'Pide reiniciar'
$msi = @($script:acciones | Where-Object { $_ -like 'msiexec.exe *' })
Debe ($msi.Count -eq 1) 'No ejecuta msiexec una vez'
Debe ($msi[0] -like '*/package "*podman-installer-windows-amd64.msi" /quiet /norestart MSIINSTALLPERUSER=1 MACHINE_PROVIDER=wsl /l`*v "*podman-instalacion.log"') ('Orden: ' + $msi[0])
Debe (-not ($script:acciones -match 'elevado')) 'Pide permisos para Podman'
Debe ($script:mensajes -contains (Texto 'podman_instalando')) 'No avisa de Podman'
Debe ($env:PATH.StartsWith((Join-Path $env:LOCALAPPDATA 'Programs\Podman'))) 'No añade Podman al PATH'
Debe ((Instalado).podman -eq '{11111111-2222-3333-4444-555555555555}') 'No anota el Podman instalado'
""")

    def test_msi_alterado_no_se_ejecuta(self):
        self.ejecutar(r"""
$mensaje = $null
try { [void](Instalar-Requisitos) } catch { $mensaje = $_.Exception.Message }
Debe ($mensaje -eq (Texto 'suma')) ('Mensaje: ' + $mensaje)
Debe (-not ($script:acciones -match 'msiexec')) 'Ejecuta un instalador alterado'
""", sumas_falsas=(MSI,))

    def test_msi_fallido_da_codigo_y_registro(self):
        self.ejecutar(r"""
$script:codigoMsi = 1618
$mensaje = $null
try { [void](Instalar-Requisitos) } catch { $mensaje = $_.Exception.Message }
Debe ($mensaje -eq ((Texto 'podman_fallo') -f 1618, (Join-Path $script:Privado 'podman-instalacion.log'))) ('Mensaje: ' + $mensaje)
""")

    def test_sin_maquina_la_crea_rootful_y_la_enciende(self):
        self.ejecutar(r"""
Podman-Instalado
$script:maquinas = '[{"Name":"vieja","VMType":"hyperv","Running":false}]'
$env:CONTAINERS_MACHINE_PROVIDER = 'hyperv'
Debe (-not (Instalar-Requisitos)) 'Pide reiniciar'
$ordenes = @($script:acciones | Where-Object { $_ -like 'podman *' }) -join ';'
Debe ($ordenes -eq 'podman machine list --format json;podman machine init --rootful --cpus 2 --memory 4096;podman machine list --format json;podman machine start') ('Órdenes: ' + $ordenes)
Debe ((Instalado).maquina -eq 'podman-machine-default') 'No anota la máquina creada'
Debe ($env:CONTAINERS_MACHINE_PROVIDER -eq 'hyperv') 'No restaura el proveedor de la persona'
Debe ($script:mensajes -contains (Texto 'maquina_creando')) 'No avisa de la máquina'
""")

    def test_sin_compose_usa_el_incluido(self):
        self.ejecutar(r"""
$script:composeListo = $false
$env:PODMAN_COMPOSE_PROVIDER = $null
Asegurar-Compose
$copia = Join-Path $script:Privado 'herramientas\docker-compose.exe'
Debe (Test-Path $copia) 'No copia docker-compose'
Debe ($env:PODMAN_COMPOSE_PROVIDER -eq $copia) 'No apunta Podman a la copia'
Debe ([IO.File]::ReadAllText($copia) -eq 'instalador requisitos/docker-compose.exe') 'Copia distinta'
# Un Compose ya instalado (Podman Desktop, Docker Desktop) se respeta.
$env:PODMAN_COMPOSE_PROVIDER = $null
$script:composeListo = $true
Remove-Item $copia
Asegurar-Compose
Debe (-not (Test-Path $copia) -and -not $env:PODMAN_COMPOSE_PROVIDER) 'Sustituye el Compose existente'
""")

    def test_compose_alterado_no_se_copia(self):
        self.ejecutar(r"""
$script:composeListo = $false
$env:PODMAN_COMPOSE_PROVIDER = $null
$mensaje = $null
try { Asegurar-Compose } catch { $mensaje = $_.Exception.Message }
Debe ($mensaje -eq (Texto 'suma')) ('Mensaje: ' + $mensaje)
Debe (-not (Test-Path (Join-Path $script:Privado 'herramientas'))) 'Copia un Compose alterado'
""", sumas_falsas=(COMPOSE,))

    def test_rancher_y_solo_comprobar_no_instalan(self):
        codigo = (RAIZ / "despliegue/windows/instalar.ps1").read_text(encoding="utf-8-sig")
        self.assertIn("if ($Motor -eq 'Podman' -and -not $SoloComprobar -and (Instalar-Requisitos)) { exit 3010 }", codigo)
        self.assertLess(codigo.index("Instalar-Requisitos"), codigo.index("Comprobar-Requisitos"))


if __name__ == "__main__":
    unittest.main()
