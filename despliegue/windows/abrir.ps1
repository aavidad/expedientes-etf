# Cualquier fallo se muestra en un aviso por encima de las demás ventanas
# (Podman Desktop puede abrirse a la vez al iniciar sesión) y queda en el
# registro de la carpeta privada, sin contraseñas. Ninguna espera es infinita:
# el aviso se cierra solo a los 10 minutos y, si no puede mostrarse, la consola
# espera una tecla durante ese mismo tiempo. Así nunca retiene el apagado.
function Mostrar-Error([string]$Mensaje, [string]$Titulo) {
    Microsoft.PowerShell.Utility\Write-Host $Mensaje
    $segundos = 600
    try {
        # 16 icono de error, 4096 por encima de todo, 65536 en primer plano.
        $aviso = New-Object -ComObject WScript.Shell
        [void]$aviso.Popup($Mensaje, $segundos, $Titulo, 16 + 4096 + 65536)
    } catch {
        $limite = [DateTime]::UtcNow.AddSeconds($segundos)
        try { while (-not [Console]::KeyAvailable -and [DateTime]::UtcNow -lt $limite) { Start-Sleep -Milliseconds 250 } } catch { }
    }
}

try {
    . (Join-Path $PSScriptRoot 'comun.ps1')
    . (Join-Path $PSScriptRoot 'accesos.ps1')
    function Write-Host { param([Parameter(Position = 0)]$Object) Microsoft.PowerShell.Utility\Write-Host $Object; Anotar-Registro ([string]$Object) }
    $Puerto = 18083
    Preparar-Directorio (Split-Path -Parent $PSScriptRoot)
    Iniciar-Registro 'abrir'
    Anotar-Registro ('abrir ' + $PSScriptRoot)
    $configuracion = Get-Content -LiteralPath (Join-Path $script:Privado 'inicio.json') -Raw -Encoding UTF8 | ConvertFrom-Json
    $Motor = $configuracion.motor
    $lista = $false
    try {
        $salud = Invoke-WebRequest -Uri ($script:Url + '/salud') -UseBasicParsing -TimeoutSec 3
        $lista = $salud.StatusCode -eq 200
    } catch { }
    if (-not $lista) {
        # La copia estable permite volver a abrir después de parar o de borrar
        # la carpeta extraída originalmente, sin volver a pedir las cuentas.
        $SinNavegador = $true
        Comprobar-Requisitos $Motor -EncenderMaquina
        Arrancar
    }
    Abrir-Navegador
    Anotar-Registro 'ok'
} catch {
    $mensaje = $_.Exception.Message
    $titulo = ''
    if (Get-Command Anotar-Registro -ErrorAction SilentlyContinue) {
        Anotar-Registro ('error: ' + $mensaje)
        Anotar-Registro ([string]$_.InvocationInfo.PositionMessage)
        if ($script:ErrorMotor) { Anotar-Registro $script:ErrorMotor }
        $titulo = Texto 'acceso_nombre'
        if ($script:Registro -and (Test-Path -LiteralPath $script:Registro)) { $mensaje += "`r`n`r`n" + ((Texto 'abrir_registro') -f $script:Registro) }
    }
    Mostrar-Error $mensaje $titulo
    exit 1
}
