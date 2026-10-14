param([ValidateSet('Podman', 'Rancher')][string]$Motor = 'Podman', [string]$DirectorioPrivado = '', [ValidateRange(1024,65535)][int]$Puerto = 18083, [switch]$SinNavegador, [switch]$SoloComprobar, [switch]$EncenderMaquina)
. (Join-Path $PSScriptRoot 'comun.ps1')
try {
    Preparar-Directorio $DirectorioPrivado
    Comprobar-Requisitos $Motor -EncenderMaquina:$EncenderMaquina
    if (-not $SoloComprobar) { Arrancar }
} catch { Write-Host $_.Exception.Message; exit 1 }
