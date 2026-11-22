// SolarisPKN MultiChat Native — Windows desktop shell.
// Pure Win32 controls and GDI+ vector marks; no WebView/Electron/CEF/browser runtime.
// The actual collectors and outbound authorization remain in the Go backend.
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <objidl.h>
#include <gdiplus.h>
#include <winhttp.h>
#include <algorithm>
#include <atomic>
#include <memory>
#include <string>
#include <thread>
#include <vector>
#include <chrono>
#pragma comment(lib,"winhttp.lib")
#pragma comment(lib,"gdiplus.lib")
#pragma comment(lib,"gdi32.lib")
#pragma comment(lib,"user32.lib")
#pragma comment(lib,"comctl32.lib")
using namespace Gdiplus;

namespace {
constexpr unsigned kPort = 8766;
constexpr UINT kPollMessage = WM_APP+101;
constexpr UINT kSentMessage = WM_APP+102;
constexpr int kIconBase=610;
constexpr int kSend=641,kAll=642,kNone=643,kChat=644,kMessage=645,kResult=646,kStatus=647;
const wchar_t *kPlatformNames[5]={L"YouTube",L"TikTok",L"Twitch",L"Kick",L"Facebook"};
const char *kPlatformKeys[5]={"youtube","tiktok","twitch","kick","facebook"};
const wchar_t *kDemoConfig=L"NativeUI\\demo-config.json";
HINSTANCE instance;
HWND windowHandle=nullptr,chatHandle=nullptr,messageHandle=nullptr,resultHandle=nullptr,statusHandle=nullptr;
HWND iconButtons[5]{},sendButton=nullptr,allButton=nullptr,noneButton=nullptr;
HFONT fontNormal=nullptr,fontStrong=nullptr,fontSmall=nullptr;
HBRUSH brushDark=nullptr,brushEdit=nullptr;
std::atomic<bool> pollBusy{false},sendBusy{false};
bool selected[5]={false,false,false,false,false},ready[5]={false,false,false,false,false},isDemo=false;
std::wstring lastChat,appDirectory;
std::wstring resultText=L"Modo de prueba: todavía no enviaste nada.";
COLORREF textColor=RGB(233,232,244);

std::wstring utf8ToWide(const std::string &s){
 if(s.empty())return {};
 int n=MultiByteToWideChar(CP_UTF8,MB_ERR_INVALID_CHARS,s.data(),int(s.size()),nullptr,0);
 if(n<1)return L"";
 std::wstring w(n,L'\0');
 MultiByteToWideChar(CP_UTF8,MB_ERR_INVALID_CHARS,s.data(),int(s.size()),w.data(),n);
 return w;
}
std::string wideToUtf8(const std::wstring &s){
 if(s.empty())return {};
 int n=WideCharToMultiByte(CP_UTF8,0,s.data(),int(s.size()),nullptr,0,nullptr,nullptr);
 std::string u(n,'\0');
 WideCharToMultiByte(CP_UTF8,0,s.data(),int(s.size()),u.data(),n,nullptr,nullptr);
 return u;
}
std::wstring trimWide(std::wstring s){
 const auto first=s.find_first_not_of(L" \t\r\n");
 if(first==std::wstring::npos)return L"";
 const auto last=s.find_last_not_of(L" \t\r\n");
 return s.substr(first,last-first+1);
}
std::string escapeJSON(const std::string &s){
 std::string out;
 for(unsigned char c:s){
  switch(c){
  case '"':out+="\\\"";break;
  case '\\':out+="\\\\";break;
  case '\n':out+="\\n";break;
  case '\r':out+="\\r";break;
  case '\t':out+="\\t";break;
  default:if(c<32){out+=' ';}else out+=char(c);break;
  }
 }
 return out;
}

struct HTTPResult {int status=0;std::string body;};
HTTPResult request(const wchar_t *method,const wchar_t *uri,const std::string &payload={},bool action=false){
 HTTPResult answer;
 HINTERNET session=WinHttpOpen(L"SolarisPKN-MultiChat-Native/0.1",WINHTTP_ACCESS_TYPE_NO_PROXY,nullptr,nullptr,0);
 if(!session)return answer;
 WinHttpSetTimeouts(session,700,700,1600,1600);
 HINTERNET connection=WinHttpConnect(session,L"127.0.0.1",kPort,0);
 if(connection){
  HINTERNET req=WinHttpOpenRequest(connection,method,uri,nullptr,WINHTTP_NO_REFERER,WINHTTP_DEFAULT_ACCEPT_TYPES,0);
  if(req){
   const wchar_t *headers=action?L"Content-Type: application/json\r\nX-SolarisPKN-Action: send\r\n":nullptr;
   DWORD headersLen=action?(DWORD)-1:0;
   bool ok=WinHttpSendRequest(req,headers,headersLen,
    payload.empty()?WINHTTP_NO_REQUEST_DATA:(LPVOID)payload.data(),
    (DWORD)payload.size(),(DWORD)payload.size(),0);
   if(ok&&WinHttpReceiveResponse(req,nullptr)){
    DWORD code=0,bytes=sizeof(code);
    if(WinHttpQueryHeaders(req,WINHTTP_QUERY_STATUS_CODE|WINHTTP_QUERY_FLAG_NUMBER,WINHTTP_HEADER_NAME_BY_INDEX,&code,&bytes,WINHTTP_NO_HEADER_INDEX))answer.status=int(code);
    char buffer[4096];DWORD got=0;
    while(WinHttpReadData(req,buffer,sizeof(buffer),&got)&&got!=0){
     if(answer.body.size()+got>120000){answer.body.clear();break;}
     answer.body.append(buffer,got);
    }
   }
   WinHttpCloseHandle(req);
  }
  WinHttpCloseHandle(connection);
 }
 WinHttpCloseHandle(session);
 return answer;
}
bool hasJSONValue(const std::string &data,const char *k,const char *v){
 return data.find(std::string("\"")+k+"\":\""+v+"\"")!=std::string::npos;
}
bool jsonBoolAtPlatform(const std::string &data,const char *key){
 std::string needle=std::string("\"platform\":\"")+key+"\"";
 auto start=data.find(needle);
 if(start==std::string::npos)return false;
 auto end=data.find('}',start);
 if(end==std::string::npos)return false;
 return data.substr(start,end-start).find("\"ready\":true")!=std::string::npos;
}
void enableSend(){
 if(!sendButton||!messageHandle)return;
 int count=0;
 for(int i=0;i<5;i++)if(selected[i]&&ready[i])count++;
 int len=GetWindowTextLengthW(messageHandle);
 EnableWindow(sendButton,!sendBusy.load()&&count>0&&len>0&&len<=200);
}
void redrawIcons(){
 for(int i=0;i<5;i++){
  if(iconButtons[i]){
   EnableWindow(iconButtons[i],ready[i]);
   InvalidateRect(iconButtons[i],nullptr,FALSE);
  }
 }
 enableSend();
}
struct PollState{
 HTTPResult feed;
 HTTPResult targets;
};
void pollAsync(HWND target){
 if(pollBusy.exchange(true))return;
 std::thread([target](){
  auto *state=new PollState{request(L"GET",L"/native/feed"),request(L"GET",L"/api/outbound/status")};
  if(!PostMessageW(target,kPollMessage,0,(LPARAM)state))delete state;
  pollBusy=false;
 }).detach();
}
void setResult(const std::wstring &s){
 resultText=s;
 if(resultHandle)SetWindowTextW(resultHandle,s.c_str());
}
void sendNow(){
 if(sendBusy.load()||!messageHandle)return;
 int len=GetWindowTextLengthW(messageHandle);
 if(len<=0||len>200)return;
 std::wstring msg(len+1,L'\0');
 GetWindowTextW(messageHandle,msg.data(),len+1);
 msg.resize(wcslen(msg.c_str()));
 msg=trimWide(msg);
 if(msg.empty()||msg.find_first_of(L"\r\n")!=std::wstring::npos)return;
 std::vector<int> targets;
 for(int i=0;i<5;i++)if(selected[i]&&ready[i])targets.push_back(i);
 if(targets.empty())return;
 if(!isDemo){
  std::wstring confirmation=L"¿Enviar este mensaje a "+std::to_wstring(targets.size())+L" plataforma(s)?\n\n"+msg;
  if(MessageBoxW(windowHandle,confirmation.c_str(),L"Confirmar envío real",MB_YESNO|MB_ICONQUESTION|MB_DEFBUTTON2)!=IDYES)return;
 }
 std::string payload="{\"message\":\""+escapeJSON(wideToUtf8(msg))+"\",\"targets\":[";
 for(size_t i=0;i<targets.size();i++){
  if(i)payload+=",";
  payload+="\""+std::string(kPlatformKeys[targets[i]])+"\"";
 }
 payload+="]}";
 if(sendBusy.exchange(true))return;
 setResult(L"Enviando a "+std::to_wstring(targets.size())+L" destino(s)...");
 enableSend();
 HWND target=windowHandle;
 std::thread([target,payload=std::move(payload)](){
  HTTPResult *result=new HTTPResult(request(L"POST",L"/api/outbound/send",payload,true));
  if(!PostMessageW(target,kSentMessage,0,(LPARAM)result))delete result;
 }).detach();
}
void colorBackground(HDC hdc,const RECT &r,COLORREF c){
 HBRUSH b=CreateSolidBrush(c);
 FillRect(hdc,&r,b);DeleteObject(b);
}
void roundedBox(Graphics &g,RectF r,Color color,REAL radius){
 GraphicsPath path;
 REAL d=radius*2;
 path.AddArc(r.X,r.Y,d,d,180,90);
 path.AddArc(r.X+r.Width-d,r.Y,d,d,270,90);
 path.AddArc(r.X+r.Width-d,r.Y+r.Height-d,d,d,0,90);
 path.AddArc(r.X,r.Y+r.Height-d,d,d,90,90);
 path.CloseFigure();
 SolidBrush fill(color);
 g.FillPath(&fill,&path);
}
void drawLogo(Graphics &g,int id,RectF r,bool colored){
 g.SetSmoothingMode(SmoothingModeAntiAlias);
 const Color grayBg(255,75,78,91),grayFg(255,165,168,178);
 const Color white(255,255,255,255);
 const REAL x=r.X,y=r.Y,w=r.Width,h=r.Height;
 switch(id){
 case 0:{
  roundedBox(g,r,colored?Color(255,255,47,68):grayBg,7.5f);
  PointF p[3]={{x+w*.41f,y+h*.27f},{x+w*.73f,y+h*.50f},{x+w*.41f,y+h*.73f}};
  SolidBrush b(colored?white:grayFg);g.FillPolygon(&b,p,3);break;
 }
 case 1:{
  roundedBox(g,r,Color(255,27,28,37),7.f);
  Pen cy(colored?Color(255,55,227,219):grayFg,5.0f),red(colored?Color(255,255,72,123):grayFg,5.0f),wh(colored?white:grayFg,4.f);
  cy.SetStartCap(LineCapRound);cy.SetEndCap(LineCapRound);
  red.SetStartCap(LineCapRound);red.SetEndCap(LineCapRound);
  wh.SetStartCap(LineCapRound);wh.SetEndCap(LineCapRound);
  auto drawNote=[&](Pen *p,REAL dx,REAL dy){
   g.DrawLine(p,x+w*.58f+dx,y+h*.15f+dy,x+w*.58f+dx,y+h*.66f+dy);
   g.DrawArc(p,x+w*.28f+dx,y+h*.50f+dy,w*.37f,h*.34f,5.f,300.f);
   g.DrawLine(p,x+w*.58f+dx,y+h*.22f+dy,x+w*.81f+dx,y+h*.34f+dy);
  };
  drawNote(&cy,-2.0f,1.6f);drawNote(&red,1.6f,-.7f);drawNote(&wh,0,0);break;
 }
 case 2:{
  PointF outer[7]={{x+w*.08f,y+h*.08f},{x+w*.94f,y+h*.08f},{x+w*.94f,y+h*.67f},{x+w*.70f,y+h*.91f},{x+w*.52f,y+h*.91f},{x+w*.31f,y+h*1.0f},{x+w*.31f,y+h*.91f}};
  SolidBrush b(colored?Color(255,164,107,250):grayBg);g.FillPolygon(&b,outer,7);
  SolidBrush dark(Color(255,35,25,53));
  g.FillRectangle(&dark,x+w*.20f,y+h*.20f,w*.61f,h*.46f);
  SolidBrush eyes(colored?white:grayFg);
  g.FillRectangle(&eyes,x+w*.41f,y+h*.30f,w*.10f,h*.25f);
  g.FillRectangle(&eyes,x+w*.61f,y+h*.30f,w*.10f,h*.25f);break;
 }
 case 3:{
  roundedBox(g,r,Color(255,22,27,24),7.f);
  SolidBrush b(colored?Color(255,75,242,43):grayFg);
  g.FillRectangle(&b,x+w*.20f,y+h*.18f,w*.20f,h*.64f);
  g.FillRectangle(&b,x+w*.43f,y+h*.43f,w*.19f,h*.18f);
  g.FillRectangle(&b,x+w*.62f,y+h*.24f,w*.18f,h*.20f);
  g.FillRectangle(&b,x+w*.62f,y+h*.61f,w*.18f,h*.20f);
  break;
 }
 case 4:{
  roundedBox(g,r,colored?Color(255,40,110,247):grayBg,7.f);
  FontFamily family(L"Segoe UI");
  Font font(&family,r.Height*.80f,FontStyleBold,UnitPixel);
  SolidBrush b(colored?white:grayFg);
  g.DrawString(L"f",-1,&font,PointF(x+w*.34f,y-h*.10f),&b);break;
 }
 }
}
void drawIconButton(DRAWITEMSTRUCT *d){
 const int index=int(d->CtlID)-kIconBase;
 if(index<0||index>=5)return;
 RECT r=d->rcItem;
 HDC hdc=d->hDC;
 Graphics g(hdc);
 g.SetSmoothingMode(SmoothingModeAntiAlias);
 const bool can=ready[index];
 const bool checked=selected[index]&&can;
 roundedBox(g,RectF((REAL)r.left,(REAL)r.top,(REAL)(r.right-r.left),(REAL)(r.bottom-r.top)),
  checked?Color(255,63,47,93):Color(255,32,33,45),8.f);
 if(checked){
  Pen outline(Color(255,186,145,250),2.f);
  g.DrawRectangle(&outline,REAL(r.left+1),REAL(r.top+1),REAL(r.right-r.left-3),REAL(r.bottom-r.top-3));
 }
 drawLogo(g,index,RectF(REAL(r.left+7),REAL(r.top+4),REAL(r.right-r.left-14),REAL(r.bottom-r.top-11)),can);
 if(checked){
  SolidBrush mark(Color(255,105,221,164));
  g.FillEllipse(&mark,REAL(r.right-15),REAL(r.top+2),11.f,11.f);
 }
}
void setFont(HWND hwnd,HFONT f){SendMessageW(hwnd,WM_SETFONT,(WPARAM)f,TRUE);}
HWND createChild(const wchar_t *cls,const wchar_t *txt,DWORD style,int id){
 HWND h=CreateWindowExW(0,cls,txt,WS_CHILD|WS_VISIBLE|style,0,0,0,0,windowHandle,(HMENU)(INT_PTR)id,instance,nullptr);
 if(h)setFont(h,fontNormal);return h;
}
std::wstring applicationPath(){
 wchar_t path[MAX_PATH]{};
 GetModuleFileNameW(nullptr,path,MAX_PATH);
 std::wstring p(path);
 auto i=p.find_last_of(L"\\/");
 return i==std::wstring::npos?L"." : p.substr(0,i);
}
bool alreadyRunning(){
 return request(L"GET",L"/health").status==200;
}
void startBackend(){
 if(alreadyRunning())return;
 std::wstring exe=appDirectory+L"\\MultiChat-native-engine.exe";
 std::wstring cfg=appDirectory+L"\\NativeUI\\demo-config.json";
 if(GetFileAttributesW(exe.c_str())==INVALID_FILE_ATTRIBUTES){return;}
 std::wstring args=L"\""+exe+L"\" -config \""+cfg+L"\" -demo -quiet";
 STARTUPINFOW si{};si.cb=sizeof(si);
 PROCESS_INFORMATION pi{};
 std::vector<wchar_t> mutableArgs(args.begin(),args.end());mutableArgs.push_back(0);
 if(CreateProcessW(exe.c_str(),mutableArgs.data(),nullptr,nullptr,FALSE,CREATE_NO_WINDOW,nullptr,appDirectory.c_str(),&si,&pi)){
  CloseHandle(pi.hThread);CloseHandle(pi.hProcess);
 }
}
void layout(){
 RECT r{};GetClientRect(windowHandle,&r);
 const int w=r.right,h=r.bottom,left=w/3;
 const int inner=left-32;
 // All controls fit vertically with no overlap, even on a 1366x768 screen.
 MoveWindow(chatHandle,16,105,inner,std::max(100,h-398),TRUE);
 const int selectorY=h-222;
 for(int i=0;i<5;i++)MoveWindow(iconButtons[i],16+i*(inner/5),selectorY,inner/5-5,49,TRUE);
 MoveWindow(allButton,16,h-252,(inner-8)/2,26,TRUE);
 MoveWindow(noneButton,24+(inner-8)/2,h-252,(inner-8)/2,26,TRUE);
 MoveWindow(messageHandle,16,h-144,inner,44,TRUE);
 MoveWindow(sendButton,16,h-90,inner,34,TRUE);
 MoveWindow(resultHandle,16,h-51,inner,43,TRUE);
 MoveWindow(statusHandle,16,61,inner,22,TRUE);
}
void updateStatus(const PollState &state){
 if(state.targets.status!=200){
  isDemo=false;
  for(int i=0;i<5;i++)ready[i]=false;
  if(statusHandle)SetWindowTextW(statusHandle,L"○ Conectando al motor de chats…");
 }else{
  isDemo=hasJSONValue(state.targets.body,"mode","demo");
  bool live=hasJSONValue(state.targets.body,"mode","live");
  for(int i=0;i<5;i++)ready[i]=jsonBoolAtPlatform(state.targets.body,kPlatformKeys[i]);
  if(statusHandle)SetWindowTextW(statusHandle,isDemo?L"◆ DEMO — no publica en Internet":live?L"● Chat conectado · envío bajo autorización":L"○ Envío real desactivado");
 }
 redrawIcons();
}
LRESULT CALLBACK windowProcedure(HWND window,UINT msg,WPARAM wp,LPARAM lp){
 switch(msg){
 case WM_CREATE:{
  windowHandle=window;
  brushDark=CreateSolidBrush(RGB(18,20,31));
  brushEdit=CreateSolidBrush(RGB(24,26,39));
  fontNormal=CreateFontW(-17,0,0,0,FW_NORMAL,FALSE,FALSE,FALSE,DEFAULT_CHARSET,OUT_DEFAULT_PRECIS,CLIP_DEFAULT_PRECIS,CLEARTYPE_QUALITY,DEFAULT_PITCH,L"Segoe UI");
  fontStrong=CreateFontW(-19,0,0,0,FW_BOLD,FALSE,FALSE,FALSE,DEFAULT_CHARSET,OUT_DEFAULT_PRECIS,CLIP_DEFAULT_PRECIS,CLEARTYPE_QUALITY,DEFAULT_PITCH,L"Segoe UI");
  fontSmall=CreateFontW(-14,0,0,0,FW_NORMAL,FALSE,FALSE,FALSE,DEFAULT_CHARSET,OUT_DEFAULT_PRECIS,CLIP_DEFAULT_PRECIS,CLEARTYPE_QUALITY,DEFAULT_PITCH,L"Segoe UI");
  statusHandle=createChild(L"STATIC",L"○ Iniciando MultiChat…",SS_LEFT,kStatus);
  setFont(statusHandle,fontSmall);
  chatHandle=createChild(L"EDIT",L"Conectando…",ES_MULTILINE|ES_READONLY|ES_AUTOVSCROLL|WS_VSCROLL|WS_BORDER,kChat);
  messageHandle=createChild(L"EDIT",L"",ES_MULTILINE|ES_AUTOVSCROLL|WS_BORDER,kMessage);
  SendMessageW(messageHandle,EM_LIMITTEXT,200,0);
  for(int i=0;i<5;i++){
   iconButtons[i]=createChild(L"BUTTON",kPlatformNames[i],BS_OWNERDRAW|WS_TABSTOP,kIconBase+i);
  }
  allButton=createChild(L"BUTTON",L"Seleccionar todas",BS_PUSHBUTTON,kAll);
  noneButton=createChild(L"BUTTON",L"Deseleccionar todas",BS_PUSHBUTTON,kNone);
  setFont(allButton,fontSmall);setFont(noneButton,fontSmall);
  sendButton=createChild(L"BUTTON",L"Enviar mensaje / comando",BS_DEFPUSHBUTTON,kSend);
  resultHandle=createChild(L"STATIC",resultText.c_str(),SS_LEFT,kResult);
  setFont(resultHandle,fontSmall);
  layout();
  enableSend();
  SetTimer(window,1,1800,nullptr);
  startBackend();
  pollAsync(window);
  return 0;
 }
 case WM_COMMAND:{
  int id=LOWORD(wp);
  if(id>=kIconBase&&id<kIconBase+5&&HIWORD(wp)==BN_CLICKED){
   int idx=id-kIconBase;
   if(ready[idx])selected[idx]=!selected[idx];
   redrawIcons();return 0;
  }
  if(id==kAll&&HIWORD(wp)==BN_CLICKED){
   for(int i=0;i<5;i++)selected[i]=ready[i];
   redrawIcons();return 0;
  }
  if(id==kNone&&HIWORD(wp)==BN_CLICKED){
   for(int i=0;i<5;i++)selected[i]=false;
   redrawIcons();return 0;
  }
  if(id==kSend&&HIWORD(wp)==BN_CLICKED){sendNow();return 0;}
  if(id==kMessage&&HIWORD(wp)==EN_CHANGE){enableSend();return 0;}
  return 0;
 }
 case WM_TIMER:
  if(wp==1)pollAsync(window);
  return 0;
 case kPollMessage:{
  std::unique_ptr<PollState> state((PollState*)lp);
  if(state->feed.status==200){
   std::wstring feed=utf8ToWide(state->feed.body);
   if(feed!=lastChat){
    lastChat=feed;
    SetWindowTextW(chatHandle,feed.c_str());
    int length=GetWindowTextLengthW(chatHandle);
    SendMessageW(chatHandle,EM_SETSEL,length,length);
    SendMessageW(chatHandle,EM_SCROLLCARET,0,0);
   }
  }
  updateStatus(*state);
  return 0;
 }
 case kSentMessage:{
  std::unique_ptr<HTTPResult> answer((HTTPResult*)lp);
  sendBusy=false;
  if(answer->status!=200){
   setResult(L"Error del motor de envío (HTTP "+std::to_wstring(answer->status)+L"). No se reintentó.");
  }else{
   auto &json=answer->body;
   int simulated=0,accepted=0,failed=0,unavailable=0;
   size_t p=0;
   while((p=json.find("\"status\":\"",p))!=std::string::npos){
    p+=10;
    auto end=json.find('"',p);
    if(end==std::string::npos)break;
    std::string s=json.substr(p,end-p);p=end+1;
    if(s=="simulated")simulated++;
    else if(s=="accepted")accepted++;
    else if(s=="unavailable")unavailable++;
    else failed++;
   }
   if(isDemo)setResult(L"DEMO: "+std::to_wstring(simulated)+L" envío(s) simulados. Nada publicado.");
   else setResult(L"API aceptó "+std::to_wstring(accepted)+L"; no disponible "+std::to_wstring(unavailable)+L"; errores "+std::to_wstring(failed)+L".");
  }
  enableSend();
  return 0;
 }
 case WM_DRAWITEM:drawIconButton((DRAWITEMSTRUCT*)lp);return TRUE;
 case WM_CTLCOLOREDIT:
 case WM_CTLCOLORSTATIC:{
  HDC hdc=(HDC)wp;
  SetTextColor(hdc,textColor);
  SetBkColor(hdc,RGB(18,20,31));
  return (LRESULT)brushDark;
 }
 case WM_ERASEBKGND:return 1;
 case WM_PAINT:{
  PAINTSTRUCT ps{};HDC hdc=BeginPaint(window,&ps);
  RECT r{};GetClientRect(window,&r);
  int left=r.right/3;
  RECT a{0,0,left,r.bottom},b{left,0,r.right,r.bottom},border{left-1,0,left,r.bottom};
  colorBackground(hdc,a,RGB(17,19,29));
  colorBackground(hdc,b,RGB(13,15,23));  // remaining 2/3 intentionally empty
  colorBackground(hdc,border,RGB(48,46,63));
  SetBkMode(hdc,TRANSPARENT);
  SetTextColor(hdc,RGB(237,234,250));
  HGDIOBJ old=SelectObject(hdc,fontStrong);
  TextOutW(hdc,16,18,L"✦ SolarisPKN MultiChat",21);
  TextOutW(hdc,16,91,L"CHAT",4);
  TextOutW(hdc,16,r.bottom-282,L"PLATAFORMAS DE ENVÍO",21);
  TextOutW(hdc,16,r.bottom-165,L"MENSAJE / COMANDO",17);
  SelectObject(hdc,old);
  EndPaint(window,&ps);
  return 0;
 }
 case WM_CLOSE:
  DestroyWindow(window);return 0;
 case WM_DESTROY:
  KillTimer(window,1);
  if(fontNormal)DeleteObject(fontNormal);
  if(fontStrong)DeleteObject(fontStrong);
  if(fontSmall)DeleteObject(fontSmall);
  if(brushDark)DeleteObject(brushDark);
  if(brushEdit)DeleteObject(brushEdit);
  PostQuitMessage(0);return 0;
 }
 return DefWindowProcW(window,msg,wp,lp);
}
} // namespace

