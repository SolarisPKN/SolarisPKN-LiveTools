package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

const twitchDeviceEndpoint = "https://id.twitch.tv/oauth2/device"
const twitchTokenEndpoint = "https://id.twitch.tv/oauth2/token"
const twitchValidateEndpoint = "https://id.twitch.tv/oauth2/validate"

type savedTwitchCredentials struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ClientID     string    `json:"client_id"`
	Login        string    `json:"login"`
	ExpiresAt    time.Time `json:"expires_at"`
}

var twitchRefreshMu sync.Mutex

func twitchCredentialsPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "SolarisPKN-LiveTools", "MultiChat", "twitch-oauth.dpapi"), nil
}
func loadTwitchCredentials() (savedTwitchCredentials, error) {
	var token savedTwitchCredentials
	path, err := twitchCredentialsPath()
	if err != nil {
		return token, err
	}
	encrypted, err := os.ReadFile(path)
	if err != nil {
		return token, err
	}
	clear, err := unprotectLocalCredential(encrypted)
	if err != nil {
		return token, err
	}
	if err := json.Unmarshal(clear, &token); err != nil {
		return token, errors.New("invalid saved Twitch credential")
	}
	if token.RefreshToken == "" || token.ClientID == "" || token.Login == "" {
		return token, errors.New("incomplete saved Twitch credential")
	}
	return token, nil
}

func saveTwitchCredentials(token savedTwitchCredentials) error {
	if token.AccessToken == "" || token.RefreshToken == "" || token.Login == "" || token.ClientID == "" {
		return errors.New("incomplete Twitch OAuth response")
	}
	encoded, err := json.Marshal(token)
	if err != nil {
		return err
	}
	protected, err := protectLocalCredential(encoded)
	if err != nil {
		return err
	}
	path, err := twitchCredentialsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "twitch-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err = tmp.Chmod(0600); err != nil {
		tmp.Close()
		return err
	}
	if _, err = tmp.Write(protected); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

type twitchOAuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Error        string `json:"error"`
	Message      string `json:"message"`
}

func twitchForm(ctx context.Context, endpoint string, values url.Values) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, errors.New("Twitch authorization endpoint unavailable")
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, data, nil
}

