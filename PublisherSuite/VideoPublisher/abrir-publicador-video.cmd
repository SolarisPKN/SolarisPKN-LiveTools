@echo off
setlocal
cd /d "%~dp0"
title SolarisPKN - Video Publisher

where node >nul 2>nul
if errorlevel 1 (
  echo No se encontro Node.js en PATH. Instala Node.js 20 o superior.
  goto :error
)
node -e "const major=Number(process.versions.node.split('.')[0]); process.exit(major >= 20 ? 0 : 1)"
if errorlevel 1 (
  echo La version de Node.js es demasiado antigua. Se requiere Node.js 20 o superior.
  goto :error
)
where npm >nul 2>nul
if errorlevel 1 (
  echo No se encontro npm en PATH.
  goto :error
)
if not exist "node_modules\" (
  echo Primera ejecucion: instalando dependencias locales...
  call npm ci --cache .npm-cache
  if errorlevel 1 goto :error
)
call npm run build
if errorlevel 1 goto :error

if "%SOLARIS_VIDEO_NO_OPEN%"=="1" (
  node scripts\server.mjs
) else (
  node scripts\server.mjs --open
)
if errorlevel 1 goto :error
endlocal
exit /b 0

:error
echo.
echo El publicador de video termino con un error.
if not "%SOLARIS_VIDEO_NO_PAUSE%"=="1" pause
endlocal
exit /b 1
