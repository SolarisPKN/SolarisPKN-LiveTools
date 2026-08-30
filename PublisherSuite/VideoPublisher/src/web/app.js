const ids=['sessionStatus','dailyLimit','longFile','shortFile','chooseLong','chooseShort','longProgress','shortProgress','longInfo','shortInfo','title','shortTitle','language','longDescription','shortCaption','hashtags','tags','canonicalUrl','publicShortUrl','coverImageUrl','altText','visibility','youtubeCategoryId','madeForKids','containsSyntheticMedia','targetPicker','prepareButton','emptyState','planState','planTitle','planStats','digest','mediaWarnings','targetCards','reviewed','confirmation','publishButton','publishResults','toast'];
const el=Object.fromEntries(ids.map((id)=>[id,document.getElementById(id)]));
const state={ sessionToken:'', platforms:[], media:{long:null,short:null}, plan:null };
const defaults=new Set(['youtube-long','youtube-short','facebook-reel','instagram-reel','tiktok','threads-video','vimeo-long','linkedin-video']);

await initialize();
el.chooseLong.addEventListener('click',()=>el.longFile.click()); el.chooseShort.addEventListener('click',()=>el.shortFile.click());
el.longFile.addEventListener('change',()=>loadMedia('long',el.longFile.files?.[0])); el.shortFile.addEventListener('change',()=>loadMedia('short',el.shortFile.files?.[0]));
el.prepareButton.addEventListener('click',prepare); el.reviewed.addEventListener('change',updatePublish); el.confirmation.addEventListener('input',updatePublish); el.publishButton.addEventListener('click',publish);
for (const control of document.querySelectorAll('input,textarea,select')) if (!['longFile','shortFile','reviewed','confirmation'].includes(control.id)) control.addEventListener('input',invalidatePlan);

async function initialize() {
  try { const response=await fetch('/api/session',{cache:'no-store'}); const data=await response.json(); if (!response.ok) throw new Error(data.error||'No se pudo iniciar.'); state.sessionToken=data.sessionToken; state.platforms=data.platforms; el.sessionStatus.textContent='Sesión local protegida'; el.dailyLimit.textContent=`Límite: ${data.dailyJobLimit} trabajos editoriales por día`; renderPicker(); }
  catch (error) { el.sessionStatus.textContent='No conectado'; toast(error.message,true); }
}

function renderPicker() {
  el.targetPicker.replaceChildren(...state.platforms.map((platform)=>{
    const label=document.createElement('label'); label.className='target-option';
    const input=document.createElement('input'); input.type='checkbox'; input.value=platform.id; input.checked=defaults.has(platform.id); input.addEventListener('change',invalidatePlan);
    const text=document.createElement('span'); const strong=document.createElement('strong'); strong.textContent=platform.name; const small=document.createElement('small');
    const status=platform.mode.startsWith('api-')?(platform.configured?'API configurada':'sin credenciales'):modeLabel(platform.mode); small.textContent=`${platform.role==='short'?'vertical':'largo'} · ${status}`; text.append(strong,small); label.append(input,text); return label;
  }));
}

async function loadMedia(role,file) {
  if (!file) return; const button=role==='long'?el.chooseLong:el.chooseShort; const progress=role==='long'?el.longProgress:el.shortProgress; const info=role==='long'?el.longInfo:el.shortInfo;
  button.disabled=true; progress.hidden=false; progress.value=0; info.textContent=`Copiando ${file.name} al espacio local…`;
  try {
    if (state.media[role]) await apiDelete(`/api/media/${state.media[role].id}`).catch(()=>{});
    const media=await upload(file,role,(percent)=>{progress.value=percent;}); state.media[role]=media;
    const dimensions=media.width&&media.height?`${media.width}×${media.height}`:'dimensiones no detectadas'; const duration=media.duration?formatDuration(media.duration):'duración no detectada';
    info.textContent=`${media.fileName} · ${formatBytes(media.size)} · ${dimensions} · ${duration} · SHA ${media.sha256.slice(0,12)}`; invalidatePlan(); toast(`${role==='short'?'Short':'Video largo'} cargado localmente.`);
  } catch (error) { info.textContent='La carga falló.'; toast(error.message,true); }
  finally { button.disabled=false; progress.hidden=true; }
}

