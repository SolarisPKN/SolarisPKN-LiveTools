using System;
using System.IO;
using System.Net;
using System.Text;

public class CPHInline
{
    // ==================== CONFIGURATION ====================
    // Replace with your StreamElements JWT token (get it from your account settings)
    private readonly string _jwtToken = "YOUR_JWT_TOKEN";
    // Replace with your channel ID (found in your StreamElements dashboard URL)
    private readonly string _channelId = "YOUR_CHANNEL_ID";
    // ======================================================

    public bool Execute()
    {
        try
        {
            string userName = GetUserName();
            if (string.IsNullOrEmpty(userName))
            {
                CPH.LogError("Failed to get username");
                return false;
            }

            CPH.LogInfo($"Fetching StreamElements points for: {userName}");
            int userPoints = GetUserPoints(userName);
            CPH.SetArgument("sepoints", userPoints);
            CPH.LogInfo($"Points retrieved for {userName}: {userPoints} - saved to %sepoints%");
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogError($"Script error: {ex.Message}");
            return false;
        }
    }

    private string GetUserName()
    {
        string userName = "";
        if (CPH.TryGetArg("username", out userName) && !string.IsNullOrEmpty(userName))
            return userName;
        if (CPH.TryGetArg("user", out userName) && !string.IsNullOrEmpty(userName))
            return userName;
        if (CPH.TryGetArg("targetUser", out userName) && !string.IsNullOrEmpty(userName))
            return userName;

        CPH.LogError("No username variable found (tried: username, user, targetUser)");
        return null;
    }

    private int GetUserPoints(string userName)
    {
        try
        {
            using (var webClient = new WebClient())
            {
                webClient.Headers["Authorization"] = $"Bearer {_jwtToken}";
                webClient.Headers["Content-Type"] = "application/json";
                webClient.Encoding = Encoding.UTF8;

                string url = $"https://api.streamelements.com/kappa/v2/points/{_channelId}/{userName.ToLower()}";
                CPH.LogInfo($"Requesting points from: {url}");

                string json = webClient.DownloadString(url);
                CPH.LogInfo($"API response: {json}");

                int points = ParsePointsFromJson(json);
                CPH.LogInfo($"Parsed points: {points}");
                return points;
            }
        }
        catch (WebException ex)
        {
            CPH.LogError($"Network error: {ex.Message}");
            if (ex.Response != null)
            {
                using (var stream = ex.Response.GetResponseStream())
                using (var reader = new StreamReader(stream))
                {
                    string errorResponse = reader.ReadToEnd();
                    CPH.LogError($"Error response: {errorResponse}");

                    if (errorResponse.Contains("\"message\""))
                    {
                        int start = errorResponse.IndexOf("\"message\"") + 10;
                        int end = errorResponse.IndexOf("\"", start);
                        if (end > start)
                        {
                            string errorMsg = errorResponse.Substring(start, end - start);
                            CPH.LogError($"API error message: {errorMsg}");
                        }
                    }
                }
            }
            return 0;
        }
        catch (Exception ex)
        {
            CPH.LogError($"Error getting points: {ex.Message}");
            return 0;
        }
    }

    private int ParsePointsFromJson(string json)
    {
        try
        {
            if (json.Contains("\"points\":"))
            {
                int pointsIndex = json.IndexOf("\"points\":") + 9;
                int endIndex = json.IndexOf(",", pointsIndex);
                if (endIndex == -1) endIndex = json.IndexOf("}", pointsIndex);
                if (endIndex > pointsIndex)
                {
                    string pointsStr = json.Substring(pointsIndex, endIndex - pointsIndex).Trim();
                    if (int.TryParse(pointsStr, out int points))
                        return points;
                }
            }

            if (json.Contains("\"data\""))
            {
                int dataIndex = json.IndexOf("\"data\"");
                if (dataIndex > 0)
                {
                    string dataSection = json.Substring(dataIndex);
                    if (dataSection.Contains("\"points\":"))
                    {
                        int pointsIndex = dataSection.IndexOf("\"points\":") + 9;
                        int endIndex = dataSection.IndexOf(",", pointsIndex);
                        if (endIndex == -1) endIndex = dataSection.IndexOf("}", pointsIndex);
                        if (endIndex > pointsIndex)
                        {
                            string pointsStr = dataSection.Substring(pointsIndex, endIndex - pointsIndex).Trim();
                            if (int.TryParse(pointsStr, out int points))
                                return points;
                        }
                    }
                }
            }

            if (json.Contains("\"user\""))
            {
                int userIndex = json.IndexOf("\"user\"");
                if (userIndex > 0)
                {
                    string userSection = json.Substring(userIndex);
                    if (userSection.Contains("\"points\":"))
                    {
                        int pointsIndex = userSection.IndexOf("\"points\":") + 9;
                        int endIndex = userSection.IndexOf(",", pointsIndex);
                        if (endIndex == -1) endIndex = userSection.IndexOf("}", pointsIndex);
                        if (endIndex > pointsIndex)
                        {
                            string pointsStr = userSection.Substring(pointsIndex, endIndex - pointsIndex).Trim();
                            if (int.TryParse(pointsStr, out int points))
                                return points;
                        }
                    }
                }
            }

            CPH.LogError("Could not find 'points' property in JSON");
            return 0;
        }
        catch (Exception ex)
        {
            CPH.LogError($"JSON parsing error: {ex.Message}");
            return 0;
        }
    }
}