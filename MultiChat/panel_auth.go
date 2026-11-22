package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const panelSessionCookie = "solarispkn_mc_session"
const panelSessionLifetime = 8 * time.Hour

type panelAuth struct {
	enabled  bool
	panelKey string
	obsKey   string
	mu       sync.Mutex
	sessions map[string]time.Time
	failures map[string]panelFailure
}

type panelFailure struct {
	Count int
	Until time.Time
}

func newPanelAuth(c PanelAuthConfig) *panelAuth {
	a := &panelAuth{enabled: c.Enabled, sessions: make(map[string]time.Time), failures: make(map[string]panelFailure)}
	if c.Enabled {
		a.panelKey = os.Getenv(c.TokenEnv)
		if c.OBSTokenEnv != "" {
			a.obsKey = os.Getenv(c.OBSTokenEnv)
		}
	}
	return a
}

func constantTimeStringEqual(a, b string) bool {
	return len(a) > 0 && len(b) > 0 && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func (a *panelAuth) isAuthenticated(r *http.Request) bool {
	if !a.enabled {
		return true
	}
	cookie, err := r.Cookie(panelSessionCookie)
	if err != nil || len(cookie.Value) < 32 || len(cookie.Value) > 128 {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	exp, ok := a.sessions[cookie.Value]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		delete(a.sessions, cookie.Value)
		return false
	}
	return true
}

func (a *panelAuth) isOBSAuthorized(r *http.Request) bool {
	if !a.enabled || a.obsKey == "" || r.URL.Path != "/obs/messages" || r.Method != "GET" {
		return false
	}
	// The native OBS plugin can supply a separate read-only token.
	return constantTimeStringEqual(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), a.obsKey)
}

func randomSessionToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func (a *panelAuth) acceptLogin(clientIP, provided string) (string, bool, error) {
	a.mu.Lock()
	failure := a.failures[clientIP]
	if time.Now().Before(failure.Until) && failure.Count >= 6 {
		a.mu.Unlock()
		return "", false, nil
	}
	if !constantTimeStringEqual(provided, a.panelKey) {
		if time.Now().After(failure.Until) {
			failure = panelFailure{}
		}
		failure.Count++
		failure.Until = time.Now().Add(time.Minute)
		a.failures[clientIP] = failure
		a.mu.Unlock()
		return "", false, nil
	}
	token, err := randomSessionToken()
	if err != nil {
		a.mu.Unlock()
		return "", false, err
	}
	now := time.Now()
	// Bound retained memory even if many independent sessions are created.
	for key, expiry := range a.sessions {
		if now.After(expiry) {
			delete(a.sessions, key)
		}
	}
	if len(a.sessions) > 128 {
		a.sessions = make(map[string]time.Time)
	}
	a.sessions[token] = now.Add(panelSessionLifetime)
	delete(a.failures, clientIP)
	a.mu.Unlock()
	return token, true, nil
}

func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	} // Local CLI clients; auth still requires an unpredictable secret.
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" && strings.EqualFold(parsed.Host, r.Host)
}

func (a *panelAuth) serveLogin(w http.ResponseWriter, r *http.Request) {
	if !a.enabled {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if r.Method == "GET" {
		if a.isAuthenticated(r) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		loginPage(w, "")
		return
	}
	if r.Method != "POST" {
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !sameOrigin(r) {
		http.Error(w, "invalid origin", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 2048)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	remote, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remote = r.RemoteAddr
	}
	token, ok, err := a.acceptLogin(remote, r.Form.Get("password"))
	if err != nil {
		http.Error(w, "login unavailable", http.StatusServiceUnavailable)
		return
	}
	if !ok {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		loginPage(w, "Credencial incorrecta o demasiados intentos. Probá de nuevo en un minuto.")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: panelSessionCookie, Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: int(panelSessionLifetime.Seconds())})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func loginPage(w http.ResponseWriter, errorMessage string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
	_, _ = io.WriteString(w, `<!doctype html><html lang="es"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>SolarisPKN MultiChat — Acceso</title><style>body{font:16px system-ui;background:#131622;color:#f2f3fb;display:grid;place-items:center;min-height:95vh}main{width:min(92vw,400px);background:#1e2331;padding:28px;border-radius:18px}label,input,button{display:block;width:100%;box-sizing:border-box;margin:12px 0}input{background:#10131a;color:white;border:1px solid #58617f;border-radius:8px;padding:12px}button{border:0;border-radius:8px;padding:12px;background:#a8bbf8;color:#101421;font-weight:700;cursor:pointer}.error{color:#ff9da3}</style></head><body><main><h2>✦ SolarisPKN MultiChat</h2><p>Acceso local a tus chats.</p>`)
	if errorMessage != "" {
		_, _ = fmt.Fprintf(w, "<p class=\"error\">%s</p>", html.EscapeString(errorMessage))
	}
	_, _ = io.WriteString(w, `<form action="/auth/login" method="post"><label for="password">Clave de acceso al panel</label><input id="password" name="password" type="password" minlength="32" autocomplete="current-password" required autofocus><button type="submit">Ingresar</button></form><small>Solo en tu equipo. No se utiliza tu contraseña de Google, Twitch ni Facebook.</small></main></body></html>`)
}

func (a *panelAuth) serveLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !sameOrigin(r) {
		http.Error(w, "invalid origin", http.StatusForbidden)
		return
	}
	if cookie, err := r.Cookie(panelSessionCookie); err == nil {
		a.mu.Lock()
		delete(a.sessions, cookie.Value)
		a.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: panelSessionCookie, Path: "/", Value: "", HttpOnly: true, MaxAge: -1, SameSite: http.SameSiteStrictMode})
	http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
}

func (a *panelAuth) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Frame-Options", "DENY")
		switch {
		case r.URL.Path == "/auth/login":
			a.serveLogin(w, r)
			return
		case r.URL.Path == "/auth/logout":
			a.serveLogout(w, r)
			return
		case r.URL.Path == "/auth/status":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]bool{"enabled": a.enabled, "authenticated": a.isAuthenticated(r)})
			return
		case r.URL.Path == "/health":
			next.ServeHTTP(w, r)
			return
		case r.URL.Path == "/api/ingest" && r.Method == "POST":
			next.ServeHTTP(w, r)
			return // independent token protection in existing handler
		}
		if !a.isAuthenticated(r) && !a.isOBSAuthorized(r) {
			if r.Method == "GET" && r.URL.Path == "/" {
				http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
				return
			}
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
