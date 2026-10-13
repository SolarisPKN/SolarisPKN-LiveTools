package main

import (
 "context"
 "encoding/json"
 "io"
 "net/http"
 "net/url"
 "os"
 "time"
)
type youtubeResponse struct{
 NextPageToken string `json:"nextPageToken"`
 PollingIntervalMillis int `json:"pollingIntervalMillis"`
 Items []struct{
  ID string `json:"id"`
  Snippet struct{Type string `json:"type"`;DisplayMessage string `json:"displayMessage"`;PublishedAt time.Time `json:"publishedAt"`} `json:"snippet"`
  AuthorDetails struct{DisplayName string `json:"displayName"`} `json:"authorDetails"`
 } `json:"items"`
}
func youtubeLoop(ctx context.Context,c YouTubeConfig,h *Hub,logf func(string,...any)){
 client:=&http.Client{Timeout:12*time.Second}
 var pageToken string
 windowStart:=time.Now();spent:=0
 base:=time.Duration(c.PollSeconds)*time.Second
 wait:=time.Duration(0)
 for ctx.Err()==nil{
  if !sleepContext(ctx,wait){return}
  if time.Since(windowStart)>=24*time.Hour{windowStart=time.Now();spent=0}
  if spent+5>c.DailyQuotaUnits{logf("[youtube] local quota budget exhausted (%d units); stopping connector",spent);return}
  spent+=5
  q:=url.Values{"part":{"snippet,authorDetails"},"liveChatId":{c.LiveChatID},"maxResults":{"200"},"key":{os.Getenv(c.APIKeyEnv)}}
  if pageToken!=""{q.Set("pageToken",pageToken)}
  req,err:=http.NewRequestWithContext(ctx,"GET","https://www.googleapis.com/youtube/v3/liveChat/messages?"+q.Encode(),nil)
  if err!=nil{return}
  resp,err:=client.Do(req)
  if err!=nil{logf("[youtube] request error: %v",err);wait=time.Minute;continue}
  raw,readErr:=io.ReadAll(io.LimitReader(resp.Body,2<<20));resp.Body.Close()
  if readErr!=nil{logf("[youtube] read error");wait=time.Minute;continue}
  if resp.StatusCode!=200{
   logf("[youtube] HTTP %d (quota, permissions or stream state); retry delayed",resp.StatusCode)
   wait=2*time.Minute
   if resp.StatusCode==401||resp.StatusCode==403||resp.StatusCode==404{return}
   continue
  }
  var payload youtubeResponse
  if err:=json.Unmarshal(raw,&payload);err!=nil{logf("[youtube] invalid response");wait=time.Minute;continue}
  for _,x:=range payload.Items{
   h.publish(Message{ID:x.ID,Platform:"youtube",User:x.AuthorDetails.DisplayName,Text:x.Snippet.DisplayMessage,Kind:x.Snippet.Type,Time:x.Snippet.PublishedAt})
  }
  pageToken=payload.NextPageToken
  wait=base
  if serverWait:=time.Duration(payload.PollingIntervalMillis)*time.Millisecond;serverWait>wait{wait=serverWait}
 }
}
