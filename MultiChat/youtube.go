package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

type youtubeResponse struct {
	NextPageToken         string `json:"nextPageToken"`
	PollingIntervalMillis int    `json:"pollingIntervalMillis"`
	Items                 []struct {
		ID      string `json:"id"`
		Snippet struct {
			Type           string    `json:"type"`
			DisplayMessage string    `json:"displayMessage"`
			PublishedAt    time.Time `json:"publishedAt"`
		} `json:"snippet"`
		AuthorDetails struct {
			DisplayName string `json:"displayName"`
		} `json:"authorDetails"`
	} `json:"items"`
}

func youtubeLookupLiveChatID(ctx context.Context, client *http.Client, videoID, apiKey, oauthToken string) (string, error) {
	q := url.Values{"part": {"liveStreamingDetails"}, "id": {videoID}}
	if apiKey != "" {
		q.Set("key", apiKey)
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/youtube/v3/videos?"+q.Encode(), nil)
	if err != nil {
		return "", err
	}
	if oauthToken != "" {
		req.Header.Set("Authorization", "Bearer "+oauthToken)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("YouTube videos.list HTTP %d", resp.StatusCode)
	}
	var data struct {
		Items []struct {
			LiveStreamingDetails struct {
				ActiveLiveChatID string `json:"activeLiveChatId"`
			} `json:"liveStreamingDetails"`
		} `json:"items"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 512<<10)).Decode(&data); err != nil {
		return "", err
	}
	if len(data.Items) == 0 {
		return "", nil
	}
	return data.Items[0].LiveStreamingDetails.ActiveLiveChatID, nil
}
func youtubeLoop(ctx context.Context, c YouTubeConfig, h *Hub, logf func(string, ...any)) {
	defer h.clearConnection("youtube")
	client := &http.Client{Timeout: 12 * time.Second}
	var pageToken string
	chatID := c.LiveChatID
	oauthToken := os.Getenv(c.OAuthTokenEnv)
	autoMode := c.LiveChatID == "" && c.VideoID == "" && (oauthToken != "" || c.UseSavedAuth)
	var lastAutoLookup time.Time
	windowStart := time.Now()
	spent := 0
	base := time.Duration(c.PollSeconds) * time.Second
	wait := time.Duration(0)
	for ctx.Err() == nil {
		if !sleepContext(ctx, wait) {
			return
		}
		if c.UseSavedAuth {
			fresh, err := currentGoogleToken(ctx)
			if err != nil {
				logf("[youtube] saved Google authorization unavailable; use -link-google or refresh access")
				wait = time.Minute
				continue
			}
			oauthToken = fresh
		}
		if autoMode && (lastAutoLookup.IsZero() || time.Since(lastAutoLookup) >= time.Minute) {
			lastAutoLookup = time.Now()
			resolved, lookupErr := youtubeActiveChatID(ctx, client, oauthToken)
			if lookupErr != nil {
				logf("[youtube] live discovery unavailable; retrying")
				wait = time.Minute
				continue
			}
			if resolved != chatID {
				chatID = resolved
				pageToken = ""
				if resolved != "" {
					logf("[youtube] switched to current broadcast")
				}
			}
		}
		if chatID == "" {
			if autoMode {
				wait = time.Minute
				continue
			}
			resolved, lookupErr := youtubeLookupLiveChatID(ctx, client, c.VideoID, os.Getenv(c.APIKeyEnv), oauthToken)
			if lookupErr != nil {
				logf("[youtube] video lookup failed; retrying")
				wait = time.Minute
				continue
			}
			if resolved == "" {
				wait = time.Minute
				continue
			}
			chatID = resolved
			logf("[youtube] active chat identified from video ID")
		}
		if time.Since(windowStart) >= 24*time.Hour {
			windowStart = time.Now()
			spent = 0
		}
		if spent+5 > c.DailyQuotaUnits {
			logf("[youtube] local quota budget exhausted (%d units); stopping connector", spent)
			return
		}
		spent += 5
		q := url.Values{"part": {"snippet,authorDetails"}, "liveChatId": {chatID}, "maxResults": {"200"}}
		if apiKey := os.Getenv(c.APIKeyEnv); apiKey != "" {
			q.Set("key", apiKey)
		}
		if pageToken != "" {
			q.Set("pageToken", pageToken)
		}
		req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/youtube/v3/liveChat/messages?"+q.Encode(), nil)
		if err != nil {
			return
		}
		if oauthToken != "" {
			req.Header.Set("Authorization", "Bearer "+oauthToken)
		}
		resp, err := client.Do(req)
		if err != nil {
			logf("[youtube] request error: %v", err)
			wait = time.Minute
			continue
		}
		raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		resp.Body.Close()
		if readErr != nil {
			logf("[youtube] read error")
			wait = time.Minute
			continue
		}
		if resp.StatusCode != 200 {
			logf("[youtube] HTTP %d (quota, permissions or stream state); retry delayed", resp.StatusCode)
			wait = 2 * time.Minute
			if autoMode && (resp.StatusCode == 403 || resp.StatusCode == 404) {
				chatID = ""
				pageToken = ""
				lastAutoLookup = time.Time{}
				wait = time.Minute
				continue
			}
			if resp.StatusCode == 404 && c.VideoID != "" {
				chatID = ""
				pageToken = ""
				wait = time.Minute
				continue
			}
			if resp.StatusCode == 401 || resp.StatusCode == 403 || resp.StatusCode == 404 {
				return
			}
			continue
		}
		h.setConnection("youtube", chatID, 90*time.Second)
		var payload youtubeResponse
		if err := json.Unmarshal(raw, &payload); err != nil {
			logf("[youtube] invalid response")
			wait = time.Minute
			continue
		}
		for _, x := range payload.Items {
			h.publish(Message{ID: x.ID, Platform: "youtube", User: x.AuthorDetails.DisplayName, Text: x.Snippet.DisplayMessage, Kind: x.Snippet.Type, Time: x.Snippet.PublishedAt})
		}
		pageToken = payload.NextPageToken
		wait = base
		if serverWait := time.Duration(payload.PollingIntervalMillis) * time.Millisecond; serverWait > wait {
			wait = serverWait
		}
	}
}
