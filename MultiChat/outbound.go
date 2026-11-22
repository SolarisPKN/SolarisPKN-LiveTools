package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

var outboundPlatforms = []string{"youtube", "tiktok", "twitch", "kick", "facebook"}

type OutboundConfig struct {
	Enabled               bool   `json:"enabled"`
	KickTokenEnv          string `json:"kick_token_env"`
	KickBroadcasterUserID int64  `json:"kick_broadcaster_user_id"`
}

type outboundTarget struct {
	Platform string `json:"platform"`
	Ready    bool   `json:"ready"`
	Detail   string `json:"detail"`
}
type outboundRequest struct {
	Message string   `json:"message"`
	Targets []string `json:"targets"`
}
type outboundResult struct {
	Platform string `json:"platform"`
	Status   string `json:"status"`
	Detail   string `json:"detail"`
}
type outboundResponse struct {
	Mode    string           `json:"mode"`
	Results []outboundResult `json:"results"`
}

type outboundService struct {
	config   Config
	demo     bool
	hub      *Hub
	mu       sync.Mutex
	lastSend time.Time
	client   *http.Client
}

func newOutboundService(c Config, h *Hub) *outboundService {
	return &outboundService{config: c, demo: c.Demo, hub: h, client: &http.Client{Timeout: 15 * time.Second}}
}
func (o *outboundService) capabilities() []outboundTarget {
	statuses := make([]outboundTarget, 0, len(outboundPlatforms))
	if o.demo {
		for _, name := range outboundPlatforms {
			statuses = append(statuses, outboundTarget{Platform: name, Ready: true, Detail: "Simulación local: no publica en Internet"})
		}
		return statuses
	}
	_, googleErr := loadGoogleCredentials()
	_, twitchErr := loadTwitchCredentials()
	for _, p := range outboundPlatforms {
		v := outboundTarget{Platform: p}
		switch {
		case !o.config.Outbound.Enabled:
			v.Detail = "Habilitá outbound.enabled con el panel protegido"
		case p == "youtube":
			v.Ready = o.config.YouTube.Enabled && (o.config.YouTube.UseSavedAuth && googleErr == nil || o.config.YouTube.OAuthTokenEnv != "" && os.Getenv(o.config.YouTube.OAuthTokenEnv) != "")
			v.Detail = "Requiere OAuth de escritura YouTube y un chat LIVE activo"
		case p == "twitch":
			v.Ready = o.config.Twitch.Enabled && o.config.Twitch.UseSavedAuth && twitchErr == nil
			v.Detail = "Requiere OAuth Twitch con user:write:chat"
		case p == "facebook":
			v.Ready = o.config.Facebook.Enabled && o.config.Facebook.AccessTokenEnv != "" && os.Getenv(o.config.Facebook.AccessTokenEnv) != ""
			v.Detail = "Requiere token de Página con permiso de publicar comentarios"
		case p == "kick":
			v.Ready = o.config.Kick.Enabled && o.config.Outbound.KickTokenEnv != "" && os.Getenv(o.config.Outbound.KickTokenEnv) != "" && o.config.Outbound.KickBroadcasterUserID > 0
			v.Detail = "Requiere OAuth Kick chat:write y broadcaster_user_id"
		case p == "tiktok":
			v.Detail = "Solo lectura: falta un adaptador autenticado de envío"
		}
		statuses = append(statuses, v)
	}
	return statuses
}

func (o *outboundService) serveStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	mode := "disabled"
	if o.demo {
		mode = "demo"
	} else if o.config.Outbound.Enabled {
		mode = "live"
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"mode": mode, "targets": o.capabilities()})
}

