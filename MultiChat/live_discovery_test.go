package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type discoveryRoundTrip func(*http.Request) (*http.Response, error)

func (f discoveryRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
func discoveryClient(t *testing.T, check func(*http.Request), payload string) *http.Client {
	t.Helper()
	return &http.Client{Transport: discoveryRoundTrip(func(r *http.Request) (*http.Response, error) {
		check(r)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(payload)), Header: make(http.Header)}, nil
	})}
}
func TestYouTubeLiveAutoDiscovery(t *testing.T) {
	client := discoveryClient(t, func(r *http.Request) {
		if r.URL.Path != "/youtube/v3/liveBroadcasts" || r.URL.Query().Get("broadcastStatus") != "active" || r.URL.Query().Get("mine") != "true" {
			t.Errorf("unexpected YouTube lookup: %s", r.URL)
		}
		if r.Header.Get("Authorization") != "Bearer test-owner-oauth" {
			t.Errorf("OAuth missing")
		}
	}, `{"items":[{"snippet":{"liveChatId":"CHAT-NEW"}}]}`)
	chat, err := youtubeActiveChatID(context.Background(), client, "test-owner-oauth")
	if err != nil || chat != "CHAT-NEW" {
		t.Fatalf("unexpected chat: %q %v", chat, err)
	}
}
func TestYouTubeAutoDiscoveryOffline(t *testing.T) {
	client := discoveryClient(t, func(*http.Request) {}, `{"items":[]}`)
	id, err := youtubeActiveChatID(context.Background(), client, "t")
	if err != nil || id != "" {
		t.Fatalf("offline should return empty, got %q %v", id, err)
	}
}
func TestFacebookLiveAutoDiscovery(t *testing.T) {
	client := discoveryClient(t, func(r *http.Request) {
		if r.URL.Path != "/v27.0/123456789/live_videos" {
			t.Errorf("wrong Facebook Page path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer page-token" {
			t.Errorf("Page token missing")
		}
	}, `{"data":[{"id":"old","status":"VOD"},{"id":"new-live","status":"LIVE_NOW"}]}`)
	id, err := facebookActiveLiveVideoID(context.Background(), client, "123456789", "page-token")
	if err != nil || id != "new-live" {
		t.Fatalf("wrong active FB video: %q %v", id, err)
	}
}
func TestYouTubeConfigActiveDiscovery(t *testing.T) {
	t.Setenv("MULTICHAT_TEST_YT_OAUTH", "demo-creator-oauth")
	path := filepath.Join(t.TempDir(), "config.json")
	data := []byte(`{"youtube":{"enabled":true,"oauth_token_env":"MULTICHAT_TEST_YT_OAUTH"}}`)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	c, err := readConfig(path)
	if err != nil || c.YouTube.OAuthTokenEnv != "MULTICHAT_TEST_YT_OAUTH" {
		t.Fatalf("auto config invalid: %+v %v", c.YouTube, err)
	}
}
func TestFacebookConfigPageDiscovery(t *testing.T) {
	t.Setenv("MULTICHAT_TEST_FB_PAGE_TOKEN", "fake-page-token")
	path := filepath.Join(t.TempDir(), "config.json")
	data := []byte(`{"facebook":{"enabled":true,"page_id":"123456789","access_token_env":"MULTICHAT_TEST_FB_PAGE_TOKEN"}}`)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	c, err := readConfig(path)
	if err != nil || c.Facebook.PageID != "123456789" {
		t.Fatalf("page mode invalid: %+v %v", c.Facebook, err)
	}
}
