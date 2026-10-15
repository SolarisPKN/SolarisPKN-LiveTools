[CmdletBinding()]
param([string]$Serial = '')
$a = @()
if ($Serial) { $a += @('-s', $Serial) }
& adb @a shell am force-stop com.solarispkn.movilcam
& adb @a forward --remove tcp:27183
Write-Host 'MovilCam detenido.'
