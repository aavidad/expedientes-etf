param([ValidateSet('Podman', 'Rancher')][string]$Motor = 'Podman', [string]$DirectorioPrivado = '', [ValidateRange(1024,65535)][int]$Puerto = 18083, [switch]$SinNavegador)
. (Join-Path $PSScriptRoot 'comun.ps1')
try {
    Preparar-Directorio $DirectorioPrivado
    Comprobar-Requisitos $Motor -EncenderMaquina
    if (-not (Test-Path $script:Entorno)) { throw (Texto 'sin_instalacion') }
    Comprobar-Paquete
    $respuesta = Read-Host (Texto 'borrar_pregunta')
    if ($respuesta -cne (Texto 'borrar_palabra')) { Write-Host (Texto 'cancelado'); exit 0 }
    [void](Componer -Argumentos @('--profile', 'herramientas', 'down', '--volumes'))
    foreach ($nombre in @('usuarios.json', 'USUARIOS.txt', 'entorno.env')) {
        Remove-Item -LiteralPath (Join-Path $script:Privado $nombre) -Force -ErrorAction SilentlyContinue
    }
    Arrancar
} catch { Write-Host $_.Exception.Message; exit 1 }
