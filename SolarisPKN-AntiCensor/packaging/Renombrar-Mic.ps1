# SPDX-License-Identifier: GPL-3.0-only
# SolarisPKN AntiCensor - restore our VB-CABLE capture endpoint's display name.
# This does not modify VB-Audio's signed INF, SYS or CAT files and never
# renames physical microphones, other virtual cables, or playback endpoints.
[CmdletBinding()]
param([switch] $Probe)

$ErrorActionPreference = 'Stop'
$DesiredName = 'SolarisPKN-Mic Lelele'

# The regular audio endpoint property store is independent of the signed
# third-party kernel driver. All API calls below are limited to eCapture.
$source = @'
using System;
using System.Collections.Generic;
using System.Runtime.InteropServices;

namespace SolarisPKNAntiCensor {
    [StructLayout(LayoutKind.Sequential)]
    public struct PROPERTYKEY {
        public Guid fmtid;
        public int pid;
        public PROPERTYKEY(string id, int number) { fmtid = new Guid(id); pid = number; }
    }

    [StructLayout(LayoutKind.Explicit, Size = 24)]
    public struct PROPVARIANT {
        [FieldOffset(0)] public ushort vt;
        [FieldOffset(8)] public IntPtr ptr;
    }

    [ComImport, Guid("BCDE0395-E52F-467C-8E3D-C4579291692E")]
    public class MMDeviceEnumerator { }

