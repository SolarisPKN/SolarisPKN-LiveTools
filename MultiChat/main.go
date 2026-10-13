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
	demo := flag.Bool("demo",false,"emit sample messages locally (no real platform connections)")
	quiet := flag.Bool("quiet",false,"do not print incoming chat in terminal")
	flag.Parse()
	cfg,err := readConfig(*configPath)
	if err != nil { log.Fatalf("config: %v",err) }
	h:=newHub(cfg.HistoryLimit)
	ctx,stop := signal.NotifyContext(context.Background(),os.Interrupt,syscall.SIGTERM)
	defer stop()
	log.Printf("SolarisPKN MultiChat | listening at http://%s | browser-free collectors",cfg.Listen)
	if !*quiet {
		ch,remove:=h.subscribe(); defer remove()
		go func(){ for { select {case <-ctx.Done():return; case m:=<-ch:fmt.Printf("[%s] %s: %s\n",m.Platform,m.User,m.Text)} } }()
	}
	if cfg.Twitch.Enabled {go twitchLoop(ctx,cfg.Twitch,h,log.Printf)}
	if cfg.YouTube.Enabled {go youtubeLoop(ctx,cfg.YouTube,h,log.Printf)}
	if cfg.Rumble.Enabled {go rumbleLoop(ctx,cfg.Rumble,h,log.Printf)}
	if *demo {go func(){n:=0;for ctx.Err()==nil {n++;h.publish(Message{ID:fmt.Sprintf("demo-%d",n),Platform:[]string{"twitch","youtube","kick","rumble"}[n%4],User:"DemoUser",Text:fmt.Sprintf("Mensaje de prueba %d ✨",n),Kind:"chat"});if !sleepContext(ctx,2*time.Second){return}}}()}
	srv:=&http.Server{Addr:cfg.Listen,Handler:newHandler(h,cfg),ReadHeaderTimeout:5*time.Second,IdleTimeout:35*time.Second}
	go func(){<-ctx.Done(); shutdownCtx,cancel:=context.WithTimeout(context.Background(),5*time.Second);defer cancel();_ = srv.Shutdown(shutdownCtx)}()
	if err:=srv.ListenAndServe();err!=nil&&err!=http.ErrServerClosed{log.Fatal(err)}
}
