package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	PanelAuth    PanelAuthConfig `json:"panel_auth"`
	Outbound     OutboundConfig  `json:"outbound"`
	Demo         bool            `json:"-"`
	Listen       string          `json:"listen"`
	HistoryLimit int             `json:"history_limit"`
	Twitch       TwitchConfig    `json:"twitch"`
	YouTube      YouTubeConfig   `json:"youtube"`
	TikTok       TikTokConfig    `json:"tiktok"`
	Kick         KickConfig      `json:"kick"`
	Facebook     FacebookConfig  `json:"facebook"`
	Rumble       RumbleConfig    `json:"rumble"`
	Ingest       IngestConfig    `json:"ingest"`
}
type TwitchConfig struct {
	Enabled      bool   `json:"enabled"`
	Channel      string `json:"channel"`
	Nick         string `json:"nick"`
	TokenEnv     string `json:"token_env"`
	UseSavedAuth bool   `json:"use_saved_twitch_auth"`
}
type YouTubeConfig struct {
	Enabled         bool   `json:"enabled"`
	LiveChatID      string `json:"live_chat_id"`
	VideoID         string `json:"video_id"`
	APIKeyEnv       string `json:"api_key_env"`
	OAuthTokenEnv   string `json:"oauth_token_env"`
	UseSavedAuth    bool   `json:"use_saved_google_auth"`
	PollSeconds     int    `json:"poll_seconds"`
	DailyQuotaUnits int    `json:"daily_quota_units"`
}
type TikTokConfig struct {
	Enabled  bool   `json:"enabled"`
	Username string `json:"username"`
}
type KickConfig struct {
	Enabled    bool   `json:"enabled"`
	Channel    string `json:"channel"`
	ChatroomID string `json:"chatroom_id"`
}
type FacebookConfig struct {
	Enabled        bool   `json:"enabled"`
	LiveVideoID    string `json:"live_video_id"`
	PageID         string `json:"page_id"`
	AccessTokenEnv string `json:"access_token_env"`
	PollSeconds    int    `json:"poll_seconds"`
}
type RumbleConfig struct {
	Enabled     bool   `json:"enabled"`
	APIURLEnv   string `json:"api_url_env"`
	PollSeconds int    `json:"poll_seconds"`
}
type IngestConfig struct {
	Enabled  bool   `json:"enabled"`
	TokenEnv string `json:"token_env"`
}
type PanelAuthConfig struct {
	Enabled     bool   `json:"enabled"`
	TokenEnv    string `json:"token_env"`
	OBSTokenEnv string `json:"obs_token_env"`
}

