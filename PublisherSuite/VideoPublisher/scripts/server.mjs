import { createHash, randomBytes, randomUUID } from 'node:crypto';
import { execFile, spawn } from 'node:child_process';
import { createWriteStream } from 'node:fs';
import { access, mkdir, readFile, rm, stat } from 'node:fs/promises';
import { constants } from 'node:fs';
import { basename, dirname, extname, join, normalize, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createServer } from 'node:http';
import { pipeline } from 'node:stream/promises';
import { buildVideoPlan, publicPlatformStatus, videoPlatforms } from '../src/core/video-plan.js';
import { publishVideoTargets } from '../src/core/publishers.js';

const root=dirname(dirname(fileURLToPath(import.meta.url)));
const publicRoot=join(root,'dist'); const mediaRoot=join(root,'media');
const host='127.0.0.1'; const explicit=Boolean(process.env.SOLARIS_VIDEO_PORT); const requested=parsePort(process.env.SOLARIS_VIDEO_PORT||'4181');
const shouldOpen=process.argv.includes('--open')&&process.env.SOLARIS_VIDEO_NO_OPEN!=='1';
const sessionToken=randomBytes(32).toString('base64url'); const sessionDirectory=join(mediaRoot,sessionToken.slice(0,12));
const mediaRegistry=new Map(); let fallback=false;
await access(join(publicRoot,'index.html'),constants.R_OK).catch(()=>{ console.error('Falta dist/index.html. Ejecutá npm run build.'); process.exit(1); });
await mkdir(sessionDirectory,{recursive:true});

const server=createServer(async (request,response)=>{
  try {
    if (!allowedHost(request.headers.host)) return send(response,400,{error:'Host no permitido.'});
    const url=new URL(request.url||'/',`http://${request.headers.host}`);
    if (url.pathname.startsWith('/api/')) return handleApi(request,response,url);
    if (!['GET','HEAD'].includes(request.method||'')) return send(response,405,'Método no permitido.','text/plain; charset=utf-8');
    return serveStatic(request,response,url);
  } catch (error) { console.error(safeMessage(error)); return send(response,500,{error:safeMessage(error)}); }
});

server.on('error',(error)=>{ if (error.code==='EADDRINUSE'&&!explicit&&!fallback) { fallback=true; console.warn(`El puerto ${requested} está ocupado; se elegirá otro.`); server.listen(0,host); return; } if (error.code==='EADDRINUSE') { console.error(`El puerto ${requested} está ocupado.`); process.exitCode=1; return; } throw error; });
server.listen(requested,host,()=>{ const port=server.address().port; const url=`http://${host}:${port}`; console.log(''); console.log('SolarisPKN Video Publisher está listo.'); console.log(`Abrí: ${url}`); console.log('Máximo local predeterminado: 2 trabajos editoriales por día.'); console.log('Ctrl+C cierra la sesión; los videos copiados quedan en media/ e ignorados por Git.'); console.log(''); if (shouldOpen) openBrowser(url); });

async function handleApi(request,response,url) {
  if (request.method==='GET'&&url.pathname==='/api/session') return send(response,200,{ sessionToken, platforms:publicPlatformStatus(), dailyJobLimit:Number.parseInt(process.env.SOLARIS_VIDEO_DAILY_JOB_LIMIT||'2',10)||2 });
  if (!authorized(request)) return send(response,403,{error:'Solicitud local no autorizada.'});
  if (request.method==='POST'&&url.pathname==='/api/media') return uploadMedia(request,response);
  if (request.method==='DELETE'&&url.pathname.startsWith('/api/media/')) return deleteMedia(request,response,url.pathname.split('/').pop());
  if (request.method==='POST'&&['/api/plan','/api/publish'].includes(url.pathname)) {
    if (!String(request.headers['content-type']||'').toLowerCase().startsWith('application/json')) return send(response,415,{error:'Se requiere application/json.'});
    const body=await readJson(request,1024*1024); const targets=validateTargets(body.targets); const media=resolveMedia(body.media);
    const plan=buildVideoPlan(body.input,media,targets);
    if (url.pathname==='/api/plan') return send(response,200,publicPlan(plan));
    if (body.confirmation!=='PUBLICAR VIDEO') return send(response,409,{error:'Escribí PUBLICAR VIDEO para confirmar.'});
    if (body.digest!==plan.digest) return send(response,409,{error:'Los archivos, metadatos o destinos cambiaron. Prepará el plan nuevamente.'});
    const selection=validateTargets(body.publishTargets); if (selection.some((id)=>!targets.includes(id))) return send(response,409,{error:'La publicación incluye un destino ajeno al plan.'});
    const result=await publishVideoTargets(plan,selection,mediaRegistry); return send(response,result.results.every((item)=>item.ok)?200:207,result);
  }
  return send(response,404,{error:'Endpoint no encontrado.'});
}

