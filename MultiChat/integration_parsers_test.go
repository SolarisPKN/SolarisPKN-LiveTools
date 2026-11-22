package main

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	pb "github.com/PirateTok/live-go/proto"
)

func TestTikTokChatParser(t *testing.T) {
	m, ok := parseTikTokChat(&pb.WebcastChatMessage{Common: &pb.Common{MsgId: 12345}, User: &pb.User{Nickname: "Luna"}, Content: "hola"})
	if !ok || m.Platform != "tiktok" || m.ID != "12345" || m.User != "Luna" || m.Text != "hola" {
		t.Fatalf("invalid tiktok parse: %#v", m)
	}
	if _, ok = parseTikTokChat(nil); ok {
		t.Fatal("nil TikTok message accepted")
	}
	if _, ok = parseTikTokChat(&pb.WebcastChatMessage{Content: " "}); ok {
		t.Fatal("empty TikTok message accepted")
	}
}
func TestKickChatParser(t *testing.T) {
	raw := []byte(`{"id":"kick-1","content":"Hola Kick","sender":{"username":"Viewer"},"created_at":"2026-11-21T21:00:00Z"}`)
	m, ok := parseKickChat("App\\Events\\ChatMessageEvent", raw)
	if !ok || m.Platform != "kick" || m.ID != "kick-1" || m.User != "Viewer" {
		t.Fatalf("invalid Kick message: %#v", m)
	}
	if _, ok = parseKickChat("App\\Events\\StreamHostEvent", raw); ok {
		t.Fatal("non-chat Kick event accepted")
	}
}
func TestFacebookParserOrdering(t *testing.T) {
	raw := []byte(`{"data":[{"id":"later","message":"dos","from":{"name":"B"},"created_time":"2026-11-21T10:00:02+0000"},{"id":"earlier","message":"uno","from":{"name":"A"},"created_time":"2026-11-21T10:00:01+0000"}]}`)
	// Graph timestamps can use +0000; parser must handle them.
	msgs, err := parseFacebookMessages(raw)
	if err != nil || len(msgs) != 2 {
		t.Fatalf("facebook parse failed: %v %+v", err, msgs)
	}
	if msgs[0].ID != "earlier" || msgs[1].ID != "later" {
		t.Fatalf("wrong Facebook order: %+v", msgs)
	}
}
func TestOBSMessagesRoute(t *testing.T) {
	h := newHub(50)
	for i := 0; i < 5; i++ {
		h.publish(Message{ID: fmt.Sprint(i), Platform: "youtube", Text: fmt.Sprint(i)})
	}
	req := httptest.NewRequest("GET", "http://127.0.0.1:8765/obs/messages?limit=2", nil)
	w := httptest.NewRecorder()
	newHandler(h, Config{}).ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("route HTTP %d: %s", w.Code, w.Body.String())
	}
	var messages []Message
	if err := json.Unmarshal(w.Body.Bytes(), &messages); err != nil {
		t.Fatal(err)
	}
	if len(messages) != 2 || messages[0].ID != "3" || messages[1].ID != "4" {
		t.Fatalf("wrong snapshot: %+v", messages)
	}
	req = httptest.NewRequest("GET", "http://127.0.0.1:8765/health", nil)
	w = httptest.NewRecorder()
	newHandler(h, Config{}).ServeHTTP(w, req)
	var status struct {
		Configured map[string]bool `json:"configured"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"youtube", "tiktok", "twitch", "kick", "facebook"} {
		if _, ok := status.Configured[name]; !ok {
			t.Fatalf("missing health status: %s", name)
		}
	}
}
