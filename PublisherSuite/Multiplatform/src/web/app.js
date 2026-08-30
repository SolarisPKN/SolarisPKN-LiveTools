const elements = Object.fromEntries([
  'sourceFile','sourceName','sourceSize','chooseSource','loadDemo','dropZone','metadataFile','metadataStatus',
  'title','description','canonicalUrl','coverImage','tags','language','targetPicker','prepareButton','emptyState',
  'planState','planTitle','planStats','digest','globalWarnings','targetCards','reviewed','confirmation','publishButton','publishResults','toast'
].map((id) => [id, document.getElementById(id)]));

const state = { source: '', metadata: {}, sessionToken: '', platforms: [], plan: null };
const defaults = new Set(['solarispkn-blog','devto','linkedin','mastodon','bluesky','threads']);
const demo = `# Automatizar sin delegar el criterio

Una sola fuente puede viajar a muchas plataformas, pero no debería fingir que todas son iguales.

## El contrato

- El artículo conserva su versión completa.
- Cada red recibe un resumen dentro de su límite.
- Las imágenes faltantes bloquean los destinos que realmente las exigen.
- La publicación sucede únicamente después de revisar el plan.

> Automatizar la repetición no significa automatizar la decisión.
`;

await initialize();

async function initialize() {
  try {
    const response = await fetch('/api/session', { cache: 'no-store' });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || 'No se pudo iniciar la sesión local.');
    state.sessionToken = data.sessionToken;
    state.platforms = data.platforms;
    renderTargetPicker();
  } catch (error) { showToast(error.message, true); }
}

elements.chooseSource.addEventListener('click', () => elements.sourceFile.click());
elements.sourceFile.addEventListener('change', () => loadSource(elements.sourceFile.files?.[0]));
elements.metadataFile.addEventListener('change', () => loadMetadata(elements.metadataFile.files?.[0]));
elements.loadDemo.addEventListener('click', () => { state.source=demo; elements.sourceName.textContent='ejemplo.mdx'; elements.sourceSize.textContent=formatBytes(new Blob([demo]).size); elements.prepareButton.disabled=false; });
elements.prepareButton.addEventListener('click', preparePlan);
elements.reviewed.addEventListener('change', updatePublishState);
elements.confirmation.addEventListener('input', updatePublishState);
elements.publishButton.addEventListener('click', publishPlan);

for (const eventName of ['dragenter','dragover']) elements.dropZone.addEventListener(eventName, (event) => { event.preventDefault(); elements.dropZone.classList.add('dragging'); });
for (const eventName of ['dragleave','drop']) elements.dropZone.addEventListener(eventName, (event) => { event.preventDefault(); elements.dropZone.classList.remove('dragging'); });
elements.dropZone.addEventListener('drop', (event) => loadSource(event.dataTransfer?.files?.[0]));

async function loadSource(file) {
  if (!file) return;
  if (file.size > 2 * 1024 * 1024) return showToast('El archivo supera 2 MiB.', true);
  state.source = await file.text();
  elements.sourceName.textContent = file.name;
  elements.sourceSize.textContent = formatBytes(file.size);
  elements.prepareButton.disabled = !state.source.trim();
}

async function loadMetadata(file) {
  if (!file) { state.metadata={}; return; }
  try {
    if (file.size > 256 * 1024) throw new Error('El JSON supera 256 KiB.');
    state.metadata = JSON.parse(await file.text());
    elements.metadataStatus.textContent = `${file.name} cargado.`;
    hydrateMetadata(state.metadata);
  } catch (error) { state.metadata={}; elements.metadataStatus.textContent=''; showToast(`Metadatos inválidos: ${error.message}`, true); }
}

function hydrateMetadata(metadata) {
  if (metadata.title && !elements.title.value) elements.title.value=metadata.title;
  if (metadata.description && !elements.description.value) elements.description.value=metadata.description;
  if ((metadata.canonicalUrl || metadata.url) && !elements.canonicalUrl.value) elements.canonicalUrl.value=metadata.canonicalUrl || metadata.url;
  if ((metadata.coverImage || metadata.image) && !elements.coverImage.value) elements.coverImage.value=metadata.coverImage || metadata.image;
  if (metadata.tags && !elements.tags.value) elements.tags.value=(Array.isArray(metadata.tags) ? metadata.tags : [metadata.tags]).join(', ');
  if (metadata.language || metadata.lang) elements.language.value=metadata.language || metadata.lang;
}

