package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

func newHandler(h *Hub, c Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'")
		_, _ = io.WriteString(w, webPage)
	})
	mux.HandleFunc("GET /compose", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'")
		_, _ = io.WriteString(w, composerPage)
	})
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "history_count": len(h.snapshot()), "configured": map[string]bool{"twitch": c.Twitch.Enabled, "youtube": c.YouTube.Enabled, "tiktok": c.TikTok.Enabled, "kick": c.Kick.Enabled, "facebook": c.Facebook.Enabled, "rumble": c.Rumble.Enabled, "ingest": c.Ingest.Enabled}})
	})
	mux.HandleFunc("GET /api/native/status", h.serveNativeStatus(c))
	mux.HandleFunc("POST /api/obs/event", h.serveOBSEvent(c))
	mux.HandleFunc("GET /auth/providers", func(w http.ResponseWriter, r *http.Request) {
		// Report configuration and locally saved credentials, NOT remote login or LIVE status.
		_, googleErr := loadGoogleCredentials()
		_, twitchErr := loadTwitchCredentials()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"youtube":  map[string]any{"enabled": c.YouTube.Enabled, "oauth_saved": googleErr == nil, "method": "google_oauth_or_api_key"},
			"twitch":   map[string]any{"enabled": c.Twitch.Enabled, "oauth_saved": twitchErr == nil, "method": "twitch_device_oauth_or_irc_token"},
			"facebook": map[string]any{"enabled": c.Facebook.Enabled, "page_token_provided": c.Facebook.AccessTokenEnv != "" && os.Getenv(c.Facebook.AccessTokenEnv) != "", "method": "meta_page_access_token"},
			"kick":     map[string]any{"enabled": c.Kick.Enabled, "method": "public_chat_guest"},
			"tiktok":   map[string]any{"enabled": c.TikTok.Enabled, "method": "public_webcast"},
		})
	})
	mux.HandleFunc("GET /obs/messages", func(w http.ResponseWriter, r *http.Request) {
		// Bounded snapshot for the native OBS GDI+ plugin; no embedded browser.
		n := 8
		if value := r.URL.Query().Get("limit"); value != "" {
			if _, err := fmt.Sscan(value, &n); err != nil {
				http.Error(w, "bad limit", 400)
				return
			}
		}
		if n < 1 {
			n = 1
		}
		if n > 16 {
			n = 16
		}
		msgs := h.snapshot()
		if len(msgs) > n {
			msgs = msgs[len(msgs)-n:]
		}
		if msgs == nil {
			msgs = []Message{}
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(msgs)
	})
	mux.HandleFunc("GET /native/feed", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		msgs := h.snapshot()
		if len(msgs) > 80 {
			msgs = msgs[len(msgs)-80:]
		}
		var out strings.Builder
		for _, m := range msgs {
			clean := func(s string) string {
				return strings.Map(func(ch rune) rune {
					if ch == '\r' || ch == '\n' || ch == '\t' || ch < 32 || ch == 127 {
						return ' '
					}
					return ch
				}, s)
			}
			fmt.Fprintf(&out, "[%s] %s: %s\r\n", clean(m.Platform), clean(m.User), clean(m.Text))
		}
		_, _ = io.WriteString(w, out.String())
	})
	mux.HandleFunc("GET /history", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(h.snapshot())
	})
	mux.HandleFunc("GET /events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Accel-Buffering", "no")
		f, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "stream unavailable", 500)
			return
		}
		ch, remove := h.subscribe()
		defer remove()
		_, _ = io.WriteString(w, ": connected\n\n")
		f.Flush()
		keepalive := time.NewTicker(20 * time.Second)
		defer keepalive.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case msg := <-ch:
				b, _ := json.Marshal(msg)
				if _, err := fmt.Fprintf(w, "data: %s\n\n", b); err != nil {
					return
				}
				f.Flush()
			case <-keepalive.C:
				if _, err := io.WriteString(w, ": ping\n\n"); err != nil {
					return
				}
				f.Flush()
			}
		}
	})
	outbox := newOutboundService(c, h)
	mux.HandleFunc("GET /api/outbound/status", outbox.serveStatus)
	mux.HandleFunc("POST /api/outbound/send", outbox.serveSend)
	mux.HandleFunc("POST /api/ingest", func(w http.ResponseWriter, r *http.Request) {
		if !c.Ingest.Enabled {
			http.NotFound(w, r)
			return
		}
		expected := os.Getenv(c.Ingest.TokenEnv)
		received := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(received), []byte(expected)) != 1 {
			http.Error(w, "unauthorized", 401)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
		var m Message
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			http.Error(w, "invalid JSON", 400)
			return
		}
		if m.Platform == "" || m.Text == "" {
			http.Error(w, "platform and text required", 400)
			return
		}
		h.publish(m)
		w.WriteHeader(202)
	})
	protected := newPanelAuth(c.PanelAuth).middleware(mux)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}
		if host != "127.0.0.1" && host != "localhost" && host != "::1" {
			http.Error(w, "invalid host", 403)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		protected.ServeHTTP(w, r)
	})
}
