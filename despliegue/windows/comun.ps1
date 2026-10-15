Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$script:Paquete = $PSScriptRoot
$script:Textos = Get-Content -LiteralPath (Join-Path $PSScriptRoot 'mensajes.json') -Raw -Encoding UTF8 | ConvertFrom-Json
$script:EsWindows = [Environment]::OSVersion.Platform -eq [PlatformID]::Win32NT

function Texto([string]$Clave) { return $script:Textos.('windows.' + $Clave) }

# Citas válidas para CreateProcess y ProcessStartInfo de PowerShell 5.1.
function Unir-Argumentos([string[]]$Argumentos) {
    return ($Argumentos | ForEach-Object {
        '"' + (($_ -replace '(\\*)"', '$1$1\"') -replace '(\\+)$', '$1$1') + '"'
    }) -join ' '
}

function Ejecutar-Motor {
    param([string[]]$Argumentos, [string]$Entrada = '')
    $inicio = New-Object System.Diagnostics.ProcessStartInfo
    $inicio.FileName = $script:Ejecutable
    $inicio.Arguments = Unir-Argumentos $Argumentos
    $inicio.UseShellExecute = $false
    $inicio.CreateNoWindow = $true
    $inicio.RedirectStandardInput = $true
    $inicio.RedirectStandardOutput = $true
    $inicio.RedirectStandardError = $true
    $inicio.StandardOutputEncoding = New-Object System.Text.UTF8Encoding($false)
    $inicio.StandardErrorEncoding = New-Object System.Text.UTF8Encoding($false)
    $proceso = New-Object System.Diagnostics.Process
    $proceso.StartInfo = $inicio
    try {
        [void]$proceso.Start()
        $salida = $proceso.StandardOutput.ReadToEndAsync()
        $errores = $proceso.StandardError.ReadToEndAsync()
        $proceso.StandardInput.Write($Entrada)
        $proceso.StandardInput.Close()
        if (-not $proceso.WaitForExit(1200000)) { $proceso.Kill(); throw (Texto 'tiempo') }
        $resultado = $salida.GetAwaiter().GetResult()
        [void]$errores.GetAwaiter().GetResult()
        if ($proceso.ExitCode -ne 0) { throw ((Texto 'motor_fallo') -f $Argumentos[0], $proceso.ExitCode) }
        return $resultado
    } finally { $proceso.Dispose() }
}

function Guardar-Privado([string]$Ruta, [string]$Contenido) {
    $temporal = $Ruta + '.nuevo'
    [IO.File]::WriteAllText($temporal, $Contenido, (New-Object Text.UTF8Encoding($false)))
    if (-not $script:EsWindows) { & chmod 600 -- $temporal; if ($LASTEXITCODE -ne 0) { throw (Texto 'privacidad') } }
    Move-Item -LiteralPath $temporal -Destination $Ruta -Force
}

