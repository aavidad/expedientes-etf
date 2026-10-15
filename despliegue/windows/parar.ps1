param([ValidateSet('Podman', 'Rancher')][string]$Motor = 'Podman', [string]$DirectorioPrivado = '', [ValidateRange(1024,65535)][int]$Puerto = 18083)
. (Join-Path $PSScriptRoot 'comun.ps1')
try {
    Preparar-Directorio $DirectorioPrivado
    # Con la máquina ya apagada la aplicación está parada: sólo se cierran
    # la ventana y el proxy propio.
    Comprobar-Requisitos $Motor -SoloMotor -PermitirApagada
    if (-not (Test-Path $script:Entorno)) { throw (Texto 'sin_instalacion') }
    if (-not $script:MaquinaApagada) { [void](Componer -Argumentos @('down')) }
    if ($script:EsWindows) {
        . (Join-Path $script:Paquete 'accesos.ps1')
        Cerrar-NavegadorPropio
        if ($script:MaquinaApagada) { Cerrar-ProxyPropio }
        elseif ($Motor -eq 'Podman') { Apagar-MaquinaPodman }
    }
    Write-Host (Texto 'parado')
} catch { Write-Host $_.Exception.Message; exit 1 }
