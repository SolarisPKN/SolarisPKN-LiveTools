[CmdletBinding()]
param([string]$Dll = "")
$ErrorActionPreference = 'Stop'
if ([IntPtr]::Size -ne 8) { throw 'Usa PowerShell x64, no x86.' }
if (-not $Dll) {
  $Dll = Join-Path $PSScriptRoot 'native\build\Release\MovilCamVirtualCamera.dll'
}
$Dll = [IO.Path]::GetFullPath($Dll)
if (-not (Test-Path -LiteralPath $Dll)) { throw "No encuentro DLL: $Dll. Compila el proyecto o descarga artefacto CI." }
$id = [Security.Principal.WindowsIdentity]::GetCurrent()
$admin = [Security.Principal.WindowsPrincipal]::new($id).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $admin) { throw 'Abri PowerShell x64 COMO ADMINISTRADOR para instalar la webcam virtual.' }
$folder = Join-Path $env:ProgramFiles 'SolarisPKN\MovilCam'
New-Item -ItemType Directory -Force -Path $folder | Out-Null
$dest = Join-Path $folder 'MovilCamVirtualCamera.dll'
Copy-Item -LiteralPath $Dll -Destination $dest -Force
$registered = Start-Process -FilePath "$env:windir\System32\regsvr32.exe" -ArgumentList @('/s', "`"$dest`"") -Wait -PassThru
if ($registered.ExitCode -ne 0) { throw "regsvr32 fallo con codigo $($registered.ExitCode)" }
Write-Host 'SolarisPKN MovilCam (DirectShow) registrada.'
Write-Host 'Ejecuta el broker antes de abrir OBS, Discord u otras apps.'
Write-Host 'La DLL registrada DEBE mantenerse en la ruta instalada.'