    [ComImport, Guid("A95664D2-9614-4F35-A746-DE8DB63617E6"),
     InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    public interface IMMDeviceEnumerator {
        void EnumAudioEndpoints(int dataFlow, int stateMask, out IMMDeviceCollection devices);
        void GetDefaultAudioEndpoint(int dataFlow, int role, out IMMDevice device);
        void GetDevice([MarshalAs(UnmanagedType.LPWStr)] string id, out IMMDevice device);
        void RegisterEndpointNotificationCallback(IntPtr client);
        void UnregisterEndpointNotificationCallback(IntPtr client);
    }

    [ComImport, Guid("0BD7A1BE-7A1A-44DB-8397-CC5392387B5E"),
     InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    public interface IMMDeviceCollection {
        void GetCount(out uint count);
        void Item(uint index, out IMMDevice device);
    }

    [ComImport, Guid("D666063F-1587-4E43-81F1-B948E807363F"),
     InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    public interface IMMDevice {
        void Activate(ref Guid iid, uint clsCtx, IntPtr activationParams, out IntPtr interfacePtr);
        void OpenPropertyStore(uint storageMode, out IPropertyStore propertyStore);
        void GetId(out IntPtr id);
        void GetState(out uint state);
    }

    [ComImport, Guid("886D8EEB-8CF2-4446-8D02-CDBA1DBDCF99"),
     InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    public interface IPropertyStore {
        void GetCount(out uint count);
        void GetAt(uint index, out PROPERTYKEY key);
        void GetValue(ref PROPERTYKEY key, out PROPVARIANT value);
        void SetValue(ref PROPERTYKEY key, ref PROPVARIANT value);
        void Commit();
    }

    public static class EndpointRenamer {
        // Microsoft Functiondiscoverykeys_devpkey.h:
        // DeviceDesc PID 2; FriendlyName PID 14.
        private static readonly PROPERTYKEY Description =
            new PROPERTYKEY("a45c254e-df1c-4efd-8020-67d146a850e0", 2);
        private static readonly PROPERTYKEY Friendly =
            new PROPERTYKEY("a45c254e-df1c-4efd-8020-67d146a850e0", 14);
        private static readonly PROPERTYKEY InterfaceFriendly =
            new PROPERTYKEY("026e516e-b814-414b-83cd-856d6fef4822", 2);

        [DllImport("ole32.dll", PreserveSig = false)]
        private static extern void PropVariantClear(ref PROPVARIANT value);

        private static string GetString(IPropertyStore store, PROPERTYKEY key) {
            PROPVARIANT value;
            store.GetValue(ref key, out value);
            try {
                if (value.ptr == IntPtr.Zero) return "";
                if (value.vt == 31) return Marshal.PtrToStringUni(value.ptr) ?? "";
                if (value.vt == 8) return Marshal.PtrToStringBSTR(value.ptr) ?? "";
                return "";
            } finally {
                PropVariantClear(ref value);
            }
        }

        private static bool ContainsVendor(string name) {
            return name.IndexOf("VB-Audio", StringComparison.OrdinalIgnoreCase) >= 0 ||
                   name.IndexOf("VB Audio", StringComparison.OrdinalIgnoreCase) >= 0;
        }

        public static string Apply(string desiredName, bool probeOnly) {
            IMMDeviceEnumerator enumerator = null;
            IMMDeviceCollection collection = null;
            var matches = new List<IMMDevice>();
            try {
                enumerator = (IMMDeviceEnumerator)new MMDeviceEnumerator();
                // eCapture = 1, all device states = 0xF.
                enumerator.EnumAudioEndpoints(1, 0xF, out collection);
                uint count;
                collection.GetCount(out count);
                for (uint i = 0; i < count; i++) {
                    IMMDevice device = null;
                    IPropertyStore props = null;
                    try {
                        collection.Item(i, out device);
                        device.OpenPropertyStore(0, out props); // read-only probe
                        string description = GetString(props, Description);
                        string friendly = GetString(props, Friendly);
                        string adapter = GetString(props, InterfaceFriendly);
                        // Both the basic VB-CABLE identity and Capture direction
                        // must match. Exclude A/B/C/D and any hardware microphone.
                        bool basic = description.Equals("CABLE Output", StringComparison.OrdinalIgnoreCase) ||
                                     friendly.StartsWith("CABLE Output (", StringComparison.OrdinalIgnoreCase);
                        bool vendor = ContainsVendor(adapter) || ContainsVendor(friendly) ||
                                      String.Equals(friendly, desiredName, StringComparison.OrdinalIgnoreCase);
                        if (basic && vendor) {
                            matches.Add(device);
                            device = null; // retained for second pass
                        }
                    } finally {
                        if (props != null) Marshal.ReleaseComObject(props);
                        if (device != null) Marshal.ReleaseComObject(device);
                    }
                }
                if (matches.Count == 0) return "MISSING";
                if (matches.Count > 1) return "AMBIGUOUS";
                IMMDevice target = matches[0];
                IPropertyStore oldStore = null;
                string oldName;
                try {
                    target.OpenPropertyStore(0, out oldStore);
                    oldName = GetString(oldStore, Friendly);
                } finally {
                    if (oldStore != null) Marshal.ReleaseComObject(oldStore);
                }
                if (String.Equals(oldName, desiredName, StringComparison.Ordinal))
                    return "ALREADY";
                if (probeOnly) return "READY";
                IPropertyStore writable = null;
                try {
                    // STGM_READWRITE = 2; user-visible endpoint name only.
                    target.OpenPropertyStore(2, out writable);
                    PROPERTYKEY key = Friendly;
                    PROPVARIANT v = new PROPVARIANT();
                    v.vt = 31; // VT_LPWSTR
                    v.ptr = Marshal.StringToCoTaskMemUni(desiredName);
                    try {
                        writable.SetValue(ref key, ref v);
                        writable.Commit();
                    } finally { Marshal.FreeCoTaskMem(v.ptr); }
                } finally {
                    if (writable != null) Marshal.ReleaseComObject(writable);
                }
                IPropertyStore verify = null;
                try {
                    target.OpenPropertyStore(0, out verify);
                    return GetString(verify, Friendly) == desiredName ? "RENAMED" : "NOT_PERSISTED";
                } finally {
                    if (verify != null) Marshal.ReleaseComObject(verify);
                }
            } finally {
                foreach (IMMDevice item in matches) Marshal.ReleaseComObject(item);
                if (collection != null) Marshal.ReleaseComObject(collection);
                if (enumerator != null) Marshal.ReleaseComObject(enumerator);
            }
        }
    }
}
'@

try {
    Add-Type -TypeDefinition $source -ErrorAction Stop
    $result = [SolarisPKNAntiCensor.EndpointRenamer]::Apply($DesiredName,[bool]$Probe)
    Write-Output "ANTICENSOR_ENDPOINT_NAME=$result"
    if ($result -eq 'AMBIGUOUS' -or $result -eq 'NOT_PERSISTED') { exit 3 }
    # MISSING is normal before VB-CABLE installation/restart. The tray retries.
    exit 0
} catch {
    Write-Error "AntiCensor endpoint rename failed: $($_.Exception.Message)"
    exit 4
}
