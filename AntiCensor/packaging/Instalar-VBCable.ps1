# SPDX-License-Identifier: GPL-3.0-only
# VB-CABLE is NOT GPL: proprietary donationware from VB-Audio Software.
# This helper never modifies/re-signs the original VB-CABLE package.
[CmdletBinding()]
param(
    [switch] $VerifyOnly,
    [ValidateSet('english','spanish','en','es')]
    [string] $Language = 'spanish'
)
function L([string]$es,[string]$en) {
    if ($Language -in @('en','english')) { return $en }
    return $es
}

$ErrorActionPreference = 'Stop'
$AppRoot = Split-Path -Parent $PSScriptRoot
$Archive = Join-Path $AppRoot 'third_party\VBCABLE_Driver_Pack46.zip'
# Dentro del checkout del repositorio, el paquete se encuentra en packaging/third_party.
if (!(Test-Path -LiteralPath $Archive)) {
    $Archive = Join-Path $PSScriptRoot 'third_party\VBCABLE_Driver_Pack46.zip'
}
$ExpectedSha256 = 'BC44E8974191A5A2427D6C979E2DAB08DBB09925F3F435DFA2F0BA3E45B6EB90'
$OfficialPage = 'https://vb-audio.com/Cable/'

function Get-VBCableDevice {
    $dev = @(Get-CimInstance -ClassName Win32_PnPEntity -ErrorAction SilentlyContinue |
        Where-Object { $_.Name -match 'VB-Audio Virtual Cable|VB-Audio Cable|VB-CABLE' -and $_.ConfigManagerErrorCode -eq 0 })
    return $dev.Count -gt 0
}

function Show-Info([string]$message, [string]$title='SolarisPKN AntiCensor') {
    try {
        Add-Type -AssemblyName System.Windows.Forms
        [void][System.Windows.Forms.MessageBox]::Show(
            $message, $title, [System.Windows.Forms.MessageBoxButtons]::OK,
            [System.Windows.Forms.MessageBoxIcon]::Information)
    } catch { Write-Host "$title : $message" }
}

function Set-AntiCensorMicName {
    # Display-name change is independent of the VB-Audio signed driver.
    # If the endpoint appears only after a reboot, AntiCensor tray retries it.
    $renamer = Join-Path $PSScriptRoot 'Renombrar-Mic.ps1'
    if (!(Test-Path -LiteralPath $renamer)) {
        Write-Warning "Mic rename helper not found; AntiCensor will retry after installation."
        return
    }
    & powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -File $renamer
    if ($LASTEXITCODE -ne 0) {
        Write-Warning "Could not rename endpoint yet; the tray will retry after Windows starts."
    }
}

if (Get-VBCableDevice) {
    Write-Host (L "VB-CABLE ya está instalado." "VB-CABLE is already installed.")
    if (-not $VerifyOnly) {
        Set-AntiCensorMicName
        Show-Info (L "VB-CABLE ya está instalado; AntiCensor intentó nombrar el micrófono como SolarisPKN-Mic Lelele." "VB-CABLE is already installed; AntiCensor checked the SolarisPKN-Mic Lelele display name.")
    }
    exit 0
}
if (!(Test-Path -LiteralPath $Archive)) {
    if ($VerifyOnly) { throw "Paquete original ausente: $Archive" }
    throw (L "No se encontró el paquete oficial. Descargalo desde vb-audio.com/Cable/" "Original VB-CABLE package not found. Download it at vb-audio.com/Cable/")
}
$Actual = (Get-FileHash -LiteralPath $Archive -Algorithm SHA256).Hash
if ($Actual -ne $ExpectedSha256) { throw (L "El SHA-256 de VB-CABLE no coincide; abortado." "VB-CABLE SHA-256 mismatch; aborting installation.") }
$TempDir = Join-Path $env:TEMP 'SolarisPKN-AntiCensor\VBCABLE-Pack46'
if (!(Test-Path $TempDir)) { New-Item -Path $TempDir -ItemType Directory -Force | Out-Null }
Expand-Archive -LiteralPath $Archive -DestinationPath $TempDir -Force
$Catalog = Join-Path $TempDir 'vbaudio_cable64_win10.cat'
$Setup = Join-Path $TempDir 'VBCABLE_Setup_x64.exe'
if (!(Test-Path $Catalog) -or !(Test-Path $Setup)) { throw (L "Faltan archivos del instalador VB-CABLE." "Missing VB-CABLE installer files.") }
$CatSig = Get-AuthenticodeSignature -LiteralPath $Catalog
if ($CatSig.Status -ne 'Valid' -or
    $CatSig.SignerCertificate.Subject -notmatch 'Microsoft Windows Hardware Compatibility Publisher') {
    throw (L "Firma Microsoft de catálogo inválida; cancelado." "Invalid Microsoft catalog signature; cancelled.")
}
if ($VerifyOnly) {
    Write-Host (L "OK: archivo intacto, catálogo Microsoft válido." "OK: original package intact, Microsoft-signed catalog valid.")
    exit 0
}
Show-Info (L (
    'AntiCensor usará VB-CABLE de VB-Audio Software (software propietario donationware, no GPLv3).' +
    [Environment]::NewLine + 'Se ejecutará el instalador original con UAC. Reiniciá al terminar.' +
    [Environment]::NewLine + 'Origen: ' + $OfficialPage
) (
    'AntiCensor uses VB-CABLE by VB-Audio Software (proprietary donationware, not GPLv3).' +
    [Environment]::NewLine + 'The original installer requires UAC. Restart Windows afterwards.' +
    [Environment]::NewLine + 'Source: ' + $OfficialPage
))
try {
    $proc = Start-Process -FilePath $Setup -Verb RunAs -WorkingDirectory $TempDir -Wait -PassThru
    Write-Host "VB-CABLE setup exit code: $($proc.ExitCode)"
    if ($proc.ExitCode -ne 0) {
        Show-Info "El instalador del fabricante terminó con código $($proc.ExitCode). Revisá su ventana y la guía de VB-Audio."
        exit 2
    }
    Set-AntiCensorMicName
    Show-Info (L (
        'Instalador VB-Audio finalizado. Reiniciá Windows. AntiCensor pondrá automáticamente el nombre SolarisPKN-Mic Lelele.'
    ) (
        'VB-Audio installer finished. Restart Windows. AntiCensor will automatically name the microphone SolarisPKN-Mic Lelele.'
    ))
} catch {
    $errorSpanish = "No se completó VB-CABLE: " + $_.Exception.Message +
                    [Environment]::NewLine + "Instalación manual: " + $OfficialPage
    $errorEnglish = "VB-CABLE installation was not completed: " + $_.Exception.Message +
                    [Environment]::NewLine + "Manual install: " + $OfficialPage
    Show-Info (L $errorSpanish $errorEnglish)
    exit 3
}
