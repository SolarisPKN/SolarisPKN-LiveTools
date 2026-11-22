package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type kickChatPayload struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
	Sender    struct {
		Username string `json:"username"`
	} `json:"sender"`
}

func parseKickChat(event string, payload []byte) (Message, bool) {
	if event != "App\\Events\\ChatMessageEvent" && event != "App\\Events\\ChatMessageSentEvent" {
		return Message{}, false
	}
	var raw kickChatPayload
	if json.Unmarshal(payload, &raw) != nil || raw.Content == "" {
		return Message{}, false
	}
	t, _ := time.Parse(time.RFC3339Nano, raw.CreatedAt)
	return Message{ID: raw.ID, Platform: "kick", User: raw.Sender.Username, Text: raw.Content, Kind: "chat", Time: t}, true
}
func newKickClientID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("multichat-%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	s := hex.EncodeToString(b)
	return s[:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:]
}
func kickPost(ctx context.Context, client *http.Client, endpoint string, input any, out any) error {
	b, err := json.Marshal(input)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://kick.com")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "SolarisPKN-MultiChat/0.2")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("kick API HTTP %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 512<<10))
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}
func kickChatroomID(ctx context.Context, client *http.Client, c KickConfig) (string, error) {
	if c.ChatroomID != "" {
		return c.ChatroomID, nil
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://kick.com/api/v2/channels/"+url.PathEscape(c.Channel), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "SolarisPKN-MultiChat/0.2")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Kick channel lookup HTTP %d; set chatroom_id manually if needed", resp.StatusCode)
	}
	var result struct {
		Chatroom struct {
			ID int64 `json:"id"`
		} `json:"chatroom"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 512<<10)).Decode(&result); err != nil {
		return "", err
	}
	if result.Chatroom.ID <= 0 {
		return "", errors.New("Kick chatroom ID not found")
	}
	return strconv.FormatInt(result.Chatroom.ID, 10), nil
}
func kickRealtimeCredentials(ctx context.Context, client *http.Client, id string) (string, string, error) {
	var conn struct {
		Data struct {
			Connections []struct {
				Provider    string `json:"provider"`
				Credentials struct {
					URL string `json:"url"`
				} `json:"credentials"`
			} `json:"connections"`
		} `json:"data"`
	}
	req := map[string]any{"client": map[string]string{"id": id, "type": "web"}, "capabilities": map[string]any{"accepted_providers": []map[string]string{{"provider": "centrifugo"}}}}
	if err := kickPost(ctx, client, "https://web.kick.com/api/v1/realtime/connection", req, &conn); err != nil {
		return "", "", err
	}
	var endpoint string
	for _, x := range conn.Data.Connections {
		if x.Provider == "centrifugo" {
			endpoint = x.Credentials.URL
			break
		}
	}
	u, err := url.Parse(endpoint)
	if err != nil || u == nil || u.Scheme != "wss" || !strings.HasSuffix(strings.ToLower(u.Hostname()), ".platform.kick.com") {
		return "", "", errors.New("untrusted Kick realtime websocket endpoint")
	}
	var auth struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := kickPost(ctx, client, "https://web.kick.com/api/v1/realtime/auth/connection", map[string]string{"client_id": id}, &auth); err != nil {
		return "", "", err
	}
	if auth.Data.Token == "" {
		return "", "", errors.New("Kick realtime guest token missing")
	}
	return endpoint, auth.Data.Token, nil
}
func kickWebsocketSession(ctx context.Context, endpoint, token, room string, h *Hub) error {
	defer h.clearConnection("kick")
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second, ReadBufferSize: 4096}
	conn, _, err := dialer.DialContext(ctx, endpoint, http.Header{"Origin": []string{"https://kick.com"}})
	if err != nil {
		return err
	}
	defer conn.Close()
	stop := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-stop:
		}
	}()
	defer close(stop)
	conn.SetReadLimit(1 << 20)
	_ = conn.SetReadDeadline(time.Now().Add(75 * time.Second))
	if err = conn.WriteJSON(map[string]any{"id": 1, "connect": map[string]string{"token": token, "name": "multichat"}}); err != nil {
		return err
	}
	connected := false
	subscribed := false
	expiry := time.Now().Add(40 * time.Minute)
	for ctx.Err() == nil && time.Now().Before(expiry) {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		_ = conn.SetReadDeadline(time.Now().Add(75 * time.Second))
		for _, line := range bytes.Split(raw, []byte("\n")) {
			if len(bytes.TrimSpace(line)) == 0 {
				continue
			}
			var frame struct {
				ID        int             `json:"id"`
				Connect   json.RawMessage `json:"connect"`
				Subscribe json.RawMessage `json:"subscribe"`
				Error     *struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
				Push *struct {
					Pub *struct {
						Data struct {
							Event string          `json:"event"`
							Data  json.RawMessage `json:"data"`
						} `json:"data"`
					} `json:"pub"`
				} `json:"push"`
			}
			if json.Unmarshal(line, &frame) != nil {
				continue
			}
			if frame.Error != nil {
				return fmt.Errorf("Kick Centrifugo error %d", frame.Error.Code)
			}
			if frame.ID == 1 && len(frame.Connect) > 0 && !connected {
				connected = true
				if err = conn.WriteJSON(map[string]any{"id": 2, "subscribe": map[string]string{"channel": "chatrooms." + room + ".v2"}}); err != nil {
					return err
				}
			}
			if frame.ID == 2 && len(frame.Subscribe) > 0 {
				subscribed = true
				h.setConnection("kick", room, 4*time.Minute)
			}
			if frame.Push != nil && frame.Push.Pub != nil {
				h.setConnection("kick", room, 4*time.Minute)
				d := frame.Push.Pub.Data
				payload := d.Data
				// Kick wraps event data as a JSON string within the Centrifugo publication.
				if len(payload) > 0 && payload[0] == '"' {
					var inner string
					if json.Unmarshal(payload, &inner) == nil {
						payload = []byte(inner)
					}
				}
				if m, ok := parseKickChat(d.Event, payload); ok {
					h.publish(m)
				}
			}
			if bytes.Equal(bytes.TrimSpace(line), []byte("{}")) {
				if err = conn.WriteMessage(websocket.TextMessage, []byte("{}")); err != nil {
					return err
				}
			}
		}
	}
	if !subscribed {
		return errors.New("Kick Centrifugo subscription not confirmed")
	}
	return nil // reconnect before 45-minute token expiry
}
func kickLoop(ctx context.Context, c KickConfig, h *Hub, logf func(string, ...any)) {
	client := &http.Client{Timeout: 12 * time.Second}
	delay := 3 * time.Second
	for ctx.Err() == nil {
		room, err := kickChatroomID(ctx, client, c)
		if err == nil {
			id := newKickClientID()
			var endpoint, token string
			endpoint, token, err = kickRealtimeCredentials(ctx, client, id)
			if err == nil {
				err = kickWebsocketSession(ctx, endpoint, token, room, h)
			}
		}
		if ctx.Err() != nil {
			return
		}
		if err == nil {
			delay = 3 * time.Second
		}
		logf("[kick] guest realtime disconnected: %v; retry pending", err)
		if !sleepContext(ctx, delay) {
			return
		}
		if delay < time.Minute {
			delay *= 2
		}
	}
}
