param([ValidateSet('Podman', 'Rancher')][string]$Motor = 'Podman', [string]$DirectorioPrivado = '', [ValidateRange(1024,65535)][int]$Puerto = 18083, [switch]$SinNavegador, [switch]$SoloComprobar, [switch]$EncenderMaquina)
. (Join-Path $PSScriptRoot 'comun.ps1')
try {
    Preparar-Directorio $DirectorioPrivado
    # Con Podman instala lo que falte; Rancher y Docker se respetan como estén.
    # 3010: Windows debe reiniciarse y la instalación sigue al volver a entrar.
    if ($Motor -eq 'Podman' -and -not $SoloComprobar -and (Instalar-Requisitos)) { exit 3010 }
    Comprobar-Requisitos $Motor -EncenderMaquina:($EncenderMaquina -or -not $SoloComprobar)
    if (-not $SoloComprobar) { Arrancar }
} catch { Write-Host $_.Exception.Message; exit 1 }
