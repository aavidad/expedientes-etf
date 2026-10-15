Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$script:Paquete = $PSScriptRoot
$script:Textos = Get-Content -LiteralPath (Join-Path $PSScriptRoot 'mensajes.json') -Raw -Encoding UTF8 | ConvertFrom-Json
$script:EsWindows = [Environment]::OSVersion.Platform -eq [PlatformID]::Win32NT
$script:Maquina = $null
$script:MaquinaApagada = $false
$script:Registro = $null
$script:ErrorMotor = $null

function Texto([string]$Clave) { return $script:Textos.('windows.' + $Clave) }

# Citas válidas para CreateProcess y ProcessStartInfo de PowerShell 5.1.
function Unir-Argumentos([string[]]$Argumentos) {
    return ($Argumentos | ForEach-Object {
        '"' + (($_ -replace '(\\*)"', '$1$1\"') -replace '(\\+)$', '$1$1') + '"'
    }) -join ' '
}

function Ejecutar-Motor {
    param([string[]]$Argumentos, [string]$Entrada = '', [int]$Segundos = 1200)
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
    $script:ErrorMotor = $null
    try {
        [void]$proceso.Start()
        $salida = $proceso.StandardOutput.ReadToEndAsync()
        $errores = $proceso.StandardError.ReadToEndAsync()
        $proceso.StandardInput.Write($Entrada)
        $proceso.StandardInput.Close()
        if (-not $proceso.WaitForExit($Segundos * 1000)) { $proceso.Kill(); throw (Texto 'tiempo') }
        $resultado = $salida.GetAwaiter().GetResult()
        $detalle = $errores.GetAwaiter().GetResult()
        if ($proceso.ExitCode -ne 0) {
            # Sólo para el registro de diagnóstico, que oculta las contraseñas.
            $script:ErrorMotor = ($Argumentos[0..([Math]::Min(2, $Argumentos.Count - 1))] -join ' ') + ': ' +
                $(if ($detalle.Length -gt 2000) { $detalle.Substring($detalle.Length - 2000) } else { $detalle })
            throw ((Texto 'motor_fallo') -f $Argumentos[0], $proceso.ExitCode)
        }
        return $resultado
    } finally { $proceso.Dispose() }
}

function Guardar-Privado([string]$Ruta, [string]$Contenido) {
    $temporal = $Ruta + '.nuevo'
    [IO.File]::WriteAllText($temporal, $Contenido, (New-Object Text.UTF8Encoding($false)))
    if (-not $script:EsWindows) { & chmod 600 -- $temporal; if ($LASTEXITCODE -ne 0) { throw (Texto 'privacidad') } }
    Move-Item -LiteralPath $temporal -Destination $Ruta -Force
}

# Registro de diagnóstico de la apertura en la carpeta privada. Antes de
# escribir se sustituyen las contraseñas de entorno.env y usuarios.json.
function Ocultar-Secretos([string]$Texto) {
    if (-not $Texto) { return $Texto }
    $secretos = @()
    try {
        if ($script:Entorno -and (Test-Path -LiteralPath $script:Entorno -PathType Leaf)) {
            foreach ($coincidencia in [regex]::Matches([IO.File]::ReadAllText($script:Entorno), '(?m)^ETF_[A-Z_]+_CONTRASENA=([^\r\n]+)')) {
                $secretos += $coincidencia.Groups[1].Value
            }
        }
        $usuarios = Join-Path $script:Privado 'usuarios.json'
        if (Test-Path -LiteralPath $usuarios -PathType Leaf) {
            foreach ($cuenta in (Get-Content -LiteralPath $usuarios -Raw -Encoding UTF8 | ConvertFrom-Json).PSObject.Properties) { $secretos += [string]$cuenta.Value }
        }
    } catch { return '***' }
    foreach ($secreto in @($secretos | Where-Object { $_.Length -ge 4 } | Sort-Object Length -Descending)) { $Texto = $Texto.Replace($secreto, '***') }
    return ($Texto -replace '(?i)((contrase.a|password|passwd|secret)[a-z_]*\s*[=:]\s*)\S+', '$1***')
}

function Iniciar-Registro([string]$Nombre) {
    $script:Registro = Join-Path $script:Privado ('registro-' + $Nombre + '.log')
    if ((Test-Path -LiteralPath $script:Registro -PathType Leaf) -and (Get-Item -LiteralPath $script:Registro).Length -gt 262144) {
        Move-Item -LiteralPath $script:Registro -Destination ($script:Registro + '.anterior') -Force
    }
}

