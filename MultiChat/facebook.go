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

type facebookResponse struct {
	Data []struct {
		ID      string `json:"id"`
		Message string `json:"message"`
		From    struct {
			Name string `json:"name"`
		} `json:"from"`
		CreatedAt string `json:"created_time"`
	} `json:"data"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func parseFacebookMessages(raw []byte) ([]Message, error) {
	var response facebookResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, err
	}
	if response.Error != nil {
		return nil, fmt.Errorf("graph error %d", response.Error.Code)
	}
	result := make([]Message, 0, len(response.Data))
	// Graph recommends reverse chronological polling; publish in chronological order.
	for i := len(response.Data) - 1; i >= 0; i-- {
		x := response.Data[i]
		if x.ID == "" || x.Message == "" {
			continue
		}
		t, parseErr := time.Parse(time.RFC3339Nano, x.CreatedAt)
		if parseErr != nil {
			t, _ = time.Parse("2006-01-02T15:04:05-0700", x.CreatedAt)
		}
		result = append(result, Message{ID: x.ID, Platform: "facebook", User: x.From.Name, Text: x.Message, Kind: "chat", Time: t})
	}
	return result, nil
}
func facebookLoop(ctx context.Context, c FacebookConfig, h *Hub, logf func(string, ...any)) {
	defer h.clearConnection("facebook")
	client := &http.Client{Timeout: 12 * time.Second}
	delay := time.Duration(c.PollSeconds) * time.Second
	currentVideoID := c.LiveVideoID
	autoMode := c.PageID != ""
	var lastAutoLookup time.Time
	for ctx.Err() == nil {
		if autoMode && (lastAutoLookup.IsZero() || time.Since(lastAutoLookup) >= time.Minute) {
			lastAutoLookup = time.Now()
			resolved, err := facebookActiveLiveVideoID(ctx, client, c.PageID, os.Getenv(c.AccessTokenEnv))
			if err != nil {
				logf("[facebook] Page live discovery unavailable; retrying")
				if !sleepContext(ctx, time.Minute) {
					return
				}
				continue
			}
			if currentVideoID != resolved {
				currentVideoID = resolved
				if resolved != "" {
					logf("[facebook] switched to active Page LIVE video")
				}
			}
		}
		if currentVideoID == "" {
			if !sleepContext(ctx, 30*time.Second) {
				return
			}
			continue
		}
		q := url.Values{"fields": {"id,message,from{name},created_time"}, "limit": {"100"}, "order": {"reverse_chronological"}, "filter": {"stream"}}
		endpoint := "https://graph.facebook.com/v27.0/" + url.PathEscape(currentVideoID) + "/comments?" + q.Encode()
		req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
		if err != nil {
			return
		}
		req.Header.Set("Authorization", "Bearer "+os.Getenv(c.AccessTokenEnv))
		resp, err := client.Do(req)
		if err != nil {
			logf("[facebook] transport error (details withheld); retry")
			if !sleepContext(ctx, delay) {
				return
			}
			continue
		}
		raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		resp.Body.Close()
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			logf("[facebook] HTTP %d: page permissions/token require attention; stopping", resp.StatusCode)
			return
		}
		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			logf("[facebook] HTTP %d; backing off", resp.StatusCode)
			if !sleepContext(ctx, time.Minute) {
				return
			}
			continue
		}
		if resp.StatusCode != 200 || readErr != nil {
			logf("[facebook] HTTP %d or response error", resp.StatusCode)
			if !sleepContext(ctx, delay) {
				return
			}
			continue
		}
		h.setConnection("facebook", currentVideoID, 65*time.Second)
		msgs, err := parseFacebookMessages(raw)
		if err != nil {
			logf("[facebook] invalid Graph response")
			if !sleepContext(ctx, delay) {
				return
			}
			continue
		}
		for _, msg := range msgs {
			h.publish(msg)
		}
		if len(msgs) == 100 {
			logf("[facebook] page limit reached: high-volume chat may require shorter polling interval")
		}
		if !sleepContext(ctx, delay) {
			return
		}
	}
}
