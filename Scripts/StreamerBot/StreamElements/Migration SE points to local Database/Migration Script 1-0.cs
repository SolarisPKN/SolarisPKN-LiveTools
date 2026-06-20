/*
 * Streamer.bot - SQLite Points Migrator
 * Compatible with Streamer.bot 1.2+
 *
 * Description:
 *   This script adds points to a local SQLite database from various sources:
 *   - Twitch user variable (per user)
 *   - Global variable
 *   - Arguments passed from a previous action
 *
 * Configuration (via Global Variables or Arguments):
 *   - dbFile         : Name of the SQLite database file (default: "PuntosRecompensas.db")
 *   - pointsVar      : Name of the variable containing points to add (default: "sepoints")
 *   - userVar        : Name of the variable containing the username (default: "userName")
 *   - sqliteDllPath  : Full path to System.Data.SQLite.dll (optional, auto-search if not set)
 *
 * Required files (place in Streamer.bot folder or subfolder lib/, dependencies/, plugins/):
 *   - System.Data.SQLite.dll
 *   - SQLite.Interop.dll
 *
 * Repository:
 *   https://github.com/SolarisPKN/SolarisPKN-LiveTools
 */

using System;
using System.IO;
using System.Reflection;

public class CPHInline
{
    // Default configuration (can be overridden via global vars or args)
    private const string DEFAULT_DB_FILE = "PuntosRecompensas.db";
    private const string DEFAULT_POINTS_VAR = "sepoints";
    private const string DEFAULT_USER_VAR = "userName";

    public bool Execute()
    {
        // 1. Get configuration (allows customization without code changes)
        string dbFile = GetConfig("dbFile", DEFAULT_DB_FILE);
        string pointsVar = GetConfig("pointsVar", DEFAULT_POINTS_VAR);
        string userVar = GetConfig("userVar", DEFAULT_USER_VAR);

        // 2. Get username (same logic as original)
        string userName = GetUserName(userVar);
        if (string.IsNullOrEmpty(userName))
        {
            CPH.LogError("❌ Error: Could not get username.");
            return false;
        }

        CPH.LogInfo($"🔍 User: {userName}");
        CPH.LogInfo($"📁 DB path: {Path.GetFullPath(dbFile)}");

        // 3. Get points to add (pass userName to avoid re-fetching)
        int pointsToAdd = GetPointsToAdd(userName, pointsVar);
        if (pointsToAdd <= 0)
        {
            CPH.LogError($"❌ Error: No valid points value found for '{pointsVar}'.");
            CPH.LogInfo("ℹ️ Check that previous action stores points in: 1) Twitch user var, 2) Global var, or 3) args.");
            return false;
        }

        CPH.LogInfo($"💰 Points to add: {pointsToAdd}");

        // 4. Execute database update
        bool success = AddPointsToDatabase(dbFile, userName, pointsToAdd);
        if (success)
        {
            CPH.LogInfo($"✅ {pointsToAdd} points added to {userName}.");
            // Clear the Twitch user variable after migration
            CPH.SetTwitchUserVar(userName.ToLower(), pointsVar, 0, false);
            CPH.LogInfo($"✅ Twitch user var '{pointsVar}' reset to 0 for {userName}.");
            return true;
        }
        else
        {
            CPH.LogWarn($"⚠️ Failed to add points to {userName}.");
            return false;
        }
    }

    // ------------------------------------------------------------------
    // Helper: Get configuration value (global var -> args -> default)
    // ------------------------------------------------------------------
    private string GetConfig(string key, string defaultValue)
    {
        string val = CPH.GetGlobalVar<string>(key, false);
        if (!string.IsNullOrEmpty(val)) return val;

        if (args.ContainsKey(key) && args[key] != null)
        {
            string argVal = args[key].ToString();
            if (!string.IsNullOrEmpty(argVal)) return argVal;
        }

        return defaultValue;
    }

