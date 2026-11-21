@echo off
setlocal
cd /d "%~dp0"
if not exist ".venv\Scripts\python.exe" (
 echo [SolarisPKN AntiCensor] Preparando dependencias...
 powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0INSTALAR.ps1"
 if errorlevel 1 (pause & exit /b 2)
)
".venv\Scripts\python.exe" "tray_app.py"
if errorlevel 1 (echo Error de AntiCensor & pause)
endlocal
