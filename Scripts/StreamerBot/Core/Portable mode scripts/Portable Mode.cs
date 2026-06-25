/*
 * Streamer.bot Installation Path Recorder
 * Compatible with Streamer.bot 1.2+
 *
 * This script saves the installation directory of Streamer.bot
 * into a global variable for use in other actions.
 *
 * Configuration (optional):
 * - Global var "installPathVar": override the variable name (default: "InstallationPath")
 * - Global var "installPathValue": override the value to save (default: current directory)
 *
 * Repository: https://github.com/SolarisPKN/SolarisPKN-LiveTools
 */

using System;

public class CPHInline
{
    // Default variable name for the installation path
    private const string DEFAULT_VAR_NAME = "InstallationPath";

    public bool Execute()
    {
        try
        {
            // 1. Get configuration (allow override via global vars or args)
            string varName = GetConfig("installPathVar", DEFAULT_VAR_NAME);
            string pathValue = GetConfig("installPathValue", Environment.CurrentDirectory);

            // 2. Save to global variable
            CPH.SetGlobalVar(varName, pathValue, true);

            // 3. Log confirmation
            CPH.LogInfo($"✅ Installation path saved to global var '{varName}': {pathValue}");

            return true;
        }
        catch (Exception ex)
        {
            CPH.LogError($"❌ Error saving installation path: {ex.Message}");
            return false;
        }
    }

    // ------------------------------------------------------------------
    // Helper: Get configuration value (global var -> args -> default)
    // ------------------------------------------------------------------
    private string GetConfig(string key, string defaultValue)
    {
        // Try global var
        string val = CPH.GetGlobalVar<string>(key, false);
        if (!string.IsNullOrEmpty(val))
            return val;

        // Try args
        if (args.ContainsKey(key) && args[key] != null)
        {
            string argVal = args[key].ToString();
            if (!string.IsNullOrEmpty(argVal))
                return argVal;
        }

        return defaultValue;
    }
}