# SPDX-License-Identifier: GPL-3.0-only
# Rename ONLY the verified VB-CABLE Basic recording endpoint using Windows UI.
# Runs in interactive user session; no kernel driver, registry ACL, Secure Boot changes.
$ErrorActionPreference='Stop'
$desired='SolarisPKN-Mic Lelele'
$log=Join-Path $env:LOCALAPPDATA 'SolarisPKN\AntiCensor\rename-gui-result.txt'
$output=New-Object System.Collections.Generic.List[string]
function Info([string]$line){$output.Add($line)}
$renameSet=$false
$dialog=$null
$openedByScript=$false
try {
 Add-Type -AssemblyName UIAutomationClient
 Add-Type -AssemblyName UIAutomationTypes
 $interop=@'
using System;
using System.Runtime.InteropServices;
using System.Text;
public static class AntiCensorSoundGui {
 [DllImport("user32.dll", CharSet=CharSet.Unicode, EntryPoint="SendMessageW")]
 public static extern IntPtr SetText(IntPtr window, uint msg, IntPtr wparam,
    [MarshalAs(UnmanagedType.LPWStr)]string text);
 [DllImport("user32.dll", CharSet=CharSet.Unicode, EntryPoint="SendMessageW")]
 public static extern IntPtr GetText(IntPtr window, uint msg, IntPtr max, StringBuilder text);
 [DllImport("user32.dll", SetLastError=true)]
 public static extern bool PostMessage(IntPtr hwnd,uint msg,IntPtr w,IntPtr l);
 [DllImport("oleacc.dll")]
 public static extern int AccessibleObjectFromWindow(IntPtr hwnd,uint objectId,
    ref Guid iid,[MarshalAs(UnmanagedType.IUnknown)]out object acc);
}
'@
 Add-Type -TypeDefinition $interop
 # Safety: only one VB-Audio recording endpoint, not a physical or virtual
 # device from another vendor.
 $endpoint=@(Get-PnpDevice -Class AudioEndpoint -PresentOnly | Where-Object {
   $_.FriendlyName -eq 'CABLE Output (VB-Audio Virtual Cable)' -or
   $_.FriendlyName -eq 'SolarisPKN-Mic Lelele (VB-Audio Virtual Cable)' -or
   $_.FriendlyName -eq 'SolarisPKN-Mic Lelele'})
 if($endpoint.Count -ne 1){throw "VB-CABLE capture endpoint not unique: $($endpoint.Count)"}
 if($endpoint[0].FriendlyName -like 'SolarisPKN-Mic Lelele*'){
  Info 'RESULT=ALREADY_RENAMED'
  Info 'ANTICENSOR_ENDPOINT_NAME=ALREADY'
  return
 }
 # Never steal the desktop focus while OBS/Roblox is open.
 $streamers=@(Get-CimInstance Win32_Process -ErrorAction SilentlyContinue |
    Where-Object {$_.Name -match '(?i)^(obs64|RobloxPlayerBeta|RobloxPlayer|streamlabs|xsplit|tiktoklive|twitchstudio)\.exe$'})
 if($streamers.Count -gt 0){
  Info 'RESULT=DEFERRED_STREAM_ACTIVE'
  Info 'ANTICENSOR_ENDPOINT_NAME=BUSY'
  return
 }
 $root=[System.Windows.Automation.AutomationElement]::RootElement
 function Get-SoundDialogs {
  $top=$root.FindAll([System.Windows.Automation.TreeScope]::Children,[System.Windows.Automation.Condition]::TrueCondition)
  return @($top|Where-Object {$_.Current.Name -in @('Sonido','Sound') -and $_.Current.ClassName -eq '#32770'})
 }
 $sound=Get-SoundDialogs
 if($sound.Count -eq 0){
  Start-Process -FilePath "$env:WINDIR\System32\control.exe" -ArgumentList 'mmsys.cpl,,1' | Out-Null
  $openedByScript=$true
  for($i=0;$i -lt 35;$i++){
   Start-Sleep -Milliseconds 300
   $sound=Get-SoundDialogs
   if($sound.Count -ge 1){break}
  }
 }
 if($sound.Count -ne 1){throw "Expected one Sound dialog, found $($sound.Count)"}
 function Get-PropertiesDialogs {
  $sub=$sound[0].FindAll([System.Windows.Automation.TreeScope]::Descendants,[System.Windows.Automation.Condition]::TrueCondition)
  return @($sub|Where-Object {
   $_.Current.ClassName -eq '#32770' -and
   $_.Current.Name -match '^(Propiedades de CABLE Output|CABLE Output Properties)$'
  })
 }
 $propertyDialogs=Get-PropertiesDialogs
 if($propertyDialogs.Count -eq 0){
  $items=$sound[0].FindAll([System.Windows.Automation.TreeScope]::Descendants,[System.Windows.Automation.Condition]::TrueCondition)
  $list=@($items|Where-Object{$_.Current.AutomationId -eq '1000' -and $_.Current.ClassName -eq 'SysListView32'})
  $propsButton=@($items|Where-Object{$_.Current.AutomationId -eq '1003' -and $_.Current.ClassName -eq 'Button'})
  if($list.Count -ne 1 -or $propsButton.Count -ne 1){throw 'Sound recording controls unavailable'}
  $listHwnd=[IntPtr]$list[0].Current.NativeWindowHandle
  $iid=[Guid]'618736E0-3C3D-11CF-810C-00AA00389B71'
  $acc=$null
  $hr=[AntiCensorSoundGui]::AccessibleObjectFromWindow($listHwnd,
      [Convert]::ToUInt32('FFFFFFFC',16),[ref]$iid,[ref]$acc)
  if($hr -ne 0 -or $null -eq $acc){throw 'No MSAA recording list'}
  $matching=@()
  for($i=1;$i -le $acc.accChildCount;$i++){
   try{if($acc.accName($i) -ceq 'CABLE Output'){$matching+= $i}}catch{}
  }
  if($matching.Count -ne 1){throw "Recording CABLE Output row count=$($matching.Count)"}
  $acc.accSelect(3,[int]$matching[0])
  Start-Sleep -Milliseconds 500
  if(-not $propsButton[0].Current.IsEnabled){throw 'Properties button is disabled'}
  $propsHwnd=[IntPtr]$propsButton[0].Current.NativeWindowHandle
  if(-not [AntiCensorSoundGui]::PostMessage($propsHwnd,0x00F5,[IntPtr]::Zero,[IntPtr]::Zero)){
   throw 'Properties button click failed'
  }
  for($i=0;$i -lt 28;$i++){
   Start-Sleep -Milliseconds 250
   $propertyDialogs=Get-PropertiesDialogs
   if($propertyDialogs.Count -ge 1){break}
  }
 }
 if($propertyDialogs.Count -ne 1){throw "VB-CABLE properties dialog missing: $($propertyDialogs.Count)"}
 $dialog=$propertyDialogs[0]
 $fields=$dialog.FindAll([System.Windows.Automation.TreeScope]::Descendants,[System.Windows.Automation.Condition]::TrueCondition)
 $edits=@($fields|Where-Object{$_.Current.AutomationId -eq '1201' -and $_.Current.ClassName -eq 'Edit' -and $_.Current.IsEnabled})
 $accept=@($fields|Where-Object{$_.Current.AutomationId -eq '1' -and $_.Current.ClassName -eq 'Button' -and $_.Current.Name -eq 'Aceptar' -and $_.Current.IsEnabled})
 $cancel=@($fields|Where-Object{$_.Current.AutomationId -eq '2' -and $_.Current.ClassName -eq 'Button' -and $_.Current.Name -eq 'Cancelar' -and $_.Current.IsEnabled})
 if($edits.Count -ne 1 -or $accept.Count -ne 1 -or $cancel.Count -ne 1){throw 'Wrong properties form controls'}
 $editHwnd=[IntPtr]$edits[0].Current.NativeWindowHandle
 $acceptHwnd=[IntPtr]$accept[0].Current.NativeWindowHandle
 $cancelHwnd=[IntPtr]$cancel[0].Current.NativeWindowHandle
 if($editHwnd -eq [IntPtr]::Zero -or $acceptHwnd -eq [IntPtr]::Zero){throw 'Missing controls'}
 $before=New-Object System.Text.StringBuilder 255
 [void][AntiCensorSoundGui]::GetText($editHwnd,0x000D,[IntPtr]255,$before)
 Info ("FIELD_BEFORE="+$before.ToString())
 if($before.ToString() -cne 'CABLE Output'){throw 'Unexpected field value: not the VB-CABLE basic mic'}
 [void][AntiCensorSoundGui]::SetText($editHwnd,0x000C,[IntPtr]::Zero,$desired)
 $renameSet=$true
 Start-Sleep -Milliseconds 300
 $after=New-Object System.Text.StringBuilder 255
 [void][AntiCensorSoundGui]::GetText($editHwnd,0x000D,[IntPtr]255,$after)
 Info ("FIELD_AFTER="+$after.ToString())
 if($after.ToString() -cne $desired){throw 'Windows Edit did not accept requested name'}
 if(-not [AntiCensorSoundGui]::PostMessage($acceptHwnd,0x00F5,[IntPtr]::Zero,[IntPtr]::Zero)){throw 'Could not apply Windows sound properties'}
 $renameSet=$false
 Start-Sleep -Seconds 2
 $updated=@(Get-PnpDevice -Class AudioEndpoint -PresentOnly | Where-Object{$_.InstanceId -eq $endpoint[0].InstanceId})
 if($updated.Count -ne 1){throw 'VB-CABLE endpoint disappeared after rename'}
 Info ("PNP_AFTER="+$updated[0].FriendlyName)
 if($updated[0].FriendlyName -like 'SolarisPKN-Mic Lelele*'){
  Info 'RESULT=RENAMED_AND_VERIFIED'
  Info 'ANTICENSOR_ENDPOINT_NAME=RENAMED'
 }else{
  Info 'RESULT=NOT_VERIFIED'
  Info 'ANTICENSOR_ENDPOINT_NAME=NOT_PERMITTED'
 }
 } catch {
 Info ('ERROR='+$_.Exception.Message)
 Info 'ANTICENSOR_ENDPOINT_NAME=NOT_PERMITTED'
 if($renameSet -and $dialog -and $cancelHwnd -ne [IntPtr]::Zero){
  try{[void][AntiCensorSoundGui]::PostMessage($cancelHwnd,0x00F5,[IntPtr]::Zero,[IntPtr]::Zero)}catch{}
 }
} finally {
 # Close only the Sound dialog we opened, never a preexisting user window.
 if($openedByScript){
  try{
   Start-Sleep -Milliseconds 350
   $openDialogs=Get-SoundDialogs
   if($openDialogs.Count -eq 1 -and $openDialogs[0].Current.IsEnabled){
    $items=$openDialogs[0].FindAll([System.Windows.Automation.TreeScope]::Children,[System.Windows.Automation.Condition]::TrueCondition)
    $cancel=@($items|Where-Object{$_.Current.AutomationId -eq '2' -and $_.Current.Name -in @('Cancelar','Cancel')})
    if($cancel.Count -eq 1){
     [void][AntiCensorSoundGui]::PostMessage([IntPtr]$cancel[0].Current.NativeWindowHandle,0x00F5,[IntPtr]::Zero,[IntPtr]::Zero)
    }
   }
  } catch {}
 }
 $output|Set-Content -LiteralPath $log -Encoding UTF8
 foreach($line in $output){if($line -like 'ANTICENSOR_ENDPOINT_NAME=*'){Write-Output $line}}
}
