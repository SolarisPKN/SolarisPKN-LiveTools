package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// youtubeActiveChatID queries the authenticated creator's current broadcast.
// YouTube's liveBroadcasts.list(mine=true) requires OAuth, not an API key.
// It is intentionally read-only and does not start or modify a broadcast.
func youtubeActiveChatID(ctx context.Context, client *http.Client, token string) (string, error) {
	q := url.Values{
		"part":            {"snippet"},
		"mine":            {"true"},
		"broadcastStatus": {"active"},
		"maxResults":      {"10"},
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/youtube/v3/liveBroadcasts?"+q.Encode(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("youtube liveBroadcasts.list HTTP %d", resp.StatusCode)
	}
	var payload struct {
		Items []struct {
			Snippet struct {
				LiveChatID string `json:"liveChatId"`
			} `json:"snippet"`
		} `json:"items"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 512<<10)).Decode(&payload); err != nil {
		return "", err
	}
	for _, item := range payload.Items {
		if item.Snippet.LiveChatID != "" {
			return item.Snippet.LiveChatID, nil
		}
	}
	return "", nil // offline or live chat disabled
}

// facebookActiveLiveVideoID asks Meta for the active live video of a Page.
// The Page ID is a stable identifier, while the live_video_id changes per broadcast.
func facebookActiveLiveVideoID(ctx context.Context, client *http.Client, pageID, token string) (string, error) {
	q := url.Values{"fields": {"id,status"}, "limit": {"25"}}
	req, err := http.NewRequestWithContext(ctx, "GET",
		"https://graph.facebook.com/v27.0/"+url.PathEscape(pageID)+"/live_videos?"+q.Encode(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("facebook Page live_videos HTTP %d", resp.StatusCode)
	}
	var payload struct {
		Data []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 512<<10)).Decode(&payload); err != nil {
		return "", err
	}
	for _, v := range payload.Data {
		status := strings.ToUpper(strings.TrimSpace(v.Status))
		if v.ID != "" && (status == "LIVE" || status == "LIVE_NOW") {
			return v.ID, nil
		}
	}
	return "", nil // no active live
}