function Anotar-Registro([string]$Texto) {
    if (-not $script:Registro) { return }
    try {
        $linea = [DateTime]::Now.ToString('yyyy-MM-dd HH:mm:ss') + '  ' + (Ocultar-Secretos $Texto) + [Environment]::NewLine
        [IO.File]::AppendAllText($script:Registro, $linea, (New-Object Text.UTF8Encoding($false)))
    } catch { }
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

function Comprobar-Equipo {
    $sistema = Get-CimInstance Win32_OperatingSystem
    if ([int]$sistema.BuildNumber -lt 19041 -or -not [Environment]::Is64BitOperatingSystem) { throw (Texto 'windows') }
    $instalada = (Get-CimInstance Win32_PhysicalMemory | Measure-Object -Property Capacity -Sum).Sum
    if ($instalada -lt 8GB) { throw (Texto 'memoria') }
    $unidad = New-Object IO.DriveInfo([IO.Path]::GetPathRoot($script:Privado))
    if ($unidad.AvailableFreeSpace -lt 15GB) { throw (Texto 'disco') }
}

# -PermitirApagada: con la máquina de Podman apagada no hay nada que hacer en
# ella; se anota en $script:MaquinaApagada en vez de dar error (parar.ps1).
function Comprobar-Requisitos([string]$Motor, [switch]$SoloMotor, [switch]$EncenderMaquina, [switch]$PermitirApagada) {
    if ($script:EsWindows -and -not $SoloMotor) {
        Comprobar-Equipo
        if (-not (Get-Command wsl.exe -ErrorAction SilentlyContinue)) { throw (Texto 'wsl') }
        $wsl = ((& wsl.exe --list --verbose 2>&1) -join "`n") -replace "`0", ''
        if ($LASTEXITCODE -ne 0 -or $wsl -notmatch '\s2\s*(\r?\n|$)') { throw (Texto 'wsl') }
        if ($Motor -eq 'Rancher') {
            if (-not (Test-Path (Join-Path $env:ProgramFiles 'Rancher Desktop'))) { throw (Texto 'rancher') }
        }
    }
    if ($Motor -eq 'Podman') { $script:Ejecutable = Buscar-Podman }
    else {
        $comando = Get-Command docker -CommandType Application -ErrorAction SilentlyContinue
        $script:Ejecutable = if ($comando) { $comando.Source } else { $null }
    }
    if (-not $script:Ejecutable) { throw (Texto 'podman') }
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
            $maquina = $seleccionada[0]
            if (Maquina-Arrancando $maquina) { $maquina = Esperar-Arranque }
            if (-not $maquina.Running) {
                if ($PermitirApagada) { $script:MaquinaApagada = $true; return }
                if (-not $EncenderMaquina) { throw (Texto 'maquina') }
                try { [void](Ejecutar-Motor -Argumentos @('machine', 'start', $script:Maquina)) }
                catch {
                    # Otro programa puede haber empezado a encenderla entre la consulta y la orden.
                    $fallo = $_
                    $maquina = Esperar-Arranque
                    if (-not $maquina.Running) { throw $fallo }
                }
            }
            Asegurar-TuberiaPodman $activa[0]
        }
        $info = Ejecutar-Motor -Argumentos @('info', '--format', 'json') | ConvertFrom-Json
        if ($info.host.security.rootless) { throw (Texto 'rootful') }
    } else {
        $contexto = (Ejecutar-Motor -Argumentos @('context', 'show')).Trim()
        if ($script:EsWindows -and $contexto -ne 'rancher-desktop') { throw (Texto 'rancher') }
    }
    if ($Motor -eq 'Podman') { Asegurar-Compose }
    else { [void](Ejecutar-Motor -Argumentos @('compose', 'version')) }
    Write-Host (Texto 'requisitos_ok')
}

# --- Requisitos que instalar.cmd instala si faltan (WREQ) -------------------
# WSL2 con permiso de administrador; Podman (MSI por usuario), su máquina WSL
# y Compose sin él. Los instaladores van en el paquete con SHA-256 fijado.
$script:ClaveReanudar = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\RunOnce'
$script:NombreReanudar = 'ExpedientesETF'

