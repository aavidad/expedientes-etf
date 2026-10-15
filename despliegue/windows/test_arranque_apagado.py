"""Simula en PowerShell la máquina de Podman encendiéndose, la apertura fallida y la parada."""
import json
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

RAIZ = Path(__file__).resolve().parents[2]
WINDOWS = RAIZ / "despliegue/windows"
PWSH = shutil.which("pwsh") or str(Path.home() / ".local/bin/pwsh")
OCULTA = "ensayo-sin-valor-real"

# Motor simulado: cada consulta de máquinas devuelve el siguiente estado de la
# lista $script:estados (el último se repite) y anota todas las órdenes.
MOTOR = r"""
Preparar-Directorio (Join-Path $Raiz 'privado')
$script:EsWindows = $true
$script:ordenes = [Collections.Generic.List[string]]::new()
$script:mensajes = [Collections.Generic.List[string]]::new()
$script:esperas = 0
$script:estados = @('{"Running":true,"Starting":false}')
$script:consulta = 0
$script:arranqueFalla = $false
$script:tuberia = $true
$script:enMarcha = ''
$script:enMarchaRoot = ''
$script:paradaFalla = $false
$script:limiteParada = $null
function Buscar-Podman { return (Join-Path $Raiz 'podman.exe') }
function Asegurar-Compose { }
function Probar-Tuberia([string]$Ruta) { return $script:tuberia }
function Start-Sleep { [CmdletBinding()] param([int]$Seconds) $script:esperas++ }
function Write-Host { param($Object) $script:mensajes.Add([string]$Object) }
function Estado([string]$Json) {
    $maquina = $Json | ConvertFrom-Json
    $maquina | Add-Member Name 'podman-machine-default'
    $maquina | Add-Member VMType 'wsl'
    return '[' + ($maquina | ConvertTo-Json -Compress) + ']'
}
function Ejecutar-Motor {
    param([string[]]$Argumentos, [string]$Entrada = '', [int]$Segundos = 1200)
    $orden = $Argumentos -join ' '
    $script:ordenes.Add($orden)
    switch -Wildcard ($orden) {
        'machine list --format json' {
            $estado = $script:estados[[Math]::Min($script:consulta, $script:estados.Count - 1)]
            $script:consulta++
            return (Estado $estado)
        }
        'system connection list --format json' { return '[{"Name":"podman-machine-default-root","URI":"ssh://root@127.0.0.1:49708/run/podman/podman.sock","Default":true}]' }
        'machine inspect *' { return '[{"ConnectionInfo":{"PodmanPipe":{"Path":"\\\\.\\pipe\\podman-machine-default"}}}]' }
        'info --format json' { return '{"host":{"security":{"rootless":false}}}' }
        'machine start *' { if ($script:arranqueFalla) { throw 'VM already running or starting' }; return '' }
        'machine stop *' { $script:limiteParada = $Segundos; if ($script:paradaFalla) { throw (Texto 'tiempo') }; return '' }
        '--connection podman-machine-default ps --quiet' { return $script:enMarcha }
        '--connection podman-machine-default-root ps --quiet' { return $script:enMarchaRoot }
    }
    return ''
}
function Debe([bool]$Condicion, [string]$Mensaje) { if (-not $Condicion) { throw ($Mensaje + ' | ' + ($script:ordenes -join '; ')) } }
function Sin-Reinicios { Debe (-not ($script:ordenes -match '^machine (start|stop)')) 'Arranca o para la máquina encima de otro programa' }
"""


