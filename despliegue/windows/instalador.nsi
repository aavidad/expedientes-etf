; Instalador de Windows de Expedientes ETF (pruebas). Lo compila paquete.py
; con makensis 3.10 a partir de la misma carpeta que el ZIP:
;   makensis /DCARPETA=... /DSALIDA=... /DVERSION=... /DIDIOMAS=... /DARCHIVOS=...
;            /DACCESO_NOMBRE=... /DACCESO_DESCRIPCION=...
; Los textos viven en idiomas/<código>/textos/windows.json (claves
; windows.instalador_*); paquete.py los convierte en IDIOMAS.
;
; Se instala por usuario, sin permiso de administrador, directamente en la
; copia estable que usan los accesos (comun.ps1, accesos.ps1), para no
; duplicar las imágenes en disco.
;
; Los accesos .lnk los gestiona sólo código NSIS (accesos.nsh): este asistente,
; su desinstalador y el lanzador «Expedientes ETF.exe». Ningún cmd.exe ni
; powershell.exe abre, crea ni borra accesos directos, porque algunos
; antivirus de comportamiento matan esa secuencia.

Unicode true
Target amd64-unicode
ManifestDPIAware true
RequestExecutionLevel user
SetCompressor /SOLID lzma
SetCompressorDictSize 64

!include MUI2.nsh
!include LogicLib.nsh
!include FileFunc.nsh
!include "${__FILEDIR__}/accesos.nsh"

!define CLAVE_DESINSTALAR "Software\Microsoft\Windows\CurrentVersion\Uninstall\Expedientes-ETF"
!define DESINSTALADOR "desinstalar-expedientes-etf.exe"

Name "$(instalador_nombre)"
OutFile "${SALIDA}"
InstallDir "$LOCALAPPDATA\Expedientes-ETF\pruebas\paquete"
ShowInstDetails show
ShowUninstDetails show
BrandingText " "
!ifdef VERSION_NUMERICA
VIProductVersion "${VERSION_NUMERICA}"
VIAddVersionKey /LANG=0 "ProductName" "Expedientes ETF"
VIAddVersionKey /LANG=0 "ProductVersion" "${VERSION}"
VIAddVersionKey /LANG=0 "FileVersion" "${VERSION}"
VIAddVersionKey /LANG=0 "FileDescription" "Expedientes ETF"
VIAddVersionKey /LANG=0 "LegalCopyright" "Alberto Avidad"
!endif

!define MUI_ICON "${CARPETA}\aplicacion.ico"
!define MUI_UNICON "${CARPETA}\aplicacion.ico"
!define MUI_ABORTWARNING

!define MUI_WELCOMEPAGE_TITLE "$(instalador_bienvenida_titulo)"
!define MUI_WELCOMEPAGE_TEXT "$(instalador_bienvenida_texto)"
!insertmacro MUI_PAGE_WELCOME
!ifdef LICENCIA
!insertmacro MUI_PAGE_LICENSE "${LICENCIA}"
!endif
!insertmacro MUI_PAGE_INSTFILES
!define MUI_FINISHPAGE_TITLE "$(instalador_fin_titulo)"
!define MUI_FINISHPAGE_TEXT "$(instalador_fin_texto)"
!insertmacro MUI_PAGE_FINISH

!define MUI_UNCONFIRMPAGE_TEXT_TOP "$(instalador_desinstalar_aviso)"
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "Spanish"
!insertmacro MUI_LANGUAGE "English"
!include "${IDIOMAS}"

Var Privado
Var Accion

; La carpeta es fija aunque se pase /D=: los scripts y los accesos la esperan ahí.
Function .onInit
    SetShellVarContext current
    StrCpy $INSTDIR "$LOCALAPPDATA\Expedientes-ETF\pruebas\paquete"
FunctionEnd

; Devuelve en $0 el número de entradas de $Privado distintas de la marca.
!macro CONTAR_ENTRADAS PREFIJO
Function ${PREFIJO}ContarEntradas
    Push $1
    Push $2
    StrCpy $0 0
    ClearErrors
    FindFirst $1 $2 "$Privado\*"
    ${DoUntil} ${Errors}
        ${If} $2 != "."
        ${AndIf} $2 != ".."
        ${AndIf} $2 != "instalacion.json"
            IntOp $0 $0 + 1
        ${EndIf}
        ClearErrors
        FindNext $1 $2
    ${Loop}
    FindClose $1
    Pop $2
    Pop $1
FunctionEnd
!macroend
!insertmacro CONTAR_ENTRADAS ""
!insertmacro CONTAR_ENTRADAS "un."
!insertmacro ACCESOS_LECTURA ""
!insertmacro ACCESOS_CREAR ""
!insertmacro ACCESOS_LECTURA "un."
!insertmacro ACCESOS_QUITAR "un."

; Instalador de 64 bits: la consola, PowerShell, WSL y Podman son los nativos.
!define CONSOLA "$WINDIR\System32\cmd.exe"

