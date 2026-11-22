package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type outboundRoundTrip func(*http.Request) (*http.Response, error)

func (f outboundRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func fakeResponse(body string, status int) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}
func TestOutboundIsDisabledUnlessPanelProtected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"outbound":{"enabled":true}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readConfig(path); err == nil {
		t.Fatal("real chat sending allowed without panel authentication")
	}
}
func TestDemoOutboundSelectionAndNoNetwork(t *testing.T) {
	hub := newHub(30)
	c := Config{Demo: true}
	handler := newHandler(hub, c)
	statusReq := httptest.NewRequest("GET", "http://127.0.0.1:8765/api/outbound/status", nil)
	statusResponse := httptest.NewRecorder()
	handler.ServeHTTP(statusResponse, statusReq)
	if statusResponse.Code != 200 {
		t.Fatalf("status HTTP %d", statusResponse.Code)
	}
	var status struct {
		Mode    string           `json:"mode"`
		Targets []outboundTarget `json:"targets"`
	}
	if err := json.Unmarshal(statusResponse.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.Mode != "demo" || len(status.Targets) != 5 {
		t.Fatal("wrong demo status")
	}
	for _, target := range status.Targets {
		if !target.Ready {
			t.Fatalf("demo platform %s unavailable", target.Platform)
		}
	}
	send := func(targets []string, message string, origin string, header bool) *httptest.ResponseRecorder {
		payload, _ := json.Marshal(outboundRequest{Message: message, Targets: targets})
		req := httptest.NewRequest("POST", "http://127.0.0.1:8765/api/outbound/send", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		if header {
			req.Header.Set("X-SolarisPKN-Action", "send")
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}
	rec := send([]string{"youtube"}, "!redes", "https://other-site.example", true)
	if rec.Code != 403 {
		t.Fatalf("cross-origin send allowed: %d", rec.Code)
	}
	rec = send([]string{"youtube"}, "hola", "", false)
	if rec.Code != 403 {
		t.Fatalf("missing action header allowed: %d", rec.Code)
	}
	rec = send([]string{"youtube", "youtube"}, "hola", "", true)
	if rec.Code != 400 {
		t.Fatalf("duplicate platform allowed: %d", rec.Code)
	}
	rec = send([]string{"other"}, "hola", "", true)
	if rec.Code != 400 {
		t.Fatalf("unregistered target allowed: %d", rec.Code)
	}
	rec = send([]string{"youtube", "tiktok", "twitch", "kick", "facebook"}, "!redes", "", true)
	if rec.Code != 200 {
		t.Fatalf("demo send failed: HTTP %d %s", rec.Code, rec.Body.String())
	}
	var result outboundResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Mode != "demo" || len(result.Results) != 5 {
		t.Fatalf("wrong fanout: %+v", result)
	}
	for _, entry := range result.Results {
		if entry.Status != "simulated" {
			t.Fatalf("fake success claimed: %+v", entry)
		}
	}
	history := hub.snapshot()
	if len(history) != 5 {
		t.Fatalf("expected five local demo echoes: %d", len(history))
	}
	for _, x := range history {
		if x.Kind != "outbound_demo" || x.Text != "!redes" {
			t.Fatalf("wrong local echo: %+v", x)
		}
	}
	rec = send([]string{"youtube"}, "second", "", true)
	if rec.Code != 429 {
		t.Fatalf("outbound rate limiter missing: %d", rec.Code)
	}
}
func TestLiveSendDisabledByDefault(t *testing.T) {
	c := Config{}
	handler := newHandler(newHub(5), c)
	req := httptest.NewRequest("POST", "http://127.0.0.1:8765/api/outbound/send", strings.NewReader(`{"message":"hello","targets":["youtube"]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-SolarisPKN-Action", "send")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 403 {
		t.Fatalf("real outbound not fail-closed: %d", rec.Code)
	}
}
func TestOutboundYouTubeOfficialEndpoint(t *testing.T) {
	t.Setenv("OUTBOUND_TEST_GOOGLE_TOKEN", "not-real-test-token")
	called := 0
	client := &http.Client{Transport: outboundRoundTrip(func(r *http.Request) (*http.Response, error) {
		called++
		if r.URL.Host != "www.googleapis.com" || r.URL.Path != "/youtube/v3/liveChat/messages" || r.URL.Query().Get("part") != "snippet" {
			t.Errorf("bad YouTube endpoint: %s", r.URL)
		}
		if r.Header.Get("Authorization") != "Bearer not-real-test-token" {
			t.Errorf("bearer missing")
		}
		var payload struct {
			Snippet struct {
				LiveChatID string `json:"liveChatId"`
				Type       string `json:"type"`
				Text       struct {
					MessageText string `json:"messageText"`
				} `json:"textMessageDetails"`
			} `json:"snippet"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.Snippet.LiveChatID != "CHAT-ID" || payload.Snippet.Type != "textMessageEvent" || payload.Snippet.Text.MessageText != "hola Youtube" {
			t.Fatalf("wrong YouTube payload: %+v", payload)
		}
		return fakeResponse(`{"id":"video-chat-message"}`, 200), nil
	})}
	o := newOutboundService(Config{YouTube: YouTubeConfig{Enabled: true, LiveChatID: "CHAT-ID", OAuthTokenEnv: "OUTBOUND_TEST_GOOGLE_TOKEN"}}, newHub(5))
	o.client = client
	if err := o.sendYouTube(context.Background(), "hola Youtube"); err != nil {
		t.Fatal(err)
	}
	if called != 1 {
		t.Fatalf("unexpected API calls: %d", called)
	}
}
func TestOutboundFacebookPageComment(t *testing.T) {
	t.Setenv("OUTBOUND_TEST_FB_TOKEN", "fake-mock-token")
	client := &http.Client{Transport: outboundRoundTrip(func(r *http.Request) (*http.Response, error) {
		if r.Method != "POST" || r.URL.Path != "/v27.0/1234/comments" {
			t.Fatalf("wrong FB request %s %s", r.Method, r.URL)
		}
		if r.Header.Get("Authorization") != "Bearer fake-mock-token" {
			t.Fatal("missing token")
		}
		raw, _ := io.ReadAll(r.Body)
		if string(raw) != "message=hola+Facebook" {
			t.Fatalf("wrong Facebook form: %s", raw)
		}
		return fakeResponse(`{"id":"new-facebook-comment"}`, 200), nil
	})}
	o := newOutboundService(Config{Facebook: FacebookConfig{Enabled: true, LiveVideoID: "1234", AccessTokenEnv: "OUTBOUND_TEST_FB_TOKEN"}}, newHub(5))
	o.client = client
	if err := o.sendFacebook(context.Background(), "hola Facebook"); err != nil {
		t.Fatal(err)
	}
}
func TestOutboundKickOfficialEndpoint(t *testing.T) {
	t.Setenv("OUTBOUND_TEST_KICK_TOKEN", "fake-kick-user-oauth")
	client := &http.Client{Transport: outboundRoundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "api.kick.com" || r.URL.Path != "/public/v1/chat" {
			t.Fatalf("unexpected Kick path: %s", r.URL)
		}
		if r.Header.Get("Authorization") != "Bearer fake-kick-user-oauth" {
			t.Fatal("Kick auth missing")
		}
		var v struct {
			Type              string `json:"type"`
			BroadcasterUserID int64  `json:"broadcaster_user_id"`
			Content           string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
			t.Fatal(err)
		}
		if v.Type != "user" || v.BroadcasterUserID != 123456 || v.Content != "!hola" {
			t.Fatalf("bad Kick payload: %+v", v)
		}
		return fakeResponse(`{}`, 200), nil
	})}
	o := newOutboundService(Config{Outbound: OutboundConfig{Enabled: true, KickTokenEnv: "OUTBOUND_TEST_KICK_TOKEN", KickBroadcasterUserID: 123456}}, newHub(5))
	o.client = client
	if err := o.sendKick(context.Background(), "!hola"); err != nil {
		t.Fatal(err)
	}
}

