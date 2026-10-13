package main

import (
 "context"
 "encoding/json"
 "errors"
 "io"
 "net/http"
 "net/url"
 "os"
 "strconv"
 "strings"
 "time"
)
type rumbleResponse struct{
 Livestreams []struct{
  ID any `json:"id"`
  Chat struct{
   RecentMessages []struct{
    ID any `json:"id"`
    Username string `json:"username"`
    Text string `json:"text"`
    CreatedOn any `json:"created_on"`
   } `json:"recent_messages"`
  } `json:"chat"`
 } `json:"livestreams"`
}
func rumbleLoop(ctx context.Context,c RumbleConfig,h *Hub,logf func(string,...any)){
 endpoint:=os.Getenv(c.APIURLEnv)
 u,err:=url.Parse(endpoint)
 if err!=nil||u.Scheme!="https"||(u.Hostname()!="rumble.com"&&!strings.HasSuffix(u.Hostname(),".rumble.com")){
  logf("[rumble] invalid API URL: HTTPS rumble.com URL required");return
 }
 client:=&http.Client{Timeout:12*time.Second,CheckRedirect:func(req *http.Request,via []*http.Request)error{
  if req.URL.Scheme!="https"||(req.URL.Hostname()!="rumble.com"&&!strings.HasSuffix(req.URL.Hostname(),".rumble.com")){return errors.New("redirect outside Rumble refused")}
  if len(via)>3{return errors.New("too many redirects")}
  return nil
 }}
 delay:=time.Duration(c.PollSeconds)*time.Second;wait:=time.Duration(0)
 for ctx.Err()==nil{
  if !sleepContext(ctx,wait){return}
  req,err:=http.NewRequestWithContext(ctx,"GET",endpoint,nil);if err!=nil{return}
  resp,err:=client.Do(req)
  if err!=nil{logf("[rumble] network error");wait=time.Minute;continue}
  raw,readErr:=io.ReadAll(io.LimitReader(resp.Body,2<<20));resp.Body.Close()
  if resp.StatusCode!=200||readErr!=nil{logf("[rumble] HTTP %d; waiting before retry",resp.StatusCode);wait=time.Minute;continue}
  var body rumbleResponse
  if err:=json.Unmarshal(raw,&body);err!=nil{logf("[rumble] invalid JSON response");wait=time.Minute;continue}
  for _,stream:=range body.Livestreams{
   for _,msg:=range stream.Chat.RecentMessages{
    id:=""
    if msg.ID!=nil{id=strings.TrimSpace(toString(msg.ID))}
    if id==""{id=toString(stream.ID)+"/"+msg.Username+"/"+msg.Text+"/"+toString(msg.CreatedOn)}
    h.publish(Message{ID:toString(stream.ID)+"/"+id,Platform:"rumble",User:msg.Username,Text:msg.Text,Kind:"chat"})
   }
  }
  wait=delay
 }
}
func toString(v any)string{
 switch t:=v.(type){
 case string:return t
 case float64:return strconv.FormatFloat(t,'f',-1,64)
 case json.Number:return t.String()
 default:return ""
 }
}
