package main

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestOBSWebSocketAuthenticatedLiveAndStop(t *testing.T) {
	t.Setenv("MULTICHAT_OBS_WS_MOCK_PASSWORD", "not-a-real-obs-password")
	changed := make(chan bool, 3)
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		socket, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer socket.Close()
		send := func(v any) bool { return socket.WriteJSON(v) == nil }
		if !send(map[string]any{"op": 0, "d": map[string]any{"rpcVersion": 1, "authentication": map[string]string{"challenge": "challenge", "salt": "salt"}}}) {
			return
		}
		var identify struct {
			Op int `json:"op"`
			D  struct {
				Authentication     string `json:"authentication"`
				EventSubscriptions int    `json:"eventSubscriptions"`
			} `json:"d"`
		}
		if socket.ReadJSON(&identify) != nil {
			return
		}
		if identify.Op != 1 || identify.D.Authentication != obsAuthentication("not-a-real-obs-password", "salt", "challenge") || identify.D.EventSubscriptions != obsOutputEvents {
			t.Error("OBS websocket identification wrong")
			return
		}
		if !send(map[string]any{"op": 2, "d": map[string]int{"negotiatedRpcVersion": 1}}) {
			return
		}
		gotService, gotStatus := false, false
		for !(gotService && gotStatus) {
			var request struct {
				Op int `json:"op"`
				D  struct {
					RequestType string `json:"requestType"`
					RequestID   string `json:"requestId"`
				} `json:"d"`
			}
			if socket.ReadJSON(&request) != nil {
				return
			}
			if request.Op != 6 {
				return
			}
			switch request.D.RequestType {
			case "GetStreamServiceSettings":
				gotService = true
				if !send(map[string]any{"op": 7, "d": map[string]any{"requestType": "GetStreamServiceSettings", "requestId": request.D.RequestID, "requestStatus": map[string]bool{"result": true}, "responseData": map[string]any{"streamServiceType": "rtmp_common", "streamServiceSettings": map[string]any{"service": "Twitch", "key": "mock-private-key-never-log"}}}}) {
					return
				}
			case "GetStreamStatus":
				gotStatus = true
				if !send(map[string]any{"op": 7, "d": map[string]any{"requestType": "GetStreamStatus", "requestId": request.D.RequestID, "requestStatus": map[string]bool{"result": true}, "responseData": map[string]bool{"outputActive": true}}}) {
					return
				}
			}
		}
		close(started)
		<-changed
		if !send(map[string]any{"op": 5, "d": map[string]any{"eventType": "StreamStateChanged", "eventData": map[string]any{"outputActive": false, "outputState": "OBS_WEBSOCKET_OUTPUT_STOPPED"}}}) {
			return
		}
		<-changed
	}))
	defer server.Close()
	_, portStr, err := net.SplitHostPort(strings.TrimPrefix(server.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{OBS: OBSConfig{Enabled: true, WebSocketEnabled: true, WebSocketPort: port, WebSocketPasswordEnv: "MULTICHAT_OBS_WS_MOCK_PASSWORD"}}
	h := newHub(10)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- watchOBSWebsocket(ctx, cfg, h) }()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("OBS handshake timed out")
	}
	await := func(want bool) {
		t.Helper()
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			s := h.connectionSnapshot(cfg).OBS
			if s.Streaming == want && (want == false || s.Service == "twitch") {
				return
			}
			time.Sleep(15 * time.Millisecond)
		}
		t.Fatalf("OBS status was not %v: %+v", want, h.connectionSnapshot(cfg).OBS)
	}
	await(true)
	changed <- true
	await(false)
	changed <- true
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("OBS watcher did not stop")
	}
}
func TestOBSWebsocketAuthFormula(t *testing.T) {
	got := obsAuthentication("password", "salt", "challenge")
	if got == "" || len(got) < 40 {
		t.Fatal("OBS ws auth formula invalid")
	}
	raw, _ := json.Marshal(map[string]string{"auth": got})
	if strings.Contains(string(raw), "password") {
		t.Fatal("password copied into identification")
	}
}
