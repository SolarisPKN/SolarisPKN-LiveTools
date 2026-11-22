package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestCredentialProtectionWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("DPAPI storage is only supported on Windows")
	}
	plain := []byte("test-refresh-token-do-not-publish-" + time.Now().Format(time.RFC3339Nano))
	protected, err := protectLocalCredential(plain)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(plain, protected) || bytes.Contains(protected, plain) {
		t.Fatal("unencrypted OAuth credential returned")
	}
	restored, err := unprotectLocalCredential(protected)
	if err != nil || !bytes.Equal(plain, restored) {
		t.Fatalf("DPAPI roundtrip failed: %v", err)
	}
	if _, err := unprotectLocalCredential([]byte("invalid ciphertext")); err == nil {
		t.Fatal("invalid ciphertext accepted")
	}
}
func TestGoogleOAuthRequiresConsent(t *testing.T) {
	t.Setenv("MULTICHAT_GOOGLE_CLIENT_ID", "")
	if err := runGoogleLink(context.Background()); err == nil {
		t.Fatal("Google linking started without registered OAuth application")
	}
}
func TestTwitchOAuthRequiresClient(t *testing.T) {
	t.Setenv("MULTICHAT_TWITCH_CLIENT_ID", "")
	if err := runTwitchLink(context.Background()); err == nil {
		t.Fatal("Twitch linking started without client ID")
	}
}
func TestConfigSavedProviderModes(t *testing.T) {
	p := filepath.Join(t.TempDir(), "oauth.json")
	config := `{"youtube":{"enabled":true,"use_saved_google_auth":true},"twitch":{"enabled":true,"channel":"solarispkn","use_saved_twitch_auth":true}}`
	if err := os.WriteFile(p, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := readConfig(p)
	if err != nil {
		t.Fatal(err)
	}
	if !c.Twitch.UseSavedAuth || !c.YouTube.UseSavedAuth {
		t.Fatal("saved provider login modes not recognized")
	}
}