Section
    SetShellVarContext current
    StrCpy $Privado "$LOCALAPPDATA\Expedientes-ETF\pruebas"
    ; Con datos ya creados, el ejecutable actúa como actualización.
    ${If} ${FileExists} "$Privado\entorno.env"
        StrCpy $Accion "actualizar"
    ${Else}
        StrCpy $Accion "instalar"
    ${EndIf}
    ; comun.ps1 sólo admite la carpeta privada si lleva su marca o está vacía.
    ${IfNot} ${FileExists} "$Privado\instalacion.json"
        Call ContarEntradas
        ${If} $0 != 0
        ${AndIfNot} ${FileExists} "$Privado\entorno.env"
            MessageBox MB_ICONSTOP|MB_OK "$(instalador_ajeno)" /SD IDOK
            Abort "$(instalador_ajeno)"
        ${EndIf}
        CreateDirectory "$Privado"
        ClearErrors
        FileOpen $1 "$Privado\instalacion.json" w
        FileWrite $1 '{"proyecto":"etf-windows","formato":1}'
        FileClose $1
        ${If} ${Errors}
            Abort "$(instalador_ajeno)"
        ${EndIf}
    ${EndIf}
    ; Antes de copiar nada: un acceso con el mismo nombre de otra instalación
    ; se conserva y la instalación no sigue.
    StrCpy $R3 "$INSTDIR"
    Call ComprobarAccesos
    ${If} $R5 == 1
        MessageBox MB_ICONSTOP|MB_OK "$(instalador_acceso_ocupado)" /SD IDOK
        Abort "$(instalador_acceso_ocupado)"
    ${EndIf}

    SetOutPath "$INSTDIR"
    File /r "${CARPETA}\*"
    WriteUninstaller "$INSTDIR\${DESINSTALADOR}"

    WriteRegStr HKCU "${CLAVE_DESINSTALAR}" "DisplayName" "$(instalador_nombre)"
    WriteRegStr HKCU "${CLAVE_DESINSTALAR}" "DisplayVersion" "${VERSION}"
    WriteRegStr HKCU "${CLAVE_DESINSTALAR}" "Publisher" "$(instalador_editor)"
    WriteRegStr HKCU "${CLAVE_DESINSTALAR}" "DisplayIcon" "$INSTDIR\aplicacion.ico"
    WriteRegStr HKCU "${CLAVE_DESINSTALAR}" "InstallLocation" "$INSTDIR"
    WriteRegStr HKCU "${CLAVE_DESINSTALAR}" "UninstallString" '"$INSTDIR\${DESINSTALADOR}"'
    WriteRegDWORD HKCU "${CLAVE_DESINSTALAR}" "NoModify" 1
    WriteRegDWORD HKCU "${CLAVE_DESINSTALAR}" "NoRepair" 1
    ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
    IntFmt $0 "0x%08X" $0
    WriteRegDWORD HKCU "${CLAVE_DESINSTALAR}" "EstimatedSize" "$0"

    ; Al actualizar, los accesos (también los antiguos que abrían PowerShell)
    ; pasan ya al lanzador. En una instalación nueva los crea el lanzador al
    ; final de instalar.cmd, cuando la aplicación ya puede abrirse.
    ${If} $Accion == "actualizar"
        StrCpy $R3 "$INSTDIR"
        Call CrearAccesos
    ${EndIf}

    ; instalar.cmd o actualizar.cmd en una consola visible: instala WSL y
    ; Podman si faltan, crea los accesos y abre la aplicación.
    DetailPrint "$(instalador_consola)"
    Exec '"${CONSOLA}" /d /c ""$INSTDIR\$Accion.cmd""'
SectionEnd

Section "Uninstall"
    SetShellVarContext current
    StrCpy $Privado "$LOCALAPPDATA\Expedientes-ETF\pruebas"
    ; desinstalar.ps1 pregunta, quita lo instalado y borra los archivos
    ; registrados de esta carpeta, desinstalar.cmd incluido. Por eso se ejecuta
    ; lo mismo que desinstalar.cmd desde una orden en memoria y no desde el .cmd.
    ${If} ${FileExists} "$Privado\entorno.env"
    ${OrIf} ${FileExists} "$Privado\instalado.json"
        ExecWait '"${CONSOLA}" /d /c ""$WINDIR\System32\WindowsPowerShell\v1.0\powershell.exe" -NoLogo -NoProfile -ExecutionPolicy Bypass -File "$INSTDIR\desinstalar.ps1" & pause"'
        ${If} ${FileExists} "$Privado\entorno.env"
        ${OrIf} ${FileExists} "$Privado\instalado.json"
            MessageBox MB_ICONEXCLAMATION|MB_OK "$(instalador_desinstalar_cancelado)" /SD IDOK
            Abort "$(instalador_desinstalar_cancelado)"
        ${EndIf}
    ${EndIf}
    ; Sólo los archivos que copió este instalador; lo añadido a mano se conserva.
    !include "${ARCHIVOS}"
    Delete "$INSTDIR\${DESINSTALADOR}"
    RMDir "$INSTDIR"
    DeleteRegKey HKCU "${CLAVE_DESINSTALAR}"
    Call un.ContarEntradas
    ${If} $0 == 0
        Delete "$Privado\instalacion.json"
        RMDir "$Privado"
        RMDir "$LOCALAPPDATA\Expedientes-ETF"
    ${EndIf}
    ; Los accesos se borran al final, desde NSIS y después de PowerShell: ningún
    ; intérprete toca un .lnk (algunos antivirus matan esa secuencia).
    StrCpy $R3 "$INSTDIR"
    Call un.QuitarAccesos
SectionEnd
