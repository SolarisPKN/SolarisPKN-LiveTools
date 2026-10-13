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
func newHandler(h *Hub,c Config) http.Handler{
 mux:=http.NewServeMux()
 mux.HandleFunc("GET /",func(w http.ResponseWriter,r *http.Request){
  if r.URL.Path!="/"{http.NotFound(w,r);return}
  w.Header().Set("Content-Type","text/html; charset=utf-8")
  w.Header().Set("Content-Security-Policy","default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'")
  _,_=io.WriteString(w,webPage)
 })
 mux.HandleFunc("GET /health",func(w http.ResponseWriter,r *http.Request){
  w.Header().Set("Content-Type","application/json")
  _=json.NewEncoder(w).Encode(map[string]any{"ok":true,"history_count":len(h.snapshot()),"configured":map[string]bool{"twitch":c.Twitch.Enabled,"youtube":c.YouTube.Enabled,"rumble":c.Rumble.Enabled,"ingest":c.Ingest.Enabled}})
 })
 mux.HandleFunc("GET /history",func(w http.ResponseWriter,r *http.Request){
  w.Header().Set("Content-Type","application/json")
  _=json.NewEncoder(w).Encode(h.snapshot())
 })
 mux.HandleFunc("GET /events",func(w http.ResponseWriter,r *http.Request){
  w.Header().Set("Content-Type","text/event-stream")
  w.Header().Set("Cache-Control","no-cache")
  w.Header().Set("X-Accel-Buffering","no")
  f,ok:=w.(http.Flusher);if !ok{http.Error(w,"stream unavailable",500);return}
  ch,remove:=h.subscribe();defer remove()
  _,_=io.WriteString(w,": connected\n\n");f.Flush()
  keepalive:=time.NewTicker(20*time.Second);defer keepalive.Stop()
  for{select{
   case <-r.Context().Done():return
   case msg:=<-ch:
    b,_:=json.Marshal(msg)
    if _,err:=fmt.Fprintf(w,"data: %s\n\n",b);err!=nil{return}
    f.Flush()
   case <-keepalive.C:
    if _,err:=io.WriteString(w,": ping\n\n");err!=nil{return}
    f.Flush()
  }}
 })
 mux.HandleFunc("POST /api/ingest",func(w http.ResponseWriter,r *http.Request){
  if !c.Ingest.Enabled{http.NotFound(w,r);return}
  expected:=os.Getenv(c.Ingest.TokenEnv)
  received:=strings.TrimPrefix(r.Header.Get("Authorization"),"Bearer ")
  if subtle.ConstantTimeCompare([]byte(received),[]byte(expected))!=1{http.Error(w,"unauthorized",401);return}
  r.Body=http.MaxBytesReader(w,r.Body,16<<10)
  var m Message
  if err:=json.NewDecoder(r.Body).Decode(&m);err!=nil{http.Error(w,"invalid JSON",400);return}
  if m.Platform==""||m.Text==""{http.Error(w,"platform and text required",400);return}
  h.publish(m);w.WriteHeader(202)
 })
 return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  host,_,err:=net.SplitHostPort(r.Host);if err!=nil{host=r.Host}
  if host!="127.0.0.1"&&host!="localhost"&&host!="::1"{http.Error(w,"invalid host",403);return}
  w.Header().Set("X-Content-Type-Options","nosniff")
  w.Header().Set("Referrer-Policy","no-referrer")
  mux.ServeHTTP(w,r)
 })
}
