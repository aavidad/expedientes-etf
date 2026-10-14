@echo off
setlocal
powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%~dp0instalar.ps1" %*
set "resultado=%errorlevel%"
pause
exit /b %resultado%
