. (Join-Path $PSScriptRoot 'comun.ps1')
. (Join-Path $PSScriptRoot 'accesos.ps1')
try {
    $Puerto = 18083
    Preparar-Directorio (Split-Path -Parent $PSScriptRoot)
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
} catch {
    Add-Type -AssemblyName PresentationFramework
    [void][System.Windows.MessageBox]::Show($_.Exception.Message, (Texto 'acceso_nombre'))
    exit 1
}
