package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Listen       string        `json:"listen"`
	HistoryLimit int           `json:"history_limit"`
	Twitch       TwitchConfig  `json:"twitch"`
	YouTube      YouTubeConfig `json:"youtube"`
	Rumble       RumbleConfig  `json:"rumble"`
	Ingest       IngestConfig  `json:"ingest"`
}
type TwitchConfig struct {
	Enabled  bool   `json:"enabled"`
	Channel  string `json:"channel"`
	Nick     string `json:"nick"`
	TokenEnv string `json:"token_env"`
}
type YouTubeConfig struct {
	Enabled         bool   `json:"enabled"`
	LiveChatID      string `json:"live_chat_id"`
	APIKeyEnv       string `json:"api_key_env"`
	PollSeconds     int    `json:"poll_seconds"`
	DailyQuotaUnits int    `json:"daily_quota_units"`
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

func readConfig(path string) (Config, error) {
	var c Config
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, err
	}
	if c.Listen == "" {
		c.Listen = "127.0.0.1:8765"
	}
	if !strings.HasPrefix(c.Listen, "127.0.0.1:") && !strings.HasPrefix(c.Listen, "localhost:") && !strings.HasPrefix(c.Listen, "[::1]:") {
		return c, errors.New("listen must be a loopback address (127.0.0.1, localhost or [::1])")
	}
	if c.HistoryLimit <= 0 { c.HistoryLimit = 250 }
	if c.HistoryLimit > 1000 { c.HistoryLimit = 1000 }
	if c.Twitch.Enabled && (c.Twitch.Channel == "" || c.Twitch.Nick == "" || os.Getenv(c.Twitch.TokenEnv) == "") {
		return c, errors.New("twitch: channel, nick and token environment variable required")
	}
	if c.YouTube.Enabled {
		if c.YouTube.LiveChatID == "" || os.Getenv(c.YouTube.APIKeyEnv) == "" {
			return c, errors.New("youtube: live_chat_id and API key environment variable required")
		}
		if c.YouTube.PollSeconds < 20 { c.YouTube.PollSeconds = 20 }
		if c.YouTube.DailyQuotaUnits <= 0 { c.YouTube.DailyQuotaUnits = 6000 }
		if c.YouTube.DailyQuotaUnits > 10000 { c.YouTube.DailyQuotaUnits = 10000 }
	}
	if c.Rumble.Enabled {
		if os.Getenv(c.Rumble.APIURLEnv) == "" { return c, errors.New("rumble: API URL environment variable required") }
		if c.Rumble.PollSeconds < 15 { c.Rumble.PollSeconds = 15 }
	}
	if c.Ingest.Enabled && len(os.Getenv(c.Ingest.TokenEnv)) < 24 {
		return c, fmt.Errorf("ingest: %s must contain a secret of at least 24 characters", c.Ingest.TokenEnv)
	}
	return c, nil
}
