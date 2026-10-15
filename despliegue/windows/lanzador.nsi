; Lanzador «Expedientes ETF.exe» de los accesos del escritorio e Inicio.
; Los antivirus de comportamiento (y los EDR corporativos) cortan PowerShell o
; cmd lanzados directamente desde un acceso directo .lnk; por eso el acceso
; abre este programa y es él quien ejecuta abrir.ps1 de su misma carpeta.
; Lo compila paquete.py con makensis 3.10:
;   makensis /DICONO=... /DSALIDA=... /DACCESO_NOMBRE=... /DACCESO_DESCRIPCION=... lanzador.nsi
; No muestra ventana propia: la consola de abrir.ps1 enseña el progreso como
; antes y sus mensajes son los de abrir.ps1 (mensajes.json).
;
; Usos:
;   Expedientes ETF.exe                  abre la aplicación (abrir.ps1).
;   Expedientes ETF.exe /accesos crear   crea o migra los accesos (accesos.nsh).
;   Expedientes ETF.exe /accesos quitar  borra los accesos propios.
; Los scripts sólo lo ejecutan y leen su código de salida: 0 bien, 1 fallo,
; 2 hay un acceso con ese nombre de otra instalación. Ningún script toca un .lnk.

Unicode true
Target amd64-unicode
ManifestDPIAware true
RequestExecutionLevel user
SilentInstall silent
SetCompressor zlib
Icon "${ICONO}"
OutFile "${SALIDA}"
Name "Expedientes ETF"
BrandingText " "

!include LogicLib.nsh
!include FileFunc.nsh
!include "${__FILEDIR__}/accesos.nsh"
!insertmacro ACCESOS_LECTURA ""
!insertmacro ACCESOS_CREAR ""
!insertmacro ACCESOS_QUITAR ""

Section
    SetShellVarContext current
    ${GetParameters} $0
    ${If} $0 == ""
        ; Ruta absoluta a la PowerShell de 64 bits de Windows: no depende del PATH.
        Exec '"$WINDIR\System32\WindowsPowerShell\v1.0\powershell.exe" -NoLogo -NoProfile -ExecutionPolicy Bypass -File "$EXEDIR\abrir.ps1"'
    ${ElseIf} $0 == "/accesos crear"
        StrCpy $R3 "$EXEDIR"
        Call ComprobarAccesos
        ${If} $R5 == 1
            SetErrorLevel 2
        ${Else}
            Call CrearAccesos
            ${If} ${Errors}
                SetErrorLevel 1
            ${Else}
                SetErrorLevel 0
            ${EndIf}
        ${EndIf}
    ${ElseIf} $0 == "/accesos quitar"
        StrCpy $R3 "$EXEDIR"
        ClearErrors
        Call QuitarAccesos
        ${If} ${Errors}
            SetErrorLevel 1
        ${Else}
            SetErrorLevel 0
        ${EndIf}
    ${Else}
        SetErrorLevel 1
    ${EndIf}
SectionEnd