function Preparar-Directorio([string]$DirectorioPrivado) {
    if (-not $DirectorioPrivado) {
        if ($script:EsWindows) { $DirectorioPrivado = Join-Path $env:LOCALAPPDATA 'Expedientes-ETF\pruebas' }
        else { $DirectorioPrivado = Join-Path ([Environment]::GetFolderPath('UserProfile')) '.local/state/expedientes-etf/pruebas' }
    }
    $script:Privado = [IO.Path]::GetFullPath($DirectorioPrivado)
    if ($script:Privado.StartsWith($script:Paquete, [StringComparison]::OrdinalIgnoreCase)) { throw (Texto 'fuera_paquete') }
    $carpeta = New-Object IO.DirectoryInfo($script:Privado)
    while ($null -ne $carpeta) {
        if (Test-Path (Join-Path $carpeta.FullName '.git')) { throw (Texto 'fuera_paquete') }
        $carpeta = $carpeta.Parent
    }
    $marca = Join-Path $script:Privado 'instalacion.json'
    if (Test-Path $script:Privado) {
        if ((Get-Item -LiteralPath $script:Privado).Attributes -band [IO.FileAttributes]::ReparsePoint) { throw (Texto 'directorio_ajeno') }
        if (Test-Path $marca) {
            $identidadInstalacion = Get-Content -LiteralPath $marca -Raw -Encoding UTF8 | ConvertFrom-Json
            if ($identidadInstalacion.proyecto -ne 'etf-windows') { throw (Texto 'directorio_ajeno') }
        } else {
            $anteriores = @('entorno.env', 'entorno.env.nuevo', 'usuarios.json', 'usuarios.json.nuevo', 'USUARIOS.txt', 'USUARIOS.txt.nuevo')
            foreach ($entrada in @(Get-ChildItem -LiteralPath $script:Privado -Force)) {
                if ($entrada.PSIsContainer -or $entrada.Name -notin $anteriores) { throw (Texto 'directorio_ajeno') }
            }
        }
    }
    [void](New-Item -ItemType Directory -Path $script:Privado -Force)
    if ($script:EsWindows) {
        foreach ($nombre in @('CONTAINER_HOST', 'CONTAINER_CONNECTION', 'DOCKER_HOST', 'DOCKER_CONTEXT')) {
            if ([Environment]::GetEnvironmentVariable($nombre)) { throw (Texto 'motor_local') }
        }
        $directorio = New-Object IO.DirectoryInfo($script:Privado)
        if ($PSVersionTable.PSEdition -eq 'Desktop') {
            $acl = $directorio.GetAccessControl([Security.AccessControl.AccessControlSections]::Access)
        } else {
            $acl = [IO.FileSystemAclExtensions]::GetAccessControl($directorio, [Security.AccessControl.AccessControlSections]::Access)
        }
        $acl.SetAccessRuleProtection($true, $false)
        foreach ($entrada in @($acl.GetAccessRules($true, $false, [Security.Principal.SecurityIdentifier]))) {
            $acl.RemoveAccessRuleSpecific($entrada)
        }
        $identidad = [Security.Principal.WindowsIdentity]::GetCurrent().User
        $regla = New-Object Security.AccessControl.FileSystemAccessRule($identidad, 'FullControl', 'ContainerInherit,ObjectInherit', 'None', 'Allow')
        $acl.AddAccessRule($regla)
        # Sólo se modifica la lista de permisos de acceso. Set-Acl en Windows
        # PowerShell 5.1 intenta escribir también auditoría y pide privilegios.
        if ($PSVersionTable.PSEdition -eq 'Desktop') { $directorio.SetAccessControl($acl) }
        else { [IO.FileSystemAclExtensions]::SetAccessControl($directorio, $acl) }
    } else {
        & chmod 700 -- $script:Privado
        if ($LASTEXITCODE -ne 0) { throw (Texto 'privacidad') }
    }
    $script:Entorno = Join-Path $script:Privado 'entorno.env'
    if (-not (Test-Path $marca)) { Guardar-Privado $marca (@{ proyecto = 'etf-windows'; formato = 1 } | ConvertTo-Json) }
    $script:PuertoLocal = if (Test-Path $script:Entorno) {
        $guardado = [IO.File]::ReadAllText($script:Entorno)
        if ($guardado -match '(?m)^ETF_WINDOWS_PUERTO=(\d+)') { [int]$Matches[1] } else { 18083 }
    } else { $Puerto }
    $script:Url = 'http://127.0.0.1:' + $script:PuertoLocal
}

