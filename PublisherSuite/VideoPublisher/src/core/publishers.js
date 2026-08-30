import { createReadStream } from 'node:fs';
import { open, mkdir, readFile, rename, rm, writeFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root=dirname(dirname(dirname(fileURLToPath(import.meta.url))));
const outputRoot=join(root,'output');
const journalPath=join(outputRoot,'publish-journal.json');

export async function publishVideoTargets(plan, targetIds, mediaRegistry, env=process.env) {
  await enforceDailyLimit(env);
  const results=[];
  for (const id of [...new Set(targetIds)]) {
    const target=plan.targets.find((item)=>item.id===id);
    if (!target) { results.push(fail(id,'El destino no forma parte del plan.')); continue; }
    if (!target.canPublish) { results.push(fail(id,blockedReason(target),target.name)); continue; }
    const media=mediaRegistry.get(target.mediaId);
    if (!media) { results.push(fail(id,'El archivo local ya no está disponible en esta sesión.',target.name)); continue; }
    try { results.push({ id, name:target.name, ok:true, ...(await publishOne(target,media,plan,env)) }); }
    catch (error) { results.push(fail(id,safeError(error),target.name)); }
  }
  const receipt={ schemaVersion:1, digest:plan.digest, createdAt:new Date().toISOString(), title:plan.metadata.title, results };
  const receiptPath=join(outputRoot,plan.digestShort,'receipt.json');
  await atomicWrite(receiptPath,`${JSON.stringify(receipt,null,2)}\n`);
  if (results.some((item)=>item.ok)) await appendJournal({ digest:plan.digest, createdAt:receipt.createdAt, targets:results.filter((item)=>item.ok).map((item)=>item.id) });
  return { ...receipt, receiptPath };
}

async function publishOne(target,media,plan,env) {
  switch (target.id) {
    case 'youtube-long': case 'youtube-short': return publishYouTube(target,media,env);
    case 'facebook-reel': return publishFacebookReel(target,media,env);
    case 'instagram-reel': return publishInstagramReel(target,env);
    case 'tiktok': return publishTikTok(target,media,env);
    case 'threads-video': return publishThreadsVideo(target,env);
    case 'vimeo-long': return publishVimeo(target,media,env);
    case 'linkedin-video': return publishLinkedInVideo(target,media,env);
    default: throw new Error('Este destino sólo tiene preparación asistida en esta versión.');
  }
}

async function publishYouTube(target,media,env) {
  const resource={ snippet:{ title:target.payload.title, description:target.payload.description, tags:target.payload.tags, categoryId:target.payload.categoryId, defaultLanguage:'es' }, status:{ privacyStatus:target.payload.privacyStatus, selfDeclaredMadeForKids:target.payload.madeForKids, containsSyntheticMedia:target.payload.containsSyntheticMedia, embeddable:true } };
  const init=await fetchChecked('YouTube','https://www.googleapis.com/upload/youtube/v3/videos?uploadType=resumable&part=snippet,status',{ method:'POST', headers:{ ...bearer(env.SOLARIS_YOUTUBE_ACCESS_TOKEN), 'Content-Type':'application/json; charset=UTF-8', 'X-Upload-Content-Length':String(media.size), 'X-Upload-Content-Type':media.mime }, body:JSON.stringify(resource) });
  const uploadUrl=init.headers.get('location'); if (!uploadUrl) throw new Error('YouTube no devolvió una URL de upload reanudable.');
  const uploaded=await fetchChecked('YouTube',uploadUrl,{ method:'PUT', headers:{ 'Content-Type':media.mime, 'Content-Length':String(media.size) }, body:createReadStream(media.path), duplex:'half' });
  const data=await uploaded.json();
  return { status:'uploaded', externalId:data.id||null, url:data.id?`https://www.youtube.com/watch?v=${data.id}`:null };
}

async function publishFacebookReel(target,media,env) {
  const version=env.SOLARIS_META_GRAPH_VERSION||'v25.0';
  const start=await postForm('Facebook',`https://graph.facebook.com/${version}/me/video_reels`,{ access_token:env.SOLARIS_FACEBOOK_PAGE_TOKEN, upload_phase:'start' });
  if (!start.video_id || !start.upload_url) throw new Error('Facebook no inició la carga del Reel.');
  await fetchChecked('Facebook',start.upload_url,{ method:'POST', headers:{ Authorization:`OAuth ${env.SOLARIS_FACEBOOK_PAGE_TOKEN}`, offset:'0', file_size:String(media.size), 'Content-Type':'application/octet-stream' }, body:createReadStream(media.path), duplex:'half' });
  const finished=await postForm('Facebook',`https://graph.facebook.com/${version}/me/video_reels`,{ access_token:env.SOLARIS_FACEBOOK_PAGE_TOKEN, video_id:start.video_id, upload_phase:'finish', video_state:target.payload.videoState, description:target.payload.description, title:target.payload.title });
  return { status:target.payload.videoState==='PUBLISHED'?'published':'draft', externalId:String(start.video_id), url:finished.post_id?`https://www.facebook.com/reel/${finished.post_id}`:null };
}

async function publishInstagramReel(target,env) {
  const version=env.SOLARIS_META_GRAPH_VERSION||'v25.0';
  const user=encodeURIComponent(env.SOLARIS_INSTAGRAM_USER_ID);
  const container=await postForm('Instagram',`https://graph.facebook.com/${version}/${user}/media`,{ media_type:'REELS', video_url:target.payload.videoUrl, caption:target.payload.caption, share_to_feed:String(target.payload.shareToFeed), access_token:env.SOLARIS_INSTAGRAM_ACCESS_TOKEN });
  if (!container.id) throw new Error('Instagram no devolvió un contenedor.');
  await waitForInstagram(container.id,version,env.SOLARIS_INSTAGRAM_ACCESS_TOKEN);
  const result=await postForm('Instagram',`https://graph.facebook.com/${version}/${user}/media_publish`,{ creation_id:container.id, access_token:env.SOLARIS_INSTAGRAM_ACCESS_TOKEN });
  return { status:'published', externalId:result.id||null, url:null };
}

async function publishThreadsVideo(target,env) {
  const base=env.SOLARIS_THREADS_API_BASE||'https://graph.threads.net/v1.0';
  const user=encodeURIComponent(env.SOLARIS_THREADS_USER_ID);
  const container=await postForm('Threads',`${base}/${user}/threads`,{ media_type:'VIDEO', video_url:target.payload.videoUrl, text:target.payload.text, alt_text:target.payload.altText, access_token:env.SOLARIS_THREADS_ACCESS_TOKEN });
  const result=await postForm('Threads',`${base}/${user}/threads_publish`,{ creation_id:container.id, access_token:env.SOLARIS_THREADS_ACCESS_TOKEN });
  return { status:'published', externalId:result.id||null, url:null };
}

async function publishTikTok(target,media,env) {
  const headers={ ...bearer(env.SOLARIS_TIKTOK_ACCESS_TOKEN), 'Content-Type':'application/json; charset=UTF-8' };
  const creator=await requestJson('TikTok','https://open.tiktokapis.com/v2/post/publish/creator_info/query/',{ method:'POST', headers, body:'{}' });
  const allowed=creator.data?.privacy_level_options||[];
  if (allowed.length && !allowed.includes(target.payload.privacyLevel)) throw new Error(`TikTok no permite ${target.payload.privacyLevel} para esta cuenta.`);
  const chunkSize=Math.min(64*1024*1024,Math.max(5*1024*1024,Math.ceil(media.size/1000)));
  const totalChunks=Math.ceil(media.size/chunkSize);
  const initBody={ post_info:{ title:target.payload.title, privacy_level:target.payload.privacyLevel, disable_comment:target.payload.disableComment, disable_duet:target.payload.disableDuet, disable_stitch:target.payload.disableStitch }, source_info:{ source:'FILE_UPLOAD', video_size:media.size, chunk_size:totalChunks===1?media.size:chunkSize, total_chunk_count:totalChunks } };
  const initialized=await requestJson('TikTok','https://open.tiktokapis.com/v2/post/publish/video/init/',{ method:'POST', headers, body:JSON.stringify(initBody) });
  if (initialized.error?.code && initialized.error.code!=='ok') throw new Error(`TikTok rechazó el inicio: ${initialized.error.message||initialized.error.code}`);
  if (!initialized.data?.upload_url) throw new Error('TikTok no devolvió una URL de upload.');
  await uploadChunks('TikTok',initialized.data.upload_url,media,chunkSize);
  return { status:'processing', externalId:initialized.data.publish_id||null, url:null };
}

async function publishVimeo(target,media,env) {
  const created=await requestJson('Vimeo','https://api.vimeo.com/me/videos',{ method:'POST', headers:{ ...bearer(env.SOLARIS_VIMEO_ACCESS_TOKEN), 'Content-Type':'application/json', Accept:'application/vnd.vimeo.*+json;version=3.4' }, body:JSON.stringify({ upload:{ approach:'tus', size:media.size }, name:target.payload.name, description:target.payload.description, privacy:{ view:target.payload.privacy } }) });
  const uploadUrl=created.upload?.upload_link; if (!uploadUrl) throw new Error('Vimeo no devolvió upload.upload_link.');
  await uploadTus(uploadUrl,media);
  return { status:'processing', externalId:created.uri||null, url:created.link||null };
}

async function publishLinkedInVideo(target,media,env) {
  const version=env.SOLARIS_LINKEDIN_VERSION||currentVersion();
  const apiHeaders={ ...bearer(env.SOLARIS_LINKEDIN_ACCESS_TOKEN), 'Linkedin-Version':version, 'X-Restli-Protocol-Version':'2.0.0', 'Content-Type':'application/json' };
  const initialized=await requestJson('LinkedIn','https://api.linkedin.com/rest/videos?action=initializeUpload',{ method:'POST', headers:apiHeaders, body:JSON.stringify({ initializeUploadRequest:{ owner:env.SOLARIS_LINKEDIN_AUTHOR_URN, fileSizeBytes:media.size, uploadCaptions:false, uploadThumbnail:false } }) });
  const value=initialized.value||{}; if (!value.video || !value.uploadInstructions?.length) throw new Error('LinkedIn no devolvió instrucciones de upload.');
  const file=await open(media.path,'r'); const partIds=[];
  try {
    for (const instruction of value.uploadInstructions) {
      const length=instruction.lastByte-instruction.firstByte+1; const buffer=Buffer.alloc(length);
      await file.read(buffer,0,length,instruction.firstByte);
      const response=await fetchChecked('LinkedIn',instruction.uploadUrl,{ method:'PUT', headers:{ 'Content-Type':'application/octet-stream', 'Content-Length':String(length) }, body:buffer });
      const etag=response.headers.get('etag'); if (etag) partIds.push(etag.replace(/^"|"$/g,''));
    }
  } finally { await file.close(); }
  await requestJson('LinkedIn','https://api.linkedin.com/rest/videos?action=finalizeUpload',{ method:'POST', headers:apiHeaders, body:JSON.stringify({ finalizeUploadRequest:{ video:value.video, uploadToken:value.uploadToken||'', uploadedPartIds:partIds } }) });
  const post=await fetchChecked('LinkedIn','https://api.linkedin.com/rest/posts',{ method:'POST', headers:apiHeaders, body:JSON.stringify({ author:env.SOLARIS_LINKEDIN_AUTHOR_URN, commentary:target.payload.commentary, visibility:'PUBLIC', distribution:{ feedDistribution:'MAIN_FEED', targetEntities:[], thirdPartyDistributionChannels:[] }, content:{ media:{ title:target.payload.title, id:value.video } }, lifecycleState:'PUBLISHED', isReshareDisabledByAuthor:false }) });
  return { status:'published', externalId:post.headers.get('x-restli-id')||value.video, url:null };
}

async function uploadChunks(label,url,media,chunkSize) {
  const file=await open(media.path,'r');
  try { for (let start=0;start<media.size;start+=chunkSize) { const end=Math.min(media.size,start+chunkSize)-1; const length=end-start+1; const buffer=Buffer.alloc(length); await file.read(buffer,0,length,start); await fetchChecked(label,url,{ method:'PUT', headers:{ 'Content-Type':'video/mp4', 'Content-Length':String(length), 'Content-Range':`bytes ${start}-${end}/${media.size}` }, body:buffer }); } }
  finally { await file.close(); }
}

async function uploadTus(url,media) {
  const chunkSize=32*1024*1024; const file=await open(media.path,'r');
  try { for (let offset=0;offset<media.size;offset+=chunkSize) { const length=Math.min(chunkSize,media.size-offset); const buffer=Buffer.alloc(length); await file.read(buffer,0,length,offset); await fetchChecked('Vimeo',url,{ method:'PATCH', headers:{ 'Tus-Resumable':'1.0.0', 'Upload-Offset':String(offset), 'Content-Type':'application/offset+octet-stream', 'Content-Length':String(length) }, body:buffer }); } }
  finally { await file.close(); }
}

async function waitForInstagram(id,version,token) {
  for (let attempt=0;attempt<24;attempt++) {
    const status=await requestJson('Instagram',`https://graph.facebook.com/${version}/${encodeURIComponent(id)}?fields=status_code,status&access_token=${encodeURIComponent(token)}`);
    if (status.status_code==='FINISHED') return;
    if (['ERROR','EXPIRED'].includes(status.status_code)) throw new Error(`Instagram no procesó el Reel: ${status.status||status.status_code}`);
    await new Promise((resolve)=>setTimeout(resolve,5000));
  }
  throw new Error('Instagram sigue procesando el Reel después de dos minutos.');
}

async function requestJson(label,url,options={}) { const response=await fetchChecked(label,url,options); return response.json().catch(()=>({})); }
async function postForm(label,url,values) { const body=new URLSearchParams(); for (const [key,value] of Object.entries(values)) if (value!==undefined && value!==null && value!=='') body.set(key,String(value)); return requestJson(label,url,{ method:'POST', headers:{ 'Content-Type':'application/x-www-form-urlencoded' }, body:body.toString() }); }
async function fetchChecked(label,url,options={}) {
  const controller=new AbortController(); const timeout=setTimeout(()=>controller.abort(),120000);
  try { const response=await fetch(url,{ ...options, signal:controller.signal, redirect:'error' }); if (!response.ok) { const detail=(await response.text().catch(()=>'' )).replace(/[\r\n]+/g,' ').slice(0,500); throw new Error(`${label} respondió HTTP ${response.status}${detail?`: ${detail}`:'.'}`); } return response; }
  catch (error) { if (error.name==='AbortError') throw new Error(`${label} no respondió dentro de dos minutos.`); throw error; }
  finally { clearTimeout(timeout); }
}

async function enforceDailyLimit(env) { const limit=Math.max(1,Math.min(10,Number.parseInt(env.SOLARIS_VIDEO_DAILY_JOB_LIMIT||'2',10)||2)); const journal=await readJournal(); const today=localDay(new Date()); if (journal.filter((item)=>localDay(new Date(item.createdAt))===today).length>=limit) throw new Error(`Se alcanzó el límite local de ${limit} trabajos de video por día.`); }
async function appendJournal(item) { const journal=await readJournal(); journal.push(item); await atomicWrite(journalPath,`${JSON.stringify(journal.slice(-100),null,2)}\n`); }
async function readJournal() { try { const value=JSON.parse(await readFile(journalPath,'utf8')); return Array.isArray(value)?value:[]; } catch { return []; } }
function localDay(date) { return new Intl.DateTimeFormat('en-CA',{ timeZone:'America/Argentina/Buenos_Aires',year:'numeric',month:'2-digit',day:'2-digit' }).format(date); }
async function atomicWrite(path,contents) { await mkdir(dirname(path),{recursive:true}); const temp=`${path}.${process.pid}.${Date.now()}.tmp`; try { await writeFile(temp,contents,{encoding:'utf8',flag:'wx'}); try { await rename(temp,path); } catch (error) { if (!['EEXIST','EPERM'].includes(error.code)) throw error; await rm(path,{force:true}); await rename(temp,path); } } catch (error) { await rm(temp,{force:true}); throw error; } }
function bearer(token) { return { Authorization:`Bearer ${token}` }; }
function blockedReason(target) { if (target.missingCredentials.length) return `Faltan credenciales: ${target.missingCredentials.join(', ')}.`; return target.warnings.join(' ')||target.limitation||'Destino no publicable.'; }
function fail(id,error,name=id) { return { id,name,ok:false,error }; }
function safeError(error) { return String(error?.message||'Error de publicación.').replace(/Bearer\s+[A-Za-z0-9._~-]+/gi,'Bearer [redactado]').replace(/access_token=[^&\s]+/gi,'access_token=[redactado]').slice(0,1200); }
function currentVersion() { const now=new Date(); return `${now.getUTCFullYear()}${String(now.getUTCMonth()+1).padStart(2,'0')}`; }
