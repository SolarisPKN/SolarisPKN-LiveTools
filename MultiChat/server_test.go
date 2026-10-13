package main
import("bytes";"encoding/json";"net/http";"net/http/httptest";"testing")
func TestProtectedIngest(t *testing.T){
 t.Setenv("MULTICHAT_INGEST_TOKEN","at-least-24-characters-long-token")
 c:=Config{Ingest:IngestConfig{Enabled:true,TokenEnv:"MULTICHAT_INGEST_TOKEN"}}
 h:=newHub(5);srv:=httptest.NewServer(newHandler(h,c));defer srv.Close()
 request:=func(token string)int{
  req,_:=http.NewRequest("POST",srv.URL+"/api/ingest",bytes.NewBufferString(`{"platform":"demo","text":"hola"}`))
  req.Header.Set("Authorization",token)
  resp,err:=srv.Client().Do(req);if err!=nil{t.Fatal(err)};defer resp.Body.Close();return resp.StatusCode
 }
 if request("Bearer bad")!=401{t.Fatal("invalid token admitted")}
 if request("Bearer at-least-24-characters-long-token")!=202{t.Fatal("valid token rejected")}
 req,_:=http.NewRequest("GET",srv.URL+"/history",nil)
 resp,err:=srv.Client().Do(req);if err!=nil{t.Fatal(err)}
 defer resp.Body.Close();var msgs []Message
 if err:=json.NewDecoder(resp.Body).Decode(&msgs);err!=nil||len(msgs)!=1{t.Fatalf("bad history: %+v %v",msgs,err)}
 req2,_:=http.NewRequest("GET",srv.URL+"/health",nil);req2.Host="evil.example"
 resp2,err:=srv.Client().Do(req2);if err!=nil{t.Fatal(err)};defer resp2.Body.Close()
 if resp2.StatusCode!=403{t.Fatal("Host check broken")}
}
