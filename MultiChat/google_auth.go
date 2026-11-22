package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const googleScope = "https://www.googleapis.com/auth/youtube.readonly"
const googleAuthorizationEndpoint = "https://accounts.google.com/o/oauth2/v2/auth"
const googleTokenEndpoint = "https://oauth2.googleapis.com/token"

type savedGoogleCredentials struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

var googleRefreshMu sync.Mutex

func savedGooglePath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "SolarisPKN-LiveTools", "MultiChat", "google-oauth.dpapi"), nil
}

func loadGoogleCredentials() (savedGoogleCredentials, error) {
	var token savedGoogleCredentials
	path, err := savedGooglePath()
	if err != nil {
		return token, err
	}
	encrypted, err := os.ReadFile(path)
	if err != nil {
		return token, err
	}
	data, err := unprotectLocalCredential(encrypted)
	if err != nil {
		return token, err
	}
	if err := json.Unmarshal(data, &token); err != nil {
		return token, errors.New("invalid saved Google credential")
	}
	if token.RefreshToken == "" {
		return token, errors.New("saved Google credential has no refresh token")
	}
	return token, nil
}

func saveGoogleCredentials(token savedGoogleCredentials) error {
	if token.AccessToken == "" || token.RefreshToken == "" {
		return errors.New("incomplete Google OAuth token response")
	}
	encoded, err := json.Marshal(token)
	if err != nil {
		return err
	}
	protected, err := protectLocalCredential(encoded)
	if err != nil {
		return err
	}
	path, err := savedGooglePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), "google-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if err = temp.Chmod(0600); err != nil {
		temp.Close()
		return err
	}
	if _, err = temp.Write(protected); err != nil {
		temp.Close()
		return err
	}
	if err = temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err = temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), path)
}

type googleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Error        string `json:"error"`
}

func requestGoogleToken(ctx context.Context, values url.Values) (googleTokenResponse, error) {
	var result googleTokenResponse
	req, err := http.NewRequestWithContext(ctx, "POST", googleTokenEndpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return result, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return result, errors.New("Google OAuth token endpoint unavailable")
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(io.LimitReader(resp.Body, 512<<10)).Decode(&result); err != nil {
		return result, errors.New("invalid Google OAuth token response")
	}
	if resp.StatusCode != http.StatusOK || result.Error != "" {
		return result, fmt.Errorf("Google OAuth rejected token exchange (HTTP %d, %s)", resp.StatusCode, result.Error)
	}
	if result.AccessToken == "" || result.ExpiresIn < 1 {
		return result, errors.New("Google OAuth response omitted access_token or expires_in")
	}
	return result, nil
}

func currentGoogleToken(ctx context.Context) (string, error) {
	googleRefreshMu.Lock()
	defer googleRefreshMu.Unlock()
	stored, err := loadGoogleCredentials()
	if err != nil {
		return "", fmt.Errorf("Google account not linked; run MultiChat -link-google")
	}
	if stored.AccessToken != "" && time.Until(stored.ExpiresAt) > 2*time.Minute {
		return stored.AccessToken, nil
	}
	clientID := os.Getenv("MULTICHAT_GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("MULTICHAT_GOOGLE_CLIENT_SECRET")
	if clientID == "" {
		return "", errors.New("Google OAuth client ID is missing for token renewal")
	}
	values := url.Values{"client_id": {clientID}, "refresh_token": {stored.RefreshToken}, "grant_type": {"refresh_token"}}
	if clientSecret != "" {
		values.Set("client_secret", clientSecret)
	}
	fresh, err := requestGoogleToken(ctx, values)
	if err != nil {
		return "", err
	}
	stored.AccessToken = fresh.AccessToken
	if fresh.RefreshToken != "" {
		stored.RefreshToken = fresh.RefreshToken
	}
	stored.ExpiresAt = time.Now().Add(time.Duration(fresh.ExpiresIn) * time.Second)
	if err := saveGoogleCredentials(stored); err != nil {
		return "", err
	}
	return stored.AccessToken, nil
}

func randomOAuthValue() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

func runGoogleLink(ctx context.Context) error {
	clientID := os.Getenv("MULTICHAT_GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("MULTICHAT_GOOGLE_CLIENT_SECRET")
	if !strings.HasSuffix(clientID, ".apps.googleusercontent.com") {
		return errors.New("set MULTICHAT_GOOGLE_CLIENT_ID to the Google Desktop OAuth client ID")
	}
	// Desktop OAuth client secrets are optional for some providers but supported here.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer listener.Close()
	state, err := randomOAuthValue()
	if err != nil {
		return err
	}
	verifier, err := randomOAuthValue()
	if err != nil {
		return err
	}
	digest := sha256.Sum256([]byte(verifier))
	redirect := "http://" + listener.Addr().String() + "/oauth/google/callback"
	q := url.Values{
		"client_id":             {clientID},
		"redirect_uri":          {redirect},
		"response_type":         {"code"},
		"scope":                 {googleScope},
		"access_type":           {"offline"},
		"prompt":                {"consent"},
		"state":                 {state},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(digest[:])},
		"code_challenge_method": {"S256"},
	}
	authorizeURL := googleAuthorizationEndpoint + "?" + q.Encode()
	type callbackResult struct {
		code string
		err  error
	}
	codeChannel := make(chan callbackResult, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /oauth/google/callback", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.Host != listener.Addr().String() || subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("state")), []byte(state)) != 1 {
			http.Error(w, "Invalid OAuth state or callback host", http.StatusForbidden)
			return
		}
		var res callbackResult
		if r.URL.Query().Get("error") != "" {
			res.err = errors.New("Google authorization was declined")
		}
		res.code = r.URL.Query().Get("code")
		if res.code == "" && res.err == nil {
			res.err = errors.New("Missing authorization code")
		}
		select {
		case codeChannel <- res:
			_, _ = io.WriteString(w, "<html><body><h2>SolarisPKN MultiChat</h2><p>Autorización recibida. Volvé a MultiChat.</p></body></html>")
		default:
			http.Error(w, "Authorization already processed", http.StatusConflict)
		}
	})
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = server.Serve(listener) }()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	fmt.Println("Abrí este enlace para autorizar YouTube en modo solo lectura:")
	fmt.Println(authorizeURL)
	if runtime.GOOS == "windows" {
		// A normal user-launched process opens the system browser. Not executed automatically at startup.
		_ = exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", authorizeURL).Start()
	}
	var result callbackResult
	select {
	case result = <-codeChannel:
	case <-time.After(3 * time.Minute):
		return errors.New("Google login timed out after 3 minutes")
	case <-ctx.Done():
		return ctx.Err()
	}
	if result.err != nil {
		return result.err
	}
	values := url.Values{"code": {result.code}, "client_id": {clientID}, "redirect_uri": {redirect}, "grant_type": {"authorization_code"}, "code_verifier": {verifier}}
	if clientSecret != "" {
		values.Set("client_secret", clientSecret)
	}
	fresh, err := requestGoogleToken(ctx, values)
	if err != nil {
		return err
	}
	if fresh.RefreshToken == "" {
		return errors.New("Google did not return an offline refresh token; reauthorize with offline consent")
	}
	token := savedGoogleCredentials{AccessToken: fresh.AccessToken, RefreshToken: fresh.RefreshToken, ExpiresAt: time.Now().Add(time.Duration(fresh.ExpiresIn) * time.Second)}
	if err := saveGoogleCredentials(token); err != nil {
		return err
	}
	fmt.Println("Google YouTube: autorizado. Token protegido localmente para el usuario de Windows actual.")
	return nil
}