func validateTwitchToken(ctx context.Context, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", twitchValidateEndpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "OAuth "+token)
	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.New("Twitch token validation unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Twitch rejected token validation HTTP %d", resp.StatusCode)
	}
	var result struct {
		Login    string   `json:"login"`
		ClientID string   `json:"client_id"`
		Scopes   []string `json:"scopes"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 128<<10)).Decode(&result); err != nil {
		return "", err
	}
	canRead := false
	for _, scope := range result.Scopes {
		if scope == "chat:read" {
			canRead = true
		}
	}
	if !canRead || result.Login == "" {
		return "", errors.New("Twitch token lacks chat:read")
	}
	return result.Login, nil
}

func currentTwitchToken(ctx context.Context) (savedTwitchCredentials, error) {
	twitchRefreshMu.Lock()
	defer twitchRefreshMu.Unlock()
	token, err := loadTwitchCredentials()
	if err != nil {
		return token, errors.New("Twitch account not linked; run MultiChat -link-twitch")
	}
	if token.AccessToken != "" && time.Until(token.ExpiresAt) > 2*time.Minute {
		return token, nil
	}
	params := url.Values{"client_id": {token.ClientID}, "refresh_token": {token.RefreshToken}, "grant_type": {"refresh_token"}}
	status, data, err := twitchForm(ctx, twitchTokenEndpoint, params)
	if err != nil {
		return token, err
	}
	var result twitchOAuthResponse
	if json.Unmarshal(data, &result) != nil || status != 200 || result.AccessToken == "" || result.RefreshToken == "" || result.ExpiresIn < 1 {
		return token, fmt.Errorf("Twitch token refresh failed HTTP %d; reconnect using -link-twitch", status)
	}
	token.AccessToken = result.AccessToken
	token.RefreshToken = result.RefreshToken // Twitch refresh tokens rotate; persist new one.
	token.ExpiresAt = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	if err := saveTwitchCredentials(token); err != nil {
		return token, err
	}
	return token, nil
}

func runTwitchLink(ctx context.Context) error {
	clientID := strings.TrimSpace(os.Getenv("MULTICHAT_TWITCH_CLIENT_ID"))
	if clientID == "" {
		return errors.New("set MULTICHAT_TWITCH_CLIENT_ID to your public Twitch app client ID")
	}
	params := url.Values{"client_id": {clientID}, "scopes": {"chat:read"}}
	status, data, err := twitchForm(ctx, twitchDeviceEndpoint, params)
	if err != nil {
		return err
	}
	if status != 200 {
		return fmt.Errorf("Twitch device authorization HTTP %d", status)
	}
	var device struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURI string `json:"verification_uri"`
		ExpiresIn       int    `json:"expires_in"`
		Interval        int    `json:"interval"`
	}
	if err := json.Unmarshal(data, &device); err != nil {
		return err
	}
	if device.DeviceCode == "" || device.UserCode == "" || device.ExpiresIn <= 0 {
		return errors.New("Twitch device flow returned incomplete response")
	}
	verifyURL, err := url.Parse(device.VerificationURI)
	if err != nil || verifyURL.Scheme != "https" || !(verifyURL.Hostname() == "www.twitch.tv" || verifyURL.Hostname() == "twitch.tv") {
		return errors.New("unexpected Twitch verification host")
	}
	if device.Interval < 5 {
		device.Interval = 5
	}
	if device.Interval > 30 {
		device.Interval = 30
	}
	fmt.Println("Vinculá Twitch para leer el chat. Entrá a:", device.VerificationURI)
	fmt.Println("Código de activación:", device.UserCode)
	if runtime.GOOS == "windows" {
		_ = exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", device.VerificationURI).Start()
	}
	until := time.Now().Add(time.Duration(device.ExpiresIn) * time.Second)
	maxUntil := time.Now().Add(15 * time.Minute)
	if until.After(maxUntil) {
		until = maxUntil
	}
	for time.Now().Before(until) {
		if !sleepContext(ctx, time.Duration(device.Interval)*time.Second) {
			return ctx.Err()
		}
		values := url.Values{"client_id": {clientID}, "scopes": {"chat:read"}, "device_code": {device.DeviceCode}, "grant_type": {"urn:ietf:params:oauth:grant-type:device_code"}}
		status, raw, err := twitchForm(ctx, twitchTokenEndpoint, values)
		if err != nil {
			return err
		}
		var result twitchOAuthResponse
		if json.Unmarshal(raw, &result) != nil {
			return errors.New("invalid Twitch token response")
		}
		if status == 200 {
			if result.AccessToken == "" || result.RefreshToken == "" || result.ExpiresIn < 1 {
				return errors.New("Twitch returned incomplete tokens")
			}
			login, err := validateTwitchToken(ctx, result.AccessToken)
			if err != nil {
				return err
			}
			token := savedTwitchCredentials{AccessToken: result.AccessToken, RefreshToken: result.RefreshToken, Login: login, ClientID: clientID, ExpiresAt: time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)}
			if err := saveTwitchCredentials(token); err != nil {
				return err
			}
			fmt.Println("Twitch chat: autorizado para", login, "(token protegido para el usuario actual de Windows).")
			return nil
		}
		msg := strings.ToLower(result.Error + " " + result.Message)
		if strings.Contains(msg, "authorization_pending") {
			continue
		}
		if strings.Contains(msg, "slow_down") {
			device.Interval += 5
			continue
		}
		if strings.Contains(msg, "access_denied") || strings.Contains(msg, "authorization_declined") {
			return errors.New("Twitch authorization declined")
		}
		if strings.Contains(msg, "expired_token") {
			return errors.New("Twitch device code expired")
		}
		return fmt.Errorf("Twitch device exchange failed HTTP %d", status)
	}
	return errors.New("Twitch device authorization expired; retry -link-twitch")
}