    // ------------------------------------------------------------------
    // Get username from various sources (same as original)
    // ------------------------------------------------------------------
    private string GetUserName(string userVarName)
    {
        // 1. Global var (e.g., %userName%)
        string userName = CPH.GetGlobalVar<string>(userVarName, false);
        if (!string.IsNullOrEmpty(userName))
        {
            CPH.LogInfo($"✅ Username from global var '{userVarName}': {userName}");
            return userName;
        }

        // 2. Arguments (common keys)
        string[] possibleKeys = { "userName", "targetUser", "user", "login", "broadcasterUserLogin", "rawInput", "input" };
        foreach (string key in possibleKeys)
        {
            if (args.ContainsKey(key) && args[key] != null)
            {
                string val = args[key].ToString();
                if (!string.IsNullOrEmpty(val))
                {
                    CPH.LogInfo($"✅ Username from arg '{key}': {val}");
                    return val;
                }
            }
        }

        CPH.LogError("❌ No username found.");
        return string.Empty;
    }

    // ------------------------------------------------------------------
    // Get points to add (priority: Twitch user var -> Global var -> args)
    // Now receives userName to avoid double-fetching
    // ------------------------------------------------------------------
    private int GetPointsToAdd(string userName, string pointsVar)
    {
        int points = 0;

        // 1. Twitch user variable (per user)
        string userVarStr = CPH.GetTwitchUserVar<string>(userName.ToLower(), pointsVar, false);
        if (!string.IsNullOrEmpty(userVarStr) && int.TryParse(userVarStr, out int fromUser))
        {
            points = fromUser;
            CPH.LogInfo($"✅ Points from Twitch user var '{pointsVar}': {points}");
            return points;
        }

        // 2. Global variable
        string globalStr = CPH.GetGlobalVar<string>(pointsVar, false);
        if (!string.IsNullOrEmpty(globalStr) && int.TryParse(globalStr, out int fromGlobal))
        {
            points = fromGlobal;
            CPH.LogInfo($"✅ Points from global var '{pointsVar}': {points}");
            return points;
        }

        // 3. Arguments
        if (args.ContainsKey(pointsVar) && args[pointsVar] != null)
        {
            string argStr = args[pointsVar].ToString();
            if (int.TryParse(argStr, out int fromArg))
            {
                points = fromArg;
                CPH.LogInfo($"✅ Points from arg '{pointsVar}': {points}");
                return points;
            }
        }

        return 0; // not found
    }

