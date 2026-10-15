; Lanzador «Expedientes ETF.exe» de los accesos del escritorio e Inicio.
; Los antivirus de comportamiento (y los EDR corporativos) cortan PowerShell o
; cmd lanzados directamente desde un acceso directo .lnk; por eso el acceso
; abre este programa y es él quien ejecuta abrir.ps1 de su misma carpeta.
; Lo compila paquete.py con makensis 3.10:
;   makensis /DICONO=... /DSALIDA=... lanzador.nsi
; No muestra ventana propia: la consola de abrir.ps1 enseña el progreso como
; antes. No tiene textos: los mensajes son los de abrir.ps1 (mensajes.json).

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

Section
    ; Ruta absoluta a la PowerShell de 64 bits de Windows: no depende del PATH.
    Exec '"$WINDIR\System32\WindowsPowerShell\v1.0\powershell.exe" -NoLogo -NoProfile -ExecutionPolicy Bypass -File "$EXEDIR\abrir.ps1"'
SectionEnd