# La carpeta del sistema se pide a Windows, no a la variable SystemRoot, para
# que el programa que se eleva con UAC sea siempre el de Windows.
function Ruta-Sistema([string]$Relativa) {
    $sistema = [Environment]::GetFolderPath('System')
    if (-not $sistema) { $sistema = Join-Path $env:SystemRoot 'System32' }
    return Join-Path $sistema $Relativa
}

function Comprobar-Requisito([string]$Relativa) {
    $indice = Join-Path $script:Paquete 'SUMAS.json'
    $ruta = Join-Path $script:Paquete $Relativa
    if (-not (Test-Path -LiteralPath $indice) -or -not (Test-Path -LiteralPath $ruta -PathType Leaf)) { throw (Texto 'suma') }
    $esperada = (Get-Content -LiteralPath $indice -Raw -Encoding UTF8 | ConvertFrom-Json).PSObject.Properties[$Relativa]
    if (-not $esperada -or (Get-FileHash -LiteralPath $ruta -Algorithm SHA256).Hash -ne $esperada.Value) { throw (Texto 'suma') }
    return $ruta
}

# Sin ventana y con la entrada cerrada: el wsl.exe de Windows sin WSL puede
# quedarse esperando una tecla. Devuelve -1 si se agota el tiempo.
function Ejecutar-Programa([string]$Programa, [string]$Argumentos, [int]$Segundos = 1800) {
    $inicio = New-Object System.Diagnostics.ProcessStartInfo
    $inicio.FileName = $Programa
    $inicio.Arguments = $Argumentos
    $inicio.UseShellExecute = $false
    $inicio.CreateNoWindow = $true
    $inicio.RedirectStandardInput = $true
    $inicio.RedirectStandardOutput = $true
    $inicio.RedirectStandardError = $true
    $proceso = New-Object System.Diagnostics.Process
    $proceso.StartInfo = $inicio
    try {
        [void]$proceso.Start()
        $salida = $proceso.StandardOutput.ReadToEndAsync()
        $errores = $proceso.StandardError.ReadToEndAsync()
        $proceso.StandardInput.Close()
        if (-not $proceso.WaitForExit($Segundos * 1000)) { $proceso.Kill(); return -1 }
        [void]$salida.GetAwaiter().GetResult()
        [void]$errores.GetAwaiter().GetResult()
        return $proceso.ExitCode
    } finally { $proceso.Dispose() }
}

# Pide el permiso con UAC; si la persona lo rechaza, Start-Process falla.
# Espera como máximo 30 minutos; si se agota devuelve -1 y la instalación
# se detiene con un mensaje en vez de quedarse esperando.
function Ejecutar-Elevado([string]$Programa, [string]$Argumentos, [int]$Segundos = 1800) {
    try { $proceso = Start-Process -FilePath $Programa -ArgumentList $Argumentos -Verb RunAs -PassThru }
    catch { throw (Texto 'uac_rechazado') }
    if (-not $proceso.WaitForExit($Segundos * 1000)) { return -1 }
    return $proceso.ExitCode
}

function Probar-Wsl {
    $wsl = Ruta-Sistema 'wsl.exe'
    if (-not (Test-Path -LiteralPath $wsl -PathType Leaf)) { return $false }
    foreach ($argumento in @('--version', '--status')) {
        if ((Ejecutar-Programa $wsl $argumento 120) -ne 0) { return $false }
    }
    return $true
}

function Arranque-Sistema { return [string](Get-CimInstance Win32_OperatingSystem).LastBootUpTime.ToUniversalTime().ToString('o') }

function Orden-Reanudacion {
    $powershell = Ruta-Sistema 'WindowsPowerShell\v1.0\powershell.exe'
    return '"' + $powershell + '" -NoLogo -NoProfile -ExecutionPolicy Bypass -NoExit -File "' +
        (Join-Path $script:Paquete 'instalar.ps1') + '" -DirectorioPrivado "' + $script:Privado + '" -Puerto ' + $script:PuertoLocal
}

function Guardar-Reanudacion([string]$Orden) {
    if (-not (Test-Path $script:ClaveReanudar)) { [void](New-Item -Path $script:ClaveReanudar -Force) }
    Set-ItemProperty -Path $script:ClaveReanudar -Name $script:NombreReanudar -Value $Orden
}

