package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNativeChatFeedBoundedAndSanitized(t *testing.T) {
	h := newHub(250)
	for i := 0; i < 88; i++ {
		h.publish(Message{ID: "unique-" + strings.Repeat("x", i%3) + string(rune(65+i)), Platform: "twitch", User: "some\nviewer", Text: "Hola\r\n!redes\tOK"})
	}
	handler := newHandler(h, Config{})
	req := httptest.NewRequest("GET", "http://127.0.0.1:8766/native/feed", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("HTTP %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Fatal("wrong native content type")
	}
	lines := strings.Split(strings.TrimSpace(rec.Body.String()), "\r\n")
	if len(lines) > 80 || len(lines) < 1 {
		t.Fatalf("expected at most 80 native feed lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "some viewer: Hola  !redes OK") {
		t.Fatalf("unsanitized feed: %q", lines[0])
	}
}
func TestNativeFeedRespectsPanelAuth(t *testing.T) {
	t.Setenv("TEST_MULTI_NATIVE_AUTH", "long-native-chat-test-panel-secret-abc123")
	h := newHub(10)
	c := Config{PanelAuth: PanelAuthConfig{Enabled: true, TokenEnv: "TEST_MULTI_NATIVE_AUTH"}}
	rec := httptest.NewRecorder()
	newHandler(h, c).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8766/native/feed", nil))
	if rec.Code != 401 {
		t.Fatalf("native feed bypassed login: HTTP %d", rec.Code)
	}
}