@unittest.skipUnless(Path(PWSH).is_file(), "PowerShell no está instalado en este puesto")
class ArranqueYApagado(unittest.TestCase):
    def paquete(self, raiz, extra=""):
        paquete = raiz / "fuente"
        paquete.mkdir(parents=True)
        for nombre in ("comun.ps1", "accesos.ps1", "abrir.ps1", "parar.ps1"):
            shutil.copyfile(WINDOWS / nombre, paquete / nombre)
        if extra:
            with (paquete / "comun.ps1").open("a", encoding="utf-8") as comun:
                comun.write("\n" + extra)
        shutil.copyfile(RAIZ / "idiomas/es/textos/windows.json", paquete / "mensajes.json")
        return paquete

    def ejecutar(self, codigo):
        with tempfile.TemporaryDirectory(prefix="wrei-test-") as temporal:
            raiz = Path(temporal)
            self.paquete(raiz)
            script = raiz / "ensayo.ps1"
            script.write_text("param([string]$Raiz)\n. (Join-Path $Raiz 'fuente/comun.ps1')\n"
                              ". (Join-Path $Raiz 'fuente/accesos.ps1')\n$Puerto = 18381\n$Motor = 'Podman'\n"
                              + MOTOR + codigo, encoding="utf-8-sig")
            resultado = subprocess.run([PWSH, "-NoProfile", "-File", str(script), str(raiz)],
                                       capture_output=True, text=True, timeout=60)
            self.assertEqual(resultado.returncode, 0, resultado.stdout + resultado.stderr)

    def test_espera_a_la_maquina_que_se_esta_encendiendo(self):
        self.ejecutar(r"""
$script:estados = @('{"Running":false,"Starting":true}', '{"Running":true,"Starting":true}', '{"Running":true,"Starting":false}')
Comprobar-Requisitos 'Podman' -SoloMotor -EncenderMaquina
Sin-Reinicios
Debe ($script:mensajes -contains (Texto 'maquina_esperando')) 'No avisa de la espera'
Debe ($script:esperas -eq 1) ('Esperas: ' + $script:esperas)
""")

    def test_maquina_que_no_termina_de_encender_da_mensaje_claro(self):
        self.ejecutar(r"""
$script:estados = @('{"Running":true,"Starting":true}')
$mensaje = $null
try { Comprobar-Requisitos 'Podman' -SoloMotor -EncenderMaquina } catch { $mensaje = $_.Exception.Message }
Debe ($mensaje -eq (Texto 'maquina_arrancando')) ('Mensaje: ' + $mensaje)
Sin-Reinicios
Debe ($script:esperas -eq 36) ('Debe esperar 3 minutos en pausas de 5 s: ' + $script:esperas)
""")

    def test_arranque_simultaneo_espera_en_vez_de_fallar(self):
        self.ejecutar(r"""
# Parada al consultar; Podman Desktop la empieza a encender antes que nosotros.
$script:estados = @('{"Running":false,"Starting":false}', '{"Running":true,"Starting":true}', '{"Running":true,"Starting":false}')
$script:arranqueFalla = $true
Comprobar-Requisitos 'Podman' -SoloMotor -EncenderMaquina
Debe (-not ($script:ordenes -match '^machine stop')) 'Para la máquina'
Debe ($script:mensajes -contains (Texto 'maquina_esperando')) 'No avisa de la espera'
# Sin nadie que la encienda, el fallo de la orden se conserva.
$script:estados = @('{"Running":false,"Starting":false}')
$script:consulta = 0
$mensaje = $null
try { Comprobar-Requisitos 'Podman' -SoloMotor -EncenderMaquina } catch { $mensaje = $_.Exception.Message }
Debe ($mensaje -eq 'VM already running or starting') ('Mensaje: ' + $mensaje)
""")

    def test_tuberia_tardia_no_reinicia_la_maquina(self):
        self.ejecutar(r"""
$script:pruebas = 0
function Probar-Tuberia([string]$Ruta) { $script:pruebas++; return ($script:pruebas -ge 5) }
Comprobar-Requisitos 'Podman' -SoloMotor -EncenderMaquina
Sin-Reinicios
Debe ($script:mensajes -notcontains (Texto 'tuberia_reintento')) 'Avisa de un reinicio que no hace'
""")

    def test_tuberia_ausente_mientras_otro_la_enciende_espera(self):
        self.ejecutar(r"""
# Encendida sin tubería durante la espera corta; otro programa sigue arrancándola.
$script:estados = @('{"Running":true,"Starting":false}', '{"Running":true,"Starting":true}', '{"Running":true,"Starting":false}')
$script:tuberia = $false
function Start-Sleep { [CmdletBinding()] param([int]$Seconds) $script:esperas++; if ($script:consulta -ge 2) { $script:tuberia = $true } }
Comprobar-Requisitos 'Podman' -SoloMotor -EncenderMaquina
Sin-Reinicios
Debe ($script:mensajes -contains (Texto 'maquina_esperando')) 'No espera al otro programa'
""")

    def test_proxy_propio_se_anota_y_se_cierra(self):
        self.ejecutar(r"""
$inicio = [DateTime]::new(2026, 10, 15, 8, 0, 0, [DateTimeKind]::Utc)
$script:detenidos = [Collections.Generic.List[int]]::new()
$script:procesos = @{}
function Start-Process {
    [CmdletBinding()] param([string]$FilePath, [string]$ArgumentList, [string]$WindowStyle, [switch]$PassThru)
    Debe ($PassThru -and $WindowStyle -eq 'Hidden') 'Proxy sin identificador o visible'
    return [pscustomobject]@{ Id = 4321; StartTime = $inicio }
}
function Get-Process { [CmdletBinding()] param([int]$Id) return $script:procesos[$Id] }
function Stop-Process { [CmdletBinding()] param([int]$Id, [switch]$Force) $script:detenidos.Add($Id) }
function Wait-Process { [CmdletBinding()] param([int]$Id, [int]$Timeout) }
$ruta = Join-Path $script:Privado 'proxy.json'
Lanzar-Proxy (Join-Path $Raiz 'win-sshproxy.exe') @('podman-machine-default')
$datos = Get-Content -LiteralPath $ruta -Raw | ConvertFrom-Json
Debe ($datos.id -eq 4321 -and [int64]$datos.inicio -eq $inicio.Ticks) 'No anota el proxy'
# Otro proceso con el mismo número no se cierra.
$script:procesos[4321] = [pscustomobject]@{ Id = 4321; ProcessName = 'win-sshproxy'; StartTime = $inicio.AddMinutes(5) }
Cerrar-ProxyPropio
Debe ($script:detenidos.Count -eq 0) 'Cierra un proceso ajeno'
Debe (-not (Test-Path $ruta)) 'Conserva una anotación caducada'
Lanzar-Proxy (Join-Path $Raiz 'win-sshproxy.exe') @('podman-machine-default')
$script:procesos[4321] = [pscustomobject]@{ Id = 4321; ProcessName = 'notepad'; StartTime = $inicio }
Cerrar-ProxyPropio
Debe ($script:detenidos.Count -eq 0) 'Cierra un proceso con otro nombre'
Lanzar-Proxy (Join-Path $Raiz 'win-sshproxy.exe') @('podman-machine-default')
$script:procesos[4321] = [pscustomobject]@{ Id = 4321; ProcessName = 'win-sshproxy'; StartTime = $inicio }
Cerrar-ProxyPropio
Debe (($script:detenidos -join ',') -eq '4321') 'No cierra el proxy propio'
Debe (-not (Test-Path $ruta)) 'No borra la anotación'
""")

    def test_parar_apaga_la_maquina_si_no_tiene_otros_contenedores(self):
        self.ejecutar(r"""
$script:cierres = 0
function Cerrar-ProxyPropio { $script:cierres++ }
$script:Maquina = 'podman-machine-default'
$script:enMarcha = "abc123`n"
Apagar-MaquinaPodman
$script:enMarcha = ''
$script:enMarchaRoot = "def456`n"
Apagar-MaquinaPodman
Debe (-not ($script:ordenes -match '^machine stop')) 'Apaga una máquina con contenedores ajenos'
Debe ($script:mensajes -contains (Texto 'maquina_en_uso') -and $script:cierres -eq 0) 'No avisa o cierra el proxy en uso'
$script:enMarchaRoot = ''
Apagar-MaquinaPodman
Debe ($script:ordenes -contains 'machine stop podman-machine-default' -and $script:limiteParada -eq 120) 'No apaga con límite de 2 minutos'
Debe ($script:mensajes -contains (Texto 'maquina_apagando') -and $script:cierres -eq 1) 'No avisa o no cierra el proxy'
# Sin respuesta de ninguna conexión no se apaga a ciegas.
$script:mensajes.Clear()
$script:ordenes.Clear()
function Ejecutar-Motor { param([string[]]$Argumentos, [string]$Entrada = '', [int]$Segundos = 1200) $script:ordenes.Add($Argumentos -join ' '); throw 'sin conexión' }
Apagar-MaquinaPodman
Debe (-not ($script:ordenes -match '^machine stop') -and $script:cierres -eq 1) 'Apaga sin saber qué contiene'
Debe ($script:mensajes -contains (Texto 'maquina_no_apagada')) 'No avisa de que sigue encendida'
$script:paradaFalla = $true
$script:mensajes.Clear()
function Ejecutar-Motor {
    param([string[]]$Argumentos, [string]$Entrada = '', [int]$Segundos = 1200)
    if ($Argumentos[0] -eq 'machine') { throw (Texto 'tiempo') }
    return ''
}
Apagar-MaquinaPodman
Debe ($script:mensajes -contains (Texto 'maquina_no_apagada') -and $script:cierres -eq 2) 'Un fallo al apagar interrumpe la parada'
""")

    def test_parar_y_desinstalar_apagan_tras_bajar_los_contenedores(self):
        parar = (WINDOWS / "parar.ps1").read_text(encoding="utf-8-sig")
        self.assertIn("Comprobar-Requisitos $Motor -SoloMotor -PermitirApagada", parar)
        self.assertLess(parar.index("'down'"), parar.index("Apagar-MaquinaPodman"))
        self.assertIn("if ($script:MaquinaApagada) { Cerrar-ProxyPropio }", parar)
        desinstalar = (WINDOWS / "desinstalar.ps1").read_text(encoding="utf-8-sig")
        for orden in ("Retirar-Requisitos", "Quitar-Reanudacion", "Eliminar-Accesos", "Eliminar-ArchivosInstalados"):
            self.assertLess(desinstalar.index("'down', '--volumes'"), desinstalar.index(orden), orden)
        self.assertLess(desinstalar.index("$wslNuestro ="), desinstalar.index("Eliminar-ArchivosInstalados"))
        for nombre in ("actualizar.ps1", "reiniciar-datos.ps1", "desinstalar.ps1"):
            self.assertIn("-EncenderMaquina", (WINDOWS / nombre).read_text(encoding="utf-8-sig"), nombre)

    def test_parar_con_la_maquina_apagada_no_falla(self):
        self.ejecutar(r"""
$script:estados = @('{"Running":false,"Starting":false}')
Comprobar-Requisitos 'Podman' -SoloMotor -PermitirApagada
Debe $script:MaquinaApagada 'No anota la máquina apagada'
Sin-Reinicios
Debe (-not ($script:ordenes -match '^info')) 'Sigue comprobando con la máquina apagada'
""")

    RETIRADA = r"""
$script:Maquina = 'podman-machine-default'
$script:cierres = 0
$script:respuesta = ''
$script:preguntas = [Collections.Generic.List[string]]::new()
$script:programas = [Collections.Generic.List[string]]::new()
$script:codigoMsi = 0
function Cerrar-ProxyPropio { $script:cierres++ }
function Read-Host { param([string]$Prompt) $script:preguntas.Add($Prompt); return $script:respuesta }
function Ejecutar-Programa([string]$Programa, [string]$Argumentos, [int]$Segundos = 1800) {
    $script:programas.Add((Split-Path -Leaf $Programa) + ' ' + $Argumentos + ' ' + $Segundos)
    return $script:codigoMsi
}
function Ruta-Sistema([string]$Relativa) { return (Join-Path $Raiz $Relativa) }
"""

    def test_desinstalar_no_toca_lo_que_ya_habia(self):
        self.ejecutar(self.RETIRADA + r"""
Retirar-Requisitos
Debe ($script:preguntas.Count -eq 0) 'Pregunta por lo que no instaló'
Debe (-not ($script:ordenes -match '^machine rm') -and $script:programas.Count -eq 0) 'Quita lo que ya estaba'
Debe ($script:ordenes -contains 'machine stop podman-machine-default') 'No apaga la máquina'
""")

    def test_desinstalar_pregunta_y_quita_maquina_y_podman_propios(self):
        self.ejecutar(self.RETIRADA + r"""
Anotar-Instalado 'maquina' 'podman-machine-default'
Anotar-Instalado 'podman' '{11111111-2222-3333-4444-555555555555}'
Anotar-Instalado 'wsl' $true
# Sin escribir la palabra se conserva todo.
Retirar-Requisitos
Debe ($script:preguntas -contains (Texto 'quitar_podman_pregunta')) 'No pregunta antes de quitar'
Debe (-not ($script:ordenes -match '^machine rm') -and $script:programas.Count -eq 0) 'Quita sin confirmar'
Debe ($script:mensajes -contains (Texto 'requisitos_conservados')) 'No avisa de que conserva'
# Con la palabra se quitan la máquina y después Podman, nunca WSL.
$script:ordenes.Clear()
$script:respuesta = Texto 'quitar_palabra'
Retirar-Requisitos
$rm = [array]::IndexOf($script:ordenes.ToArray(), 'machine rm --force podman-machine-default')
Debe ($rm -ge 0) 'No quita la máquina propia'
Debe (($script:programas -join ';') -like 'msiexec.exe /x {11111111-2222-3333-4444-555555555555} /quiet /norestart /l`*v "*podman-desinstalacion.log" 900') ('MSI: ' + ($script:programas -join ';'))
Debe (-not ($script:programas -match 'wsl')) 'Toca WSL'
Debe ($script:cierres -ge 1) 'No cierra el proxy'
""")

    def test_desinstalar_conserva_si_hay_otros_contenedores_o_falla(self):
        self.ejecutar(self.RETIRADA + r"""
Anotar-Instalado 'maquina' 'podman-machine-default'
Anotar-Instalado 'podman' '{11111111-2222-3333-4444-555555555555}'
$script:respuesta = Texto 'quitar_palabra'
$script:enMarchaRoot = "abc123`n"
Retirar-Requisitos
Debe ($script:mensajes -contains (Texto 'requisitos_en_uso') -and $script:preguntas.Count -eq 0) 'Ofrece quitar una máquina en uso'
Debe (-not ($script:ordenes -match '^machine (rm|stop)') -and $script:programas.Count -eq 0) 'Quita o apaga una máquina en uso'
# Una máquina que no se puede quitar no deja a Podman sin desinstalar a medias.
$script:enMarchaRoot = ''
function Ejecutar-Motor {
    param([string[]]$Argumentos, [string]$Entrada = '', [int]$Segundos = 1200)
    $orden = $Argumentos -join ' '
    $script:ordenes.Add($orden)
    if ($orden -like 'machine rm *') { throw 'ocupada' }
    if ($orden -eq 'machine list --format json') { return (Estado '{"Running":true,"Starting":false}') }
    return ''
}
Retirar-Requisitos
Debe ($script:mensajes -contains (Texto 'maquina_quitar_fallo') -and $script:programas.Count -eq 0) 'Desinstala Podman con la máquina sin quitar'
# Un fallo del MSI se avisa con su registro y no detiene la desinstalación.
Anotar-Instalado 'maquina' 'otra'
$script:codigoMsi = 1603
Retirar-Requisitos
Debe ($script:mensajes -contains ((Texto 'podman_quitar_fallo') -f 1603, (Join-Path $script:Privado 'podman-desinstalacion.log'))) 'No avisa del fallo del MSI'
# Sin código de producto remite a Configuración.
Anotar-Instalado 'podman' $null
Retirar-Requisitos
Debe ($script:mensajes -contains (Texto 'podman_quitar_manual')) 'No remite a Configuración'
""")

    def test_desinstalar_retira_archivos_propios_de_la_carpeta(self):
        self.ejecutar(r"""
foreach ($nombre in @('instalado.json', 'wsl-pendiente.txt', 'podman-instalacion.log', 'proxy.json', 'registro-abrir.log')) {
    [IO.File]::WriteAllText((Join-Path $script:Privado $nombre), 'x')
}
[void](New-Item -ItemType Directory (Join-Path $script:Privado 'herramientas'))
[IO.File]::WriteAllText((Join-Path $script:Privado 'herramientas\docker-compose.exe'), 'x')
Eliminar-ArchivosInstalados
Debe (-not (Test-Path $script:Privado)) ('Quedan archivos: ' + (@(Get-ChildItem -Recurse $script:Privado | ForEach-Object Name) -join ', '))
""")

    def test_abrir_fallido_avisa_por_encima_y_deja_registro_sin_contrasenas(self):
        with tempfile.TemporaryDirectory(prefix="wrei-test-") as temporal:
            raiz = Path(temporal)
            privado = raiz / "privado"
            privado.mkdir()
            (privado / "instalacion.json").write_text(json.dumps({"proyecto": "etf-windows", "formato": 1}))
            (privado / "inicio.json").write_text(json.dumps({"motor": "Podman"}))
            (privado / "entorno.env").write_text(f"ETF_WINDOWS_PUERTO=18381\nETF_APP_CONTRASENA={OCULTA}\n")
            (privado / "usuarios.json").write_text(json.dumps({"ines.moreno": "Cuenta-Inventada-77"}))
            mostrado = raiz / "mostrado.txt"
            simulacion = r"""
function Invoke-WebRequest { throw 'sin respuesta' }
function Mostrar-Error([string]$Mensaje, [string]$Titulo) {
    [IO.File]::WriteAllText('""" + str(mostrado) + r"""', $Titulo + '|' + $Mensaje)
}
function Comprobar-Requisitos {
    Write-Host (Texto 'maquina_esperando')
    $script:ErrorMotor = 'machine start: fallo con """ + OCULTA + r""" y Cuenta-Inventada-77 password=OtraClave1'
    throw (Texto 'maquina_arrancando')
}
"""
            paquete = self.paquete(privado / "paquete-falso", simulacion)
            destino = privado / "paquete"
            paquete.rename(destino)
            resultado = subprocess.run([PWSH, "-NoProfile", "-File", str(destino / "abrir.ps1")],
                                       capture_output=True, text=True, timeout=60)
            self.assertEqual(resultado.returncode, 1, resultado.stdout + resultado.stderr)
            textos = json.loads((RAIZ / "idiomas/es/textos/windows.json").read_text(encoding="utf-8"))
            titulo, mensaje = mostrado.read_text(encoding="utf-8").split("|", 1)
            registro = privado / "registro-abrir.log"
            self.assertEqual(titulo, textos["windows.acceso_nombre"])
            self.assertTrue(mensaje.startswith(textos["windows.maquina_arrancando"]), mensaje)
            self.assertIn(textos["windows.abrir_registro"].format(str(registro)), mensaje)
            contenido = registro.read_text(encoding="utf-8")
            self.assertIn(textos["windows.maquina_esperando"], contenido)
            self.assertIn("error: " + textos["windows.maquina_arrancando"], contenido)
            self.assertIn("machine start: fallo con ***", contenido)
            for secreto in (OCULTA, "Cuenta-Inventada-77", "OtraClave1"):
                self.assertNotIn(secreto, contenido)

    def test_aviso_por_encima_y_sin_esperas_infinitas(self):
        codigo = (WINDOWS / "abrir.ps1").read_text(encoding="utf-8-sig")
        aviso = codigo[codigo.index("function Mostrar-Error"):codigo.index("\ntry {")]
        self.assertIn("$aviso.Popup($Mensaje, $segundos, $Titulo, 16 + 4096 + 65536)", aviso)
        self.assertIn("$segundos = 600", aviso)
        self.assertIn("[DateTime]::UtcNow -lt $limite", aviso)
        self.assertNotIn("Read-Host", aviso)
        self.assertLess(codigo.index("function Mostrar-Error"), codigo.index(". (Join-Path $PSScriptRoot 'comun.ps1')"))
        comun = (WINDOWS / "comun.ps1").read_text(encoding="utf-8-sig")
        elevado = comun[comun.index("function Ejecutar-Elevado"):comun.index("function Probar-Wsl")]
        self.assertNotIn("-Wait", elevado)
        self.assertIn("WaitForExit($Segundos * 1000)", elevado)

if __name__ == "__main__":
    unittest.main()