func readConfig(path string) (Config, error) {
	var c Config
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if err = json.Unmarshal(b, &c); err != nil {
		return c, err
	}
	if c.Listen == "" {
		c.Listen = "127.0.0.1:8765"
	}
	if !strings.HasPrefix(c.Listen, "127.0.0.1:") && !strings.HasPrefix(c.Listen, "localhost:") && !strings.HasPrefix(c.Listen, "[::1]:") {
		return c, errors.New("listen must use loopback")
	}
	if c.HistoryLimit <= 0 {
		c.HistoryLimit = 250
	}
	if c.HistoryLimit > 1000 {
		c.HistoryLimit = 1000
	}
	if c.PanelAuth.Enabled {
		if c.PanelAuth.TokenEnv == "" || len(os.Getenv(c.PanelAuth.TokenEnv)) < 32 {
			return c, errors.New("panel_auth: token_env must contain a secret of at least 32 characters")
		}
		if c.PanelAuth.OBSTokenEnv != "" {
			obsKey := os.Getenv(c.PanelAuth.OBSTokenEnv)
			if len(obsKey) < 32 || obsKey == os.Getenv(c.PanelAuth.TokenEnv) {
				return c, errors.New("panel_auth: obs_token_env must be at least 32 characters and different from the panel token")
			}
		}
	}
	if c.Outbound.Enabled && !c.PanelAuth.Enabled {
		return c, errors.New("outbound: panel_auth must be enabled before real chat sending")
	}
	if c.Outbound.KickBroadcasterUserID < 0 {
		return c, errors.New("outbound: invalid Kick broadcaster_user_id")
	}
	if c.Twitch.Enabled && c.Twitch.Channel == "" {
		return c, errors.New("twitch: channel required")
	}
	if c.Twitch.Enabled && !c.Twitch.UseSavedAuth && (c.Twitch.Nick == "" || c.Twitch.TokenEnv == "" || os.Getenv(c.Twitch.TokenEnv) == "") {
		return c, errors.New("twitch: nick and token environment required unless using saved Twitch OAuth")
	}
	if c.YouTube.Enabled {
		hasYouTubeAPIKey := c.YouTube.APIKeyEnv != "" && os.Getenv(c.YouTube.APIKeyEnv) != ""
		hasYouTubeOAuth := c.YouTube.UseSavedAuth || (c.YouTube.OAuthTokenEnv != "" && os.Getenv(c.YouTube.OAuthTokenEnv) != "")
		if !hasYouTubeAPIKey && !hasYouTubeOAuth {
			return c, errors.New("youtube: API key or OAuth token environment required")
		}
		if c.YouTube.LiveChatID == "" && c.YouTube.VideoID == "" && !hasYouTubeOAuth {
			return c, errors.New("youtube: current stream autodiscovery requires OAuth token")
		}
		if c.YouTube.PollSeconds < 20 {
			c.YouTube.PollSeconds = 20
		}
		if c.YouTube.DailyQuotaUnits <= 0 {
			c.YouTube.DailyQuotaUnits = 6000
		}
		if c.YouTube.DailyQuotaUnits > 10000 {
			c.YouTube.DailyQuotaUnits = 10000
		}
	}
	if c.TikTok.Enabled {
		c.TikTok.Username = strings.TrimPrefix(strings.TrimSpace(c.TikTok.Username), "@")
		if c.TikTok.Username == "" || strings.ContainsAny(c.TikTok.Username, " /\\\r\n\t") {
			return c, errors.New("tiktok: valid username required")
		}
	}
	if c.Kick.Enabled {
		c.Kick.Channel = strings.TrimSpace(c.Kick.Channel)
		if c.Kick.Channel == "" && c.Kick.ChatroomID == "" {
			return c, errors.New("kick: channel or chatroom_id required")
		}
		if strings.ContainsAny(c.Kick.Channel, " /\\\r\n\t") {
			return c, errors.New("kick: invalid channel")
		}
		if len(c.Kick.ChatroomID) > 20 {
			return c, errors.New("kick: invalid chatroom_id")
		}
		for _, digit := range c.Kick.ChatroomID {
			if digit < '0' || digit > '9' {
				return c, errors.New("kick: chatroom_id must be numeric")
			}
		}
	}
	if c.Facebook.Enabled {
		if (c.Facebook.LiveVideoID == "" && c.Facebook.PageID == "") || c.Facebook.AccessTokenEnv == "" || os.Getenv(c.Facebook.AccessTokenEnv) == "" {
			return c, errors.New("facebook: live_video_id or page_id, and Page access token required")
		}
		for _, digit := range c.Facebook.PageID {
			if digit < '0' || digit > '9' {
				return c, errors.New("facebook: numeric page_id required")
			}
		}
		for _, digit := range c.Facebook.LiveVideoID {
			if digit < '0' || digit > '9' {
				return c, errors.New("facebook: numeric live_video_id required")
			}
		}
		if c.Facebook.PollSeconds < 10 {
			c.Facebook.PollSeconds = 10
		}
	}
	if c.Rumble.Enabled {
		if c.Rumble.APIURLEnv == "" || os.Getenv(c.Rumble.APIURLEnv) == "" {
			return c, errors.New("rumble: API URL environment variable required")
		}
		if c.Rumble.PollSeconds < 15 {
			c.Rumble.PollSeconds = 15
		}
	}
	if c.Ingest.Enabled && (c.Ingest.TokenEnv == "" || len(os.Getenv(c.Ingest.TokenEnv)) < 24) {
		return c, fmt.Errorf("ingest: %s must contain a secret of at least 24 characters", c.Ingest.TokenEnv)
	}
	return c, nil
}
