[CmdletBinding()]
param(
  [ValidateSet('back','front')][string]$Lens='back',
  [ValidateRange(5,60)][int]$Fps=60,
  [ValidateRange(1,24)][int]$BitrateMbps=12,
  [string]$Serial='',
  [ValidateSet('auto','intel','nvidia','amd','software')][string]$Gpu='auto',
  [string]$BridgeExe='',
  [switch]$SoftwareDecode
)
$ErrorActionPreference='Stop'
if ($Fps -ne 60) { throw 'El dispositivo virtual DirectShow de esta version trabaja a 60 FPS. Para 30 FPS usa la fuente multimedia directa de OBS.' }
& (Join-Path $PSScriptRoot 'start-movilcam.ps1') -Lens $Lens -Fps $Fps -BitrateMbps $BitrateMbps -Serial $Serial
if (-not $BridgeExe) {
  $BridgeExe = Join-Path $PSScriptRoot 'native\build\Release\MovilCamBridge.exe'
}
if (-not (Test-Path -LiteralPath $BridgeExe)) {
  throw 'Falta MovilCamBridge.exe. Compila o descargalo del workflow Windows native.'
}
Write-Host 'Broker iniciado. La camara permanece apagada hasta que una app use la fuente DirectShow.'
if ($SoftwareDecode) { $Gpu = 'software' }
Write-Host "Decodificador elegido: $Gpu"
& $BridgeExe --gpu $Gpu
