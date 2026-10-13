package main

// This single optional local page never loads any platform video/chat webpage.
const webPage = `<!doctype html><html lang="es"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>SolarisPKN MultiChat</title><style>
:root{color-scheme:dark;font:14px/1.45 system-ui,Segoe UI,sans-serif;background:#11131a;color:#eef1ff}
*{box-sizing:border-box}body{margin:0}header{padding:12px 16px;background:#1b1d27;display:flex;justify-content:space-between;align-items:center;position:sticky;top:0}header strong{font-size:15px}#status{font-size:12px;color:#adb4c8}main{padding:8px 12px 24px;max-width:960px;margin:auto}#log{list-style:none;padding:0;margin:0}li{padding:6px 8px;border-bottom:1px solid #292b34;overflow-wrap:anywhere}.platform{font-size:11px;padding:2px 6px;border-radius:5px;background:#34394c;color:#c8d5fc;margin-right:7px}.name{font-weight:700;color:#edf0fa;margin-right:6px}.body{color:#e0e3f0}.time{float:right;color:#868da5;font-size:11px;margin-left:10px}
</style></head><body><header><strong>✦ SolarisPKN MultiChat</strong><span id="status">Conectando…</span></header><main><ol id="log" aria-live="polite"></ol></main>
<script>
'use strict';const log=document.getElementById('log'),status=document.getElementById('status');const max=250;const ids=new Set();
function draw(m){const key=m.platform+':'+m.id;if(m.id&&ids.has(key))return;if(m.id)ids.add(key);
const li=document.createElement('li');const badge=document.createElement('span');badge.className='platform';badge.textContent=m.platform;
const user=document.createElement('span');user.className='name';user.textContent=m.user+':';
const body=document.createElement('span');body.className='body';body.textContent=m.text;
const when=document.createElement('span');when.className='time';when.textContent=new Date(m.time).toLocaleTimeString();
li.append(badge,user,body,when);if(m.id)li.dataset.msgkey=key;log.append(li);while(log.childElementCount>max){const old=log.firstElementChild;if(old.dataset.msgkey)ids.delete(old.dataset.msgkey);old.remove()}window.scrollTo(0,document.body.scrollHeight);
}
fetch('/history').then(r=>r.json()).then(items=>items.forEach(draw)).catch(()=>{});
const stream=new EventSource('/events');stream.onopen=()=>status.textContent='● Conectado';stream.onerror=()=>status.textContent='○ Reconectando…';stream.onmessage=e=>{try{draw(JSON.parse(e.data))}catch{}};
</script></body></html>`
