# SPDX-License-Identifier: GPL-3.0-only
# Downloads only the original VB-CABLE Basic Pack46 from VB-Audio's HTTPS host.
# Never executes it. Packaging is separate from GPLv3 application source.
$ErrorActionPreference = 'Stop'
$archive = Join-Path $PSScriptRoot 'third_party\VBCABLE_Driver_Pack46.zip'
$official = 'https://download.vb-audio.com/Download_CABLE/VBCABLE_Driver_Pack46.zip'
$expected = 'BC44E8974191A5A2427D6C979E2DAB08DBB09925F3F435DFA2F0BA3E45B6EB90'
New-Item -ItemType Directory -Path (Split-Path $archive) -Force | Out-Null
if (-not (Test-Path -LiteralPath $archive)) {
    Write-Host 'Downloading unchanged VB-CABLE Basic from its official vendor...'
    $temporary = $archive + '.download'
    try {
        Invoke-WebRequest -Uri $official -UseBasicParsing -OutFile $temporary -TimeoutSec 80
        $actual = (Get-FileHash -LiteralPath $temporary -Algorithm SHA256).Hash
        if ($actual -ne $expected) { throw "VB-CABLE package SHA256 mismatch: $actual" }
        Move-Item -LiteralPath $temporary -Destination $archive -Force
    } finally {
        if (Test-Path $temporary) { Remove-Item -LiteralPath $temporary -Force }
    }
}
$hash = (Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash
if ($hash -ne $expected) { throw "Invalid VB-CABLE ZIP hash: $hash" }
Write-Host "VB-CABLE original package verified: $hash"
