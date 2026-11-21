# SPDX-License-Identifier: GPL-3.0-only
$ErrorActionPreference='Stop'
$root=(Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$py=Join-Path $root '.venv\Scripts\python.exe'
$builder=Join-Path $PSScriptRoot 'tools\InnoSetup6\ISCC.exe'
if(-not (Test-Path $builder)) {
    # Reuse a locally installed compiler when available; never bundle its binaries.
    $candidates=@(
        'C:\Program Files (x86)\Inno Setup 6\ISCC.exe',
        'C:\Program Files\Inno Setup 6\ISCC.exe',
        'D:\SolarisPKN-IA\Proyectos-Locales\SolarisPKN-AntiCheat-Mic\packaging\tools\InnoSetup6\ISCC.exe'
    )
    foreach($candidate in $candidates) {
        if(Test-Path -LiteralPath $candidate) { $builder=$candidate; break }
    }
}
if(-not (Test-Path $py)){throw 'First run INSTALAR.ps1 / Primero ejecutá INSTALAR.ps1.'}
& $py -m pip show pyinstaller 2>&1 | Out-Null
if($LASTEXITCODE -ne 0) {
    & $py -m pip --disable-pip-version-check install --only-binary=:all: 'pyinstaller==6.22.3'
    if($LASTEXITCODE -ne 0){throw 'Could not install PyInstaller / No se pudo instalar PyInstaller.'}
}
$ico=Join-Path $PSScriptRoot 'anticensor.ico'
Push-Location $root
try {
    & $py (Join-Path $PSScriptRoot 'MAKE_ICON.py')
    if($LASTEXITCODE -ne 0){throw 'No se pudo crear icono original.'}
    $argsPy=@('-m','PyInstaller','--noconfirm','--clean','--onefile','--windowed',
        '--name','SolarisPKN-AntiCensor','--icon',$ico,
        '--distpath',(Join-Path $root 'dist'),
        '--workpath',(Join-Path $PSScriptRoot 'build'),
        '--specpath',$PSScriptRoot,'--paths',$root,
        '--collect-all','sounddevice','--collect-all','pystray',
        '--hidden-import','webrtcvad','--hidden-import','_webrtcvad',
        '--add-data',((Join-Path $root 'docs\assets\cover.webp') + ';docs/assets'),
        (Join-Path $root 'tray_app.py'))
    & $py @argsPy
    if($LASTEXITCODE -ne 0){throw 'PyInstaller fallo.'}
    $exe=Join-Path $root 'dist\SolarisPKN-AntiCensor.exe'
    if(-not (Test-Path $exe)){throw 'No se genero EXE.'}
    Write-Host "EXE=$exe SHA256=$((Get-FileHash $exe).Hash)"
    if(Test-Path $builder){
        # Public repository does not track vendor binaries. Obtain verified official
        # unchanged archive only when building a Windows installer.
        & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot 'FETCH_VBCABLE.ps1')
        if($LASTEXITCODE -ne 0){throw 'Failed to verify the official VB-CABLE package.'}
        & $builder (Join-Path $PSScriptRoot 'AntiCensor.iss')
        if($LASTEXITCODE -ne 0){throw 'Inno Setup fallo.'}
        Write-Host 'Instalador generado en dist/'
    }else{Write-Warning 'Inno Setup no instalado: solamente EXE portable.'}
} finally {Pop-Location}
