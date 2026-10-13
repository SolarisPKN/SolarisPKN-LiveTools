@echo off
setlocal
cd /d "%~dp0"
if not exist "config.json" (
  copy /Y "config.example.json" "config.json" >nul
  echo Configuracion de ejemplo creada. Editar antes de conectar plataformas reales.
)
if exist "MultiChat.exe" (
  "MultiChat.exe" -config "config.json" %*
) else (
  where go >nul 2>&1
  if errorlevel 1 (
    echo Falta MultiChat.exe o Go. Instala Go 1.22+ o compila el proyecto.
    exit /b 1
  )
  go run . -config "config.json" %*
)
endlocal
