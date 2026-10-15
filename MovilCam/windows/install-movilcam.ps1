[CmdletBinding()]
param([string]$Apk = "", [string]$Serial = "")
$ErrorActionPreference = 'Stop'
$package = 'com.solarispkn.movilcam'
if (-not (Get-Command adb -ErrorAction SilentlyContinue)) { throw 'ADB no esta en PATH.' }
$a = @()
if ($Serial) { $a += @('-s', $Serial) }
if (-not $Apk) { $Apk = Join-Path $PSScriptRoot '..\app\build\outputs\apk\debug\app-debug.apk' }
$Apk = [IO.Path]::GetFullPath($Apk)
if (-not (Test-Path $Apk)) { throw "APK no encontrado: $Apk. Compila el APK o descargalo de GitHub Actions." }
& adb @a install -r $Apk
if ($LASTEXITCODE -ne 0) { throw 'adb install fallo' }
& adb @a shell pm grant $package android.permission.CAMERA
if ($LASTEXITCODE -ne 0) { Write-Warning 'Dar permiso CAMERA manualmente.' }
Write-Host 'Instalado. Ahora ejecuta start-movilcam.ps1.'
