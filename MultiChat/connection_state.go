package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type ConnectionSignal struct {
	Identity     string    `json:"identity,omitempty"`
	LastVerified time.Time `json:"last_verified,omitempty"`
	ValidUntil   time.Time `json:"-"`
}
type OBSStreamStatus struct {
	Streaming  bool      `json:"streaming"`
	Service    string    `json:"service"`
	ObservedAt time.Time `json:"observed_at,omitempty"`
	Source     string    `json:"source"`
}
type nativeConnection struct {
	Platform   string `json:"platform"`
	State      string `json:"state"`
	Online     bool   `json:"online"`
	Configured bool   `json:"configured"`
	SendReady  bool   `json:"send_ready"`
	Identity   string `json:"identity,omitempty"`
}
type nativeStatus struct {
	Mode      string             `json:"mode"`
	OBS       OBSStreamStatus    `json:"obs"`
	Platforms []nativeConnection `json:"platforms"`
}

func (h *Hub) setConnection(platform, identity string, ttl time.Duration) {
	if ttl < 10*time.Second {
		ttl = 10 * time.Second
	}
	if ttl > 5*time.Minute {
		ttl = 5 * time.Minute
	}
	identity = truncate(strings.TrimSpace(identity), 100)
	now := time.Now()
	h.connectionMu.Lock()
	h.connections[platform] = ConnectionSignal{Identity: identity, LastVerified: now, ValidUntil: now.Add(ttl)}
	h.connectionMu.Unlock()
}
func (h *Hub) clearConnection(platform string) {
	h.connectionMu.Lock()
	delete(h.connections, platform)
	h.connectionMu.Unlock()
}
func (h *Hub) connectionSnapshot(c Config) nativeStatus {
	result := nativeStatus{Mode: "off", Platforms: make([]nativeConnection, 0, len(outboundPlatforms))}
	if c.Demo {
		result.Mode = "demo"
	} else if c.Outbound.Enabled {
		result.Mode = "live"
	}
	h.connectionMu.RLock()
	result.OBS = h.obsStatus
	verified := make(map[string]ConnectionSignal, len(h.connections))
	for p, v := range h.connections {
		verified[p] = v
	}
	h.connectionMu.RUnlock()
	prepared := newOutboundService(c, h).capabilities()
	ready := map[string]bool{}
	for _, p := range prepared {
		ready[p.Platform] = p.Ready
	}
	configured := map[string]bool{"youtube": c.YouTube.Enabled, "tiktok": c.TikTok.Enabled, "twitch": c.Twitch.Enabled, "kick": c.Kick.Enabled, "facebook": c.Facebook.Enabled}
	for _, p := range outboundPlatforms {
		v, ok := verified[p]
		alive := ok && time.Now().Before(v.ValidUntil) && !c.Demo
		state := "offline"
		if c.Demo {
			state = "demo"
		} else if alive {
			state = "online"
		} else if configured[p] {
			state = "waiting"
		}
		result.Platforms = append(result.Platforms, nativeConnection{Platform: p, State: state, Online: alive, Configured: configured[p], SendReady: ready[p], Identity: v.Identity})
	}
	return result
}
func (h *Hub) serveNativeStatus(c Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(h.connectionSnapshot(c))
	}
}

type OBSConfig struct {
	Enabled                 bool   `json:"enabled"`
	TokenEnv                string `json:"token_env"`
	StartCollectorsOnStream bool   `json:"start_collectors_on_stream"`
	ScanAllConfigured       bool   `json:"scan_all_configured"`
	WebSocketEnabled        bool   `json:"websocket_enabled"`
	WebSocketPort           int    `json:"websocket_port"`
	WebSocketPasswordEnv    string `json:"websocket_password_env"`
}
type obsEvent struct {
	Event   string `json:"event"`
	Service string `json:"service"`
}
type OBSCoordinator struct {
	ctx     context.Context
	hub     *Hub
	config  Config
	mu      sync.Mutex
	cancel  context.CancelFunc
	active  bool
	service string
}

