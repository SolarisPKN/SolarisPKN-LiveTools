package main
import("testing";"time")
func TestBoundedHub(t *testing.T){
 h:=newHub(2)
 if !h.publish(Message{Platform:"Twitch",User:" A ",Text:" Hola ",ID:"1"}){t.Fatal("message rejected")}
 if h.publish(Message{Platform:"twitch",Text:"duplicate",ID:"1"}){t.Fatal("replay accepted")}
 h.publish(Message{Platform:"youtube",Text:"two",ID:"2"})
 h.publish(Message{Platform:"rumble",Text:"three",ID:"3"})
 if x:=h.snapshot();len(x)!=2||x[0].Platform!="youtube"{t.Fatalf("bad history: %+v",x)}
}
func TestTwitchParsing(t *testing.T){
 m,ok:=parseTwitchLine("@display-name=Hola;id=abc;tmi-sent-ts=1700000000000 :nick!n@n PRIVMSG #canal :Buenas")
 if !ok||m.Text!="Buenas"||m.User!="Hola"||m.ID!="abc"||m.Time.IsZero(){t.Fatalf("bad parse: %+v %v",m,ok)}
 if _,ok:=parseTwitchLine("PING :tmi.twitch.tv");ok{t.Fatal("PING was chat")}
 _=time.Now()
}