async function uploadMedia(request,response) {
  const role=String(request.headers['x-video-role']||''); if (!['long','short'].includes(role)) return send(response,400,{error:'Rol de video inválido.'});
  const size=Number.parseInt(request.headers['content-length']||'',10); const max=Math.max(1024*1024,Number.parseInt(process.env.SOLARIS_VIDEO_MAX_BYTES||String(20*1024*1024*1024),10));
  if (!Number.isSafeInteger(size)||size<=0||size>max) return send(response,413,{error:`El archivo debe medir entre 1 byte y ${formatBytes(max)}.`});
  const mime=String(request.headers['content-type']||'application/octet-stream').split(';')[0].toLowerCase(); if (!mime.startsWith('video/')&&mime!=='application/octet-stream') return send(response,415,{error:'El archivo no tiene un tipo MIME de video.'});
  const original=safeFileName(decodeURIComponent(String(request.headers['x-file-name']||`${role}.mp4`))); const id=randomUUID(); const target=join(sessionDirectory,`${id}${safeExtension(original)}`); const hash=createHash('sha256'); let received=0;
  request.on('data',(chunk)=>{ received+=chunk.length; hash.update(chunk); if (received>max) request.destroy(new Error('El archivo superó el máximo.')); });
  try { await pipeline(request,createWriteStream(target,{flags:'wx'})); if (received!==size) throw new Error('La carga quedó incompleta.'); const probe=await probeVideo(target); const media={ id,role,fileName:original,path:target,size:received,sha256:hash.digest('hex'),mime,duration:probe.duration,width:probe.width,height:probe.height,codec:probe.codec }; mediaRegistry.set(id,media); return send(response,201,publicMedia(media)); }
  catch (error) { await rm(target,{force:true}); throw error; }
}

async function deleteMedia(_request,response,id) { const media=mediaRegistry.get(id); if (!media) return send(response,404,{error:'Archivo no encontrado en esta sesión.'}); mediaRegistry.delete(id); await rm(media.path,{force:true}); return send(response,200,{deleted:id}); }

async function probeVideo(path) {
  try { const output=await execFilePromise('ffprobe',['-v','error','-select_streams','v:0','-show_entries','stream=width,height,codec_name:format=duration','-of','json',path],15000); const data=JSON.parse(output); const stream=data.streams?.[0]||{}; return { duration:Number(data.format?.duration)||null,width:Number(stream.width)||null,height:Number(stream.height)||null,codec:stream.codec_name||null }; }
  catch { return {duration:null,width:null,height:null,codec:null}; }
}

