@echo off
setlocal
cd /d "%~dp0"
title SolarisPKN - Publicador Multiplataforma

where node >nul 2>nul
if errorlevel 1 (
  echo No se encontro Node.js en PATH.
  echo Instala Node.js 20 o superior y volve a ejecutar este archivo.
  goto :error
)

node -e "const major=Number(process.versions.node.split('.')[0]); process.exit(major >= 20 ? 0 : 1)"
if errorlevel 1 (
  echo La version instalada de Node.js es demasiado antigua.
  echo Instala Node.js 20 o superior y volve a ejecutar este archivo.
  goto :error
)

where npm >nul 2>nul
if errorlevel 1 (
  echo No se encontro npm en PATH.
  echo Reinstala Node.js incluyendo npm y volve a ejecutar este archivo.
  goto :error
)

if not exist "..\Linkedin\node_modules\" (
  echo Primera ejecucion: preparando el conversor Markdown/MDX compartido...
  pushd "..\Linkedin"
  call npm ci --cache .npm-cache
  if errorlevel 1 (
    popd
    goto :error
  )
  popd
  echo.
)

if not exist "node_modules\" (
  echo Instalando dependencias del publicador multiplataforma...
  call npm ci --cache .npm-cache
  if errorlevel 1 goto :error
  echo.
)

echo Preparando SolarisPKN Multiplatform Publisher...
call npm run build
if errorlevel 1 goto :error

echo.
if "%SOLARIS_PUBLISHER_NO_OPEN%"=="1" (
  node scripts\server.mjs
) else (
  node scripts\server.mjs --open
)

if errorlevel 1 goto :error
endlocal
exit /b 0

:error
echo.
echo El publicador termino con un error.
if not "%SOLARIS_PUBLISHER_NO_PAUSE%"=="1" pause
endlocal
exit /b 1