func (o *outboundService) serveSend(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	// Unlike /api/ingest, outbound writes always require browser-origin safety checks.
	if !sameOrigin(r) || r.Header.Get("X-SolarisPKN-Action") != "send" || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "unsafe origin or missing action header", http.StatusForbidden)
		return
	}
	if !o.demo && !o.config.Outbound.Enabled {
		http.Error(w, "real sending is disabled", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request outboundRequest
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "invalid JSON request", http.StatusBadRequest)
		return
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		http.Error(w, "unexpected trailing JSON", http.StatusBadRequest)
		return
	}
	request.Message = strings.TrimSpace(request.Message)
	chars := utf8.RuneCountInString(request.Message)
	if chars < 1 || chars > 200 || strings.ContainsAny(request.Message, "\r\n\x00") {
		http.Error(w, "message must contain 1-200 characters and no line breaks", http.StatusBadRequest)
		return
	}
	if len(request.Targets) == 0 || len(request.Targets) > len(outboundPlatforms) {
		http.Error(w, "select 1-5 platforms", http.StatusBadRequest)
		return
	}
	seen := map[string]bool{}
	for _, name := range request.Targets {
		valid := false
		for _, known := range outboundPlatforms {
			if name == known {
				valid = true
				break
			}
		}
		if !valid || seen[name] {
			http.Error(w, "invalid or duplicate platform selection", http.StatusBadRequest)
			return
		}
		seen[name] = true
	}
	o.mu.Lock()
	if time.Since(o.lastSend) < 2*time.Second {
		o.mu.Unlock()
		http.Error(w, "outbound cooldown: wait 2 seconds", http.StatusTooManyRequests)
		return
	}
	o.lastSend = time.Now()
	o.mu.Unlock()

	response := outboundResponse{Mode: "live", Results: make([]outboundResult, len(request.Targets))}
	if o.demo {
		response.Mode = "demo"
		for i, p := range request.Targets {
			response.Results[i] = outboundResult{Platform: p, Status: "simulated", Detail: "Simulado localmente: nada fue publicado"}
			o.hub.publish(Message{ID: fmt.Sprintf("demo-sent-%d-%d", time.Now().UnixNano(), i), Platform: p, User: "TU MENSAJE · DEMO", Text: request.Message, Kind: "outbound_demo", Time: time.Now()})
		}
		_ = json.NewEncoder(w).Encode(response)
		return
	}
	capabilities := o.capabilities()
	enabled := map[string]bool{}
	for _, target := range capabilities {
		enabled[target.Platform] = target.Ready
	}
	var wg sync.WaitGroup
	for i, p := range request.Targets {
		if !enabled[p] {
			response.Results[i] = outboundResult{Platform: p, Status: "unavailable", Detail: "No disponible: falta permiso de escritura, token o conector de envío"}
			continue
		}
		wg.Add(1)
		go func(i int, p string) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(r.Context(), 18*time.Second)
			defer cancel()
			err := o.sendTo(ctx, p, request.Message)
			if err != nil {
				response.Results[i] = outboundResult{Platform: p, Status: "error", Detail: err.Error()}
				return
			}
			response.Results[i] = outboundResult{Platform: p, Status: "accepted", Detail: "Aceptado por la API; aparición en el chat no confirmada"}
		}(i, p)
	}
	wg.Wait()
	_ = json.NewEncoder(w).Encode(response)
}

func (o *outboundService) sendTo(ctx context.Context, platform, message string) error {
	switch platform {
	case "youtube":
		return o.sendYouTube(ctx, message)
	case "twitch":
		return o.sendTwitch(ctx, message)
	case "facebook":
		return o.sendFacebook(ctx, message)
	case "kick":
		return o.sendKick(ctx, message)
	default:
		return errors.New("envío a esta plataforma no implementado")
	}
}

func (o *outboundService) sendYouTube(ctx context.Context, message string) error {
	c := o.config.YouTube
	token := os.Getenv(c.OAuthTokenEnv)
	if c.UseSavedAuth {
		var err error
		token, err = currentGoogleToken(ctx)
		if err != nil {
			return errors.New("YouTube OAuth no disponible; vincular la cuenta")
		}
	}
	if token == "" {
		return errors.New("YouTube requiere OAuth con permiso de escribir mensajes")
	}
	chatID := c.LiveChatID
	var err error
	if chatID == "" && c.VideoID != "" {
		chatID, err = youtubeLookupLiveChatID(ctx, o.client, c.VideoID, "", token)
	}
	if err != nil {
		return fmt.Errorf("YouTube chat lookup: %w", err)
	}
	if chatID == "" && c.VideoID == "" {
		chatID, err = youtubeActiveChatID(ctx, o.client, token)
	}
	if err != nil {
		return fmt.Errorf("YouTube current LIVE lookup: %w", err)
	}
	if chatID == "" {
		return errors.New("YouTube: no se encontró un chat LIVE activo")
	}
	body := map[string]any{"snippet": map[string]any{"liveChatId": chatID, "type": "textMessageEvent", "textMessageDetails": map[string]string{"messageText": message}}}
	encoded, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://www.googleapis.com/youtube/v3/liveChat/messages?part=snippet", bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	return acceptedJSONResponse(o.client, req, "youtube", func(raw []byte) error {
		var value struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(raw, &value) != nil || value.ID == "" {
			return errors.New("YouTube: respuesta sin ID de mensaje; estado de entrega desconocido")
		}
		return nil
	})
}

func (o *outboundService) sendTwitch(ctx context.Context, message string) error {
	c := o.config.Twitch
	if !c.UseSavedAuth {
		return errors.New("Twitch necesita OAuth vinculado con user:write:chat")
	}
	token, err := currentTwitchToken(ctx)
	if err != nil {
		return errors.New("Twitch OAuth no disponible; vincular la cuenta")
	}
	return o.sendTwitchAuthorized(ctx, message, token)
}