function upload(file,role,onProgress) {
  return new Promise((resolve,reject)=>{ const xhr=new XMLHttpRequest(); xhr.open('POST','/api/media'); xhr.setRequestHeader('X-Solaris-Session',state.sessionToken); xhr.setRequestHeader('X-Video-Role',role); xhr.setRequestHeader('X-File-Name',encodeURIComponent(file.name)); xhr.setRequestHeader('Content-Type',file.type||'application/octet-stream'); xhr.upload.addEventListener('progress',(event)=>{if(event.lengthComputable)onProgress(Math.round(event.loaded/event.total*100));}); xhr.addEventListener('load',()=>{let data={};try{data=JSON.parse(xhr.responseText);}catch{} if(xhr.status>=200&&xhr.status<300)resolve(data);else reject(new Error(data.error||`HTTP ${xhr.status}`));}); xhr.addEventListener('error',()=>reject(new Error('Se interrumpió la copia local del video.'))); xhr.send(file); });
}

async function prepare() {
  const targets=selectedTargets(); if (!targets.length) return toast('Elegí al menos un destino.',true);
  el.prepareButton.disabled=true; el.prepareButton.textContent='Preparando…';
  try { state.plan=await api('/api/plan',{input:inputData(),media:mediaIds(),targets}); renderPlan(); toast('Plan preparado. Nada fue publicado.'); }
  catch (error) { toast(error.message,true); }
  finally { el.prepareButton.disabled=false; el.prepareButton.textContent='Preparar vista previa'; }
}

function renderPlan() {
  const plan=state.plan; el.emptyState.hidden=true; el.planState.hidden=false; el.planTitle.textContent=plan.metadata.title;
  const loaded=[plan.media.long?'1 largo':null,plan.media.short?'1 short':null].filter(Boolean).join(' + ')||'sin archivos'; el.planStats.textContent=`${loaded} · ${plan.targets.length} destinos`; el.digest.textContent=plan.digest;
  const warnings=[]; if (!plan.media.long) warnings.push('No hay video largo cargado.'); if (!plan.media.short) warnings.push('No hay short cargado.');
  if (plan.media.short?.duration>180) warnings.push('El archivo vertical supera tres minutos: YouTube no lo clasificará como Short.');
  if (plan.media.short?.width&&plan.media.short?.height&&plan.media.short.height<plan.media.short.width) warnings.push('El short es horizontal.');
  el.mediaWarnings.replaceChildren(...(warnings.length?warnings.map((text)=>notice(text)): [notice('Los archivos base cumplen las comprobaciones estructurales disponibles.',false)]));
  el.targetCards.replaceChildren(...plan.targets.map(targetCard)); el.reviewed.checked=false; el.confirmation.value=''; el.publishResults.textContent=''; updatePublish();
}

function targetCard(target) {
  const card=document.createElement('article'); card.className='target-card'; const header=document.createElement('header'); const check=document.createElement('input'); check.type='checkbox'; check.dataset.publishTarget=''; check.value=target.id; check.disabled=!target.canPublish; check.checked=target.canPublish&&defaults.has(target.id); check.addEventListener('change',updatePublish);
  const heading=document.createElement('div'); const h=document.createElement('h3'); h.textContent=target.name; const p=document.createElement('p'); p.textContent=`${target.role==='short'?'archivo vertical':'archivo largo'} · ${modeLabel(target.mode)}`; heading.append(h,p);
  const badge=document.createElement('span'); badge.className=`badge ${target.canPublish?'ready':'blocked'}`; badge.textContent=target.canPublish?'API lista':target.mode.startsWith('api-')?'bloqueado':'asistido'; header.append(check,heading,badge);
  const details=document.createElement('details'); const summary=document.createElement('summary'); summary.textContent='Ver adaptación'; const pre=document.createElement('pre'); pre.textContent=JSON.stringify(target.payload,null,2); details.append(summary,pre); card.append(header,details);
  for (const text of [...target.warnings,target.missingCredentials.length?`Faltan: ${target.missingCredentials.join(', ')}`:'',target.limitation].filter(Boolean)) card.append(notice(text));
  return card;
}