function Comprobar-Requisitos([string]$Motor, [switch]$SoloMotor, [switch]$EncenderMaquina) {
    if ($script:EsWindows -and -not $SoloMotor) {
        $sistema = Get-CimInstance Win32_OperatingSystem
        if ([int]$sistema.BuildNumber -lt 19041 -or -not [Environment]::Is64BitOperatingSystem) { throw (Texto 'windows') }
        $instalada = (Get-CimInstance Win32_PhysicalMemory | Measure-Object -Property Capacity -Sum).Sum
        if ($instalada -lt 8GB) { throw (Texto 'memoria') }
        if (-not (Get-Command wsl.exe -ErrorAction SilentlyContinue)) { throw (Texto 'wsl') }
        $wsl = ((& wsl.exe --list --verbose 2>&1) -join "`n") -replace "`0", ''
        if ($LASTEXITCODE -ne 0 -or $wsl -notmatch '\s2\s*(\r?\n|$)') { throw (Texto 'wsl') }
        if ($Motor -eq 'Rancher') {
            if (-not (Test-Path (Join-Path $env:ProgramFiles 'Rancher Desktop'))) { throw (Texto 'rancher') }
        }
        $unidad = New-Object IO.DriveInfo([IO.Path]::GetPathRoot($script:Privado))
        if ($unidad.AvailableFreeSpace -lt 15GB) { throw (Texto 'disco') }
    }
    $nombre = if ($Motor -eq 'Podman') { 'podman' } else { 'docker' }
    $comando = Get-Command $nombre -CommandType Application -ErrorAction SilentlyContinue
    if (-not $comando) { throw (Texto 'podman') }
    $script:Ejecutable = $comando.Source
    if ($Motor -eq 'Podman') {
        if ($script:EsWindows) {
            # Windows PowerShell 5.1 entrega una matriz JSON como un único
            # objeto de la canalización; se enumeran sus elementos explícitamente.
            $maquinas = @(Ejecutar-Motor -Argumentos @('machine', 'list', '--format', 'json') | ConvertFrom-Json | ForEach-Object { $_ })
            $conexiones = @(Ejecutar-Motor -Argumentos @('system', 'connection', 'list', '--format', 'json') | ConvertFrom-Json | ForEach-Object { $_ })
            $activa = @($conexiones | Where-Object { $_.Default })
            if ($activa.Count -ne 1 -or $activa[0].URI -notmatch '^ssh://[^@]+@(127\.0\.0\.1|localhost|\[::1\]):') { throw (Texto 'motor_local') }
            $seleccionada = @($maquinas | Where-Object { $_.VMType -eq 'wsl' -and $activa[0].Name -in @($_.Name, ($_.Name + '-root')) })
            if ($seleccionada.Count -ne 1) { throw (Texto 'maquina') }
            $script:Maquina = $seleccionada[0].Name
            if (-not $seleccionada[0].Running) {
                if (-not $EncenderMaquina) { throw (Texto 'maquina') }
                [void](Ejecutar-Motor -Argumentos @('machine', 'start', $script:Maquina))
            }
            Asegurar-TuberiaPodman $activa[0]
        }
        $info = Ejecutar-Motor -Argumentos @('info', '--format', 'json') | ConvertFrom-Json
        if ($info.host.security.rootless) { throw (Texto 'rootful') }
    } else {
        $contexto = (Ejecutar-Motor -Argumentos @('context', 'show')).Trim()
        if ($script:EsWindows -and $contexto -ne 'rancher-desktop') { throw (Texto 'rancher') }
    }
    [void](Ejecutar-Motor -Argumentos @('compose', 'version'))
    Write-Host (Texto 'requisitos_ok')
}

function Probar-Tuberia([string]$Ruta) { return [bool](Test-Path -Path $Ruta) }

function Esperar-Tuberia([string]$Ruta, [int]$Segundos) {
    for ($i = 0; $i -le $Segundos; $i++) {
        if (Probar-Tuberia $Ruta) { return $true }
        if ($i -lt $Segundos) { Start-Sleep -Seconds 1 }
    }
    return $false
}

function Lanzar-Proxy([string]$Ejecutable, [string[]]$Argumentos) {
    Start-Process -FilePath $Ejecutable -ArgumentList (Unir-Argumentos $Argumentos) -WindowStyle Hidden | Out-Null
}

function Buscar-ProxyPodman {
    $carpetas = @((Split-Path -Parent $script:Ejecutable))
    if ($env:ProgramFiles) { $carpetas += Join-Path $env:ProgramFiles 'RedHat\Podman' }
    if ($env:LOCALAPPDATA) { $carpetas += Join-Path $env:LOCALAPPDATA 'Programs\Podman' }
    foreach ($carpeta in $carpetas) {
        $ruta = Join-Path $carpeta 'win-sshproxy.exe'
        if (Test-Path -LiteralPath $ruta -PathType Leaf) { return $ruta }
    }
    return $null
}

