package main

// Built-in OBS WebSocket v5 stream-state watcher.
// No OBS plugin DLL, browser, stream-key logging, or remote socket required.
// Opt-in: obs.enabled + obs.websocket_enabled, port restricted to 127.0.0.1.
import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

const obsOutputEvents = 1 << 6

type obsWSFrame struct {
	Op int             `json:"op"`
	D  json.RawMessage `json:"d"`
}
type obsWSHello struct {
	Authentication *struct {
		Challenge string `json:"challenge"`
		Salt      string `json:"salt"`
	} `json:"authentication"`
}
type obsWSResponse struct {
	RequestType   string `json:"requestType"`
	RequestID     string `json:"requestId"`
	RequestStatus struct {
		Result bool `json:"result"`
	} `json:"requestStatus"`
	ResponseData json.RawMessage `json:"responseData"`
}

func obsAuthentication(password, salt, challenge string) string {
	digest := sha256.Sum256([]byte(password + salt))
	secret := base64.StdEncoding.EncodeToString(digest[:])
	result := sha256.Sum256([]byte(secret + challenge))
	return base64.StdEncoding.EncodeToString(result[:])
}
func (h *Hub) applyOBSStream(c Config, event, service, source string) {
	service = classifyOBSService(service)
	h.connectionMu.Lock()
	h.obsStatus = OBSStreamStatus{Streaming: event == "started", Service: service, Source: source, ObservedAt: time.Now()}
	h.connectionMu.Unlock()
	if c.OBS.StartCollectorsOnStream && h.coordinator != nil {
		h.coordinator.onEvent(event, service)
	}
}
func obsWSRequest(conn *websocket.Conn, name, id string) error {
	return conn.WriteJSON(map[string]any{"op": 6, "d": map[string]string{"requestType": name, "requestId": id}})
}
func watchOBSWebsocket(ctx context.Context, c Config, h *Hub) error {
	port := c.OBS.WebSocketPort
	if port == 0 {
		port = 4455
	}
	addr := "ws://127.0.0.1:" + strconv.Itoa(port)
	dialer := websocket.Dialer{HandshakeTimeout: 7 * time.Second, NetDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{Timeout: 7 * time.Second}).DialContext(ctx, "tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	}}
	conn, _, err := dialer.DialContext(ctx, addr, nil)
	if err != nil {
		return errors.New("OBS WebSocket local server not available")
	}
	defer conn.Close()
	session, stop := context.WithCancel(ctx)
	defer stop()
	defer h.applyOBSStream(c, "stopped", "unknown", "OBS-websocket-disconnected")

	// OBS WebSocket 5: Hello (0), Identify (1), Identified (2).
	conn.SetReadLimit(1 << 20)
	_ = conn.SetReadDeadline(time.Now().Add(20 * time.Second))
	var hello obsWSFrame
	if err := conn.ReadJSON(&hello); err != nil || hello.Op != 0 {
		return errors.New("OBS WebSocket Hello unavailable")
	}
	var hi obsWSHello
	if json.Unmarshal(hello.D, &hi) != nil {
		return errors.New("invalid OBS WebSocket Hello")
	}
	identify := map[string]any{"rpcVersion": 1, "eventSubscriptions": obsOutputEvents}
	if hi.Authentication != nil {
		password := os.Getenv(c.OBS.WebSocketPasswordEnv)
		if password == "" {
			return errors.New("OBS WebSocket requires password environment variable")
		}
		identify["authentication"] = obsAuthentication(password, hi.Authentication.Salt, hi.Authentication.Challenge)
	}
	if err := conn.WriteJSON(map[string]any{"op": 1, "d": identify}); err != nil {
		return errors.New("OBS WebSocket Identify failed")
	}
	var identified obsWSFrame
	if err := conn.ReadJSON(&identified); err != nil || identified.Op != 2 {
		return errors.New("OBS WebSocket authentication failed")
	}
	if err := obsWSRequest(conn, "GetStreamServiceSettings", "service-initial"); err != nil {
		return err
	}
	if err := obsWSRequest(conn, "GetStreamStatus", "status-initial"); err != nil {
		return err
	}
	service := "unknown"
	streaming := false
	gotService, gotStream := false, false
	activeReported := false
	_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(90 * time.Second)) })
	go func() {
		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-session.Done():
				return
			case <-ticker.C:
				_ = conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(2*time.Second))
			}
		}
	}()
	// Shutdown of the local controller aborts OBS read without touching OBS itself.
	go func() {
		select {
		case <-session.Done():
			_ = conn.Close()
		}
	}()
	for ctx.Err() == nil {
		var frame obsWSFrame
		if err := conn.ReadJSON(&frame); err != nil {
			return errors.New("OBS WebSocket disconnected")
		}
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		switch frame.Op {
		case 7:
			var result obsWSResponse
			if json.Unmarshal(frame.D, &result) != nil || !result.RequestStatus.Result {
				continue
			}
			switch result.RequestType {
			case "GetStreamServiceSettings":
				var data struct {
					StreamServiceType     string `json:"streamServiceType"`
					StreamServiceSettings struct {
						Service string `json:"service"`
					} `json:"streamServiceSettings"`
				}
				if json.Unmarshal(result.ResponseData, &data) != nil {
					continue
				}
				// Stream keys and server addresses (which may also occur in this payload)
				// are deliberately not copied or logged.
				service = classifyOBSService(data.StreamServiceSettings.Service)
				if service == "unknown" {
					service = classifyOBSService(data.StreamServiceType)
				}
				gotService = true
			case "GetStreamStatus":
				var data struct {
					OutputActive bool `json:"outputActive"`
				}
				if json.Unmarshal(result.ResponseData, &data) != nil {
					continue
				}
				streaming = data.OutputActive
				gotStream = true
			}
			if gotService && gotStream {
				if streaming {
					h.applyOBSStream(c, "started", service, "OBS-websocket")
					activeReported = true
				} else {
					h.applyOBSStream(c, "stopped", service, "OBS-websocket")
					activeReported = false
				}
				gotStream = false // do not repeat on unrelated responses
			}
		case 5:
			var evt struct {
				EventType string `json:"eventType"`
				EventData struct {
					OutputActive bool   `json:"outputActive"`
					OutputState  string `json:"outputState"`
				} `json:"eventData"`
			}
			if json.Unmarshal(frame.D, &evt) != nil || evt.EventType != "StreamStateChanged" {
				continue
			}
			streaming = evt.EventData.OutputActive
			if streaming && !activeReported {
				// Always refresh current service at each start: OBS profiles may change
				// Twitch -> YouTube between streams. Never rely on last stream service.
				_ = obsWSRequest(conn, "GetStreamServiceSettings", "service-start")
				gotStream = true
				gotService = false
			} else if !streaming {
				h.applyOBSStream(c, "stopped", service, "OBS-websocket")
				activeReported = false
			}
		}
	}
	return fmt.Errorf("OBS watcher stopped: %w", ctx.Err())
}
func obsWebsocketLoop(ctx context.Context, c Config, h *Hub, logf func(string, ...any)) {
	delay := 2 * time.Second
	for ctx.Err() == nil {
		err := watchOBSWebsocket(ctx, c, h)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			// Intentionally no stream keys, passwords, or raw OBS responses in logs.
			logf("[obs] WebSocket desconectado o sin autenticación; reintento local en %v", delay)
		}
		if !sleepContext(ctx, delay) {
			return
		}
		if delay < 30*time.Second {
			delay *= 2
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}
		}
	}
}