func (o *outboundService) sendTwitchAuthorized(ctx context.Context, message string, token savedTwitchCredentials) error {
	c := o.config.Twitch
	if token.AccessToken == "" || token.ClientID == "" {
		return errors.New("Twitch access token o Client ID ausente")
	}
	// Verify scope and authenticated user before writing. A chat:read-only token must never send.
	valReq, err := http.NewRequestWithContext(ctx, "GET", "https://id.twitch.tv/oauth2/validate", nil)
	if err != nil {
		return err
	}
	valReq.Header.Set("Authorization", "OAuth "+token.AccessToken)
	valResp, err := o.client.Do(valReq)
	if err != nil {
		return errors.New("Twitch OAuth validation network error")
	}
	defer valResp.Body.Close()
	if valResp.StatusCode != 200 {
		return fmt.Errorf("Twitch OAuth validation HTTP %d", valResp.StatusCode)
	}
	var validation struct {
		UserID   string   `json:"user_id"`
		ClientID string   `json:"client_id"`
		Scopes   []string `json:"scopes"`
	}
	if err := json.NewDecoder(io.LimitReader(valResp.Body, 64<<10)).Decode(&validation); err != nil {
		return errors.New("Twitch validation invalid JSON")
	}
	permitted := false
	for _, scope := range validation.Scopes {
		if scope == "user:write:chat" {
			permitted = true
		}
	}
	if !permitted {
		return errors.New("Twitch permiso user:write:chat ausente; volver a vincular con -link-twitch")
	}
	if validation.UserID == "" || validation.ClientID != token.ClientID {
		return errors.New("Twitch authenticated sender mismatch")
	}
	q := url.Values{"login": {strings.TrimPrefix(strings.TrimSpace(c.Channel), "#")}}
	getReq, err := http.NewRequestWithContext(ctx, "GET", "https://api.twitch.tv/helix/users?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	getReq.Header.Set("Authorization", "Bearer "+token.AccessToken)
	getReq.Header.Set("Client-Id", token.ClientID)
	getResp, err := o.client.Do(getReq)
	if err != nil {
		return errors.New("Twitch channel ID lookup network error")
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != 200 {
		return fmt.Errorf("Twitch channel ID HTTP %d", getResp.StatusCode)
	}
	var users struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(getResp.Body, 64<<10)).Decode(&users); err != nil || len(users.Data) == 0 || users.Data[0].ID == "" {
		return errors.New("Twitch broadcaster ID unavailable")
	}
	body := map[string]string{"broadcaster_id": users.Data[0].ID, "sender_id": validation.UserID, "message": message}
	encoded, _ := json.Marshal(body)
	post, err := http.NewRequestWithContext(ctx, "POST", "https://api.twitch.tv/helix/chat/messages", bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	post.Header.Set("Authorization", "Bearer "+token.AccessToken)
	post.Header.Set("Client-Id", token.ClientID)
	post.Header.Set("Content-Type", "application/json")
	return acceptedJSONResponse(o.client, post, "twitch", func(raw []byte) error {
		var result struct {
			Data []struct {
				IsSent     bool `json:"is_sent"`
				DropReason *struct {
					Code string `json:"code"`
				} `json:"drop_reason"`
			} `json:"data"`
		}
		if json.Unmarshal(raw, &result) != nil || len(result.Data) == 0 {
			return errors.New("Twitch: respuesta sin confirmación de aceptación")
		}
		if !result.Data[0].IsSent {
			detail := "Twitch rechazó el mensaje"
			if result.Data[0].DropReason != nil {
				detail += ": " + result.Data[0].DropReason.Code
			}
			return errors.New(detail)
		}
		return nil
	})
}

func (o *outboundService) sendFacebook(ctx context.Context, message string) error {
	c := o.config.Facebook
	token := os.Getenv(c.AccessTokenEnv)
	if token == "" {
		return errors.New("Facebook: falta token autorizado de Página")
	}
	id := c.LiveVideoID
	if c.PageID != "" {
		var err error
		id, err = facebookActiveLiveVideoID(ctx, o.client, c.PageID, token)
		if err != nil {
			return fmt.Errorf("Facebook LIVE lookup: %w", err)
		}
	}
	if id == "" {
		return errors.New("Facebook: no hay video LIVE activo")
	}
	params := url.Values{"message": {message}}
	req, err := http.NewRequestWithContext(ctx, "POST", "https://graph.facebook.com/v27.0/"+url.PathEscape(id)+"/comments", strings.NewReader(params.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return acceptedJSONResponse(o.client, req, "facebook", func(raw []byte) error {
		var result struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(raw, &result) != nil || result.ID == "" {
			return errors.New("Facebook: no devolvió un ID de comentario; entrega desconocida")
		}
		return nil
	})
}
func (o *outboundService) sendKick(ctx context.Context, message string) error {
	c := o.config.Outbound
	token := os.Getenv(c.KickTokenEnv)
	if token == "" || c.KickBroadcasterUserID <= 0 {
		return errors.New("Kick: se necesita OAuth chat:write y broadcaster_user_id")
	}
	body := map[string]any{"type": "user", "broadcaster_user_id": c.KickBroadcasterUserID, "content": message}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.kick.com/public/v1/chat", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	return acceptedJSONResponse(o.client, req, "kick", nil)
}

func acceptedJSONResponse(client *http.Client, req *http.Request, platform string, check func([]byte) error) error {
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%s: error de conexión; entrega no confirmada", platform)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	if err != nil {
		return fmt.Errorf("%s: error leyendo respuesta; entrega desconocida", platform)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s: HTTP %d (permiso, sesión, límite o chat inactivo)", platform, resp.StatusCode)
	}
	if check != nil {
		return check(body)
	}
	return nil
}
