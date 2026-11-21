# SPDX-License-Identifier: GPL-3.0-only
# Registry HKCU Run entry: install a previously verified update BEFORE app starts.
# No audio stream is active at Windows sign-in; no OS reboot is triggered.
$ErrorActionPreference='Continue'
$app=Split-Path -Parent $PSScriptRoot
$apply=Join-Path $PSScriptRoot 'Apply-PendingUpdate.ps1'
if(Test-Path -LiteralPath $apply) {
    & powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -File $apply | Out-Null
}
$exe=Join-Path $app 'SolarisPKN-AntiCensor.exe'
if(Test-Path -LiteralPath $exe) {
    Start-Process -FilePath $exe -WorkingDirectory $app | Out-Null
}
