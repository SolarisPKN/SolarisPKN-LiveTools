/*
 * First Chatter Logger for Streamer.bot
 * Tracks first-time chatters in a JSON file.
 * 
 * Dependencies:
 * - Newtonsoft.Json (comes bundled with Streamer.bot)
 * 
 * Configuration (via Global Variables or Arguments):
 * - basePath        : Base directory (default: Streamer.bot installation folder)
 * - jsonFolder      : Subfolder for JSON file (default: "TWITCH")
 * - jsonFile        : JSON filename (default: "firstchatters.json")
 * 
 * Public Methods:
 * - Execute()          : Adds current chatter to the list if not already present.
 * - GetFirstChattersCount() : Sends total count in chat.
 * - CheckFirstChatter()     : Checks if a user is in the list and replies in chat.
 * 
 * Repository: https://github.com/SolarisPKN/SolarisPKN-LiveTools
 */

using System;
using System.IO;
using System.Collections.Generic;
using Newtonsoft.Json;

public class CPHInline
{
    // Defaults (can be overridden)
    private const string DEFAULT_BASE_VAR = "Instalacion";    // Existing global var from your previous script
    private const string DEFAULT_JSON_FOLDER = "TWITCH";
    private const string DEFAULT_JSON_FILE = "firstchatters.json";

    // ------------------------------------------------------------------
    // Main method: Add current user to the JSON list if new
    // ------------------------------------------------------------------
    public bool Execute()
    {
        try
        {
            // Get configuration
            string basePath = GetConfig("basePath", null);
            if (string.IsNullOrEmpty(basePath))
                basePath = GetConfig(DEFAULT_BASE_VAR, AppDomain.CurrentDomain.BaseDirectory);

            string jsonFolder = GetConfig("jsonFolder", DEFAULT_JSON_FOLDER);
            string jsonFile = GetConfig("jsonFile", DEFAULT_JSON_FILE);
            string fullPath = Path.Combine(basePath, jsonFolder, jsonFile);

            CPH.LogInfo($"📁 JSON path: {fullPath}");

            // Get username
            string userName = args.ContainsKey("userName") ? args["userName"].ToString() : null;
            if (string.IsNullOrEmpty(userName))
            {
                CPH.LogError("❌ No username received from trigger.");
                return false;
            }
            userName = userName.Trim().ToLower();

            // Ensure directory exists
            string directory = Path.GetDirectoryName(fullPath);
            if (!Directory.Exists(directory))
            {
                Directory.CreateDirectory(directory);
                CPH.LogInfo($"📂 Created directory: {directory}");
            }

            // Load existing list
            List<FirstChatter> chatters = new List<FirstChatter>();
            if (File.Exists(fullPath))
            {
                string jsonContent = File.ReadAllText(fullPath);
                chatters = JsonConvert.DeserializeObject<List<FirstChatter>>(jsonContent) ?? new List<FirstChatter>();
                CPH.LogInfo($"📖 Loaded {chatters.Count} chatters from JSON.");
            }
            else
            {
                CPH.LogInfo("📄 JSON file does not exist; will create new.");
            }

            // Check if user already exists
            bool userExists = chatters.Exists(c => c.Username.Equals(userName, StringComparison.OrdinalIgnoreCase));
            if (userExists)
            {
                CPH.LogInfo($"ℹ️ User '{userName}' already in list, ignoring.");
                return true;
            }

            // Add new user
            FirstChatter newChatter = new FirstChatter
            {
                Username = userName,
                FirstChatDate = DateTime.Now.ToString("yyyy-MM-dd HH:mm:ss"),
                AddedDate = DateTime.Now.ToString("yyyy-MM-dd HH:mm:ss"),
                Platform = "twitch"
            };
            chatters.Add(newChatter);

            // Save
            string newJson = JsonConvert.SerializeObject(chatters, Formatting.Indented);
            File.WriteAllText(fullPath, newJson);
            CPH.LogInfo($"✅ Added '{userName}' to JSON. Total: {chatters.Count}");

            // Optional: send a welcome message in chat
            // CPH.SendMessage($"🎉 Welcome for the first time, {userName}!");

            return true;
        }
        catch (Exception ex)
        {
            CPH.LogError($"💥 Error adding user: {ex.Message}");
            CPH.LogError($"🔍 Stack: {ex.StackTrace}");
            return false;
        }
    }

    // ------------------------------------------------------------------
    // Method: Get total count and send to chat
    // ------------------------------------------------------------------
    public bool GetFirstChattersCount()
    {
        try
        {
            string fullPath = GetJsonPath();
            if (!File.Exists(fullPath))
            {
                CPH.SendMessage("No first chatters have been recorded yet.");
                return true;
            }

            string json = File.ReadAllText(fullPath);
            List<FirstChatter> chatters = JsonConvert.DeserializeObject<List<FirstChatter>>(json);
            CPH.SendMessage($"🎉 We have {chatters.Count} first chatters in the community!");
            CPH.LogInfo($"📊 Stats: {chatters.Count} first chatters.");
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogError($"❌ Error getting stats: {ex.Message}");
            return false;
        }
    }

    // ------------------------------------------------------------------
    // Method: Check if a given user is a first chatter
    // ------------------------------------------------------------------
    public bool CheckFirstChatter()
    {
        try
        {
            string fullPath = GetJsonPath();
            string userName = args.ContainsKey("userName") ? args["userName"].ToString() : null;
            if (string.IsNullOrEmpty(userName))
            {
                CPH.LogError("❌ No username provided for check.");
                return false;
            }
            userName = userName.Trim().ToLower();

            if (!File.Exists(fullPath))
            {
                CPH.SendMessage($"❌ {userName} is not in the first chatters list.");
                return true;
            }

            string json = File.ReadAllText(fullPath);
            List<FirstChatter> chatters = JsonConvert.DeserializeObject<List<FirstChatter>>(json);
            bool isFirst = chatters.Exists(c => c.Username.Equals(userName, StringComparison.OrdinalIgnoreCase));

            if (isFirst)
            {
                CPH.SendMessage($"✅ {userName} is a first chatter!");
                CPH.LogInfo($"✅ {userName} is a first chatter.");
            }
            else
            {
                CPH.SendMessage($"❌ {userName} is not a first chatter.");
                CPH.LogInfo($"❌ {userName} is NOT a first chatter.");
            }
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogError($"❌ Error checking user: {ex.Message}");
            return false;
        }
    }

    // ------------------------------------------------------------------
    // Helper: Get configuration value (global var → args → default)
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
    // Helper: Build full JSON file path using current config
    // ------------------------------------------------------------------
    private string GetJsonPath()
    {
        string basePath = GetConfig("basePath", null);
        if (string.IsNullOrEmpty(basePath))
            basePath = GetConfig(DEFAULT_BASE_VAR, AppDomain.CurrentDomain.BaseDirectory);

        string jsonFolder = GetConfig("jsonFolder", DEFAULT_JSON_FOLDER);
        string jsonFile = GetConfig("jsonFile", DEFAULT_JSON_FILE);
        return Path.Combine(basePath, jsonFolder, jsonFile);
    }

    // ------------------------------------------------------------------
    // Data class
    // ------------------------------------------------------------------
    public class FirstChatter
    {
        public string Username { get; set; }
        public string FirstChatDate { get; set; }
        public string AddedDate { get; set; }
        public string Platform { get; set; }
    }
}