// SolarisPKN MultiChat — compact Windows native sidebar, no Chromium/WebView/Electron.
// UI only: all platform network and OBS stream activity lives in the Go engine.
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <objidl.h>
#include <richedit.h>
#include <gdiplus.h>
#include <winhttp.h>
#include <shellapi.h>
#include <algorithm>
#include <atomic>
#include <memory>
#include <string>
#include <thread>
#include <vector>
#include <chrono>
#include <cwchar>
#include <cmath>
#pragma comment(lib,"winhttp.lib")
#pragma comment(lib,"gdiplus.lib")
#pragma comment(lib,"gdi32.lib")
#pragma comment(lib,"user32.lib")
#pragma comment(lib,"shell32.lib")
#pragma comment(lib,"comctl32.lib")
using namespace Gdiplus;
namespace {
constexpr INTERNET_PORT kPort=8766;
constexpr UINT kPollMessage=WM_APP+101,kSentMessage=WM_APP+102;
constexpr int kIconBase=610,kGearBase=620;
constexpr int kChat=644,kMessage=645,kResult=646,kStatus=647,kSend=648,kAll=649,kNone=650,kFontMinus=651,kFontPlus=652;
const wchar_t* kNames[5]={L"YouTube",L"TikTok",L"Twitch",L"Kick",L"Facebook"};
const char* kKeys[5]={"youtube","tiktok","twitch","kick","facebook"};
const wchar_t* kShortNames[5]={L"YT",L"TT",L"TW",L"K",L"FB"};
HINSTANCE instance;
HWND windowHandle=nullptr,chatHandle=nullptr,messageHandle=nullptr,resultHandle=nullptr,statusHandle=nullptr;
HWND icons[5]{},gears[5]{},sendButton=nullptr,allButton=nullptr,noneButton=nullptr,minusButton=nullptr,plusButton=nullptr;
HFONT fontBody=nullptr,fontHeading=nullptr,fontSmall=nullptr;
HBRUSH brushBackground=nullptr,brushInput=nullptr;
std::atomic<bool> pollBusy{false},sendBusy{false};
bool selected[5]{},sendReady[5]{},online[5]{},configured[5]{},isDemo=false,obsStreaming=false,blink=true;
int fontPixels=13;
std::wstring lastChat,appDirectory,preferencesPath;
std::wstring resultText=L"Elegí un destino para enviar.";
const COLORREF textColor=RGB(228,230,242);
std::wstring wideUTF8(const std::string& s){
 if(s.empty())return {};
 int n=MultiByteToWideChar(CP_UTF8,MB_ERR_INVALID_CHARS,s.data(),int(s.size()),nullptr,0);
 if(n<=0)return L"";
 std::wstring result(size_t(n),L'\0');
 MultiByteToWideChar(CP_UTF8,MB_ERR_INVALID_CHARS,s.data(),int(s.size()),result.data(),n);
 return result;
}
std::string utf8Wide(const std::wstring& s){
 if(s.empty())return {};
 int n=WideCharToMultiByte(CP_UTF8,0,s.data(),int(s.size()),nullptr,0,nullptr,nullptr);
 if(n<=0)return {};
 std::string out(size_t(n),'\0');
 WideCharToMultiByte(CP_UTF8,0,s.data(),int(s.size()),out.data(),n,nullptr,nullptr);
 return out;
}
std::wstring trimmed(const std::wstring& s){
 const auto first=s.find_first_not_of(L" \n\r\t");
 if(first==std::wstring::npos)return {};
 const auto last=s.find_last_not_of(L" \n\r\t");
 return s.substr(first,last-first+1);
}
std::string jsonEscape(const std::string& s){
 std::string out;out.reserve(s.size()+10);
 for(unsigned char c:s){
  switch(c){case '"':out+="\\\"";break;case '\\':out+="\\\\";break;case '\r':out+="\\r";break;case '\n':out+="\\n";break;case '\t':out+="\\t";break;default:if(c<32)out+=' ';else out+=(char)c;break;}
 }
 return out;
}
struct HTTPResult{int status=0;std::string body;};
HTTPResult request(const wchar_t* method,const wchar_t* path,const std::string& payload={},bool action=false){
 HTTPResult ans;
 HINTERNET session=WinHttpOpen(L"SolarisPKN-NativeMultiChat/0.3",WINHTTP_ACCESS_TYPE_NO_PROXY,nullptr,nullptr,0);
 if(!session)return ans;
 WinHttpSetTimeouts(session,650,650,1800,1800);
 HINTERNET connection=WinHttpConnect(session,L"127.0.0.1",kPort,0);
 if(connection){
  HINTERNET req=WinHttpOpenRequest(connection,method,path,nullptr,WINHTTP_NO_REFERER,WINHTTP_DEFAULT_ACCEPT_TYPES,0);
  if(req){
   std::wstring headers;
   wchar_t nativeToken[256]{};
   DWORD n=GetEnvironmentVariableW(L"MULTICHAT_NATIVE_TOKEN",nativeToken,256);
   if(n>=32&&n<256){
    std::wstring secret(nativeToken,n);
    if(secret.find_first_of(L"\r\n")==std::wstring::npos)
     headers=L"Authorization: Bearer "+secret+L"\r\n";
   }
   if(action)headers+=L"Content-Type: application/json\r\nX-SolarisPKN-Action: send\r\n";
   if(WinHttpSendRequest(req,headers.empty()?WINHTTP_NO_ADDITIONAL_HEADERS:headers.c_str(),headers.empty()?0:DWORD(-1),payload.empty()?WINHTTP_NO_REQUEST_DATA:(LPVOID)payload.data(),(DWORD)payload.size(),(DWORD)payload.size(),0)
      &&WinHttpReceiveResponse(req,nullptr)){
    DWORD status=0,len=sizeof(status);
    if(WinHttpQueryHeaders(req,WINHTTP_QUERY_STATUS_CODE|WINHTTP_QUERY_FLAG_NUMBER,WINHTTP_HEADER_NAME_BY_INDEX,&status,&len,WINHTTP_NO_HEADER_INDEX))ans.status=int(status);
    char buf[4096];DWORD read=0;
    while(WinHttpReadData(req,buf,sizeof(buf),&read)&&read){
     if(ans.body.size()+read>120000){ans.body.clear();break;}
     ans.body.append(buf,read);
    }
   }
   WinHttpCloseHandle(req);
  }
  WinHttpCloseHandle(connection);
 }
 WinHttpCloseHandle(session);
 return ans;
}
bool jsonString(const std::string& s,const char* field,const char* value){
 return s.find(std::string("\"")+field+"\":\""+value+"\"")!=std::string::npos;
}
bool jsonFieldFor(const std::string& json,const char* platform,const char* field){
 auto start=json.find(std::string("\"platform\":\"")+platform+"\"");
 if(start==std::string::npos)return false;
 auto end=json.find('}',start);
 if(end==std::string::npos)return false;
 return json.substr(start,end-start).find(std::string("\"")+field+"\":true")!=std::string::npos;
}
std::wstring appPath(){
 wchar_t file[MAX_PATH]{};GetModuleFileNameW(nullptr,file,MAX_PATH);
 std::wstring path(file);auto i=path.find_last_of(L"\\/");
 return i==std::wstring::npos?L".":path.substr(0,i);
}
void applyFont(HWND h,HFONT f){if(h)SendMessageW(h,WM_SETFONT,(WPARAM)f,TRUE);}
void rebuildFonts(){
 HFONT previousBody=fontBody,previousHeading=fontHeading,previousSmall=fontSmall;
 fontBody=CreateFontW(-fontPixels,0,0,0,FW_NORMAL,FALSE,FALSE,FALSE,DEFAULT_CHARSET,OUT_DEFAULT_PRECIS,CLIP_DEFAULT_PRECIS,CLEARTYPE_QUALITY,DEFAULT_PITCH,L"Segoe UI");
 fontHeading=CreateFontW(-(fontPixels+3),0,0,0,FW_SEMIBOLD,FALSE,FALSE,FALSE,DEFAULT_CHARSET,OUT_DEFAULT_PRECIS,CLIP_DEFAULT_PRECIS,CLEARTYPE_QUALITY,DEFAULT_PITCH,L"Segoe UI");
 fontSmall=CreateFontW(-(std::max(10,fontPixels-2)),0,0,0,FW_NORMAL,FALSE,FALSE,FALSE,DEFAULT_CHARSET,OUT_DEFAULT_PRECIS,CLIP_DEFAULT_PRECIS,CLEARTYPE_QUALITY,DEFAULT_PITCH,L"Segoe UI");
 for(HWND h:{chatHandle,messageHandle,sendButton})applyFont(h,fontBody);
 for(HWND h:{resultHandle,statusHandle,allButton,noneButton,minusButton,plusButton})applyFont(h,fontSmall);
 if(previousBody)DeleteObject(previousBody);
 if(previousHeading)DeleteObject(previousHeading);
 if(previousSmall)DeleteObject(previousSmall);
 if(windowHandle)InvalidateRect(windowHandle,nullptr,TRUE);
}
void saveFont(){
 const std::wstring value=std::to_wstring(fontPixels);
 WritePrivateProfileStringW(L"UI",L"font_pixels",value.c_str(),preferencesPath.c_str());
}
HWND child(const wchar_t* klass,const wchar_t* title,DWORD style,int id,DWORD exStyle=0){
 return CreateWindowExW(exStyle,klass,title,WS_CHILD|WS_VISIBLE|style,0,0,0,0,windowHandle,(HMENU)(INT_PTR)id,instance,nullptr);
}
void canSend(){
 if(!sendButton||!messageHandle)return;
 bool chosen=false;
 for(int i=0;i<5;i++)if(selected[i]&&sendReady[i])chosen=true;
 const int len=GetWindowTextLengthW(messageHandle);
 EnableWindow(sendButton,chosen&&len>0&&len<=200&&!sendBusy.load());
}
void redrawIcons(){
 for(int i=0;i<5;i++){if(icons[i])InvalidateRect(icons[i],nullptr,FALSE);if(gears[i])InvalidateRect(gears[i],nullptr,FALSE);}
 canSend();
}
void result(const std::wstring& s){
 resultText=s;
 if(resultHandle)SetWindowTextW(resultHandle,s.c_str());
}
struct PollState{HTTPResult chat,targets,connection;};
void beginPoll(HWND target){
 if(pollBusy.exchange(true))return;
 std::thread([target](){
  std::unique_ptr<PollState> state(new PollState{request(L"GET",L"/native/feed"),request(L"GET",L"/api/outbound/status"),request(L"GET",L"/api/native/status")});
  if(PostMessageW(target,kPollMessage,0,(LPARAM)state.get()))state.release();
  pollBusy.store(false);
 }).detach();
}
void beginSend(){
 if(!messageHandle||sendBusy.load())return;
 int len=GetWindowTextLengthW(messageHandle);
 if(len<1||len>200)return;
 std::wstring raw(size_t(len+1),L'\0');
 GetWindowTextW(messageHandle,raw.data(),len+1);
 raw.resize(wcslen(raw.c_str()));
 std::wstring txt=trimmed(raw);
 if(txt.empty()||txt.find_first_of(L"\r\n")!=std::wstring::npos)return;
 std::vector<int> targets;
 for(int i=0;i<5;i++)if(selected[i]&&sendReady[i])targets.push_back(i);
 if(targets.empty())return;
 if(!isDemo){
  std::wstring prompt=L"¿Enviar a "+std::to_wstring(targets.size())+L" chat(s)?\n\n"+txt;
  if(MessageBoxW(windowHandle,prompt.c_str(),L"Confirmar publicación",MB_YESNO|MB_ICONQUESTION|MB_DEFBUTTON2)!=IDYES)return;
 }
 std::string payload="{\"message\":\""+jsonEscape(utf8Wide(txt))+"\",\"targets\":[";
 for(size_t i=0;i<targets.size();i++){if(i)payload+=",";payload+="\""+std::string(kKeys[targets[i]])+"\"";}
 payload+="]}";
 if(sendBusy.exchange(true))return;
 result(L"Procesando "+std::to_wstring(targets.size())+L" destino(s)…");
 canSend();
 HWND target=windowHandle;
 std::thread([target,payload=std::move(payload)](){
  std::unique_ptr<HTTPResult> output(new HTTPResult(request(L"POST",L"/api/outbound/send",payload,true)));
  if(PostMessageW(target,kSentMessage,0,(LPARAM)output.get()))output.release();
 }).detach();
}
void paintBackground(HDC hdc,const RECT& r,COLORREF c){
 HBRUSH b=CreateSolidBrush(c);FillRect(hdc,&r,b);DeleteObject(b);
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

void drawIcon(DRAWITEMSTRUCT* d){
 int i=int(d->CtlID)-kIconBase;
 if(i<0||i>4)return;
 RECT r=d->rcItem;
 int width=r.right-r.left,height=r.bottom-r.top;
 Graphics g(d->hDC);g.SetSmoothingMode(SmoothingModeAntiAlias);
 bool can=sendReady[i],on=selected[i]&&can;
 roundedBox(g,RectF((REAL)r.left,(REAL)r.top,(REAL)width,(REAL)height),
  on?Color(255,45,40,77):Color(255,24,29,41),8.f);
 if(on){
  Pen border(Color(255,147,122,239),1.4f);
 }
 drawLogo(g,i,RectF((REAL)(r.left+(width-31)/2),(REAL)(r.top+17),31.f,31.f),can);
 const Color dot=online[i]?(blink?Color(255,61,230,130):Color(255,13,78,47)):configured[i]?Color(255,178,120,55):isDemo?Color(255,123,91,184):Color(255,76,82,94);
 SolidBrush b(dot);g.FillEllipse(&b,(REAL)(r.left+7),(REAL)(r.top+5),7.f,7.f);
 if(on){Pen border(Color(255,160,132,241),1.4f);g.DrawRectangle(&border,REAL(r.left+1),REAL(r.top+1),REAL(width-3),REAL(height-3));}
}
void drawGear(DRAWITEMSTRUCT* d){
 RECT r=d->rcItem;
 Graphics g(d->hDC);g.SetSmoothingMode(SmoothingModeAntiAlias);
 SolidBrush bg(Color(255,29,32,46));g.FillRectangle(&bg,REAL(r.left),REAL(r.top),REAL(r.right-r.left),REAL(r.bottom-r.top));
 REAL cx=(r.right+r.left)/2.0f,cy=(r.top+r.bottom)/2.0f;
 Pen stroke(Color(255,164,169,191),1.4f);
 for(int k=0;k<8;k++){
  double a=k*3.141592653589793/4;
  g.DrawLine(&stroke,cx+(REAL)(std::cos(a)*4),cy+(REAL)(std::sin(a)*4),cx+(REAL)(std::cos(a)*6),cy+(REAL)(std::sin(a)*6));
 }
 g.DrawEllipse(&stroke,cx-3.7f,cy-3.7f,7.4f,7.4f);
 g.DrawEllipse(&stroke,cx-1,cy-1,2.f,2.f);
}
void drawTextButton(DRAWITEMSTRUCT* d){
 const int id=int(d->CtlID);
 bool sending=id==kSend,disabled=(d->itemState&ODS_DISABLED)!=0;
 bool pressed=(d->itemState&ODS_SELECTED)!=0;
 RECT r=d->rcItem;
 Graphics g(d->hDC);g.SetSmoothingMode(SmoothingModeAntiAlias);
 Color bg=sending?(disabled?Color(255,53,54,67):pressed?Color(255,98,82,167):Color(255,104,82,198)):
      pressed?Color(255,48,50,66):Color(255,31,35,49);
 roundedBox(g,RectF(REAL(r.left),REAL(r.top),REAL(r.right-r.left),REAL(r.bottom-r.top)),bg,6.f);
 wchar_t text[64]{};
 GetWindowTextW(d->hwndItem,text,64);
 SetBkMode(d->hDC,TRANSPARENT);
 SetTextColor(d->hDC,disabled?RGB(130,134,150):RGB(238,239,252));
 HFONT f=(id==kSend?fontBody:fontSmall);
 HGDIOBJ old=SelectObject(d->hDC,f);
 DrawTextW(d->hDC,text,-1,&r,DT_CENTER|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX);
 SelectObject(d->hDC,old);
}
void renderChat(const std::wstring& content){
 if(!chatHandle||content==lastChat)return;
 lastChat=content;
 // RichEdit normalizes CRLF to one paragraph separator; color offsets
 // must be computed in the same normalized text.
 std::wstring visible=content;
 for(size_t p=0;(p=visible.find(L"\r\n",p))!=std::wstring::npos;){visible.replace(p,2,L"\r");p++;}
 SendMessageW(chatHandle,WM_SETREDRAW,FALSE,0);
 SetWindowTextW(chatHandle,visible.c_str());
 // Native RichEdit colors platform badges without a browser rendering pipeline.
 long offset=0;
 for(size_t i=0;i<visible.size();){
  if(visible[i]==L'['){
   auto e=visible.find(L']',i+1);
   if(e!=std::wstring::npos&&e-i<22){
    std::wstring platform=visible.substr(i+1,e-i-1);
    COLORREF color=RGB(165,169,191);
    if(platform==L"youtube")color=RGB(244,90,106);
    else if(platform==L"tiktok")color=RGB(91,217,204);
    else if(platform==L"twitch")color=RGB(172,136,249);
    else if(platform==L"kick")color=RGB(119,216,105);
    else if(platform==L"facebook")color=RGB(124,163,247);
    CHARRANGE range{LONG(i),LONG(e+1)};
    SendMessageW(chatHandle,EM_EXSETSEL,0,(LPARAM)&range);
    CHARFORMAT2W fmt{};fmt.cbSize=sizeof(fmt);fmt.dwMask=CFM_COLOR|CFM_BOLD;fmt.dwEffects=CFE_BOLD;fmt.crTextColor=color;
    SendMessageW(chatHandle,EM_SETCHARFORMAT,SCF_SELECTION,(LPARAM)&fmt);
   }
  }
  auto next=visible.find(L'\r',i);
  if(next==std::wstring::npos)break;
  i=next+1;
 }
 const int len=GetWindowTextLengthW(chatHandle);
 SendMessageW(chatHandle,EM_SETSEL,len,len);
 SendMessageW(chatHandle,EM_SCROLLCARET,0,0);
 SendMessageW(chatHandle,WM_SETREDRAW,TRUE,0);
 InvalidateRect(chatHandle,nullptr,FALSE);
}
void showConfig(int id){
 if(id<0||id>4)return;
 std::wstring platform=kNames[id];
 std::wstring instructions;
 if(id==0)instructions=L"YouTube: conectá Google OAuth y habilitá tu emisión activa. Necesitás MULTICHAT_GOOGLE_CLIENT_ID.";
 if(id==1)instructions=L"TikTok: configurá tu @usuario. La lectura LIVE es pública; enviar mensajes aún requiere otro adaptador.";
 if(id==2)instructions=L"Twitch: vinculá el OAuth de dispositivo con chat:read y user:write:chat.";
 if(id==3)instructions=L"Kick: configurá canal y credencial OAuth chat:write para enviar. El lector público no autentica la cuenta.";
 if(id==4)instructions=L"Facebook: configurá ID de Página, token y permisos Meta. El chat depende del LIVE activo.";
 std::wstring state=online[id]?L"Conexión verificada":configured[id]?L"Configurada, sin conexión verificada":L"Sin conectar";
 std::wstring msg=platform+L"\n"+state+L"\n\n"+instructions+L"\n\nSí: abrir ajustes locales para editar el canal y la autorización.\nNo: "+std::wstring(id==0||id==2?L"vincular OAuth de esta plataforma":L"cerrar");
 int choice=MessageBoxW(windowHandle,msg.c_str(),L"Configurar plataforma",MB_YESNOCANCEL|MB_ICONINFORMATION);
 if(choice==IDYES){
  const std::wstring sample=appDirectory+L"\\config.solarispkn.example.json";
  const std::wstring custom=appDirectory+L"\\NativeUI\\config.local.json";
  if(GetFileAttributesW(custom.c_str())==INVALID_FILE_ATTRIBUTES){
   if(GetFileAttributesW(sample.c_str())!=INVALID_FILE_ATTRIBUTES)CopyFileW(sample.c_str(),custom.c_str(),TRUE);
  }
  if(GetFileAttributesW(custom.c_str())==INVALID_FILE_ATTRIBUTES){MessageBoxW(windowHandle,L"No se encontró la plantilla de configuración.",L"MultiChat",MB_OK|MB_ICONERROR);return;}
  std::wstring params=L"\""+custom+L"\"";
  ShellExecuteW(windowHandle,L"open",L"notepad.exe",params.c_str(),nullptr,SW_SHOWNORMAL);
  result(L"Configuración guardada al editar; se aplica al reiniciar el motor en modo real.");
 }
 else if(choice==IDNO&&(id==0||id==2)){
  const std::wstring exe=appDirectory+L"\\MultiChat-native-engine.exe";
  const wchar_t* flag=id==0?L"-link-google":L"-link-twitch";
  std::wstring command=L"\""+exe+L"\" "+flag;
  std::vector<wchar_t> args(command.begin(),command.end());args.push_back(L'\0');
  STARTUPINFOW si{};si.cb=sizeof(si);
  PROCESS_INFORMATION pi{};
  if(CreateProcessW(exe.c_str(),args.data(),nullptr,nullptr,FALSE,CREATE_NEW_CONSOLE,nullptr,appDirectory.c_str(),&si,&pi)){
   CloseHandle(pi.hThread);CloseHandle(pi.hProcess);
   result(L"Autorización iniciada. Revisá el diálogo externo.");
  }else MessageBoxW(windowHandle,L"No se pudo iniciar OAuth. Revisá el ejecutable del motor.",L"MultiChat",MB_OK|MB_ICONERROR);
 }
}
void placeControls(){
 if(!windowHandle)return;
 RECT r{};GetClientRect(windowHandle,&r);
 int w=std::max(310,int(r.right)),h=std::max(480,int(r.bottom));
 const int pad=12,inner=w-2*pad;
 MoveWindow(minusButton,w-77,13,27,25,TRUE);
 MoveWindow(plusButton,w-44,13,27,25,TRUE);
 MoveWindow(statusHandle,pad,44,inner,22,TRUE);
 const int bottom=h-246;
 MoveWindow(chatHandle,pad,75,inner,std::max(135,bottom-75),TRUE);
 const int toggleY=h-212;
 MoveWindow(allButton,w-160,toggleY,72,22,TRUE);
 MoveWindow(noneButton,w-85,toggleY,73,22,TRUE);
 const int iconY=h-185,cell=inner/5;
 for(int i=0;i<5;i++){
  int x=pad+i*cell;
  int width=(i==4?inner-i*cell:cell)-5;
  MoveWindow(icons[i],x,iconY,width,59,TRUE);
  MoveWindow(gears[i],x+width-20,iconY+2,18,18,TRUE);
 }
 MoveWindow(messageHandle,pad,h-104,inner,37,TRUE);
 MoveWindow(sendButton,pad,h-59,inner,29,TRUE);
 MoveWindow(resultHandle,pad,h-26,inner,20,TRUE);
 InvalidateRect(windowHandle,nullptr,TRUE);
}
void startBackend(){
 if(request(L"GET",L"/health").status==200)return;
 const std::wstring exe=appDirectory+L"\\MultiChat-native-engine.exe";
 const std::wstring cfg=appDirectory+L"\\NativeUI\\demo-config.json";
 if(GetFileAttributesW(exe.c_str())==INVALID_FILE_ATTRIBUTES)return;
 std::wstring command=L"\""+exe+L"\" -config \""+cfg+L"\" -demo -quiet";
 STARTUPINFOW si{};si.cb=sizeof(si);
 PROCESS_INFORMATION pi{};
 std::vector<wchar_t> args(command.begin(),command.end());args.push_back(0);
 if(CreateProcessW(exe.c_str(),args.data(),nullptr,nullptr,FALSE,CREATE_NO_WINDOW,nullptr,appDirectory.c_str(),&si,&pi)){
  CloseHandle(pi.hThread);CloseHandle(pi.hProcess);
 }
}
LRESULT CALLBACK procedure(HWND hWnd,UINT msg,WPARAM wp,LPARAM lp){
 switch(msg){
 case WM_CREATE:{
  windowHandle=hWnd;
  brushBackground=CreateSolidBrush(RGB(15,18,28));
  brushInput=CreateSolidBrush(RGB(24,28,41));
  fontPixels=std::clamp(int(GetPrivateProfileIntW(L"UI",L"font_pixels",13,preferencesPath.c_str())),10,19);
  LoadLibraryW(L"Msftedit.dll");
  chatHandle=child(MSFTEDIT_CLASS,L"",ES_MULTILINE|ES_READONLY|ES_AUTOVSCROLL|WS_VSCROLL|WS_BORDER,kChat);
  SendMessageW(chatHandle,EM_SETBKGNDCOLOR,0,RGB(23,27,40));
  messageHandle=child(L"EDIT",L"",ES_MULTILINE|ES_AUTOVSCROLL|WS_BORDER,kMessage);
  SendMessageW(messageHandle,EM_LIMITTEXT,200,0);
  for(int i=0;i<5;i++){
   icons[i]=child(L"BUTTON",kNames[i],BS_OWNERDRAW|WS_TABSTOP,kIconBase+i);
   gears[i]=child(L"BUTTON",L"Configurar",BS_OWNERDRAW|WS_TABSTOP,kGearBase+i);
  }
  allButton=child(L"BUTTON",L"Todas",BS_OWNERDRAW,kAll);
  noneButton=child(L"BUTTON",L"Ninguna",BS_OWNERDRAW,kNone);
  minusButton=child(L"BUTTON",L"A−",BS_OWNERDRAW,kFontMinus);
  plusButton=child(L"BUTTON",L"A+",BS_OWNERDRAW,kFontPlus);
  sendButton=child(L"BUTTON",L"Enviar mensaje / comando",BS_OWNERDRAW,kSend);
  statusHandle=child(L"STATIC",L"Sin conectar · esperando motor",SS_LEFT,kStatus);
  resultHandle=child(L"STATIC",resultText.c_str(),SS_LEFT,kResult);
  rebuildFonts();
  placeControls();
  canSend();
  SetTimer(hWnd,1,1600,nullptr);
  SetTimer(hWnd,2,550,nullptr);
  startBackend();
  beginPoll(hWnd);
  return 0;
 }
 case WM_SIZE:placeControls();return 0;
 case WM_GETMINMAXINFO:{
  auto *info=(MINMAXINFO*)lp;
  info->ptMinTrackSize.x=350;info->ptMinTrackSize.y=500;
  return 0;
 }
 case WM_COMMAND:{
  int id=LOWORD(wp);
  if(id>=kIconBase&&id<kIconBase+5&&HIWORD(wp)==BN_CLICKED){
   int p=id-kIconBase;
   if(sendReady[p])selected[p]=!selected[p];
   redrawIcons();return 0;
  }
  if(id>=kGearBase&&id<kGearBase+5&&HIWORD(wp)==BN_CLICKED){showConfig(id-kGearBase);return 0;}
  if(id==kAll&&HIWORD(wp)==BN_CLICKED){for(int i=0;i<5;i++)selected[i]=sendReady[i];redrawIcons();return 0;}
  if(id==kNone&&HIWORD(wp)==BN_CLICKED){for(int i=0;i<5;i++)selected[i]=false;redrawIcons();return 0;}
  if(id==kSend&&HIWORD(wp)==BN_CLICKED){beginSend();return 0;}
  if(id==kMessage&&HIWORD(wp)==EN_CHANGE){canSend();return 0;}
  if((id==kFontMinus||id==kFontPlus)&&HIWORD(wp)==BN_CLICKED){
   fontPixels=std::clamp(fontPixels+(id==kFontPlus?1:-1),10,19);
   rebuildFonts();saveFont();return 0;
  }
  return 0;
 }
 case WM_TIMER:
  if(wp==1)beginPoll(hWnd);
  if(wp==2){blink=!blink;for(int i=0;i<5;i++)if(online[i])InvalidateRect(icons[i],nullptr,FALSE);}
  return 0;
 case kPollMessage:{
  std::unique_ptr<PollState> payload((PollState*)lp);
  if(payload->chat.status==200)renderChat(wideUTF8(payload->chat.body));
  if(payload->targets.status==200){
   isDemo=jsonString(payload->targets.body,"mode","demo");
   for(int i=0;i<5;i++)sendReady[i]=jsonFieldFor(payload->targets.body,kKeys[i],"ready");
  }else{
   isDemo=false;for(int i=0;i<5;i++)sendReady[i]=false;
  }
  if(payload->connection.status==200){
   obsStreaming=payload->connection.body.find("\"streaming\":true")!=std::string::npos;
   for(int i=0;i<5;i++){
    online[i]=jsonFieldFor(payload->connection.body,kKeys[i],"online");
    configured[i]=jsonFieldFor(payload->connection.body,kKeys[i],"configured");
   }
  }else{
   obsStreaming=false;for(int i=0;i<5;i++){online[i]=false;configured[i]=false;}
  }
  if(statusHandle){
   std::wstring s=isDemo?L"DEMO · sin publicar":payload->targets.status==200?L"Conectado · "+std::wstring(obsStreaming?L"OBS EN VIVO":L"OBS en espera"):L"Motor desconectado";
   SetWindowTextW(statusHandle,s.c_str());
  }
  redrawIcons();return 0;
 }
 case kSentMessage:{
  std::unique_ptr<HTTPResult> output((HTTPResult*)lp);
  sendBusy=false;
  if(output->status!=200){
   result(L"Error HTTP "+std::to_wstring(output->status)+L". No se reintentó.");
  }else{
   int simulated=0,accepted=0,unavailable=0,errors=0;
   const std::string& json=output->body;
   size_t p=0;
   while((p=json.find("\"status\":\"",p))!=std::string::npos){
    p+=10;
    auto end=json.find('"',p);
    if(end==std::string::npos)break;
    const std::string value=json.substr(p,end-p);p=end+1;
    if(value=="simulated")simulated++;
    else if(value=="accepted")accepted++;
    else if(value=="unavailable")unavailable++;
    else errors++;
   }
   if(isDemo)result(L"DEMO: "+std::to_wstring(simulated)+L" simulado(s), ninguno publicado.");
   else result(L"Aceptados "+std::to_wstring(accepted)+L" · sin permisos "+std::to_wstring(unavailable)+L" · error "+std::to_wstring(errors));
  }
  canSend();return 0;
 }
 case WM_DRAWITEM:{
  auto *item=(DRAWITEMSTRUCT*)lp;
  int id=int(item->CtlID);
  if(id>=kIconBase&&id<kIconBase+5)drawIcon(item);
  else if(id>=kGearBase&&id<kGearBase+5)drawGear(item);
  else drawTextButton(item);
  return TRUE;
 }
 case WM_CTLCOLOREDIT:{
  HDC dc=(HDC)wp;SetTextColor(dc,textColor);SetBkColor(dc,RGB(24,28,41));
  return (LRESULT)brushInput;
 }
 case WM_CTLCOLORSTATIC:{
  HDC dc=(HDC)wp;SetTextColor(dc,RGB(178,185,205));SetBkColor(dc,RGB(15,18,28));
  return (LRESULT)brushBackground;
 }
 case WM_ERASEBKGND:return TRUE;
 case WM_PAINT:{
  PAINTSTRUCT ps{};HDC dc=BeginPaint(hWnd,&ps);
  RECT r{};GetClientRect(hWnd,&r);
  paintBackground(dc,r,RGB(15,18,28));
  RECT header={0,0,r.right,73};paintBackground(dc,header,RGB(20,24,37));
  RECT divider={0,72,r.right,73};paintBackground(dc,divider,RGB(46,52,72));
  SetBkMode(dc,TRANSPARENT);SetTextColor(dc,RGB(230,233,249));
  HGDIOBJ old=SelectObject(dc,fontHeading);
  const wchar_t* title=L"✦ MultiChat";
  TextOutW(dc,12,13,title,(int)wcslen(title));
  SelectObject(dc,fontSmall);
  const wchar_t* opts=L"DESTINOS";
  TextOutW(dc,12,r.bottom-211,opts,(int)wcslen(opts));
  const wchar_t* inputLabel=L"MENSAJE / COMANDO";
  TextOutW(dc,12,r.bottom-123,inputLabel,(int)wcslen(inputLabel));
  SelectObject(dc,old);
  EndPaint(hWnd,&ps);
  return 0;
 }
 case WM_CLOSE:DestroyWindow(hWnd);return 0;
 case WM_DESTROY:{
  KillTimer(hWnd,1);KillTimer(hWnd,2);
  if(fontBody)DeleteObject(fontBody);if(fontHeading)DeleteObject(fontHeading);if(fontSmall)DeleteObject(fontSmall);
  if(brushBackground)DeleteObject(brushBackground);if(brushInput)DeleteObject(brushInput);
  PostQuitMessage(0);return 0;
 }
 }
 return DefWindowProcW(hWnd,msg,wp,lp);
}
} // namespace

int APIENTRY wWinMain(HINSTANCE app,HINSTANCE,LPWSTR,int show){
 instance=app;appDirectory=appPath();preferencesPath=appDirectory+L"\\NativeUI\\preferences.ini";
 SetProcessDPIAware();
 GdiplusStartupInput input{};
 ULONG_PTR token=0;
 if(GdiplusStartup(&token,&input,nullptr)!=Ok)return 3;
 WNDCLASSW wc{};wc.lpfnWndProc=procedure;wc.hInstance=app;wc.lpszClassName=L"SolarisPKNMultiChatCompact_v2";
 wc.hCursor=LoadCursorW(nullptr,IDC_ARROW);wc.hIcon=LoadIconW(nullptr,IDI_APPLICATION);
 if(!RegisterClassW(&wc)){GdiplusShutdown(token);return 4;}
 RECT work{};SystemParametersInfoW(SPI_GETWORKAREA,0,&work,0);
 const int availableW=int(work.right-work.left),availableH=int(work.bottom-work.top);
 // One-third-screen *width*, vertical chat format, not a full 16:9 suite window.
 const int clientW=std::clamp(availableW/3,350,520);
 // Account for Windows caption/border so the send button stays on screen.
 const int clientH=std::clamp(int(clientW*1.45),460,std::max(460,availableH-65));
 const DWORD style=WS_OVERLAPPEDWINDOW;
 RECT outer{0,0,clientW,clientH};AdjustWindowRectEx(&outer,style,FALSE,0);
 const int totalW=int(outer.right-outer.left),totalH=int(outer.bottom-outer.top);
 const int x=work.right-totalW-12,y=work.top+std::max(4,(availableH-totalH)/2);
 HWND wnd=CreateWindowExW(0,wc.lpszClassName,L"SolarisPKN MultiChat · Compacto",style,
  x,y,totalW,totalH,nullptr,nullptr,app,nullptr);
 if(!wnd){GdiplusShutdown(token);return 5;}
 ShowWindow(wnd,show);UpdateWindow(wnd);
 MSG m{};while(GetMessageW(&m,nullptr,0,0)>0){TranslateMessage(&m);DispatchMessageW(&m);}
 GdiplusShutdown(token);return (int)m.wParam;
}
