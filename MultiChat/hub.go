package main

import (
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type Message struct {
	ID       string    `json:"id,omitempty"`
	Platform string    `json:"platform"`
	User     string    `json:"user"`
	Text     string    `json:"text"`
	Kind     string    `json:"kind,omitempty"`
	Time     time.Time `json:"time"`
}
type Hub struct {
	mu sync.RWMutex
	recent []Message
	seen map[string]struct{}
	subscribers map[chan Message]struct{}
	cap int
}
func newHub(limit int) *Hub {
	if limit < 1 { limit = 250 }
	return &Hub{cap:limit, seen:make(map[string]struct{}), subscribers:make(map[chan Message]struct{})}
}
func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n { return s }
	return string([]rune(s)[:n])
}
func (h *Hub) publish(m Message) bool {
	m.Platform = truncate(strings.ToLower(strings.TrimSpace(m.Platform)),32)
	m.User = truncate(strings.TrimSpace(m.User),80)
	m.Text = truncate(strings.TrimSpace(m.Text),2000)
	m.ID = truncate(strings.TrimSpace(m.ID),160)
	m.Kind = truncate(strings.TrimSpace(m.Kind),32)
	if m.Platform == "" || m.Text == "" { return false }
	if m.User == "" { m.User = "unknown" }
	if m.Time.IsZero() { m.Time = time.Now().UTC() }
	key := ""
	if m.ID != "" { key = m.Platform+"/"+m.ID }
	h.mu.Lock(); defer h.mu.Unlock()
	if key != "" {
		if _, ok := h.seen[key]; ok { return false }
		h.seen[key] = struct{}{}
	}
	h.recent = append(h.recent,m)
	if len(h.recent) > h.cap { h.recent = h.recent[len(h.recent)-h.cap:] }
	if len(h.seen) > h.cap*4 {
		h.seen = make(map[string]struct{}, h.cap*2)
		for _, old := range h.recent { if old.ID != "" { h.seen[old.Platform+"/"+old.ID] = struct{}{} } }
	}
	for ch := range h.subscribers { select { case ch <- m: default: } }
	return true
}
func (h *Hub) subscribe() (chan Message,func()) {
	ch := make(chan Message,128)
	h.mu.Lock(); h.subscribers[ch] = struct{}{}; h.mu.Unlock()
	return ch,func(){h.mu.Lock(); delete(h.subscribers,ch); h.mu.Unlock()}
}
func (h *Hub) snapshot() []Message {
	h.mu.RLock(); defer h.mu.RUnlock()
	return append([]Message(nil),h.recent...)
}
