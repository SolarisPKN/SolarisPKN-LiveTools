[CmdletBinding()]
$ErrorActionPreference = 'Stop'
$id = [Security.Principal.WindowsIdentity]::GetCurrent()
$admin = [Security.Principal.WindowsPrincipal]::new($id).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $admin) { throw 'Abri PowerShell COMO ADMINISTRADOR.' }
$folder = Join-Path $env:ProgramFiles 'SolarisPKN\MovilCam'
$dll = Join-Path $folder 'MovilCamVirtualCamera.dll'
if (Test-Path -LiteralPath $dll) {
  & "$env:windir\System32\regsvr32.exe" /s /u $dll
  if ($LASTEXITCODE -ne 0) { throw "No se pudo desregistrar DLL: $LASTEXITCODE" }
  Remove-Item -LiteralPath $dll
  Write-Host 'MovilCam camera virtual desregistrada.'
} else {
  Write-Host 'No esta instalada en Program Files.'
}
