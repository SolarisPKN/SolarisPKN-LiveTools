import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';

const catalog = JSON.parse(readFileSync(new URL('../../config/video-platforms.json', import.meta.url), 'utf8'));

const credentialMap = {
  'youtube-long':['SOLARIS_YOUTUBE_ACCESS_TOKEN'],
  'youtube-short':['SOLARIS_YOUTUBE_ACCESS_TOKEN'],
  'facebook-reel':['SOLARIS_FACEBOOK_PAGE_TOKEN'],
  'instagram-reel':['SOLARIS_INSTAGRAM_ACCESS_TOKEN','SOLARIS_INSTAGRAM_USER_ID'],
  'tiktok':['SOLARIS_TIKTOK_ACCESS_TOKEN'],
  'threads-video':['SOLARIS_THREADS_ACCESS_TOKEN','SOLARIS_THREADS_USER_ID'],
  'vimeo-long':['SOLARIS_VIMEO_ACCESS_TOKEN'],
  'linkedin-video':['SOLARIS_LINKEDIN_ACCESS_TOKEN','SOLARIS_LINKEDIN_AUTHOR_URN']
};

export const videoPlatforms = catalog.destinations.map((item) => ({ ...item, credentials:credentialMap[item.id] || [] }));

export function publicPlatformStatus(env = process.env) {
  return videoPlatforms.map((platform) => {
    const missingCredentials = platform.credentials.filter((name) => !env[name]);
    return { ...platform, configured:missingCredentials.length===0, missingCredentials };
  });
}

export function buildVideoPlan(input, mediaByRole, targets, env = process.env) {
  const metadata = normalizeInput(input);
  const uniqueTargets = [...new Set(targets.map(String))];
  const known = new Set(videoPlatforms.map((item) => item.id));
  if (!uniqueTargets.length || uniqueTargets.some((id) => !known.has(id))) throw new Error('La selección de destinos no es válida.');
  const long = mediaByRole.long || null;
  const short = mediaByRole.short || null;
  const plans = uniqueTargets.map((id) => prepareTarget(videoPlatforms.find((item) => item.id===id), metadata, { long, short }, env));
  const digestInput = {
    metadata,
    media:{ long:long ? fingerprint(long) : null, short:short ? fingerprint(short) : null },
    targets:uniqueTargets.slice().sort()
  };
  const digest = createHash('sha256').update(JSON.stringify(digestInput)).digest('hex');
  return { schemaVersion:1, createdAt:new Date().toISOString(), digest, digestShort:digest.slice(0,12), metadata, media:{ long:publicMedia(long), short:publicMedia(short) }, targets:plans };
}

