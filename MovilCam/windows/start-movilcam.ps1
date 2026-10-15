[CmdletBinding()]
param(
  [ValidateSet('back','front')][string]$Lens = 'back',
  [ValidateRange(320,3840)][int]$Width = 1920,
  [ValidateRange(240,2160)][int]$Height = 1080,
  [ValidateRange(5,60)][int]$Fps = 30,
  [ValidateRange(1,24)][int]$BitrateMbps = 8,
  [string]$Serial = ''
)
$ErrorActionPreference = 'Stop'
if (-not (Get-Command adb -ErrorAction SilentlyContinue)) { throw 'adb.exe no encontrado en PATH.' }
$a = @()
if ($Serial) { $a += @('-s', $Serial) }
$state = (& adb @a get-state 2>&1 | Out-String).Trim()
if ($LASTEXITCODE -ne 0 -or $state -ne 'device') {
  throw "Android no autorizado o desconectado: $state"
}
& adb @a forward tcp:27183 tcp:27183
if ($LASTEXITCODE -ne 0) { throw 'No se pudo crear el puente ADB' }
$bitrate = $BitrateMbps * 1000000
& adb @a shell am start -n 'com.solarispkn.movilcam/.MainActivity' --ez autostart true --es lens $Lens --ei width $Width --ei height $Height --ei fps $Fps --ei bitrate $bitrate
if ($LASTEXITCODE -ne 0) { throw 'Fallo iniciando MovilCam (instalacion/permisos)' }
Write-Host 'MovilCam iniciado. En OBS: Fuente multimedia, desmarcar Archivo local'
Write-Host 'Entrada: tcp://127.0.0.1:27183'
Write-Host 'Formato de entrada: h264'
Write-Host 'Diagnostico: adb logcat -s MovilCam:V'