function renderTargetPicker() {
  elements.targetPicker.replaceChildren(...state.platforms.map((platform) => {
    const label=document.createElement('label'); label.className='target-option';
    const input=document.createElement('input'); input.type='checkbox'; input.value=platform.id; input.checked=defaults.has(platform.id); input.addEventListener('change', invalidatePlan);
    const text=document.createElement('span');
    const mode=modeLabel(platform.mode);
    const status=platform.mode==='bundle' ? 'listo' : platform.mode==='assisted' ? 'manual' : platform.mode==='media-only' ? 'requiere media' : platform.configured ? 'listo' : 'sin credenciales';
    text.innerHTML=`<strong>${escapeHtml(platform.name)}</strong><small>${escapeHtml(mode)} · ${escapeHtml(status)}</small>`;
    label.append(input,text); return label;
  }));
}

function invalidatePlan() {
  if (!state.plan) return;
  state.plan=null; elements.planState.hidden=true; elements.emptyState.hidden=false; elements.reviewed.checked=false; elements.confirmation.value=''; updatePublishState();
}

async function preparePlan() {
  if (!state.source.trim()) return showToast('Cargá un Markdown primero.', true);
  const targets=selectedTargets();
  if (!targets.length) return showToast('Elegí al menos un destino.', true);
  elements.prepareButton.disabled=true; elements.prepareButton.textContent='Preparando…';
  try {
    const response=await api('/api/plan', { input: buildInput(), targets });
    state.plan=response; renderPlan(); showToast('Plan preparado. Nada fue publicado.');
  } catch (error) { showToast(error.message,true); }
  finally { elements.prepareButton.disabled=false; elements.prepareButton.textContent='Preparar vista previa'; }
}

function renderPlan() {
  const plan=state.plan;
  elements.emptyState.hidden=true; elements.planState.hidden=false;
  elements.planTitle.textContent=plan.content.title;
  elements.planStats.textContent=`${plan.content.stats.words} palabras · ${plan.content.stats.images} imágenes · ${plan.targets.length} destinos`;
  elements.digest.textContent=plan.digest;
  elements.globalWarnings.replaceChildren(...(plan.content.warnings.length ? plan.content.warnings.map((warning) => notice(warning.message || String(warning))) : [notice('El MDX no generó advertencias estructurales.', false)]));
  elements.targetCards.replaceChildren(...plan.targets.map(targetCard));
  elements.reviewed.checked=false; elements.confirmation.value=''; elements.publishResults.textContent=''; updatePublishState();
}

function targetCard(target) {
  const article=document.createElement('article'); article.className='target-card';
  const header=document.createElement('header');
  const selectable=target.canPublish;
  const check=document.createElement('input'); check.type='checkbox'; check.value=target.id; check.dataset.publishTarget=''; check.disabled=!selectable; check.checked=selectable && defaults.has(target.id); check.addEventListener('change',updatePublishState);
  const heading=document.createElement('div'); heading.innerHTML=`<h3>${escapeHtml(target.name)}</h3><p>${escapeHtml(modeLabel(target.mode))}${target.characterCount!==null ? ` · ${target.characterCount} caracteres` : ''}</p>`;
  const badge=document.createElement('span'); badge.className=`badge ${selectable ? 'ready' : 'blocked'}`; badge.textContent=selectable ? (target.mode==='bundle' ? 'bundle listo' : 'API lista') : target.mode==='assisted' ? 'asistido' : target.mode==='media-only' ? 'sólo media' : 'bloqueado';
  header.append(check,heading,badge);
  const pre=document.createElement('pre'); pre.textContent=JSON.stringify(target.payload,null,2);
  const details=document.createElement('details'); const summary=document.createElement('summary'); summary.textContent='Ver payload adaptado'; details.append(summary,pre);
  const notes=document.createElement('div'); notes.className='target-notes';
  const messages=[target.limitation,...target.warnings,target.missingCredentials.length ? `Faltan: ${target.missingCredentials.join(', ')}` : ''].filter(Boolean);
  notes.replaceChildren(...messages.map((message)=>notice(message)));
  const copy=document.createElement('button'); copy.className='text-button'; copy.type='button'; copy.textContent='Copiar adaptación'; copy.addEventListener('click',()=>copyText(extractCopyText(target.payload)));
  article.append(header,details,notes,copy); return article;
}

