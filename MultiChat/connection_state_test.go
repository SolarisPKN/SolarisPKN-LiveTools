package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNativeConnectedRequiresVerifiedHeartbeat(t *testing.T) {
	h := newHub(10)
	c := Config{Demo: true}
	s := h.connectionSnapshot(c)
	if len(s.Platforms) != 5 {
		t.Fatalf("missing five states: %d", len(s.Platforms))
	}
	for _, p := range s.Platforms {
		if p.Online || p.State != "demo" {
			t.Fatalf("demo falsely green: %+v", p)
		}
	}
	c.Demo = false
	c.Twitch.Enabled = true
	s = h.connectionSnapshot(c)
	if s.Platforms[2].Online || s.Platforms[2].State != "waiting" {
		t.Fatalf("configured Twitch falsely online: %+v", s.Platforms[2])
	}
	h.setConnection("twitch", "solarispkn", 90*time.Second)
	s = h.connectionSnapshot(c)
	if !s.Platforms[2].Online || s.Platforms[2].State != "online" || s.Platforms[2].Identity != "solarispkn" {
		t.Fatalf("real heartbeat missing: %+v", s.Platforms[2])
	}
	h.clearConnection("twitch")
	if h.connectionSnapshot(c).Platforms[2].Online {
		t.Fatal("offline must clear green lamp")
	}
	h.setConnection("youtube", "LIVE123", time.Second)
	h.connectionMu.Lock()
	expired := h.connections["youtube"]
	expired.ValidUntil = time.Now().Add(-time.Second)
	h.connections["youtube"] = expired
	h.connectionMu.Unlock()
	if h.connectionSnapshot(c).Platforms[0].Online {
		t.Fatal("expired chat status still online")
	}
}
func TestOBSEventAuthenticatedOnlyAndNoAccidentalLive(t *testing.T) {
	t.Setenv("MULTICHAT_OBS_EVENT_TEST", "separate-test-only-obs-event-auth-32-characters")
	cfg := Config{OBS: OBSConfig{Enabled: true, TokenEnv: "MULTICHAT_OBS_EVENT_TEST"}}
	h := newHub(10)
	handler := newHandler(h, cfg)
	send := func(event, token, origin string, header bool) *httptest.ResponseRecorder {
		payload, _ := json.Marshal(obsEvent{Event: event, Service: "Twitch"})
		req := httptest.NewRequest("POST", "http://127.0.0.1:8766/api/obs/event", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		if header {
			req.Header.Set("X-SolarisPKN-Action", "obs")
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}
	if send("started", "", "", true).Code != 401 {
		t.Fatal("OBS event accepted without token")
	}
	if send("started", os.Getenv("MULTICHAT_OBS_EVENT_TEST"), "https://evil.example", true).Code != 403 {
		t.Fatal("cross-origin OBS allowed")
	}
	if send("started", os.Getenv("MULTICHAT_OBS_EVENT_TEST"), "", false).Code != 403 {
		t.Fatal("missing action header allowed")
	}
	if send("started", os.Getenv("MULTICHAT_OBS_EVENT_TEST"), "", true).Code != 200 {
		t.Fatal("OBS start rejected")
	}
	if !h.connectionSnapshot(cfg).OBS.Streaming || h.connectionSnapshot(cfg).OBS.Service != "twitch" {
		t.Fatal("OBS start status not recorded correctly")
	}
	if send("stopped", os.Getenv("MULTICHAT_OBS_EVENT_TEST"), "", true).Code != 200 {
		t.Fatal("OBS stop rejected")
	}
	if h.connectionSnapshot(cfg).OBS.Streaming {
		t.Fatal("OBS stop did not clear flag")
	}
	if send("invented", os.Getenv("MULTICHAT_OBS_EVENT_TEST"), "", true).Code != 400 {
		t.Fatal("invalid OBS event accepted")
	}
}
func TestOBSConfigEventAuthRequired(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"obs":{"enabled":true,"token_env":"MISSING_OBS_SECRET","start_collectors_on_stream":true}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readConfig(path); err == nil {
		t.Fatal("OBS enabled without event secret")
	}
}
func TestOBSServiceClassification(t *testing.T) {
	cases := map[string]string{"Twitch": "twitch", "YouTube - RTMPS": "youtube", "Facebook Live": "facebook", "Custom": "unknown", "Kick": "kick", "tiktok LIVE": "tiktok"}
	for label, expected := range cases {
		if got := classifyOBSService(label); got != expected {
			t.Fatalf("%s => %s want %s", label, got, expected)
		}
	}
}

func TestOBSEventAndNativeBearerWithPanelLogin(t *testing.T) {
	t.Setenv("MC_PANEL_AUTH_TEST", "the-panel-secret-more-than-thirty-two-chars")
	t.Setenv("MC_NATIVE_AUTH_TEST", "a-distinct-native-secret-more-than-32-chars")
	t.Setenv("MC_OBS_AUTH_TEST", "a-distinct-obs-event-secret-more-than-32-chars")
	c := Config{PanelAuth: PanelAuthConfig{Enabled: true, TokenEnv: "MC_PANEL_AUTH_TEST", NativeTokenEnv: "MC_NATIVE_AUTH_TEST"},
		OBS: OBSConfig{Enabled: true, TokenEnv: "MC_OBS_AUTH_TEST"}}
	h := newHub(10)
	handler := newHandler(h, c)
	status := httptest.NewRequest("GET", "http://127.0.0.1:8766/api/native/status", nil)
	blocked := httptest.NewRecorder()
	handler.ServeHTTP(blocked, status)
	if blocked.Code != 401 {
		t.Fatalf("missing native credential must fail, %d", blocked.Code)
	}
	status.Header.Set("Authorization", "Bearer "+os.Getenv("MC_NATIVE_AUTH_TEST"))
	allowed := httptest.NewRecorder()
	handler.ServeHTTP(allowed, status)
	if allowed.Code != 200 {
		t.Fatalf("native status should pass native Bearer, %d", allowed.Code)
	}
	obsReq := httptest.NewRequest("POST", "http://127.0.0.1:8766/api/obs/event", bytes.NewReader([]byte(`{"event":"started","service":"Twitch"}`)))
	obsReq.Header.Set("Authorization", "Bearer "+os.Getenv("MC_OBS_AUTH_TEST"))
	obsReq.Header.Set("Content-Type", "application/json")
	obsReq.Header.Set("X-SolarisPKN-Action", "obs")
	approved := httptest.NewRecorder()
	handler.ServeHTTP(approved, obsReq)
	if approved.Code != 200 {
		t.Fatalf("OBS separate token was rejected by panel middleware %d body %s", approved.Code, approved.Body.String())
	}
	wrong := httptest.NewRequest("POST", "http://127.0.0.1:8766/api/obs/event", bytes.NewReader([]byte(`{"event":"stopped","service":"Twitch"}`)))
	wrong.Header.Set("Authorization", "Bearer "+os.Getenv("MC_NATIVE_AUTH_TEST"))
	wrong.Header.Set("Content-Type", "application/json")
	wrong.Header.Set("X-SolarisPKN-Action", "obs")
	denied := httptest.NewRecorder()
	handler.ServeHTTP(denied, wrong)
	if denied.Code != 401 {
		t.Fatalf("native bearer must not impersonate OBS, HTTP %d", denied.Code)
	}
}
