param([ValidateSet('Podman', 'Rancher')][string]$Motor = 'Podman', [string]$DirectorioPrivado = '', [ValidateRange(1024,65535)][int]$Puerto = 18083)
. (Join-Path $PSScriptRoot 'comun.ps1')
. (Join-Path $PSScriptRoot 'accesos.ps1')
try {
    Preparar-Directorio $DirectorioPrivado
    Comprobar-Requisitos $Motor -SoloMotor
    if (-not (Test-Path $script:Entorno)) { throw (Texto 'sin_instalacion') }
    $respuesta = Read-Host (Texto 'desinstalar_pregunta')
    if ($respuesta -cne (Texto 'desinstalar_palabra')) { Write-Host (Texto 'cancelado'); exit 0 }
    [void](Componer -Argumentos @('--profile', 'herramientas', 'down', '--volumes'))
    Cerrar-NavegadorPropio
    Eliminar-Accesos
    Eliminar-ArchivosInstalados
    Write-Host (Texto 'desinstalado')
} catch { Write-Host $_.Exception.Message; exit 1 }