# Tras reiniciar Windows, Podman puede encender la máquina WSL sin levantar
# win-sshproxy; sin su tubería Compose no llega al motor. Se reintenta una vez
# con parada y arranque y, si no basta, se lanza el proxy como lo hace Podman.
function Asegurar-TuberiaPodman($Conexion) {
    $tuberia = $null
    try {
        $inspeccion = @(Ejecutar-Motor -Argumentos @('machine', 'inspect', $script:Maquina) | ConvertFrom-Json | ForEach-Object { $_ })
        $tuberia = $inspeccion[0].ConnectionInfo.PodmanPipe.Path
    } catch { }
    if (-not $tuberia) {
        $nombre = if ($script:Maquina -like 'podman*') { $script:Maquina } else { 'podman-' + $script:Maquina }
        $tuberia = '\\.\pipe\' + $nombre
    }
    if ($tuberia -notmatch '^\\\\\.\\pipe\\[A-Za-z0-9._-]+$') { throw (Texto 'tuberia') }
    if (Esperar-Tuberia $tuberia 0) { return }
    Write-Host (Texto 'tuberia_reintento')
    try { [void](Ejecutar-Motor -Argumentos @('machine', 'stop', $script:Maquina)) } catch { }
    [void](Ejecutar-Motor -Argumentos @('machine', 'start', $script:Maquina))
    if (Esperar-Tuberia $tuberia 10) { return }
    $proxy = Buscar-ProxyPodman
    $identidad = if ($Conexion.PSObject.Properties['Identity']) { [string]$Conexion.Identity } else { '' }
    if ($proxy -and $identidad -and (Test-Path -LiteralPath $identidad -PathType Leaf)) {
        Write-Host (Texto 'tuberia_proxy')
        $datos = if ($env:XDG_DATA_HOME) { $env:XDG_DATA_HOME } else { Join-Path ([Environment]::GetFolderPath('UserProfile')) '.local\share' }
        $estado = Join-Path $datos ('containers\podman\machine\wsl\' + $script:Maquina)
        [void](New-Item -ItemType Directory -Path $estado -Force)
        $destino = 'npipe:////./pipe/' + $tuberia.Substring(9)
        Lanzar-Proxy $proxy @($script:Maquina, $estado, $destino, $Conexion.URI, $identidad)
        if (Esperar-Tuberia $tuberia 30) { return }
    }
    throw (Texto 'tuberia')
}

# El antivirus responde a PING antes de que el primer análisis real sea ágil.
# Se analiza un texto inocuo hasta que clamd lo revisa limpio (máximo 5 minutos),
# para que la primera subida no quede sin revisar.
function Esperar-Antivirus([int]$Intentos = 60, [int]$Pausa = 5) {
    Write-Host (Texto 'antivirus_espera')
    for ($i = 1; $i -le $Intentos; $i++) {
        try {
            $respuesta = Componer -Argumentos @('exec', '-T', 'clamav', 'timeout', '20', 'clamdscan', '--no-summary', '-') -Entrada 'etf'
            if ($respuesta -match 'stream: OK') { return }
        } catch { }
        if ($i -lt $Intentos) {
            if ($i % 6 -eq 0) { Write-Host (Texto 'antivirus_espera') }
            Start-Sleep -Seconds $Pausa
        }
    }
    throw (Texto 'antivirus')
}

function Componer {
    param([string[]]$Argumentos, [string]$Entrada = '')
    # Las variables heredadas no deben sustituir secretos del archivo privado.
    $anteriores = @{}
    foreach ($variable in @(Get-ChildItem Env: | Where-Object { $_.Name -match '^(ETF_|COMPOSE_)' })) {
        $anteriores[$variable.Name] = $variable.Value
        Remove-Item ('Env:' + $variable.Name)
    }
    try {
        Ejecutar-Motor -Argumentos (@('compose', '--project-name', 'etf-windows', '--env-file', $script:Entorno,
            '--file', (Join-Path $script:Paquete 'compose.json')) + $Argumentos) -Entrada $Entrada
    } finally {
        foreach ($nombre in $anteriores.Keys) { Set-Item ('Env:' + $nombre) $anteriores[$nombre] }
    }
}

function Generar-Secretos {
    $contenido = if (Test-Path $script:Entorno) { [IO.File]::ReadAllText($script:Entorno) } else { '' }
    if ($contenido -notmatch '(?m)^ETF_WINDOWS_PUERTO=') { $contenido += 'ETF_WINDOWS_PUERTO=' + $script:PuertoLocal + "`n" };
    $azar = [Security.Cryptography.RandomNumberGenerator]::Create()
    try {
        foreach ($rol in @('INSTALADOR', 'MIGRADOR', 'APP', 'COPIA', 'CALCULO_MEMORIA')) {
            $clave = 'ETF_' + $rol + '_CONTRASENA'
            if ($contenido -notmatch ('(?m)^' + $clave + '=')) {
                $bytes = New-Object byte[] 32
                $azar.GetBytes($bytes)
                $contenido = $contenido.TrimEnd() + "`n" + $clave + '=' + [Convert]::ToBase64String($bytes) + "`n"
            }
        }
    } finally { $azar.Dispose() }
    Guardar-Privado $script:Entorno $contenido.TrimStart()
}

function Comprobar-Paquete {
    if (-not (Test-Path (Join-Path $script:Paquete 'SUMAS.json'))) { throw (Texto 'suma') }
    $sumas = Get-Content -LiteralPath (Join-Path $script:Paquete 'SUMAS.json') -Raw -Encoding UTF8 | ConvertFrom-Json
    foreach ($entrada in $sumas.PSObject.Properties) {
        $ruta = [IO.Path]::GetFullPath((Join-Path $script:Paquete $entrada.Name))
        if (-not $ruta.StartsWith($script:Paquete + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) { throw (Texto 'suma') }
        if ((Get-FileHash -LiteralPath $ruta -Algorithm SHA256).Hash -ne $entrada.Value) { throw (Texto 'suma') }
    }
    Write-Host (Texto 'sumas_ok')
}

function Arrancar {
    Comprobar-Paquete
    Generar-Secretos
    # La aplicación no lee idiomas ni temas mientras se preparan (FX8); la
    # instalación, la apertura y la actualización pasan siempre por aquí.
    [void](Componer -Argumentos @('stop', 'entrada-demo', 'app'))
    Write-Host (Texto 'imagenes')
    [void](Ejecutar-Motor -Argumentos @('load', '--input', (Join-Path $script:Paquete 'imagenes.tar')))
    $usuarios = Componer -Argumentos @('run', '--rm', '--no-deps', '-T', 'herramientas', 'preparar') -Entrada ([IO.File]::ReadAllText($script:Entorno))
    $cuentas = $usuarios | ConvertFrom-Json
    Guardar-Privado (Join-Path $script:Privado 'usuarios.json') ($cuentas | ConvertTo-Json)
    Write-Host (Texto 'arranque')
    # FX2 se ejecuta explícitamente en cada actualización, sin depender de que
    # el proveedor Compose reutilice o vuelva a crear contenedores finalizados.
    [void](Componer -Argumentos @('up', '-d', 'postgres'))
    [void](Componer -Argumentos @('run', '--rm', 'preparar-roles'))
    [void](Componer -Argumentos @('run', '--rm', 'migrar'))
    [void](Componer -Argumentos @('up', '-d', '--wait', '--wait-timeout', '300', 'clamav', 'ldap', 'app', 'entrada-demo'))
    Esperar-Antivirus
    $limite = [DateTime]::UtcNow.AddMinutes(5)
    do {
        try { $salud = Invoke-WebRequest -Uri ($script:Url + '/salud') -UseBasicParsing -TimeoutSec 5; if ($salud.StatusCode -eq 200) { break } } catch { }
        if ([DateTime]::UtcNow -ge $limite) { throw (Texto 'salud') }
        Start-Sleep -Seconds 2
    } while ($true)
    [void](Componer -Argumentos @('run', '--rm', '--no-deps', '-T', 'herramientas', 'sembrar'))
    $filas = @((Texto 'usuarios_cabecera'))
    foreach ($cuenta in $cuentas.PSObject.Properties) { $filas += $cuenta.Name + '    ' + $cuenta.Value }
    Guardar-Privado (Join-Path $script:Privado 'USUARIOS.txt') ($filas -join "`r`n")
    $ubicacionUsuarios = if ($script:EsWindows) { '%LOCALAPPDATA%\Expedientes-ETF\pruebas\USUARIOS.txt' } else { Join-Path $script:Privado 'USUARIOS.txt' }
    Write-Host ((Texto 'listo') -f $script:Url, $ubicacionUsuarios)
    if ($script:EsWindows) {
        . (Join-Path $script:Paquete 'accesos.ps1')
        Crear-Accesos
    }
    if ($script:EsWindows -and -not $SinNavegador) {
        Abrir-Navegador
        Start-Process notepad.exe ('"' + (Join-Path $script:Privado 'USUARIOS.txt') + '"')
    }
}