async function serveStatic(request,response,url) { const pathname=decodeURIComponent(url.pathname); const requestedPath=pathname==='/'?'index.html':pathname.replace(/^\/+/, ''); const path=resolve(publicRoot,normalize(requestedPath)); if (relative(publicRoot,path).startsWith('..')) return send(response,403,'Ruta no permitida.','text/plain; charset=utf-8'); const info=await stat(path).catch(()=>null); if (!info?.isFile()) return send(response,404,'Archivo no encontrado.','text/plain; charset=utf-8'); const data=await readFile(path); security(response); response.statusCode=200; response.setHeader('Content-Type',mimeType(path)); response.setHeader('Content-Length',data.byteLength); response.setHeader('Cache-Control',pathname==='/'?'no-store':'no-cache'); response.end(request.method==='HEAD'?undefined:data); }
function resolveMedia(value={}) { const result={}; for (const role of ['long','short']) { const id=String(value?.[`${role}Id`]||''); if (id) { const item=mediaRegistry.get(id); if (!item||item.role!==role) throw new Error(`El archivo ${role} no está disponible.`); result[role]=item; } } return result; }
function publicPlan(plan) { return { ...plan, targets:plan.targets.map((target)=>({ ...target, credentials:undefined })) }; }
function publicMedia(media) { return { id:media.id,role:media.role,fileName:media.fileName,size:media.size,sha256:media.sha256,mime:media.mime,duration:media.duration,width:media.width,height:media.height,codec:media.codec }; }
function validateTargets(value) { if (!Array.isArray(value)||!value.length||value.length>videoPlatforms.length) throw new Error('Elegí al menos un destino.'); const known=new Set(videoPlatforms.map((item)=>item.id)); const result=[...new Set(value.map(String))]; if (result.some((id)=>!known.has(id))) throw new Error('Destino desconocido.'); return result; }
async function readJson(request,limit) { const chunks=[]; let size=0; for await (const chunk of request) { size+=chunk.length; if (size>limit) throw new Error('La solicitud JSON es demasiado grande.'); chunks.push(chunk); } try { return JSON.parse(Buffer.concat(chunks).toString('utf8')); } catch { throw new Error('JSON inválido.'); } }
function authorized(request) { return sameOrigin(request)&&request.headers['x-solaris-session']===sessionToken; }
function sameOrigin(request) { try { const origin=new URL(String(request.headers.origin||'')); return origin.protocol==='http:'&&['127.0.0.1','localhost'].includes(origin.hostname)&&origin.host===request.headers.host; } catch { return false; } }
function allowedHost(value='') { return ['127.0.0.1','localhost'].includes(value.split(':')[0].toLowerCase()); }
function security(response) { response.setHeader('Content-Security-Policy',"default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data: blob:; media-src blob:; connect-src 'self'; object-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"); response.setHeader('Referrer-Policy','no-referrer'); response.setHeader('X-Content-Type-Options','nosniff'); response.setHeader('X-Frame-Options','DENY'); response.setHeader('Permissions-Policy','camera=(), microphone=(), geolocation=(), payment=()'); }
function send(response,status,body,type='application/json; charset=utf-8') { security(response); const value=typeof body==='string'?body:`${JSON.stringify(body)}\n`; response.statusCode=status; response.setHeader('Content-Type',type); response.setHeader('Content-Length',Buffer.byteLength(value)); response.end(value); }
function mimeType(path) { return ({'.html':'text/html; charset=utf-8','.js':'text/javascript; charset=utf-8','.css':'text/css; charset=utf-8','.map':'application/json; charset=utf-8'})[extname(path).toLowerCase()]||'application/octet-stream'; }
function safeFileName(value) { return basename(value).replace(/[<>:"/\\|?*\x00-\x1F]/g,'_').slice(0,180)||'video.mp4'; }
function safeExtension(value) { const ext=extname(value).toLowerCase(); return /^\.(mp4|mov|m4v|webm|mkv|avi)$/.test(ext)?ext:'.video'; }
function parsePort(value) { const port=Number.parseInt(value,10); if (!Number.isInteger(port)||port<1024||port>65535) throw new Error('SOLARIS_VIDEO_PORT debe estar entre 1024 y 65535.'); return port; }
function openBrowser(url) { const options={detached:true,stdio:'ignore',windowsHide:true}; const child=process.platform==='win32'?spawn('cmd.exe',['/d','/s','/c','start','',url],options):process.platform==='darwin'?spawn('open',[url],options):spawn('xdg-open',[url],options); child.unref(); }
function execFilePromise(command,args,timeout) { return new Promise((resolve,reject)=>execFile(command,args,{windowsHide:true,timeout,maxBuffer:1024*1024},(error,stdout)=>error?reject(error):resolve(stdout))); }
function safeMessage(error) { return String(error?.message||'Error interno.').replace(/Bearer\s+[A-Za-z0-9._~-]+/gi,'Bearer [redactado]').slice(0,1000); }
function formatBytes(value) { return `${(value/1024/1024/1024).toFixed(1)} GiB`; }
