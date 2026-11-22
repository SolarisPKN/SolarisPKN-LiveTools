package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	configPath := flag.String("config", "config.json", "configuration file path")
	demo := flag.Bool("demo", false, "emit sample messages locally (no real platform connections)")
	quiet := flag.Bool("quiet", false, "do not print incoming chat in terminal")
	linkGoogle := flag.Bool("link-google", false, "authorize YouTube readonly in the browser and store Google OAuth protected for current Windows user")
	unlinkGoogle := flag.Bool("unlink-google", false, "erase local saved Google OAuth credentials (does not revoke Google account permissions)")
	linkTwitch := flag.Bool("link-twitch", false, "authorize Twitch chat:read via the Twitch device code flow")
	unlinkTwitch := flag.Bool("unlink-twitch", false, "erase locally saved Twitch OAuth credentials")
	flag.Parse()
	if *linkTwitch {
		if err := runTwitchLink(context.Background()); err != nil {
			log.Fatalf("Twitch link failed: %v", err)
		}
		return
	}
	if *unlinkTwitch {
		path, err := twitchCredentialsPath()
		if err != nil {
			log.Fatal(err)
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			log.Fatal(err)
		}
		log.Println("Twitch OAuth credentials erased locally. Revoke the app in Twitch if desired.")
		return
	}
	if *linkGoogle {
		if err := runGoogleLink(context.Background()); err != nil {
			log.Fatalf("Google link failed: %v", err)
		}
		return
	}
	if *unlinkGoogle {
		path, err := savedGooglePath()
		if err != nil {
			log.Fatal(err)
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			log.Fatal(err)
		}
		log.Println("Local Google OAuth credentials removed; you can revoke permissions in your Google Account separately.")
		return
	}
	cfg, err := readConfig(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	h := newHub(cfg.HistoryLimit)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log.Printf("SolarisPKN MultiChat | listening at http://%s | browser-free collectors", cfg.Listen)
	if !*quiet {
		ch, remove := h.subscribe()
		defer remove()
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case m := <-ch:
					fmt.Printf("[%s] %s: %s\n", m.Platform, m.User, m.Text)
				}
			}
		}()
	}
	if !*demo {
		if cfg.Twitch.Enabled {
			go twitchLoop(ctx, cfg.Twitch, h, log.Printf)
		}
		if cfg.YouTube.Enabled {
			go youtubeLoop(ctx, cfg.YouTube, h, log.Printf)
		}
		if cfg.TikTok.Enabled {
			go tiktokLoop(ctx, cfg.TikTok, h, log.Printf)
		}
		if cfg.Kick.Enabled {
			go kickLoop(ctx, cfg.Kick, h, log.Printf)
		}
		if cfg.Facebook.Enabled {
			go facebookLoop(ctx, cfg.Facebook, h, log.Printf)
		}
		if cfg.Rumble.Enabled {
			go rumbleLoop(ctx, cfg.Rumble, h, log.Printf)
		}
	}
	if *demo {
		go func() {
			n := 0
			for ctx.Err() == nil {
				n++
				h.publish(Message{ID: fmt.Sprintf("demo-%d", n), Platform: []string{"youtube", "tiktok", "twitch", "kick", "facebook", "rumble"}[n%6], User: "DemoUser", Text: fmt.Sprintf("Mensaje de prueba %d ✨", n), Kind: "chat"})
				if !sleepContext(ctx, 2*time.Second) {
					return
				}
			}
		}()
	}
	srv := &http.Server{Addr: cfg.Listen, Handler: newHandler(h, cfg), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 35 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