function Quitar-Reanudacion {
    Remove-ItemProperty -Path $script:ClaveReanudar -Name $script:NombreReanudar -ErrorAction SilentlyContinue
}

# Devuelve $true si Windows tiene que reiniciarse antes de seguir. La marca
# guarda el arranque en que se instaló WSL: si tras reiniciar sigue sin estar
# listo, se da un error en vez de pedir reinicios sin fin.
function Asegurar-Wsl {
    $marca = Join-Path $script:Privado 'wsl-pendiente.txt'
    if (Probar-Wsl) {
        if (Test-Path -LiteralPath $marca) { Remove-Item -LiteralPath $marca -Force }
        Quitar-Reanudacion
        return $false
    }
    $arranque = Arranque-Sistema
    if (Test-Path -LiteralPath $marca) {
        if ([IO.File]::ReadAllText($marca).Trim() -ne $arranque) { throw (Texto 'wsl_sin_activar') }
    } else {
        $wsl = Ruta-Sistema 'wsl.exe'
        if (-not (Test-Path -LiteralPath $wsl -PathType Leaf)) { throw ((Texto 'wsl_fallo') -f 'wsl.exe') }
        Write-Host (Texto 'wsl_instalando')
        $codigo = Ejecutar-Elevado $wsl '--install --no-distribution'
        if ($codigo -ne 0 -and $codigo -ne 3010) { throw ((Texto 'wsl_fallo') -f $codigo) }
        Anotar-Instalado 'wsl' $true
        if (Probar-Wsl) { return $false }
        Guardar-Privado $marca $arranque
    }
    Guardar-Reanudacion (Orden-Reanudacion)
    Write-Host (Texto 'reiniciar')
    return $true
}

function Anadir-Ruta([string]$Carpeta) {
    $separador = [IO.Path]::PathSeparator
    if (@($env:PATH -split [regex]::Escape($separador)) -notcontains $Carpeta) { $env:PATH = $Carpeta + $separador + $env:PATH }
}

# Podman por usuario, por equipo o el antiguo de Red Hat; el primero hallado
# se añade al PATH de este proceso.
function Buscar-Podman {
    $comando = Get-Command podman -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($comando) { return $comando.Source }
    $carpetas = @()
    if ($env:LOCALAPPDATA) { $carpetas += Join-Path $env:LOCALAPPDATA 'Programs\Podman' }
    if ($env:ProgramFiles) { $carpetas += @((Join-Path $env:ProgramFiles 'Podman'), (Join-Path $env:ProgramFiles 'RedHat\Podman')) }
    foreach ($carpeta in $carpetas) {
        $ruta = Join-Path $carpeta 'podman.exe'
        if (Test-Path -LiteralPath $ruta -PathType Leaf) { Anadir-Ruta $carpeta; return $ruta }
    }
    return $null
}

function Asegurar-Podman {
    if (Buscar-Podman) { return }
    $msi = Comprobar-Requisito 'requisitos/podman-installer-windows-amd64.msi'
    $registro = Join-Path $script:Privado 'podman-instalacion.log'
    $producto = Leer-CodigoProducto $msi
    Write-Host (Texto 'podman_instalando')
    $codigo = Ejecutar-Programa (Ruta-Sistema 'msiexec.exe') (
        '/package "' + $msi + '" /quiet /norestart MSIINSTALLPERUSER=1 MACHINE_PROVIDER=wsl /l*v "' + $registro + '"')
    if (($codigo -ne 0 -and $codigo -ne 3010) -or -not (Buscar-Podman)) { throw ((Texto 'podman_fallo') -f $codigo, $registro) }
    Anotar-Instalado 'podman' $producto
}

