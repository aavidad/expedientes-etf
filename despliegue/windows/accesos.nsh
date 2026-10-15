; Accesos «Expedientes ETF» del escritorio y de Inicio. Sólo los crea, migra y
; borra código NSIS (el lanzador «Expedientes ETF.exe» y el instalador .exe):
; ningún cmd.exe ni powershell.exe abre, crea ni borra un .lnk. Algunos
; antivirus de comportamiento (Fibratus, EDR corporativos) matan la secuencia
; «un intérprete abre un .lnk, crea un ejecutable o script en AppData y lanza
; otro intérprete», y la instalación o la desinstalación quedaban a medias.
;
; Lo incluyen lanzador.nsi e instalador.nsi después de LogicLib. Quien compila
; define ACCESO_NOMBRE y ACCESO_DESCRIPCION desde idiomas/es/textos/windows.json
; (los mismos textos que mensajes.json de los scripts).
;
; Convenio de registros: $R3 carpeta del paquete (la del lanzador); $R0 acceso
; que se examina; $R1 y $R2 su destino y sus argumentos; $R4 = 1 si es propio;
; $R5 = 1 si algún acceso pertenece a otra instalación.

!ifndef ACCESO_NOMBRE
    !error "Falta ACCESO_NOMBRE"
!endif
!ifndef ACCESO_DESCRIPCION
    !error "Falta ACCESO_DESCRIPCION"
!endif

!define LANZADOR_EXE "Expedientes ETF.exe"
!define CLSID_ENLACE "{00021401-0000-0000-C000-000000000046}"
!define IID_IENLACE "{000214F9-0000-0000-C000-000000000046}"
!define IID_IFICHERO "{0000010B-0000-0000-C000-000000000046}"
; Accesos de versiones anteriores a 1.0.4: PowerShell con abrir.ps1.
!define POWERSHELL_ANTERIOR "$WINDIR\System32\WindowsPowerShell\v1.0\powershell.exe"
!define ARGUMENTOS_ANTERIORES "-NoLogo -NoProfile -ExecutionPolicy Bypass"

; ACCESOS_LECTURA va siempre; ACCESOS_CREAR y ACCESOS_QUITAR según el uso,
; para no compilar funciones sin llamar. PREFIJO es "" o "un.".
!macro ACCESOS_LECTURA PREFIJO
; Lee con IShellLinkW el destino ($R1) y los argumentos ($R2) del acceso $R0.
Function ${PREFIJO}LeerAcceso
    Push $0
    Push $1
    Push $2
    StrCpy $R1 ""
    StrCpy $R2 ""
    System::Call "ole32::CoInitializeEx(p 0, i 2) i"
    System::Call "ole32::CoCreateInstance(g '${CLSID_ENLACE}', p 0, i 1, g '${IID_IENLACE}', *p .r1) i .r0"
    ${If} $0 = 0
        ; IUnknown::QueryInterface(IPersistFile)
        System::Call "$1->0(g '${IID_IFICHERO}', *p .r2) i .r0"
        ${If} $0 = 0
            ; IPersistFile::Load(ruta, STGM_READ)
            System::Call "$2->5(w R0, i 0) i .r0"
            ${If} $0 = 0
                ; IShellLinkW::GetPath y IShellLinkW::GetArguments
                System::Call "$1->3(w .R1, i ${NSIS_MAX_STRLEN}, p 0, i 0) i"
                System::Call "$1->10(w .R2, i ${NSIS_MAX_STRLEN}) i"
            ${EndIf}
            System::Call "$2->2()"
        ${EndIf}
        System::Call "$1->2()"
    ${EndIf}
    Pop $2
    Pop $1
    Pop $0
FunctionEnd

; $R4 = 1 si el acceso $R0 es de esta instalación: el lanzador de $R3 sin
; argumentos o, de versiones anteriores, PowerShell con abrir.ps1 de $R3.
Function ${PREFIJO}EsAccesoPropio
    Call ${PREFIJO}LeerAcceso
    StrCpy $R4 0
    ${If} $R1 == "$R3\${LANZADOR_EXE}"
        ${If} $R2 == ""
            StrCpy $R4 1
        ${EndIf}
    ${ElseIf} $R1 == "${POWERSHELL_ANTERIOR}"
        ${If} $R2 == '${ARGUMENTOS_ANTERIORES} -File "$R3\abrir.ps1"'
        ${OrIf} $R2 == '${ARGUMENTOS_ANTERIORES} -WindowStyle Hidden -File "$R3\abrir.ps1"'
            StrCpy $R4 1
        ${EndIf}
    ${EndIf}
FunctionEnd

!macroend

!macro ACCESOS_CREAR PREFIJO
; $R5 = 1 si alguno de los dos accesos existe y es de otra instalación.
Function ${PREFIJO}ComprobarAccesos
    StrCpy $R5 0
    ${If} ${FileExists} "$DESKTOP\${ACCESO_NOMBRE}.lnk"
        StrCpy $R0 "$DESKTOP\${ACCESO_NOMBRE}.lnk"
        Call ${PREFIJO}EsAccesoPropio
        ${If} $R4 != 1
            StrCpy $R5 1
        ${EndIf}
    ${EndIf}
    ${If} ${FileExists} "$SMPROGRAMS\${ACCESO_NOMBRE}.lnk"
        StrCpy $R0 "$SMPROGRAMS\${ACCESO_NOMBRE}.lnk"
        Call ${PREFIJO}EsAccesoPropio
        ${If} $R4 != 1
            StrCpy $R5 1
        ${EndIf}
    ${EndIf}
FunctionEnd

; Crea o sustituye los dos accesos para que abran el lanzador de $R3, con su
; icono y descripción. Antes hay que llamar a ComprobarAccesos. Si falla,
; deja activado el indicador de error.
Function ${PREFIJO}CrearAccesos
    ; El directorio de trabajo del acceso es $OUTDIR.
    SetOutPath "$R3"
    ClearErrors
    CreateShortCut "$DESKTOP\${ACCESO_NOMBRE}.lnk" "$R3\${LANZADOR_EXE}" "" "$R3\aplicacion.ico" 0 SW_SHOWNORMAL "" "${ACCESO_DESCRIPCION}"
    ${IfNot} ${Errors}
        CreateShortCut "$SMPROGRAMS\${ACCESO_NOMBRE}.lnk" "$R3\${LANZADOR_EXE}" "" "$R3\aplicacion.ico" 0 SW_SHOWNORMAL "" "${ACCESO_DESCRIPCION}"
    ${EndIf}
FunctionEnd

!macroend

!macro ACCESOS_QUITAR PREFIJO
; Borra sólo los accesos propios (actuales o anteriores); los ajenos se conservan.
Function ${PREFIJO}QuitarAccesos
    ${If} ${FileExists} "$DESKTOP\${ACCESO_NOMBRE}.lnk"
        StrCpy $R0 "$DESKTOP\${ACCESO_NOMBRE}.lnk"
        Call ${PREFIJO}EsAccesoPropio
        ${If} $R4 == 1
            Delete "$DESKTOP\${ACCESO_NOMBRE}.lnk"
        ${EndIf}
    ${EndIf}
    ${If} ${FileExists} "$SMPROGRAMS\${ACCESO_NOMBRE}.lnk"
        StrCpy $R0 "$SMPROGRAMS\${ACCESO_NOMBRE}.lnk"
        Call ${PREFIJO}EsAccesoPropio
        ${If} $R4 == 1
            Delete "$SMPROGRAMS\${ACCESO_NOMBRE}.lnk"
        ${EndIf}
    ${EndIf}
FunctionEnd
!macroend
