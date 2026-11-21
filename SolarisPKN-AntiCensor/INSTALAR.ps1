# SPDX-License-Identifier: GPL-3.0-only
# Prepara solamente dependencias de la aplicación GPLv3 AntiCensor.
# Nunca instala controladores ni modifica Windows, Secure Boot o audio predeterminado.
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$venv = Join-Path $root '.venv'
$vp = Join-Path $venv 'Scripts\python.exe'
if (-not (Test-Path $vp)) {
    $candidates = @(
        (Join-Path $env:LOCALAPPDATA 'Programs\Python\Python312\python.exe'),
        'C:\Python312\python.exe',
        'C:\Program Files\Python312\python.exe'
    )
    $py = $candidates | Where-Object { Test-Path $_ } | Select-Object -First 1
    if (-not $py) { throw 'Se requiere Python 3.12 x64. Instalalo y ejecuta de nuevo.' }
    & $py -m venv $venv
    if ($LASTEXITCODE -ne 0) { throw 'No se pudo crear el entorno virtual local.' }
}
& $vp -m pip --disable-pip-version-check install --only-binary=:all: -r (Join-Path $root 'requirements.txt')
if ($LASTEXITCODE -ne 0) { throw 'Fallo instalando las dependencias locales.' }
& $vp -m py_compile (Join-Path $root 'calibration.py') (Join-Path $root 'mic.py')
if ($LASTEXITCODE -ne 0) { throw 'Error de sintaxis en AntiCensor.' }
Write-Host 'SolarisPKN AntiCensor preparado. Ejecuta ABRIR.bat.'