int APIENTRY wWinMain(HINSTANCE app,HINSTANCE,LPWSTR,int show){
 instance=app;
 appDirectory=applicationPath();
 GdiplusStartupInput input{};
 ULONG_PTR gdiplusToken=0;
 if(GdiplusStartup(&gdiplusToken,&input,nullptr)!=Ok)return 3;
 WNDCLASSW wc{};
 wc.lpfnWndProc=windowProcedure;
 wc.hInstance=app;
 wc.lpszClassName=L"SolarisPKNMultiChatNative_v1";
 wc.hCursor=LoadCursorW(nullptr,IDC_ARROW);
 wc.hIcon=LoadIconW(nullptr,IDI_APPLICATION);
 if(!RegisterClassW(&wc)){GdiplusShutdown(gdiplusToken);return 4;}
 RECT work{};SystemParametersInfoW(SPI_GETWORKAREA,0,&work,0);
 int desiredW=std::min<int>(1200,int(work.right-work.left)-36);
 int desiredH=desiredW*9/16;
 RECT frame{0,0,desiredW,desiredH};
 const DWORD style=WS_OVERLAPPED|WS_CAPTION|WS_SYSMENU|WS_MINIMIZEBOX;
 AdjustWindowRectEx(&frame,style,FALSE,0);
 while(frame.bottom-frame.top>(work.bottom-work.top-10)&&desiredW>760){
  desiredW-=16;desiredH=desiredW*9/16;
  frame={0,0,desiredW,desiredH};AdjustWindowRectEx(&frame,style,FALSE,0);
 }
 int outerW=frame.right-frame.left,outerH=frame.bottom-frame.top;
 int x=work.left+((work.right-work.left)-outerW)/2,y=work.top+((work.bottom-work.top)-outerH)/2;
 HWND wnd=CreateWindowExW(0,wc.lpszClassName,L"SolarisPKN MultiChat · Nativo",style,x,y,outerW,outerH,nullptr,nullptr,app,nullptr);
 if(!wnd){GdiplusShutdown(gdiplusToken);return 5;}
 ShowWindow(wnd,show);UpdateWindow(wnd);
 MSG msg{};
 while(GetMessageW(&msg,nullptr,0,0)>0){
  TranslateMessage(&msg);DispatchMessageW(&msg);
 }
 GdiplusShutdown(gdiplusToken);
 return int(msg.wParam);
}
