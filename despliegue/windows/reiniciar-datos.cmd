@echo off
setlocal
powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%~dp0reiniciar-datos.ps1" %*
set "resultado=%errorlevel%"
pause
exit /b %resultado%
