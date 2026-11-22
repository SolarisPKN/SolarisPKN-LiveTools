package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testPanelSecret = "a-very-long-secret-32-character-local-panel-key"
const testOBSSecret = "an-independent-long-secret-for-OBS-only-access"

func TestPanelAuthConfiguration(t *testing.T) {
	t.Setenv("TEST_MULTICHAT_PANEL_KEY", testPanelSecret)
	t.Setenv("TEST_MULTICHAT_OBS_KEY", testOBSSecret)
	temp := filepath.Join(t.TempDir(), "config.json")
	text := `{"panel_auth":{"enabled":true,"token_env":"TEST_MULTICHAT_PANEL_KEY","obs_token_env":"TEST_MULTICHAT_OBS_KEY"}}`
	if err := os.WriteFile(temp, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	config, err := readConfig(temp)
	if err != nil || !config.PanelAuth.Enabled {
		t.Fatalf("expected secure config: %v", err)
	}
	t.Setenv("TEST_MULTICHAT_OBS_KEY", testPanelSecret)
	if _, err := readConfig(temp); err == nil {
		t.Fatal("identical panel and OBS tokens accepted")
	}
	t.Setenv("TEST_MULTICHAT_PANEL_KEY", "weak")
	if _, err := readConfig(temp); err == nil {
		t.Fatal("weak panel secret accepted")
	}
}

func TestPanelAuthenticationAndReadOnlyOBS(t *testing.T) {
	t.Setenv("TEST_MULTICHAT_PANEL_KEY", testPanelSecret)
	t.Setenv("TEST_MULTICHAT_OBS_KEY", testOBSSecret)
	h := newHub(25)
	h.publish(Message{ID: "chat-1", Platform: "youtube", User: "viewer", Text: "hola"})
	cfg := Config{PanelAuth: PanelAuthConfig{Enabled: true, TokenEnv: "TEST_MULTICHAT_PANEL_KEY", OBSTokenEnv: "TEST_MULTICHAT_OBS_KEY"}}
	server := httptest.NewServer(newHandler(h, cfg))
	defer server.Close()
	client := server.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	request := func(method, path string, body io.Reader, cookie *http.Cookie, auth string) *http.Response {
		t.Helper()
		req, err := http.NewRequest(method, server.URL+path, body)
		if err != nil {
			t.Fatal(err)
		}
		if cookie != nil {
			req.AddCookie(cookie)
		}
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		if method == "POST" {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}
	for _, path := range []string{"/history", "/events", "/obs/messages", "/auth/providers"} {
		r := request("GET", path, nil, nil, "")
		r.Body.Close()
		if r.StatusCode != 401 {
			t.Fatalf("unprotected %s: HTTP %d", path, r.StatusCode)
		}
	}
	r := request("GET", "/", nil, nil, "")
	r.Body.Close()
	if r.StatusCode != 303 || r.Header.Get("Location") != "/auth/login" {
		t.Fatalf("expected login redirect, got %d", r.StatusCode)
	}
	r = request("GET", "/auth/login", nil, nil, "")
	r.Body.Close()
	if r.StatusCode != 200 {
		t.Fatalf("login page HTTP %d", r.StatusCode)
	}
	r = request("GET", "/health", nil, nil, "")
	r.Body.Close()
	if r.StatusCode != 200 {
		t.Fatalf("health inaccessible: %d", r.StatusCode)
	}
	bad := url.Values{"password": {"bad"}}
	r = request("POST", "/auth/login", strings.NewReader(bad.Encode()), nil, "")
	r.Body.Close()
	if r.StatusCode != 401 {
		t.Fatalf("weak login accepted %d", r.StatusCode)
	}
	good := url.Values{"password": {testPanelSecret}}
	r = request("POST", "/auth/login", strings.NewReader(good.Encode()), nil, "")
	r.Body.Close()
	if r.StatusCode != 303 || len(r.Cookies()) == 0 {
		t.Fatalf("valid login failed %d", r.StatusCode)
	}
	cookie := r.Cookies()[0]
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode {
		t.Fatal("unsafe session cookie")
	}
	r = request("GET", "/history", nil, cookie, "")
	var messages []Message
	err := json.NewDecoder(r.Body).Decode(&messages)
	r.Body.Close()
	if err != nil || len(messages) != 1 || messages[0].Text != "hola" {
		t.Fatalf("authenticated history failed %v", err)
	}
	r = request("GET", "/obs/messages", nil, nil, "Bearer "+testOBSSecret)
	var viaOBS []Message
	err = json.NewDecoder(r.Body).Decode(&viaOBS)
	r.Body.Close()
	if err != nil || len(viaOBS) != 1 {
		t.Fatalf("OBS bearer failed: %v", err)
	}
	r = request("GET", "/history", nil, nil, "Bearer "+testOBSSecret)
	r.Body.Close()
	if r.StatusCode != 401 {
		t.Fatal("OBS read-only key allowed to read full history")
	}
	r = request("GET", "/events", nil, nil, "Bearer "+testOBSSecret)
	r.Body.Close()
	if r.StatusCode != 401 {
		t.Fatal("OBS key allowed SSE")
	}
	r = request("POST", "/auth/logout", strings.NewReader(""), cookie, "")
	r.Body.Close()
	if r.StatusCode != 303 {
		t.Fatalf("logout failed: %d", r.StatusCode)
	}
	r = request("GET", "/history", nil, cookie, "")
	r.Body.Close()
	if r.StatusCode != 401 {
		t.Fatal("logged-out session remained valid")
	}
}

func TestPanelLoginCrossOriginRejected(t *testing.T) {
	t.Setenv("TEST_MULTICHAT_PANEL_KEY", testPanelSecret)
	handler := newHandler(newHub(5), Config{PanelAuth: PanelAuthConfig{Enabled: true, TokenEnv: "TEST_MULTICHAT_PANEL_KEY"}})
	req := httptest.NewRequest("POST", "http://127.0.0.1:8765/auth/login", strings.NewReader(url.Values{"password": {testPanelSecret}}.Encode()))
	req.Header.Set("Origin", "https://attacker.example")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatalf("cross-site login accepted: HTTP %d", w.Code)
	}
}
