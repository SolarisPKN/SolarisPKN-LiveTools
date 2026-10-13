package main

import (
 "bufio"
 "context"
 "crypto/tls"
 "errors"
 "fmt"
 "io"
 "net"
 "os"
 "strings"
 "time"
)
func twitchLoop(ctx context.Context,c TwitchConfig,hub *Hub,logf func(string,...any)){
 delay:=time.Second
 for ctx.Err()==nil{
  err:=twitchSession(ctx,c,hub)
  if ctx.Err()!=nil{return}
  logf("[twitch] reconnecting after: %v",err)
  if !sleepContext(ctx,delay){return}
  if delay<time.Minute{delay*=2}
 }
}
func twitchSession(ctx context.Context,c TwitchConfig,hub *Hub)error{
 d:=&net.Dialer{Timeout:10*time.Second,KeepAlive:20*time.Second}
 raw,err:=tls.DialWithDialer(d,"tcp","irc.chat.twitch.tv:6697",&tls.Config{MinVersion:tls.VersionTLS12,ServerName:"irc.chat.twitch.tv"})
 if err!=nil{return err}
 defer raw.Close()
 sessionCtx,endSession:=context.WithCancel(ctx);defer endSession()
 go func(){<-sessionCtx.Done();raw.Close()}()
 channel:=strings.TrimPrefix(strings.ToLower(c.Channel),"#")
 token:=strings.TrimPrefix(os.Getenv(c.TokenEnv),"oauth:")
 if strings.ContainsAny(channel," \r\n")||strings.ContainsAny(c.Nick," \r\n"){return errors.New("invalid twitch channel or nick")}
 for _,line:=range []string{"PASS oauth:"+token,"NICK "+strings.ToLower(c.Nick),"CAP REQ :twitch.tv/tags twitch.tv/commands","JOIN #"+channel}{
  if _,err=io.WriteString(raw,line+"\r\n");err!=nil{return err}
 }
 scanner:=bufio.NewScanner(raw);scanner.Buffer(make([]byte,2048),64*1024)
 for scanner.Scan(){
  line:=scanner.Text()
  if strings.HasPrefix(line,"PING "){
   if _,err=io.WriteString(raw,"PONG "+strings.TrimPrefix(line,"PING ")+"\r\n");err!=nil{return err}
   continue
  }
  if strings.Contains(line,"Login authentication failed")||strings.Contains(line,"Improperly formatted auth"){return errors.New("Twitch OAuth rejected")}
  if m,ok:=parseTwitchLine(line);ok{hub.publish(m)}
 }
 if err:=scanner.Err();err!=nil{return err}
 return fmt.Errorf("Twitch IRC connection closed")
}
func parseTwitchLine(line string)(Message,bool){
 var m Message
 tags:=""
 if strings.HasPrefix(line,"@"){
  idx:=strings.IndexByte(line,' ')
  if idx<0{return m,false}
  tags=line[1:idx];line=line[idx+1:]
 }
 if !strings.HasPrefix(line,":"){return m,false}
 idx:=strings.Index(line," PRIVMSG #");if idx<0{return m,false}
 prefix:=line[1:idx]
 colon:=strings.Index(line[idx+10:]," :");if colon<0{return m,false}
 body:=line[idx+10+colon+2:]
 user:=strings.SplitN(prefix,"!",2)[0]
 m=Message{Platform:"twitch",User:user,Text:body,Kind:"chat"}
 for _,tag:=range strings.Split(tags,";"){
  p:=strings.SplitN(tag,"=",2);if len(p)<2{continue}
  switch p[0]{
   case "id":m.ID=p[1]
   case "display-name":if p[1]!=""{m.User=p[1]}
   case "tmi-sent-ts":if ms,err:=time.ParseDuration(p[1]+"ms");err==nil{m.Time=time.Unix(0,int64(ms)).UTC()}
  }
 }
 return m,true
}
func sleepContext(ctx context.Context,d time.Duration)bool{
 t:=time.NewTimer(d);defer t.Stop()
 select{case <-ctx.Done():return false;case <-t.C:return true}
}