# Código de producto del MSI incluido, para poder desinstalar exactamente
# lo que se instaló. Si no se puede leer, desinstalar remite a Configuración.
function Leer-CodigoProducto([string]$Msi) {
    $objetos = @()
    try {
        $instalador = New-Object -ComObject WindowsInstaller.Installer
        $objetos += $instalador
        $base = $instalador.GetType().InvokeMember('OpenDatabase', 'InvokeMethod', $null, $instalador, @($Msi, 0))
        $objetos += $base
        $vista = $base.GetType().InvokeMember('OpenView', 'InvokeMethod', $null, $base, @("SELECT Value FROM Property WHERE Property = 'ProductCode'"))
        $objetos += $vista
        [void]$vista.GetType().InvokeMember('Execute', 'InvokeMethod', $null, $vista, $null)
        $fila = $vista.GetType().InvokeMember('Fetch', 'InvokeMethod', $null, $vista, $null)
        $objetos += $fila
        $codigo = [string]$fila.GetType().InvokeMember('StringData', 'GetProperty', $null, $fila, @(1))
        [void]$vista.GetType().InvokeMember('Close', 'InvokeMethod', $null, $vista, $null)
        if ($codigo -match '^\{[0-9A-Fa-f-]{36}\}$') { return $codigo }
    } catch {
    } finally {
        [array]::Reverse($objetos)
        foreach ($objeto in $objetos) { try { [void][Runtime.InteropServices.Marshal]::ReleaseComObject($objeto) } catch { } }
    }
    return $null
}

# Lo que instala esta aplicación queda en instalado.json de la carpeta
# privada; desinstalar.cmd sólo ofrece quitar lo anotado ahí.
function Leer-Instalado {
    $ruta = Join-Path $script:Privado 'instalado.json'
    if (Test-Path -LiteralPath $ruta -PathType Leaf) {
        $datos = Get-Content -LiteralPath $ruta -Raw -Encoding UTF8 | ConvertFrom-Json
        if ($datos) { return $datos }
    }
    return [pscustomobject]@{}
}

function Anotar-Instalado([string]$Clave, $Valor) {
    $datos = Leer-Instalado
    $datos | Add-Member -NotePropertyName $Clave -NotePropertyValue $Valor -Force
    Guardar-Privado (Join-Path $script:Privado 'instalado.json') ($datos | ConvertTo-Json)
}

function Valor-Instalado($Datos, [string]$Clave) {
    if ($Datos.PSObject.Properties[$Clave]) { return $Datos.$Clave }
    return $null
}

function Asegurar-MaquinaPodman {
    $script:Ejecutable = Buscar-Podman
    $maquinas = @(Ejecutar-Motor -Argumentos @('machine', 'list', '--format', 'json') | ConvertFrom-Json | ForEach-Object { $_ })
    if (@($maquinas | Where-Object { $_.VMType -eq 'wsl' }).Count) { return }
    Write-Host (Texto 'maquina_creando')
    $proveedor = $env:CONTAINERS_MACHINE_PROVIDER
    $env:CONTAINERS_MACHINE_PROVIDER = 'wsl'
    try { [void](Ejecutar-Motor -Argumentos @('machine', 'init', '--rootful', '--cpus', '2', '--memory', '4096')) }
    finally { $env:CONTAINERS_MACHINE_PROVIDER = $proveedor }
    $creada = @(Ejecutar-Motor -Argumentos @('machine', 'list', '--format', 'json') | ConvertFrom-Json | ForEach-Object { $_ } | Where-Object { $_.VMType -eq 'wsl' })
    if ($creada.Count -eq 1) { Anotar-Instalado 'maquina' ([string]$creada[0].Name) }
    Write-Host (Texto 'maquina_encendiendo')
    [void](Ejecutar-Motor -Argumentos @('machine', 'start'))
}

# Si Podman no encuentra proveedor de Compose, usa el docker-compose del
# paquete, copiado tras comprobar su suma a la carpeta privada de la cuenta.
function Asegurar-Compose {
    try { [void](Ejecutar-Motor -Argumentos @('compose', 'version')); return } catch { }
    $copia = Join-Path $script:Privado 'herramientas\docker-compose.exe'
    if (Test-Path -LiteralPath (Join-Path $script:Paquete 'requisitos/docker-compose.exe') -PathType Leaf) {
        $incluido = Comprobar-Requisito 'requisitos/docker-compose.exe'
        if (-not (Test-Path -LiteralPath $copia -PathType Leaf) -or
            (Get-FileHash -LiteralPath $copia -Algorithm SHA256).Hash -ne (Get-FileHash -LiteralPath $incluido -Algorithm SHA256).Hash) {
            Write-Host (Texto 'compose_instalando')
            [void](New-Item -ItemType Directory -Path (Split-Path -Parent $copia) -Force)
            Copy-Item -LiteralPath $incluido -Destination ($copia + '.nuevo') -Force
            Move-Item -LiteralPath ($copia + '.nuevo') -Destination $copia -Force
        }
    }
    if (Test-Path -LiteralPath $copia -PathType Leaf) { $env:PODMAN_COMPOSE_PROVIDER = $copia }
    try { [void](Ejecutar-Motor -Argumentos @('compose', 'version')) } catch { throw (Texto 'compose') }
}

