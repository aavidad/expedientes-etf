param([ValidateSet('Podman', 'Rancher')][string]$Motor = 'Podman', [string]$DirectorioPrivado = '', [ValidateRange(1024,65535)][int]$Puerto = 18083)
. (Join-Path $PSScriptRoot 'comun.ps1')
. (Join-Path $PSScriptRoot 'accesos.ps1')
try {
    Preparar-Directorio $DirectorioPrivado
    Comprobar-Requisitos $Motor -SoloMotor -EncenderMaquina
    # Una instalación que se cortó antes de arrancar puede haber dejado Podman
    # o su máquina anotados sin datos de la aplicación; también se pueden quitar.
    $hayDatos = Test-Path $script:Entorno
    if (-not $hayDatos -and -not (Test-Path (Join-Path $script:Privado 'instalado.json'))) { throw (Texto 'sin_instalacion') }
    $respuesta = Read-Host (Texto 'desinstalar_pregunta')
    if ($respuesta -cne (Texto 'desinstalar_palabra')) { Write-Host (Texto 'cancelado'); exit 0 }
    if ($hayDatos) { [void](Componer -Argumentos @('--profile', 'herramientas', 'down', '--volumes')) }
    Cerrar-NavegadorPropio
    $wslNuestro = [bool](Leer-Instalado).PSObject.Properties['wsl']
    if ($script:EsWindows -and $Motor -eq 'Podman') { Retirar-Requisitos }
    if ($script:EsWindows) { Quitar-Reanudacion }
    Eliminar-Accesos
    Eliminar-ArchivosInstalados
    Write-Host (Texto 'desinstalado')
    if ($wslNuestro) { Write-Host (Texto 'wsl_conservado') }
} catch { Write-Host $_.Exception.Message; exit 1 }
