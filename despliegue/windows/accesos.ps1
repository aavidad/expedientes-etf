function Buscar-Navegador {
    $carpetas = @(${env:ProgramFiles(x86)}, $env:ProgramFiles, $env:LOCALAPPDATA) | Where-Object { $_ }
    foreach ($producto in @('Microsoft\Edge\Application\msedge.exe', 'Google\Chrome\Application\chrome.exe')) {
        foreach ($carpeta in $carpetas) {
            $ruta = Join-Path $carpeta $producto
            if (Test-Path -LiteralPath $ruta -PathType Leaf) { return $ruta }
        }
    }
    throw (Texto 'navegador')
}

function Abrir-Navegador {
    $navegador = Buscar-Navegador
    $direccion = $script:Url -replace '^http://127\.0\.0\.1:', 'http://localhost:'
    $perfil = Join-Path $script:Privado 'perfil-navegador'
    $argumentos = @('--app=' + $direccion)
    $argumentos += '--user-data-dir="' + $perfil + '"'
    $argumentos += @('--no-first-run', '--no-default-browser-check')
    Start-Process -FilePath $navegador -ArgumentList $argumentos
}

function Ruta-PaqueteInstalado { return Join-Path $script:Privado 'paquete' }

function Guardar-PaqueteInstalado {
    $destino = Ruta-PaqueteInstalado
    if ([IO.Path]::GetFullPath($script:Paquete).TrimEnd('\') -eq [IO.Path]::GetFullPath($destino).TrimEnd('\')) { return }
    $nuevo = $destino + '.nuevo'
    $anterior = $destino + '.anterior-' + [Guid]::NewGuid().ToString('N')
    if (Test-Path $nuevo) {
        if (Test-Path (Join-Path $nuevo 'SUMAS.json')) { Eliminar-PaqueteRegistrado $nuevo }
        elseif (-not @(Get-ChildItem -LiteralPath $nuevo -Force).Count) { Remove-Item -LiteralPath $nuevo -Force }
        if (Test-Path $nuevo) { throw (Texto 'directorio_ajeno') }
    }
    [void](New-Item -ItemType Directory -Path $nuevo)
    try {
        $sumas = Get-Content -LiteralPath (Join-Path $script:Paquete 'SUMAS.json') -Raw -Encoding UTF8 | ConvertFrom-Json
        Guardar-Privado (Join-Path $nuevo 'SUMAS.json') ($sumas | ConvertTo-Json -Depth 4)
        foreach ($nombre in @($sumas.PSObject.Properties.Name)) {
            $entrada = [IO.Path]::GetFullPath((Join-Path $script:Paquete $nombre))
            $salida = [IO.Path]::GetFullPath((Join-Path $nuevo $nombre))
            if (-not $entrada.StartsWith($script:Paquete + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase) -or
                -not $salida.StartsWith($nuevo + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) { throw (Texto 'suma') }
            [void](New-Item -ItemType Directory -Path (Split-Path -Parent $salida) -Force)
            Copy-Item -LiteralPath $entrada -Destination $salida
        }
        if (Test-Path $destino) { Move-Item -LiteralPath $destino -Destination $anterior }
        try { Move-Item -LiteralPath $nuevo -Destination $destino }
        catch {
            if (Test-Path $anterior) { Move-Item -LiteralPath $anterior -Destination $destino }
            throw
        }
        if (Test-Path (Join-Path $anterior 'SUMAS.json')) { Eliminar-PaqueteRegistrado $anterior }
        elseif ((Test-Path $anterior) -and -not @(Get-ChildItem -LiteralPath $anterior -Force).Count) { Remove-Item -LiteralPath $anterior -Force }
    } finally {
        if (Test-Path $nuevo) { Remove-Item -LiteralPath $nuevo -Recurse -Force }
    }
}

function Comprobar-RutaSinEnlaces([string]$Ruta, [string]$Raiz) {
    $actual = [IO.Path]::GetFullPath($Ruta)
    while ($actual.StartsWith($Raiz, [StringComparison]::OrdinalIgnoreCase)) {
        if (Test-Path -LiteralPath $actual) {
            if ((Get-Item -LiteralPath $actual -Force).Attributes -band [IO.FileAttributes]::ReparsePoint) { throw (Texto 'directorio_ajeno') }
        }
        if ($actual -eq $Raiz) { break }
        $actual = Split-Path -Parent $actual
    }
}

function Eliminar-DirectoriosVacios([string]$Ruta) {
    foreach ($entrada in @(Get-ChildItem -LiteralPath $Ruta -Force)) {
        if ($entrada.PSIsContainer -and -not ($entrada.Attributes -band [IO.FileAttributes]::ReparsePoint)) {
            Eliminar-DirectoriosVacios $entrada.FullName
        }
    }
    if (-not @(Get-ChildItem -LiteralPath $Ruta -Force).Count) { Remove-Item -LiteralPath $Ruta -Force }
}

function Eliminar-PaqueteRegistrado([string]$Cache) {
    Comprobar-RutaSinEnlaces $Cache $Cache
    $sumas = Get-Content -LiteralPath (Join-Path $Cache 'SUMAS.json') -Raw -Encoding UTF8 | ConvertFrom-Json
    # Sólo se retiran los archivos registrados. Las notas o archivos añadidos
    # por una persona se conservan, también al actualizar la copia estable.
    foreach ($nombre in @($sumas.PSObject.Properties.Name) + @('SUMAS.json')) {
        $ruta = [IO.Path]::GetFullPath((Join-Path $Cache $nombre))
        if (-not $ruta.StartsWith($Cache + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) { throw (Texto 'suma') }
        Comprobar-RutaSinEnlaces $ruta $Cache
        if (Test-Path -LiteralPath $ruta -PathType Leaf) { Remove-Item -LiteralPath $ruta -Force }
    }
    Eliminar-DirectoriosVacios $Cache
}

function Rutas-Accesos {
    $nombre = (Texto 'acceso_nombre') + '.lnk'
    foreach ($tipo in @('DesktopDirectory', 'Programs')) {
        Join-Path ([Environment]::GetFolderPath($tipo)) $nombre
    }
}

# Los accesos abren un lanzador .exe y no PowerShell: los antivirus de
# comportamiento cortan los scripts lanzados desde un acceso directo.
function Ruta-Lanzador { return Join-Path (Ruta-PaqueteInstalado) 'Expedientes ETF.exe' }

function Es-AccesoPropio($Enlace) {
    if ($Enlace.TargetPath -eq (Ruta-Lanzador) -and -not $Enlace.Arguments) { return $true }
    # Accesos de versiones anteriores, que se sustituyen al actualizar.
    $programa = Join-Path (Ruta-PaqueteInstalado) 'abrir.ps1'
    $argumentos = '-NoLogo -NoProfile -ExecutionPolicy Bypass -File "' + $programa + '"'
    $anteriores = $argumentos.Replace(' -File ', ' -WindowStyle Hidden -File ')
    return $Enlace.TargetPath -eq (Join-Path $env:SystemRoot 'System32\WindowsPowerShell\v1.0\powershell.exe') -and
        $Enlace.Arguments -in @($argumentos, $anteriores)
}

function Eliminar-ArchivosInstalados {
    $cache = Ruta-PaqueteInstalado
    if (Test-Path $cache) {
        Eliminar-PaqueteRegistrado $cache
    }
    $perfil = Join-Path $script:Privado 'perfil-navegador'
    if (Test-Path $perfil) {
        if ((Get-Item $perfil).Attributes -band [IO.FileAttributes]::ReparsePoint) { throw (Texto 'directorio_ajeno') }
        $limite = [DateTime]::UtcNow.AddSeconds(20)
        do {
            try { Remove-Item -LiteralPath $perfil -Recurse -Force; break }
            catch {
                if ([DateTime]::UtcNow -ge $limite) { throw }
                Start-Sleep -Milliseconds 500
            }
        } while (Test-Path $perfil)
    }
    foreach ($nombre in @('inicio.json', 'entorno.env', 'usuarios.json', 'USUARIOS.txt', 'proxy.json', 'registro-abrir.log',
            'registro-abrir.log.anterior', 'instalado.json', 'wsl-pendiente.txt', 'podman-instalacion.log', 'herramientas\docker-compose.exe')) {
        $ruta = Join-Path $script:Privado $nombre
        if (Test-Path -LiteralPath $ruta -PathType Leaf) { Remove-Item -LiteralPath $ruta -Force }
    }
    $herramientas = Join-Path $script:Privado 'herramientas'
    if ((Test-Path -LiteralPath $herramientas -PathType Container) -and -not @(Get-ChildItem -LiteralPath $herramientas -Force).Count) {
        Remove-Item -LiteralPath $herramientas -Force
    }
    # Si quedan notas añadidas, la marca permite reinstalar en esta misma carpeta.
    $restantes = @(Get-ChildItem -LiteralPath $script:Privado -Force | Where-Object { $_.Name -ne 'instalacion.json' })
    if (-not $restantes.Count) {
        Remove-Item -LiteralPath (Join-Path $script:Privado 'instalacion.json') -Force
        Remove-Item -LiteralPath $script:Privado -Force
    }
}

function Crear-Accesos {
    if (-not $script:EsWindows) { return }
    [void](Buscar-Navegador)
    Guardar-PaqueteInstalado
    Guardar-Privado (Join-Path $script:Privado 'inicio.json') (@{ motor = $Motor } | ConvertTo-Json)
    $carpeta = Ruta-PaqueteInstalado
    $shell = New-Object -ComObject WScript.Shell
    foreach ($ruta in @(Rutas-Accesos)) {
        $enlace = $shell.CreateShortcut($ruta)
        if ((Test-Path $ruta) -and -not (Es-AccesoPropio $enlace)) { throw (Texto 'acceso_ocupado') }
        $enlace.TargetPath = Ruta-Lanzador
        $enlace.Arguments = ''
        $enlace.WorkingDirectory = $carpeta
        $enlace.IconLocation = (Join-Path $carpeta 'aplicacion.ico') + ',0'
        $enlace.Description = Texto 'acceso_descripcion'
        $enlace.Save()
    }
    Write-Host (Texto 'accesos_listos')
}

function Eliminar-Accesos {
    if (-not $script:EsWindows) { return }
    $shell = New-Object -ComObject WScript.Shell
    foreach ($ruta in @(Rutas-Accesos)) {
        if ((Test-Path $ruta) -and (Es-AccesoPropio ($shell.CreateShortcut($ruta)))) {
            Remove-Item -LiteralPath $ruta -Force
        }
    }
}

function Cerrar-NavegadorPropio {
    $perfil = [regex]::Escape((Join-Path $script:Privado 'perfil-navegador'))
    $procesos = @(Get-CimInstance Win32_Process -Filter "Name = 'msedge.exe' OR Name = 'chrome.exe'")
    $propios = @{}
    foreach ($proceso in $procesos) {
        if ($proceso.CommandLine -and $proceso.CommandLine -match ('--user-data-dir="?' + $perfil + '"?(\s|$)')) {
            $propios[[int]$proceso.ProcessId] = $true
        }
    }
    # Los auxiliares de Edge y Chrome pueden omitir el perfil en sus argumentos.
    # Se incluyen sólo descendientes de procesos que sí declaran nuestro perfil.
    do {
        $nuevos = 0
        foreach ($proceso in $procesos) {
            if ($proceso.CommandLine -and $proceso.CommandLine -notmatch '--user-data-dir(?:=|\s)' -and
                $propios.ContainsKey([int]$proceso.ParentProcessId) -and -not $propios.ContainsKey([int]$proceso.ProcessId)) {
                $propios[[int]$proceso.ProcessId] = $true
                $nuevos++
            }
        }
    } while ($nuevos)
    foreach ($id in @($propios.Keys)) { Stop-Process -Id $id -Force -ErrorAction SilentlyContinue }
    foreach ($id in @($propios.Keys)) { Wait-Process -Id $id -Timeout 10 -ErrorAction SilentlyContinue }
}