# Devuelve $true si hay que reiniciar Windows antes de seguir.
function Instalar-Requisitos {
    if (-not $script:EsWindows) { return $false }
    Write-Host (Texto 'requisitos_preparando')
    Comprobar-Equipo
    if (Asegurar-Wsl) { return $true }
    Asegurar-Podman
    Asegurar-MaquinaPodman
    return $false
}

function Probar-Tuberia([string]$Ruta) { return [bool](Test-Path -Path $Ruta) }

function Esperar-Tuberia([string]$Ruta, [int]$Segundos) {
    for ($i = 0; $i -le $Segundos; $i++) {
        if (Probar-Tuberia $Ruta) { return $true }
        if ($i -lt $Segundos) { Start-Sleep -Seconds 1 }
    }
    return $false
}

# Estado de la máquina elegida; $null si ya no aparece.
function Leer-Maquina {
    $maquinas = @(Ejecutar-Motor -Argumentos @('machine', 'list', '--format', 'json') | ConvertFrom-Json | ForEach-Object { $_ })
    return @($maquinas | Where-Object { $_.Name -eq $script:Maquina }) | Select-Object -First 1
}

function Maquina-Arrancando($Maquina) {
    return [bool]($Maquina -and $Maquina.PSObject.Properties['Starting'] -and $Maquina.Starting)
}

# Podman Desktop puede estar encendiendo la máquina justo al iniciar sesión.
# Se espera a que termine (hasta 3 minutos) en vez de pararla o arrancarla
# encima, que deja la máquina a medias y sin la tubería de Compose.
function Esperar-Arranque([int]$Segundos = 180, [int]$Pausa = 5) {
    $maquina = Leer-Maquina
    $vueltas = [Math]::Ceiling($Segundos / $Pausa)
    for ($i = 0; (Maquina-Arrancando $maquina) -and $i -lt $vueltas; $i++) {
        if ($i % 6 -eq 0) { Write-Host (Texto 'maquina_esperando') }
        Start-Sleep -Seconds $Pausa
        $maquina = Leer-Maquina
    }
    if (Maquina-Arrancando $maquina) { throw (Texto 'maquina_arrancando') }
    if (-not $maquina) { throw (Texto 'maquina') }
    return $maquina
}

# El proxy lanzado aquí se anota (identificador y hora de inicio) para que
# parar.ps1 lo cierre sin confundirlo con otro proceso que reutilice el número.
function Lanzar-Proxy([string]$Ejecutable, [string[]]$Argumentos) {
    Cerrar-ProxyPropio
    $proceso = Start-Process -FilePath $Ejecutable -ArgumentList (Unir-Argumentos $Argumentos) -WindowStyle Hidden -PassThru
    Guardar-Proxy $proceso
}

function Guardar-Proxy($Proceso) {
    Guardar-Privado (Join-Path $script:Privado 'proxy.json') (@{ id = [int]$Proceso.Id; inicio = [int64]$Proceso.StartTime.ToUniversalTime().Ticks } | ConvertTo-Json)
}

function Cerrar-ProxyPropio {
    $ruta = Join-Path $script:Privado 'proxy.json'
    if (-not (Test-Path -LiteralPath $ruta -PathType Leaf)) { return }
    try {
        $datos = Get-Content -LiteralPath $ruta -Raw -Encoding UTF8 | ConvertFrom-Json
        $proceso = Get-Process -Id ([int]$datos.id) -ErrorAction SilentlyContinue
        if ($proceso -and $proceso.ProcessName -eq 'win-sshproxy' -and
            [int64]$proceso.StartTime.ToUniversalTime().Ticks -eq [int64]$datos.inicio) {
            Stop-Process -Id $proceso.Id -Force -ErrorAction SilentlyContinue
            Wait-Process -Id $proceso.Id -Timeout 10 -ErrorAction SilentlyContinue
        }
    } catch { }
    finally { Remove-Item -LiteralPath $ruta -Force -ErrorAction SilentlyContinue }
}