func TestOutboundTwitchHelixSendAndScope(t *testing.T) {
	for _, testCase := range []struct {
		Name        string
		Scopes      string
		ExpectError bool
	}{
		{"write_allowed", `["chat:read","user:write:chat"]`, false},
		{"read_only_rejected", `["chat:read"]`, true},
	} {
		t.Run(testCase.Name, func(t *testing.T) {
			count := 0
			client := &http.Client{Transport: outboundRoundTrip(func(r *http.Request) (*http.Response, error) {
				count++
				switch r.URL.Host + r.URL.Path {
				case "id.twitch.tv/oauth2/validate":
					if r.Header.Get("Authorization") != "OAuth oauth-test-token" {
						t.Error("Twitch OAuth validation bearer missing")
					}
					return fakeResponse(`{"user_id":"777","client_id":"test-client","scopes":`+testCase.Scopes+`}`, 200), nil
				case "api.twitch.tv/helix/users":
					if r.URL.Query().Get("login") != "solarispkn" {
						t.Errorf("unexpected channel login %s", r.URL)
					}
					return fakeResponse(`{"data":[{"id":"444"}]}`, 200), nil
				case "api.twitch.tv/helix/chat/messages":
					if r.Header.Get("Client-Id") != "test-client" {
						t.Error("missing Client-Id")
					}
					var b struct {
						BroadcasterID string `json:"broadcaster_id"`
						SenderID      string `json:"sender_id"`
						Message       string `json:"message"`
					}
					if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
						t.Fatal(err)
					}
					if b.BroadcasterID != "444" || b.SenderID != "777" || b.Message != "!redes" {
						t.Errorf("wrong Twitch outbound: %+v", b)
					}
					return fakeResponse(`{"data":[{"is_sent":true}]}`, 200), nil
				default:
					t.Fatalf("unexpected network call %s", r.URL)
					return nil, nil
				}
			})}
			sender := newOutboundService(Config{Twitch: TwitchConfig{Channel: "#solarispkn", UseSavedAuth: true}}, newHub(3))
			sender.client = client
			token := savedTwitchCredentials{AccessToken: "oauth-test-token", ClientID: "test-client", Login: "solarispkn"}
			err := sender.sendTwitchAuthorized(context.Background(), "!redes", token)
			if (err != nil) != testCase.ExpectError {
				t.Fatalf("unexpected send outcome %v", err)
			}
			if testCase.ExpectError && count != 1 {
				t.Fatal("read-only token reached write API")
			}
			if !testCase.ExpectError && count != 3 {
				t.Fatalf("missing Twitch HTTP requests %d", count)
			}
		})
	}
}