async function publishPlan() {
  const targets=[...document.querySelectorAll('[data-publish-target]:checked')].map((input)=>input.value);
  if (!targets.length) return showToast('No hay destinos publicables marcados.',true);
  elements.publishButton.disabled=true; elements.publishButton.textContent='Publicando…'; elements.publishResults.textContent='';
  try {
    const response=await api('/api/publish',{ input:buildInput(), targets:state.plan.targets.map((target)=>target.id), publishTargets:targets, digest:state.plan.digest, confirmation:elements.confirmation.value });
    renderReceipts(response); showToast(response.results.every((item)=>item.ok) ? 'Publicación terminada.' : 'La publicación terminó con resultados parciales.', !response.results.some((item)=>item.ok));
  } catch (error) { showToast(error.message,true); elements.publishResults.textContent=error.message; }
  finally { elements.publishButton.textContent='Publicar destinos seleccionados'; updatePublishState(); }
}

function renderReceipts(receipt) {
  const list=document.createElement('ul');
  for (const result of receipt.results) {
    const item=document.createElement('li'); item.className=result.ok?'ok':'fail';
    const suffix=result.ok ? [result.status,result.url].filter(Boolean).join(' · ') : result.error;
    item.textContent=`${result.name}: ${suffix}`; list.append(item);
  }
  const path=document.createElement('p'); path.textContent=`Recibo local: ${receipt.receiptPath}`;
  elements.publishResults.replaceChildren(list,path);
}

function buildInput() {
  return { source:state.source, options:{ metadata:state.metadata, title:elements.title.value.trim(), description:elements.description.value.trim(), canonicalUrl:elements.canonicalUrl.value.trim(), coverImage:elements.coverImage.value.trim(), tags:elements.tags.value.split(',').map((item)=>item.trim()).filter(Boolean), language:elements.language.value.trim() || 'es' } };
}

async function api(path, body) {
  const response=await fetch(path,{method:'POST',headers:{'Content-Type':'application/json','X-Solaris-Session':state.sessionToken},body:JSON.stringify(body)});
  const data=await response.json().catch(()=>({error:`HTTP ${response.status}`}));
  if (!response.ok && response.status!==207) throw new Error(data.error || `HTTP ${response.status}`);
  return data;
}

function selectedTargets() { return [...elements.targetPicker.querySelectorAll('input:checked')].map((input)=>input.value); }
function updatePublishState() { const selected=document.querySelectorAll('[data-publish-target]:checked').length; elements.publishButton.disabled=!(state.plan && selected && elements.reviewed.checked && elements.confirmation.value==='PUBLICAR'); }
function notice(text,warning=true) { const p=document.createElement('p'); p.className=warning?'notice warning':'notice ok'; p.textContent=text; return p; }
function extractCopyText(payload) { return payload.text || payload.commentary || payload.caption || payload.body_markdown || payload.markdown || payload.content || JSON.stringify(payload,null,2); }
async function copyText(text) { try { await navigator.clipboard.writeText(String(text)); showToast('Adaptación copiada.'); } catch { showToast('No se pudo copiar automáticamente.',true); } }
function modeLabel(mode) { return ({api:'API oficial','api-media':'API + media','bundle':'bundle local',assisted:'publicación asistida','media-only':'requiere archivo multimedia'})[mode] || mode; }
function formatBytes(value) { return value<1024?`${value} B`:value<1024*1024?`${(value/1024).toFixed(1)} KiB`:`${(value/1024/1024).toFixed(1)} MiB`; }
function escapeHtml(value) { return String(value).replace(/[&<>"]/g,(char)=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;'}[char])); }
function showToast(message,error=false) { elements.toast.textContent=message; elements.toast.className=`toast${error?' error':''}`; elements.toast.hidden=false; clearTimeout(showToast.timer); showToast.timer=setTimeout(()=>{elements.toast.hidden=true;},4500); }
