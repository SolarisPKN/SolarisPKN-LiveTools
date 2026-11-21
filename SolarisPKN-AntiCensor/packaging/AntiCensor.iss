; SolarisPKN AntiCensor GPLv3 installer. No unsigned kernel driver!
#define ProjectRoot AddBackslash(SourcePath) + ".."
#define Version "0.8.5"
[Setup]
AppId={{8FE390B9-D2BC-43F7-91E6-23AFBE4A0B32}
AppName=SolarisPKN AntiCensor
AppVersion={#Version}
AppVerName=SolarisPKN AntiCensor {#Version} (preview)
AppPublisher=SolarisPKN-Labs
AppPublisherURL=https://labs.solarispkn.com.ar/
AppSupportURL=https://labs.solarispkn.com.ar/
AppUpdatesURL=https://github.com/SolarisPKN/SolarisPKN-LiveTools/releases
DefaultDirName={localappdata}\Programs\SolarisPKN\AntiCensor
DefaultGroupName=SolarisPKN AntiCensor
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
ArchitecturesAllowed=x64os
OutputDir={#ProjectRoot}\dist
OutputBaseFilename=SolarisPKN-AntiCensor-Setup-v{#Version}
UninstallDisplayIcon={app}\SolarisPKN-AntiCensor.exe
LicenseFile={#ProjectRoot}\LICENSE
WizardStyle=modern
Compression=lzma2
SolidCompression=yes
CloseApplications=yes
RestartApplications=no
SetupLogging=yes
VersionInfoVersion=0.8.5.0
VersionInfoCompany=SolarisPKN
VersionInfoDescription=SolarisPKN AntiCensor GPLv3 Installer

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"
Name: "spanish"; MessagesFile: "compiler:Languages\Spanish.isl"

[CustomMessages]
english.TaskAutoStart=Launch with Windows and keep the icon in the system tray
spanish.TaskAutoStart=Iniciar con Windows y mantener el icono junto al reloj
english.TaskDesktop=Create a desktop shortcut
spanish.TaskDesktop=Crear acceso directo en el escritorio
english.TaskDependency=Install VB-CABLE (only if missing; requires administrator access and restart)
english.TaskClean=Clean VB-CABLE driver (troubleshooting only; warning before removing)
spanish.TaskDependency=Instalar VB-CABLE (solo si falta; requiere administrador y reinicio)
spanish.TaskClean=Limpiar driver VB-CABLE (solo por errores; pide confirmación)
english.TaskStartupGroup=Startup:
spanish.TaskStartupGroup=Inicio automático:
english.TaskShortcuts=Shortcuts:
spanish.TaskShortcuts=Accesos directos:
english.TaskExternal=External audio dependency:
spanish.TaskExternal=Complemento externo para transmitir audio:
english.RunApplication=Launch SolarisPKN AntiCensor
spanish.RunApplication=Abrir SolarisPKN AntiCensor


[Tasks]
Name: "autostart"; Description: "{cm:TaskAutoStart}"; GroupDescription: "{cm:TaskStartupGroup}"; Flags: checkedonce
Name: "desktopicon"; Description: "{cm:TaskDesktop}"; GroupDescription: "{cm:TaskShortcuts}"; Flags: unchecked
Name: "vbcable"; Description: "{cm:TaskDependency}"; GroupDescription: "{cm:TaskExternal}"; Flags: checkedonce; Check: not IsVendorInstalled
Name: "clean"; Description: "{cm:TaskClean}"; GroupDescription: "{cm:TaskExternal}"; Flags: unchecked; Check: IsVendorInstalled

[Files]
Source: "{#ProjectRoot}\dist\SolarisPKN-AntiCensor.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#ProjectRoot}\LICENSE"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#ProjectRoot}\mic.py"; DestDir: "{app}\source"; Flags: ignoreversion
Source: "{#ProjectRoot}\calibration.py"; DestDir: "{app}\source"; Flags: ignoreversion
Source: "{#ProjectRoot}\i18n.py"; DestDir: "{app}\source"; Flags: ignoreversion
Source: "{#ProjectRoot}\theme.py"; DestDir: "{app}\source"; Flags: ignoreversion
Source: "{#ProjectRoot}\vbcable_backend.py"; DestDir: "{app}\source"; Flags: ignoreversion
Source: "{#ProjectRoot}\device_visibility.py"; DestDir: "{app}\source"; Flags: ignoreversion
Source: "{#ProjectRoot}\voice_gate.py"; DestDir: "{app}\source"; Flags: ignoreversion
Source: "{#ProjectRoot}\packaging\third_party\VBCABLE_Driver_Pack46.zip"; DestDir: "{app}\third_party"; Flags: ignoreversion
Source: "{#ProjectRoot}\packaging\third_party\VBCABLE-NOTICE.txt"; DestDir: "{app}\third_party"; Flags: ignoreversion
Source: "{#ProjectRoot}\packaging\Instalar-VBCable.ps1"; DestDir: "{app}\scripts"; Flags: ignoreversion
Source: "{#ProjectRoot}\packaging\Apply-PendingUpdate.ps1"; DestDir: "{app}\scripts"; Flags: ignoreversion
Source: "{#ProjectRoot}\packaging\Start-AntiCensor.ps1"; DestDir: "{app}\scripts"; Flags: ignoreversion
Source: "{#ProjectRoot}\packaging\Renombrar-Mic.ps1"; DestDir: "{app}\scripts"; Flags: ignoreversion
Source: "{#ProjectRoot}\packaging\Renombrar-Mic-GUI.ps1"; DestDir: "{app}\scripts"; Flags: ignoreversion
Source: "{#ProjectRoot}\packaging\Renombrar-Mic.ps1"; DestDir: "{app}\source\packaging"; Flags: ignoreversion
Source: "{#ProjectRoot}\packaging\Renombrar-Mic-GUI.ps1"; DestDir: "{app}\source\packaging"; Flags: ignoreversion
Source: "{#ProjectRoot}\packaging\Instalar-VBCable.ps1"; DestDir: "{app}\source\packaging"; Flags: ignoreversion
Source: "{#ProjectRoot}\updater.py"; DestDir: "{app}\source"; Flags: ignoreversion
Source: "{#ProjectRoot}\tray_app.py"; DestDir: "{app}\source"; Flags: ignoreversion
Source: "{#ProjectRoot}\version.py"; DestDir: "{app}\source"; Flags: ignoreversion
Source: "{#ProjectRoot}\requirements.txt"; DestDir: "{app}\source"; Flags: ignoreversion
Source: "{#ProjectRoot}\tests\test_calibration.py"; DestDir: "{app}\source\tests"; Flags: ignoreversion
Source: "{#ProjectRoot}\tests\test_i18n.py"; DestDir: "{app}\source\tests"; Flags: ignoreversion
Source: "{#ProjectRoot}\tests\test_pcm_synth.py"; DestDir: "{app}\source\tests"; Flags: ignoreversion
Source: "{#ProjectRoot}\tests\test_updater.py"; DestDir: "{app}\source\tests"; Flags: ignoreversion
Source: "{#ProjectRoot}\tests\test_vbcable_backend.py"; DestDir: "{app}\source\tests"; Flags: ignoreversion
Source: "{#ProjectRoot}\tests\test_endpoint_rename.py"; DestDir: "{app}\source\tests"; Flags: ignoreversion
Source: "{#ProjectRoot}\tests\test_expert_mode.py"; DestDir: "{app}\source\tests"; Flags: ignoreversion
Source: "{#ProjectRoot}\tests\test_voice_gate.py"; DestDir: "{app}\source\tests"; Flags: ignoreversion
Source: "{#ProjectRoot}\tests\test_reference_device_list.py"; DestDir: "{app}\source\tests"; Flags: ignoreversion
Source: "{#ProjectRoot}\tests\test_branding.py"; DestDir: "{app}\source\tests"; Flags: ignoreversion
Source: "{#ProjectRoot}\tests\test_fixed_repo_and_name.py"; DestDir: "{app}\source\tests"; Flags: ignoreversion
Source: "{#ProjectRoot}\packaging\AntiCensor.iss"; DestDir: "{app}\source\packaging"; Flags: ignoreversion
Source: "{#ProjectRoot}\packaging\BUILD_EXE.ps1"; DestDir: "{app}\source\packaging"; Flags: ignoreversion
Source: "{#ProjectRoot}\packaging\Apply-PendingUpdate.ps1"; DestDir: "{app}\source\packaging"; Flags: ignoreversion
Source: "{#ProjectRoot}\packaging\Start-AntiCensor.ps1"; DestDir: "{app}\source\packaging"; Flags: ignoreversion
Source: "{#ProjectRoot}\packaging\FETCH_VBCABLE.ps1"; DestDir: "{app}\source\packaging"; Flags: ignoreversion
Source: "{#ProjectRoot}\packaging\MAKE_ICON.py"; DestDir: "{app}\source\packaging"; Flags: ignoreversion
Source: "{#ProjectRoot}\README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#ProjectRoot}\README.es.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#ProjectRoot}\README_INSTALADOR.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#ProjectRoot}\README_INSTALADOR.es.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#ProjectRoot}\docs\assets\cover.webp"; DestDir: "{app}\docs\assets"; Flags: ignoreversion
Source: "{#ProjectRoot}\docs\assets\*.svg"; DestDir: "{app}\docs\assets"; Flags: ignoreversion
Source: "{#ProjectRoot}\docs\assets\*.gif"; DestDir: "{app}\docs\assets"; Flags: ignoreversion
Source: "{#ProjectRoot}\docs\*.md"; DestDir: "{app}\docs"; Flags: ignoreversion
Source: "{#ProjectRoot}\packaging\GENERAR_GRAFICOS.py"; DestDir: "{app}\source\packaging"; Flags: ignoreversion

[Icons]
Name: "{group}\SolarisPKN AntiCensor"; Filename: "{app}\SolarisPKN-AntiCensor.exe"
Name: "{autodesktop}\SolarisPKN AntiCensor"; Filename: "{app}\SolarisPKN-AntiCensor.exe"; Tasks: desktopicon

[Registry]
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; ValueType: string; ValueName: "SolarisPKN-AntiCensor"; ValueData: """{sys}\WindowsPowerShell\v1.0\powershell.exe"" -NoProfile -WindowStyle Hidden -ExecutionPolicy Bypass -File ""{app}\scripts\Start-AntiCensor.ps1"""; Tasks: autostart; Flags: uninsdeletevalue

[Run]
Filename: "{sys}\WindowsPowerShell\v1.0\powershell.exe"; Parameters: "-NoProfile -ExecutionPolicy Bypass -File ""{app}\scripts\Instalar-VBCable.ps1"" -Language ""{language}"""; Tasks: vbcable; Flags: waituntilterminated skipifsilent
Filename: "{sys}\WindowsPowerShell\v1.0\powershell.exe"; Parameters: "-NoProfile -ExecutionPolicy Bypass -File ""{app}\scripts\Instalar-VBCable.ps1"" -Clean -Language ""{language}"""; Tasks: clean; Flags: waituntilterminated skipifsilent
Filename: "{sys}\WindowsPowerShell\v1.0\powershell.exe"; Parameters: "-NoProfile -NonInteractive -ExecutionPolicy Bypass -File ""{app}\scripts\Renombrar-Mic.ps1"""; Flags: waituntilterminated
Filename: "{app}\SolarisPKN-AntiCensor.exe"; Description: "{cm:RunApplication}"; Flags: nowait postinstall skipifsilent

[Code]
function IsVendorInstalled: Boolean;
var
  Locator, Service, Devices, Item: Variant;
  Name: String;
  I: Integer;
begin
  Result := False;
  try
    Locator := CreateOleObject('WbemScripting.SWbemLocator');
    Service := Locator.ConnectServer('.', 'root\cimv2');
    Devices := Service.ExecQuery('SELECT Name FROM Win32_PnPEntity WHERE Name LIKE ''%VB-Audio%''');
    for I := 0 to Integer(Devices.Count) - 1 do begin
      Item := Devices.ItemIndex(I);
      Name := Item.Name;
      if Pos('VB-Audio Virtual Cable', Name) > 0 then begin
        Result := True;
        Break;
      end;
    end;
  except
    Log('VB-CABLE detection could not enumerate WMI; install task remains available');
  end;
end;