func newOBSCoordinator(ctx context.Context, c Config, h *Hub) *OBSCoordinator {
	return &OBSCoordinator{ctx: ctx, config: c, hub: h}
}
func (o *OBSCoordinator) onEvent(event, service string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if event == "stopped" {
		if o.cancel != nil {
			o.cancel()
			o.cancel = nil
		}
		o.active = false
		o.service = ""
		for _, p := range outboundPlatforms {
			o.hub.clearConnection(p)
		}
		return
	}

	if event != "started" {
		return
	}
	if o.active {
		if o.service == service {
			return
		}
		if o.cancel != nil {
			o.cancel()
			o.cancel = nil
		}
		o.active = false
		for _, p := range outboundPlatforms {
			o.hub.clearConnection(p)
		}
	}
	ctx, cancel := context.WithCancel(o.ctx)
	o.cancel = cancel
	o.active = true
	o.service = service
	// Only connectors explicitly enabled in config are started; each must
	// independently locate a LIVE chat / complete its own handshake.
	c := o.config
	h := o.hub
	if c.YouTube.Enabled && (c.OBS.ScanAllConfigured || service == "youtube") {
		go youtubeLoop(ctx, c.YouTube, h, func(string, ...any) {})
	}
	if c.TikTok.Enabled && (c.OBS.ScanAllConfigured || service == "tiktok") {
		go tiktokLoop(ctx, c.TikTok, h, func(string, ...any) {})
	}
	if c.Twitch.Enabled && (c.OBS.ScanAllConfigured || service == "twitch") {
		go twitchLoop(ctx, c.Twitch, h, func(string, ...any) {})
	}
	if c.Kick.Enabled && (c.OBS.ScanAllConfigured || service == "kick") {
		go kickLoop(ctx, c.Kick, h, func(string, ...any) {})
	}
	if c.Facebook.Enabled && (c.OBS.ScanAllConfigured || service == "facebook") {
		go facebookLoop(ctx, c.Facebook, h, func(string, ...any) {})
	}
	if c.Rumble.Enabled && c.OBS.ScanAllConfigured {
		go rumbleLoop(ctx, c.Rumble, h, func(string, ...any) {})
	}
}

func classifyOBSService(label string) string {
	normalized := strings.ToLower(strings.TrimSpace(label))
	switch {
	case strings.Contains(normalized, "youtube"):
		return "youtube"
	case strings.Contains(normalized, "twitch"):
		return "twitch"
	case strings.Contains(normalized, "facebook"):
		return "facebook"
	case strings.Contains(normalized, "kick"):
		return "kick"
	case strings.Contains(normalized, "tiktok"):
		return "tiktok"
	default:
		return "unknown"
	}
}
func (h *Hub) serveOBSEvent(c Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if !c.OBS.Enabled {
			http.NotFound(w, r)
			return
		}
		expected := os.Getenv(c.OBS.TokenEnv)
		provided := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if len(expected) < 32 || subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if !sameOrigin(r) || r.Header.Get("X-SolarisPKN-Action") != "obs" || r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "invalid event origin or format", http.StatusForbidden)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1024)
		var event obsEvent
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&event); err != nil || !(event.Event == "started" || event.Event == "stopped") || len(event.Service) > 128 {
			http.Error(w, "invalid OBS event", http.StatusBadRequest)
			return
		}
		service := classifyOBSService(event.Service)
		h.connectionMu.Lock()
		h.obsStatus = OBSStreamStatus{Streaming: event.Event == "started", Service: service, Source: "OBS-plugin", ObservedAt: time.Now()}
		h.connectionMu.Unlock()
		if c.OBS.StartCollectorsOnStream && h.coordinator != nil {
			h.coordinator.onEvent(event.Event, service)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"accepted": true, "streaming": event.Event == "started", "service": service})
	}
}
