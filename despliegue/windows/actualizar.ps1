param([ValidateSet('Podman', 'Rancher')][string]$Motor = 'Podman', [string]$DirectorioPrivado = '', [ValidateRange(1024,65535)][int]$Puerto = 18083, [switch]$SinNavegador)
. (Join-Path $PSScriptRoot 'comun.ps1')
try {
    Preparar-Directorio $DirectorioPrivado
    Comprobar-Requisitos $Motor -EncenderMaquina
    if (-not (Test-Path $script:Entorno)) { throw (Texto 'sin_instalacion') }
    Comprobar-Paquete
    [void](Componer -Argumentos @('stop', 'entrada-demo', 'app'))
    Arrancar
    [void](Componer -Argumentos @('restart', 'entrada-demo'))
    Write-Host (Texto 'actualizado')
} catch { Write-Host $_.Exception.Message; exit 1 }