function prepareTarget(platform, metadata, media, env) {
  const selectedMedia = media[platform.role];
  const missingCredentials = platform.credentials.filter((name) => !env[name]);
  const warnings = [];
  let limitation = '';
  let payload = {};

  if (!selectedMedia) warnings.push(`Falta el archivo ${platform.role === 'short' ? 'vertical corto' : 'de video largo'}.`);
  if (platform.role==='short' && selectedMedia) {
    if (selectedMedia.duration && selectedMedia.duration > 180) warnings.push('El vertical supera 3 minutos y YouTube no lo clasificará como Short.');
    if (selectedMedia.width && selectedMedia.height && selectedMedia.height < selectedMedia.width) warnings.push('El archivo corto no es vertical ni cuadrado.');
  }
  if (platform.id==='facebook-reel' && selectedMedia) {
    if (selectedMedia.duration && (selectedMedia.duration < 4 || selectedMedia.duration > 60)) warnings.push('Facebook Reels API requiere entre 4 y 60 segundos para este flujo.');
    if (selectedMedia.width && selectedMedia.height && (selectedMedia.width < 540 || selectedMedia.height < 960)) warnings.push('Facebook Reels API requiere al menos 540×960.');
    if (selectedMedia.width && selectedMedia.height && Math.abs(selectedMedia.width/selectedMedia.height-9/16)>0.02) warnings.push('Facebook Reels API requiere relación 9:16 para este flujo.');
  }
  if (platform.id==='instagram-reel' && selectedMedia) {
    if (selectedMedia.duration && (selectedMedia.duration < 3 || selectedMedia.duration > 900)) warnings.push('Instagram Reels requiere entre 3 segundos y 15 minutos.');
    if (selectedMedia.size > 1024*1024*1024) warnings.push('Instagram Reels admite hasta 1 GB en este flujo.');
  }
  if (platform.id==='linkedin-video' && selectedMedia) {
    if (selectedMedia.duration && (selectedMedia.duration < 3 || selectedMedia.duration > 1800)) warnings.push('LinkedIn requiere entre 3 segundos y 30 minutos.');
    if (selectedMedia.size < 75*1024 || selectedMedia.size > 500*1024*1024) warnings.push('LinkedIn requiere un MP4 de 75 KB a 500 MB.');
    if (!/\.mp4$/i.test(selectedMedia.fileName)) warnings.push('LinkedIn requiere formato MP4 para este flujo.');
  }
  if (platform.id==='instagram-reel' && !metadata.publicShortUrl) warnings.push('Instagram necesita una URL HTTPS pública del archivo vertical.');
  if (platform.id==='threads-video' && !metadata.publicShortUrl) warnings.push('Threads necesita una URL HTTPS pública del archivo vertical.');
  if (platform.id==='pinterest-video' && !metadata.coverImageUrl) warnings.push('Pinterest requiere una portada pública además del video.');

  const longDescription = withLink(metadata.longDescription, metadata.canonicalUrl);
  const shortCaption = withTags(withLink(metadata.shortCaption || firstSentence(metadata.longDescription), metadata.canonicalUrl), metadata.hashtags);
  switch (platform.id) {
    case 'youtube-long': payload={ title:cut(metadata.title,100), description:cut(withTags(longDescription,metadata.hashtags),5000), tags:metadata.tags, categoryId:metadata.youtubeCategoryId, privacyStatus:metadata.visibility, madeForKids:metadata.madeForKids, containsSyntheticMedia:metadata.containsSyntheticMedia }; break;
    case 'youtube-short': payload={ title:cut(metadata.shortTitle || metadata.title,100), description:cut(shortCaption,5000), tags:metadata.tags, categoryId:metadata.youtubeCategoryId, privacyStatus:metadata.visibility, madeForKids:metadata.madeForKids, containsSyntheticMedia:metadata.containsSyntheticMedia }; break;
    case 'facebook-reel': payload={ title:cut(metadata.shortTitle || metadata.title,255), description:cut(shortCaption,2200), videoState:metadata.visibility==='public'?'PUBLISHED':'DRAFT' }; break;
    case 'instagram-reel': payload={ caption:cut(shortCaption,2200), videoUrl:metadata.publicShortUrl, shareToFeed:true }; break;
    case 'tiktok': payload={ title:cut(shortCaption,2200), privacyLevel:metadata.visibility==='public'?'PUBLIC_TO_EVERYONE':'SELF_ONLY', disableComment:false, disableDuet:false, disableStitch:false }; break;
    case 'threads-video': payload={ text:cut(shortCaption,500), videoUrl:metadata.publicShortUrl, altText:cut(metadata.altText,1000) }; break;
    case 'vimeo-long': payload={ name:cut(metadata.title,128), description:cut(longDescription,5000), privacy:metadata.visibility==='public'?'anybody':'nobody' }; break;
    case 'linkedin-video': payload={ commentary:cut(shortCaption,3000), title:cut(metadata.shortTitle || metadata.title,200), visibility:'PUBLIC' }; break;
    default: payload={ title:metadata.title, shortTitle:metadata.shortTitle, longDescription, shortCaption, tags:metadata.tags, canonicalUrl:metadata.canonicalUrl };
  }

  const automatic = platform.mode==='api-local' || platform.mode==='api-url';
  if (!automatic) limitation = limitationFor(platform.id, platform.mode);
  const hardWarnings = warnings.filter((warning) => /Falta|necesita|requiere|supera|máximo|admite hasta|no es vertical/.test(warning));
  return { ...platform, missingCredentials, warnings, limitation, payload, mediaId:selectedMedia?.id || null, canPublish:automatic && missingCredentials.length===0 && hardWarnings.length===0 };
}

