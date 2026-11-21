# SPDX-License-Identifier: GPL-3.0-only
# Runs only AFTER AntiCensor quits or before it launches at Windows login.
# Never closes apps, audio streams, windows, drivers, or Windows itself.
[CmdletBinding()]
param([int]$ParentPid = 0)
$ErrorActionPreference = 'Stop'
$state = Join-Path $env:LOCALAPPDATA 'SolarisPKN\AntiCensor'
$manifest = Join-Path $state 'pending_update.json'
$settings = Join-Path $state 'updates.json'
$updates = [IO.Path]::GetFullPath((Join-Path $state 'Updates')) + [IO.Path]::DirectorySeparatorChar
$log = Join-Path $state 'update-install.log'
$mutex = New-Object System.Threading.Mutex($false,'Local\SolarisPKN-AntiCensor-UpdateInstall')
$locked = $false
function Trace([string]$line) {
    try {
        New-Item -ItemType Directory -Force -Path $state | Out-Null
        Add-Content -LiteralPath $log -Value ("{0:o} {1}" -f (Get-Date),$line) -Encoding UTF8
    } catch {}
}
try {
    $locked = $mutex.WaitOne(0)
    if (!$locked) { exit 0 }
    if ($ParentPid -gt 0) {
        try { Wait-Process -Id $ParentPid -Timeout 120 -ErrorAction Stop } catch { }
    }
    if (!(Test-Path -LiteralPath $manifest)) { exit 0 }
    # If the app is running in any user session, never touch its executable.
    $running = @(Get-CimInstance Win32_Process -Filter "Name='SolarisPKN-AntiCensor.exe'" -ErrorAction SilentlyContinue)
    if ($running.Count -gt 0) { Trace 'DEFERRED: AntiCensor process is running'; exit 0 }

    $data = Get-Content -LiteralPath $manifest -Raw -Encoding UTF8 | ConvertFrom-Json
    if ($data.source -eq 'automatic') {
        $stateInfo = if(Test-Path $settings){Get-Content $settings -Raw -Encoding UTF8 | ConvertFrom-Json}else{$null}
        if ($null -eq $stateInfo -or $stateInfo.auto_updates -ne $true) {
            Trace 'SKIPPED: automatic updates disabled'; exit 0
        }
    } elseif ($data.source -ne 'manual') { throw 'Unexpected update source' }

    if ([string]$data.version -notmatch '^\d+\.\d+\.\d+$') { throw 'Invalid version format' }
    if ([string]$data.tag -notmatch '^(?:solarispkn-)?anticensor-v\d+\.\d+\.\d+$') { throw 'Untrusted release tag' }
    if ([string]$data.sha256 -notmatch '^[0-9a-fA-F]{64}$') { throw 'Invalid SHA256 manifest' }
    $path=[IO.Path]::GetFullPath([string]$data.file)
    if (!$path.StartsWith($updates,[StringComparison]::OrdinalIgnoreCase)) { throw 'Update path outside private staging directory' }
    $expectedFile='SolarisPKN-AntiCensor-Setup-v'+[string]$data.version+'.exe'
    if ([IO.Path]::GetFileName($path) -cne $expectedFile) { throw 'Unexpected installer filename' }
    if (!(Test-Path -LiteralPath $path -PathType Leaf)) { throw 'Staged installer missing' }
    $actual=(Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash
    if ($actual -ine [string]$data.sha256) { throw 'SHA256 mismatch: refusing staged update' }
    if ((Get-Item -LiteralPath $path).Length -ne [long]$data.bytes) { throw 'Size mismatch' }

    Trace ('Installing deferred update ' + $data.version)
    $proc=Start-Process -FilePath $path -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES',
        '/NORESTART','/TASKS="autostart"') -Wait -PassThru
    if ($proc.ExitCode -ne 0) { throw ('Installer exit code '+$proc.ExitCode) }
    Remove-Item -LiteralPath $manifest -Force
    Remove-Item -LiteralPath $path -Force -ErrorAction SilentlyContinue
    Trace ('Installed successfully ' + $data.version)
    exit 0
} catch {
    Trace ('Update left pending: ' + $_.Exception.Message)
    exit 3
} finally {
    if ($locked) { $mutex.ReleaseMutex() }
    $mutex.Dispose()
}