# Con la máquina de Podman encendida, Windows 10 puede quedarse en
# «Reiniciando». Parar la apaga si ya no tiene contenedores en marcha, de esta
# ni de otras aplicaciones, y cierra el proxy que hayamos lanzado.
# Contenedores en marcha en las dos conexiones de la máquina (con y sin
# permisos de administrador); $null si ninguna responde.
function Contenedores-EnMarcha {
    $enMarcha = @()
    $respuestas = 0
    foreach ($conexion in @($script:Maquina, ($script:Maquina + '-root'))) {
        try {
            $enMarcha += @(([string](Ejecutar-Motor -Argumentos @('--connection', $conexion, 'ps', '--quiet'))) -split '\r?\n' | Where-Object { $_.Trim() })
            $respuestas++
        } catch { }
    }
    if (-not $respuestas) { return $null }
    return ,$enMarcha
}

function Apagar-MaquinaPodman {
    if (-not $script:Maquina) { return }
    # Si ninguna conexión responde no se sabe qué hay dentro y no se apaga.
    $enMarcha = Contenedores-EnMarcha
    if ($null -eq $enMarcha) { Write-Host (Texto 'maquina_no_apagada'); return }
    if ($enMarcha.Count) { Write-Host (Texto 'maquina_en_uso'); return }
    Write-Host (Texto 'maquina_apagando')
    try { [void](Ejecutar-Motor -Argumentos @('machine', 'stop', $script:Maquina) -Segundos 120) }
    catch { Write-Host (Texto 'maquina_no_apagada') }
    Cerrar-ProxyPropio
}

# desinstalar.cmd: ofrece quitar sólo la máquina y el Podman que anotó esta
# aplicación al instalarlos, y pregunta antes. Lo que ya había se conserva y
# WSL no se toca. Un fallo se avisa sin detener el resto de la desinstalación.
function Retirar-Requisitos {
    $instalado = Leer-Instalado
    $maquina = [string](Valor-Instalado $instalado 'maquina')
    $podmanNuestro = [bool]$instalado.PSObject.Properties['podman']
    $maquinas = @(Ejecutar-Motor -Argumentos @('machine', 'list', '--format', 'json') | ConvertFrom-Json | ForEach-Object { $_ })
    $maquinaNuestra = $maquina -and @($maquinas | Where-Object { $_.Name -eq $maquina }).Count -eq 1
    if (-not $maquinaNuestra -and -not $podmanNuestro) { Apagar-MaquinaPodman; return }
    $enMarcha = Contenedores-EnMarcha
    if ($null -eq $enMarcha -or $enMarcha.Count) { Write-Host (Texto 'requisitos_en_uso'); return }
    $pregunta = if ($podmanNuestro) { Texto 'quitar_podman_pregunta' } else { Texto 'quitar_maquina_pregunta' }
    if ((Read-Host $pregunta) -cne (Texto 'quitar_palabra')) {
        Write-Host (Texto 'requisitos_conservados')
        Apagar-MaquinaPodman
        return
    }
    if ($maquinaNuestra) {
        Write-Host (Texto 'maquina_quitando')
        try { [void](Ejecutar-Motor -Argumentos @('machine', 'rm', '--force', $maquina) -Segundos 300) }
        catch { Write-Host (Texto 'maquina_quitar_fallo'); return }
    }
    Cerrar-ProxyPropio
    if (-not $podmanNuestro) { return }
    $producto = [string](Valor-Instalado $instalado 'podman')
    if ($producto -notmatch '^\{[0-9A-Fa-f-]{36}\}$') { Write-Host (Texto 'podman_quitar_manual'); return }
    $registro = Join-Path $script:Privado 'podman-desinstalacion.log'
    Write-Host (Texto 'podman_quitando')
    $codigo = Ejecutar-Programa (Ruta-Sistema 'msiexec.exe') ('/x ' + $producto + ' /quiet /norestart /l*v "' + $registro + '"') 900
    if ($codigo -notin @(0, 1605, 3010)) { Write-Host ((Texto 'podman_quitar_fallo') -f $codigo, $registro); return }
    Remove-Item -LiteralPath $registro -Force -ErrorAction SilentlyContinue
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
    # Podman puede abrir la tubería unos segundos después de encender la
    # máquina; antes de reiniciarla se espera y se comprueba que no se esté
    # encendiendo desde otro programa.
    if (Esperar-Tuberia $tuberia 15) { return }
    [void](Esperar-Arranque)
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