    // ------------------------------------------------------------------
    // Database logic (same as original, but with dynamic DLL location)
    // ------------------------------------------------------------------
    private bool AddPointsToDatabase(string databaseFile, string userName, int pointsToAdd)
    {
        try
        {
            // Locate System.Data.SQLite.dll
            string sqliteDllPath = LocateSqliteDll();
            if (string.IsNullOrEmpty(sqliteDllPath))
            {
                CPH.LogError("❌ System.Data.SQLite.dll not found. Please place it in Streamer.bot folder or a subfolder (lib/, dependencies/, plugins/).");
                return false;
            }

            CPH.LogInfo($"🔍 Loading SQLite from: {sqliteDllPath}");
            Assembly sqliteAssembly = Assembly.LoadFrom(sqliteDllPath);
            Type sqliteConnectionType = sqliteAssembly.GetType("System.Data.SQLite.SQLiteConnection");
            Type sqliteCommandType = sqliteAssembly.GetType("System.Data.SQLite.SQLiteCommand");

            if (sqliteConnectionType == null || sqliteCommandType == null)
            {
                CPH.LogError("❌ SQLite types not found.");
                return false;
            }

            string connectionString = $"Data Source={databaseFile};Version=3;";
            object connection = Activator.CreateInstance(sqliteConnectionType, new object[] { connectionString });

            // Open connection
            MethodInfo openMethod = sqliteConnectionType.GetMethod("Open", Type.EmptyTypes);
            if (openMethod == null)
            {
                CPH.LogError("❌ Open method not found.");
                return false;
            }
            openMethod.Invoke(connection, null);
            CPH.LogInfo("🔌 Database opened.");

            try
            {
                // Create table if not exists
                object createCmd = Activator.CreateInstance(sqliteCommandType, new object[] { connection });
                string createSql = @"
                    CREATE TABLE IF NOT EXISTS Usuarios (
                        Id TEXT PRIMARY KEY,
                        Puntos INTEGER NOT NULL DEFAULT 0,
                        UltimaActualizacion DATETIME DEFAULT CURRENT_TIMESTAMP
                    )";
                PropertyInfo cmdTextProp = sqliteCommandType.GetProperty("CommandText", typeof(string));
                if (cmdTextProp == null)
                {
                    CPH.LogError("❌ CommandText property not found.");
                    return false;
                }
                cmdTextProp.SetValue(createCmd, createSql);
                MethodInfo execNonQuery = sqliteCommandType.GetMethod("ExecuteNonQuery", Type.EmptyTypes);
                if (execNonQuery == null)
                {
                    CPH.LogError("❌ ExecuteNonQuery method not found.");
                    return false;
                }
                execNonQuery.Invoke(createCmd, null);
                CPH.LogInfo("🗃️ Table created/verified.");

                // Dispose createCmd
                MethodInfo dispose = sqliteCommandType.GetMethod("Dispose", Type.EmptyTypes);
                if (dispose != null) dispose.Invoke(createCmd, null);

                // Insert or replace adding points
                object insertCmd = Activator.CreateInstance(sqliteCommandType, new object[] { connection });
                string insertSql = @"
                    INSERT OR REPLACE INTO Usuarios (Id, Puntos) 
                    VALUES (@id, COALESCE((SELECT Puntos FROM Usuarios WHERE Id = @id), 0) + @puntos)";
                cmdTextProp.SetValue(insertCmd, insertSql);

                // Add parameters
                PropertyInfo paramsProp = sqliteCommandType.GetProperty("Parameters");
                if (paramsProp == null)
                {
                    CPH.LogError("❌ Parameters property not found.");
                    return false;
                }
                object parameters = paramsProp.GetValue(insertCmd);
                MethodInfo addWithValue = parameters.GetType().GetMethod("AddWithValue", new Type[] { typeof(string), typeof(object) });
                if (addWithValue == null)
                {
                    CPH.LogError("❌ AddWithValue method not found.");
                    return false;
                }
                addWithValue.Invoke(parameters, new object[] { "@id", userName });
                addWithValue.Invoke(parameters, new object[] { "@puntos", pointsToAdd });

                int rowsAffected = (int)execNonQuery.Invoke(insertCmd, null);
                CPH.LogInfo($"📊 Rows affected: {rowsAffected}");

                if (dispose != null) dispose.Invoke(insertCmd, null);

                return rowsAffected > 0;
            }
            finally
            {
                // Close connection
                MethodInfo closeMethod = sqliteConnectionType.GetMethod("Close", Type.EmptyTypes);
                if (closeMethod != null) closeMethod.Invoke(connection, null);
                MethodInfo disposeConn = sqliteConnectionType.GetMethod("Dispose", Type.EmptyTypes);
                if (disposeConn != null) disposeConn.Invoke(connection, null);
                CPH.LogInfo("🔒 Database closed.");
            }
        }
        catch (Exception ex)
        {
            CPH.LogError($"❌ DB error: {ex.Message}");
            CPH.LogError($"Stack: {ex.StackTrace}");
            return false;
        }
    }

    // ------------------------------------------------------------------
    // Locate System.Data.SQLite.dll in common locations
    // ------------------------------------------------------------------
    private string LocateSqliteDll()
    {
        // 1. If a custom path is configured via global var or args
        string configuredPath = GetConfig("sqliteDllPath", "");
        if (!string.IsNullOrEmpty(configuredPath) && File.Exists(configuredPath))
            return configuredPath;

        // 2. Search in Streamer.bot base directory and subfolders
        string baseDir = AppDomain.CurrentDomain.BaseDirectory;
        string[] subdirs = { "", "lib", "dependencies", "plugins" };
        foreach (string sub in subdirs)
        {
            string path = Path.Combine(baseDir, sub, "System.Data.SQLite.dll");
            if (File.Exists(path))
                return path;
        }

        // 3. Also check the executing script's directory (just in case)
        string executingDir = Path.GetDirectoryName(Assembly.GetExecutingAssembly().Location);
        if (!string.IsNullOrEmpty(executingDir) && executingDir != baseDir)
        {
            string localPath = Path.Combine(executingDir, "System.Data.SQLite.dll");
            if (File.Exists(localPath))
                return localPath;
        }

        return null;
    }
}