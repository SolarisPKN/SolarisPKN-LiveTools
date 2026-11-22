package main

// Pop-out command/chat composer. No external fonts, embeds or scripts.
// Message content and all provider responses are rendered with textContent.
const composerPage = `<!doctype html>
<html lang="es"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>SolarisPKN MultiChat · Centro de envíos</title>
<style>
:root{color-scheme:dark;font-family:Inter,Segoe UI,system-ui,sans-serif;background:#101019;color:#f4f3fe;font-size:14px}
*{box-sizing:border-box}
body{margin:0;background:radial-gradient(ellipse at 80% 4%,#2b2045 0,transparent 48%),#101019;min-height:100vh}
header{padding:23px 24px 17px;border-bottom:1px solid #303041;display:flex;justify-content:space-between;align-items:center;gap:14px}
.logo{display:flex;align-items:center;gap:10px;font-size:16px;font-weight:800}
.mark{border:1px solid #9e88ee;border-radius:10px;background:linear-gradient(135deg,#564a9c,#8e62e6);padding:7px 10px;font-size:18px}
a{color:#beaefa;text-decoration:none}a:hover{text-decoration:underline}
main{padding:20px 22px 32px;max-width:760px;margin:auto}
h1{font-size:23px;font-weight:820;letter-spacing:-.5px;margin:0 0 6px}
small,.subtitle{color:#a7a9bc}
.subtitle{line-height:1.5;margin:0 0 16px}
#mode{display:inline-flex;border-radius:20px;padding:5px 11px;font-size:11px;font-weight:800;border:1px solid #5e536d;color:#dacaf7;background:#231c33}
.section{border:1px solid #313043;border-radius:15px;background:#191925;overflow:hidden;margin:18px 0}
.sectionHeading{display:flex;flex-wrap:wrap;align-items:center;justify-content:space-between;gap:10px;padding:14px 16px;border-bottom:1px solid #303043}
.sectionHeading strong{font-size:14px}
.actions{display:flex;gap:6px;flex-wrap:wrap}
button{font:inherit;border:1px solid #4d4b66;border-radius:8px;background:#282637;color:#ece9fe;padding:8px 10px;cursor:pointer;transition:background .1s}
button:hover:not(:disabled){background:#393550}
button:disabled{cursor:default;opacity:.5}
.platforms{display:grid;gap:0}
.platform{display:flex;align-items:center;gap:10px;min-height:65px;padding:9px 14px;border-bottom:1px solid #282736}
.platform:last-child{border-bottom:0}
.platform:has(input:checked){background:#262339}
.platform input{accent-color:#b399ff;width:18px;height:18px;flex:none;cursor:pointer}
.platform .pic{display:grid;place-items:center;flex:none;width:33px;height:33px;border:1px solid #444151;border-radius:10px;font-weight:900;color:#eee}
.platform .copy{flex:1;min-width:0}
.platform .name{font-weight:750}
.platform .hint{font-size:11px;color:#aaa9bd;overflow-wrap:anywhere;line-height:1.4}
.platform .solo{padding:5px 8px;font-size:11px;color:#c4b7eb}
.platform .dot{display:inline-block;width:7px;height:7px;border-radius:50%;margin-right:5px;background:#948fa0}
.platform .dot.ready{background:#65cea0}
.platform .dot.wait{background:#e7b16b}
.platform .dot.sim{background:#b39afa}
.toolbar{display:flex;align-items:center;justify-content:space-between;gap:10px;flex-wrap:wrap;padding:13px 16px}
textarea{display:block;width:100%;background:#11121d;color:#f9f7ff;border:1px solid #4c4b67;border-radius:11px;resize:vertical;min-height:112px;max-height:300px;font:15px/1.5 Segoe UI,system-ui,sans-serif;padding:13px;outline:none}
textarea:focus{border-color:#b2a0f8;box-shadow:0 0 0 3px #9071dc22}
.msgarea{padding:14px 16px}
.caption{font-size:12px;color:#a1a1b7;line-height:1.5;margin:10px 0 0}
.foot{display:flex;justify-content:space-between;align-items:center;gap:12px;flex-wrap:wrap;padding:0 16px 16px}
.primary{background:linear-gradient(135deg,#9b82ed,#6f6cd9);color:#fff;border-color:#aaa0ed;font-weight:760;padding:12px 16px;min-width:174px}
.primary:hover:not(:disabled){background:linear-gradient(135deg,#b39bff,#8580f1)}
#preview{padding:12px 15px;border:1px dashed #454158;background:#1a1928;border-radius:11px;font-size:12px;color:#cbc6dc;white-space:pre-wrap;overflow-wrap:anywhere}
#results{list-style:none;margin:0;padding:0}
#results li{border-top:1px solid #303043;padding:12px 15px;display:flex;align-items:baseline;justify-content:space-between;gap:15px}
#results li:first-child{border-top:0}
#results b{font-weight:700}
#results .good{color:#83d5af}
#results .bad{color:#f0a4a3}
#results .neutral{color:#c9aefd}
#results span{font-size:12px;text-align:right;max-width:64%;overflow-wrap:anywhere}
#results .placeholder{color:#8e8ca6;text-align:left}
#general{margin:0 0 10px;color:#b6b6ca;font-size:12px}
.note{padding:13px 16px;border:1px solid #424052;background:#211d30;border-radius:12px;font-size:12px;color:#cfc7e4;line-height:1.5}
@media(max-width:510px){main{padding:15px 12px}header{padding:15px}.platform{padding:9px}.foot{align-items:stretch}.primary{width:100%}}
</style>
</head><body>
<header><div class="logo"><span class="mark">✦</span><span>SolarisPKN <span style="color:#baa8ff">MultiChat</span></span></div><a href="/" aria-label="Volver al chat">← Volver al chat</a></header>
<main>
 <h1>Centro de envíos</h1>
 <p class="subtitle">Escribí una vez y elegí a qué chats mandar el mensaje o comando. Cada plataforma responde por separado.</p>
 <span id="mode">Consultando modo…</span>
 <div class="section">
  <div class="sectionHeading"><strong>Destinos <span id="count" style="color:#bca6f0">(0/5)</span></strong>
   <div class="actions"><button id="all" type="button">Seleccionar todas</button><button id="none" type="button">Deseleccionar todas</button></div></div>
  <div id="platforms" class="platforms"></div>
  <div class="toolbar"><small>“Solo” deja seleccionado únicamente ese chat.</small><button id="refresh" type="button">↻ Revisar conexión</button></div>
 </div>
 <div class="section">
  <div class="sectionHeading"><strong>Mensaje o comando</strong><span id="length" class="subtitle" style="margin:0;font-size:12px">0/200</span></div>
  <div class="msgarea">
   <textarea id="message" maxlength="200" placeholder="Ej.: ¡Bienvenidos al stream! ✨  o  !redes" aria-label="Escribí el mensaje o comando"></textarea>
   <p class="caption">Los comandos con / o ! se envían como texto. Solo se ejecutan si esa plataforma y tus permisos los reconocen.</p>
  </div>
  <div class="foot"><span id="selectionPreview" class="subtitle" style="margin:0">Sin plataformas seleccionadas</span><button id="send" class="primary" type="button" disabled>Enviar a seleccionadas</button></div>
 </div>
 <div id="preview">Seleccioná destinatarios y escribí el mensaje.</div>
 <div class="section"><div class="sectionHeading"><strong>Resultado por plataforma</strong><span class="subtitle" style="margin:0;font-size:11px">Sin reintentos automáticos</span></div><p id="general" style="padding:12px 15px;margin:0">Todavía no hiciste envíos.</p><ul id="results" aria-live="polite"></ul></div>
 <p class="note"><strong>Seguridad:</strong> el modo demo no publica nada; en modo real se solicita confirmación antes de enviar. Una marca verde indica aceptación por API, no garantiza que el mensaje se vea en pantalla. TikTok necesita todavía un adaptador autenticado de escritura.</p>
</main>
<script>
'use strict';
const names={youtube:'YouTube',tiktok:'TikTok',twitch:'Twitch',kick:'Kick',facebook:'Facebook'};
const symbols={youtube:'YT',tiktok:'♪',twitch:'TW',kick:'K',facebook:'f'};
const accents={youtube:'#df6670',tiktok:'#6cdbd5',twitch:'#ad8fff',kick:'#9be67d',facebook:'#8dacf8'};
const selected=new Set(), cards=new Map();let mode='disabled',currentStatuses=[],busy=false;
const $=id=>document.getElementById(id);
const cleanText=s=>String(s||'').trim();
function renderPlatforms(){
 const parent=$('platforms');parent.replaceChildren();cards.clear();
 for(const key of Object.keys(names)){
  const info=currentStatuses.find(x=>x.platform===key)||{platform:key,ready:false,detail:'Cargando…'};
  const line=document.createElement('div');line.className='platform';
  const check=document.createElement('input');check.type='checkbox';check.id='select-'+key;check.checked=selected.has(key);
  check.addEventListener('change',()=>{check.checked?selected.add(key):selected.delete(key);refreshSelection()});
  const label=document.createElement('label');label.htmlFor=check.id;label.className='pic';label.textContent=symbols[key];label.style.color=accents[key];
  const copy=document.createElement('div');copy.className='copy';
  const name=document.createElement('div');name.className='name';name.textContent=names[key];
  const hint=document.createElement('div');hint.className='hint';
  const dot=document.createElement('span');dot.className='dot '+(mode==='demo'?'sim':info.ready?'ready':'wait');
  const detail=document.createElement('span');detail.textContent=(mode==='demo'?'Simulación · ':info.ready?'Preparado* · ':'No preparado · ')+info.detail;
  hint.append(dot,detail);copy.append(name,hint);
  const solo=document.createElement('button');solo.className='solo';solo.type='button';solo.textContent='Solo';
  solo.title='Enviar únicamente a '+names[key];solo.addEventListener('click',()=>{selected.clear();selected.add(key);renderPlatforms();refreshSelection()});
  line.append(check,label,copy,solo);parent.append(line);cards.set(key,check);
 }
}
function refreshSelection(){
 for(const [key,control] of cards)control.checked=selected.has(key);
 $('count').textContent='('+selected.size+'/5)';
 const msg=cleanText($('message').value);
 $('length').textContent=Array.from($('message').value).length+'/200';
 $('selectionPreview').textContent=selected.size?selected.size+' destino'+(selected.size===1?'':'s')+' elegido'+(selected.size===1?'':'s'):'Sin plataformas seleccionadas';
 $('send').disabled=busy||!msg||!selected.size||Array.from(msg).length>200||mode==='disabled';
 $('send').textContent=mode==='demo'?'Simular envío':'Enviar a seleccionadas';
 const dest=[...selected].map(x=>names[x]).join(' · ');
 $('preview').textContent=selected.size?'Vista previa → '+dest+'\n'+(msg||'(escribí un mensaje)'):'Seleccioná destinatarios y escribí el mensaje.';
}
async function loadStatus(){
 try{
  const resp=await fetch('/api/outbound/status',{cache:'no-store'});
  if(resp.status===401){location.href='/auth/login';return}
  if(!resp.ok)throw Error('HTTP '+resp.status);
  const data=await resp.json();mode=data.mode||'disabled';currentStatuses=data.targets||[];
  $('mode').textContent=mode==='demo'?'◆ MODO DEMO · sin publicaciones':mode==='live'?'● ENVÍOS REALES · autorización necesaria':'○ ENVÍO REAL DESACTIVADO';
  renderPlatforms();refreshSelection();
 }catch(e){$('mode').textContent='No se pudo consultar el servicio';$('general').textContent='Error de conexión: '+e.message}
}
$('all').addEventListener('click',()=>{for(const p of Object.keys(names))selected.add(p);refreshSelection()});
$('none').addEventListener('click',()=>{selected.clear();refreshSelection()});
$('refresh').addEventListener('click',loadStatus);
$('message').addEventListener('input',refreshSelection);
$('message').addEventListener('keydown',e=>{if(e.ctrlKey&&e.key==='Enter'){e.preventDefault();$('send').click()}});
$('send').addEventListener('click',async()=>{
 if(busy)return;
 const message=cleanText($('message').value),targets=Object.keys(names).filter(x=>selected.has(x));
 if(!targets.length||!message||Array.from(message).length>200)return;
 if(mode==='live'){
  const allowed=window.confirm('¿Enviar este texto a '+targets.map(x=>names[x]).join(', ')+'?\n\n'+message);
  if(!allowed)return;
 }
 busy=true;refreshSelection();$('general').textContent='Procesando '+targets.length+' destino(s)…';$('results').replaceChildren();
 try{
  const resp=await fetch('/api/outbound/send',{method:'POST',headers:{'Content-Type':'application/json','X-SolarisPKN-Action':'send'},credentials:'same-origin',body:JSON.stringify({message,targets})});
  if(resp.status===401){location.href='/auth/login';return}
  if(!resp.ok)throw Error('HTTP '+resp.status+': '+(await resp.text()).slice(0,240));
  const data=await resp.json();
  $('general').textContent=data.mode==='demo'?'Se simularon los envíos, sin publicar en ninguna red.':'Envío procesado. Revisá cada plataforma:';
  for(const entry of data.results||[]){
   const row=document.createElement('li'),title=document.createElement('b'),detail=document.createElement('span');
   title.textContent=(names[entry.platform]||entry.platform)+': '+(entry.status==='simulated'?'SIMULADO':entry.status==='accepted'?'ACEPTADO':entry.status==='unavailable'?'SIN AUTORIZAR':'ERROR');
   title.className=entry.status==='accepted'?'good':entry.status==='simulated'?'neutral':'bad';
   detail.textContent=entry.detail||'';row.append(title,detail);$('results').append(row);
  }
 }catch(e){$('general').textContent='No se pudo procesar el envío: '+e.message}
 finally{busy=false;refreshSelection()}
});
loadStatus();
</script></body></html>`