async function publish() {
  const publishTargets=[...document.querySelectorAll('[data-publish-target]:checked')].map((input)=>input.value); if (!publishTargets.length) return toast('No hay APIs publicables marcadas.',true);
  el.publishButton.disabled=true; el.publishButton.textContent='Publicando en cola…'; el.publishResults.textContent='';
  try { const result=await api('/api/publish',{input:inputData(),media:mediaIds(),targets:state.plan.targets.map((target)=>target.id),publishTargets,digest:state.plan.digest,confirmation:el.confirmation.value}); renderResults(result); toast(result.results.every((item)=>item.ok)?'Cola completada.':'La cola terminó con resultados parciales.',!result.results.some((item)=>item.ok)); }
  catch (error) { el.publishResults.textContent=error.message; toast(error.message,true); }
  finally { el.publishButton.textContent='Publicar destinos marcados'; updatePublish(); }
}

function renderResults(result) { const ul=document.createElement('ul'); for (const row of result.results) { const li=document.createElement('li'); li.className=row.ok?'ok':'fail'; li.textContent=`${row.name}: ${row.ok?[row.status,row.url].filter(Boolean).join(' · '):row.error}`; ul.append(li); } const path=document.createElement('p'); path.textContent=`Recibo local: ${result.receiptPath}`; el.publishResults.replaceChildren(ul,path); }
function inputData() { return { title:el.title.value,shortTitle:el.shortTitle.value,longDescription:el.longDescription.value,shortCaption:el.shortCaption.value,hashtags:split(el.hashtags.value),tags:split(el.tags.value),canonicalUrl:el.canonicalUrl.value,publicShortUrl:el.publicShortUrl.value,coverImageUrl:el.coverImageUrl.value,altText:el.altText.value,language:el.language.value,visibility:el.visibility.value,youtubeCategoryId:el.youtubeCategoryId.value,madeForKids:el.madeForKids.checked,containsSyntheticMedia:el.containsSyntheticMedia.checked }; }
function mediaIds() { return {longId:state.media.long?.id||'',shortId:state.media.short?.id||''}; }
function selectedTargets() { return [...el.targetPicker.querySelectorAll('input:checked')].map((input)=>input.value); }
function invalidatePlan() { if (!state.plan) return; state.plan=null; el.planState.hidden=true; el.emptyState.hidden=false; el.reviewed.checked=false; el.confirmation.value=''; updatePublish(); }
function updatePublish() { const count=document.querySelectorAll('[data-publish-target]:checked').length; el.publishButton.disabled=!(state.plan&&count&&el.reviewed.checked&&el.confirmation.value==='PUBLICAR VIDEO'); }
async function api(path,body) { const response=await fetch(path,{method:'POST',headers:{'Content-Type':'application/json','X-Solaris-Session':state.sessionToken},body:JSON.stringify(body)}); const data=await response.json().catch(()=>({error:`HTTP ${response.status}`})); if(!response.ok&&response.status!==207)throw new Error(data.error||`HTTP ${response.status}`); return data; }
async function apiDelete(path) { const response=await fetch(path,{method:'DELETE',headers:{'X-Solaris-Session':state.sessionToken}}); if(!response.ok)throw new Error('No se pudo quitar el archivo anterior.'); }
function notice(text,warning=true) { const p=document.createElement('p'); p.className=`notice${warning?'':' ok'}`; p.textContent=text; return p; }
function split(value) { return String(value||'').split(',').map((item)=>item.trim()).filter(Boolean); }
function modeLabel(value) { return ({'api-local':'API · archivo local','api-url':'API · URL pública','assisted':'preparación asistida','no-upload-api':'sin API de upload','planned':'conector planificado','instance-dependent':'depende de instancia','paid':'API paga'})[value]||value; }
function formatBytes(value) { return value<1024*1024?`${(value/1024).toFixed(1)} KiB`:value<1024*1024*1024?`${(value/1024/1024).toFixed(1)} MiB`:`${(value/1024/1024/1024).toFixed(2)} GiB`; }
function formatDuration(value) { const seconds=Math.round(value); return `${Math.floor(seconds/60)}:${String(seconds%60).padStart(2,'0')}`; }
function toast(message,error=false) { el.toast.textContent=message; el.toast.className=`toast${error?' error':''}`; el.toast.hidden=false; clearTimeout(toast.timer); toast.timer=setTimeout(()=>{el.toast.hidden=true;},5000); }
