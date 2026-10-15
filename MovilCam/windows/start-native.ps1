[CmdletBinding()]
param(
  [ValidateSet('back','front')][string]$Lens='back',
  [ValidateRange(5,60)][int]$Fps=30,
  [ValidateRange(1,24)][int]$BitrateMbps=8,
  [string]$Serial='',
  [string]$BridgeExe='',
  [switch]$SoftwareDecode
)
$ErrorActionPreference='Stop'
& (Join-Path $PSScriptRoot 'start-movilcam.ps1') -Lens $Lens -Fps $Fps -BitrateMbps $BitrateMbps -Serial $Serial
if (-not $BridgeExe) {
  $BridgeExe = Join-Path $PSScriptRoot 'native\build\Release\MovilCamBridge.exe'
}
if (-not (Test-Path -LiteralPath $BridgeExe)) {
  throw 'Falta MovilCamBridge.exe. Compila o descargalo del workflow Windows native.'
}
Write-Host 'Broker iniciado. La camara permanece apagada hasta que una app use la fuente DirectShow.'
if ($SoftwareDecode) { & $BridgeExe --software } else { & $BridgeExe }
