param([ValidateSet('Podman', 'Rancher')][string]$Motor = 'Podman', [string]$DirectorioPrivado = '', [ValidateRange(1024,65535)][int]$Puerto = 18083)
. (Join-Path $PSScriptRoot 'comun.ps1')
try {
    Preparar-Directorio $DirectorioPrivado
    Comprobar-Requisitos $Motor -SoloMotor
    if (-not (Test-Path $script:Entorno)) { throw (Texto 'sin_instalacion') }
    [void](Componer -Argumentos @('down'))
    if ($script:EsWindows) {
        . (Join-Path $script:Paquete 'accesos.ps1')
        Cerrar-NavegadorPropio
    }
    Write-Host (Texto 'parado')
} catch { Write-Host $_.Exception.Message; exit 1 }