function normalizeInput(value={}) {
  const title=clean(value.title,200); if (!title) throw new Error('El título es obligatorio.');
  const longDescription=clean(value.longDescription,12000); if (!longDescription) throw new Error('La descripción larga es obligatoria.');
  return {
    title, shortTitle:clean(value.shortTitle,200), longDescription, shortCaption:clean(value.shortCaption,4000),
    canonicalUrl:httpsUrl(value.canonicalUrl,'URL canónica',true), publicShortUrl:httpsUrl(value.publicShortUrl,'URL pública del short',true), coverImageUrl:httpsUrl(value.coverImageUrl,'Portada pública',true),
    altText:clean(value.altText,1500), hashtags:normalizeTags(value.hashtags,20,true), tags:normalizeTags(value.tags,50,false),
    language:clean(value.language,20)||'es', visibility:['public','private','unlisted'].includes(value.visibility)?value.visibility:'private',
    youtubeCategoryId:/^\d+$/.test(String(value.youtubeCategoryId||''))?String(value.youtubeCategoryId):'28',
    madeForKids:Boolean(value.madeForKids), containsSyntheticMedia:Boolean(value.containsSyntheticMedia)
  };
}

function publicMedia(media) { return media ? { id:media.id, role:media.role, fileName:media.fileName, size:media.size, sha256:media.sha256, mime:media.mime, duration:media.duration, width:media.width, height:media.height, codec:media.codec } : null; }
function fingerprint(media) { return { id:media.id, size:media.size, sha256:media.sha256, duration:media.duration, width:media.width, height:media.height }; }
function clean(value,max) { return String(value||'').replace(/\r\n?/g,'\n').trim().slice(0,max); }
function cut(value,max) { const text=String(value||''); return text.length<=max?text:`${text.slice(0,Math.max(0,max-1)).trimEnd()}…`; }
function firstSentence(value) { return String(value||'').split(/(?<=[.!?])\s+/)[0] || ''; }
function withLink(value,url) { return [String(value||'').trim(),url].filter(Boolean).join('\n\n'); }
function withTags(value,tags) { return [String(value||'').trim(),tags.map((tag)=>`#${tag}`).join(' ')].filter(Boolean).join('\n\n'); }
function normalizeTags(value,max,stripHash) { const source=Array.isArray(value)?value:String(value||'').split(','); return [...new Set(source.map((item)=>String(item).trim().replace(stripHash?/^#+/:'','').replace(/\s+/g,stripHash?'':' ').replace(stripHash?/[^\p{L}\p{N}_]/gu:/[<>]/g,'')).filter(Boolean))].slice(0,max); }
function httpsUrl(value,label,optional) { const text=String(value||'').trim(); if (!text && optional) return ''; let url; try { url=new URL(text); } catch { throw new Error(`${label} no es válida.`); } if (url.protocol!=='https:') throw new Error(`${label} debe usar HTTPS.`); return url.href; }
function limitationFor(id,mode) {
  const specific={ dailymotion:'La API v2 admite upload, pero el contrato de creación/perfil está en transición y queda asistido hasta validar una cuenta real.', twitch:'La Helix API pública consulta videos; no ofrece carga de videos pregrabados.', kick:'La API pública de Kick expone canales, directos y chat, no upload de videos.', rumble:'No se verificó un endpoint público oficial de upload para cuentas personales.', nimo:'Sólo se encontró SDK de reproducción/broadcast, no publicación de VOD.', soop:'El SDK público encontrado es para extensiones y chat, no carga de VOD.', imdb:'IMDb no es un host editorial con API pública de publicación para perfiles de usuario.', 'mastodon-video':'Contrato documentado; conector binario pendiente para respetar límites de la instancia.', 'bluesky-video':'La carga usa un servicio de video asíncrono y necesita validar límites actuales de la cuenta.', 'pinterest-video':'Contrato documentado; falta implementar multipart S3 y portada.', 'flickr-video':'La API de upload existe, pero usa OAuth 1.0a y límites de cuenta; queda asistida.', 'pixelfed-video':'Depende de la instancia y de su compatibilidad real con la API de Mastodon.', 'x-video':'La escritura y carga de media en X es paga por uso.' };
  return specific[id] || `Modo ${mode}: preparación de metadatos sin publicación automática.`;
}
