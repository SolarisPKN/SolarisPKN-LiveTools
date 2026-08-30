import { mkdir, rename, rm, writeFile } from 'node:fs/promises';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const moduleRoot = dirname(dirname(dirname(fileURLToPath(import.meta.url))));
const outputRoot = resolve(moduleRoot, 'output');

export async function publishTargets(plan, targetIds, env = process.env) {
  const selected = [...new Set(targetIds)];
  const results = [];

  for (const id of selected) {
    const target = plan.targets.find((item) => item.id === id);
    if (!target) {
      results.push(failure(id, 'La plataforma no forma parte del plan confirmado.'));
      continue;
    }
    if (!target.canPublish) {
      results.push(failure(id, blockedReason(target)));
      continue;
    }

    try {
      const receipt = await publishOne(target, plan, env);
      results.push({ id, name: target.name, ok: true, ...receipt });
    } catch (error) {
      results.push(failure(id, safeError(error), target.name));
    }
  }

  const receipt = {
    schemaVersion: 1,
    digest: plan.digest,
    createdAt: new Date().toISOString(),
    content: { slug: plan.content.slug, title: plan.content.title, canonicalUrl: plan.content.canonicalUrl },
    results
  };
  const receiptPath = join(outputRoot, plan.digestShort, `${plan.content.slug}.receipt.json`);
  await atomicWrite(receiptPath, `${JSON.stringify(receipt, null, 2)}\n`);
  return { ...receipt, receiptPath };
}

async function publishOne(target, plan, env) {
  switch (target.id) {
    case 'solarispkn-blog': return writeLabsBundle(plan);
    case 'devto': return publishDevto(target.payload, env);
    case 'blogger': return publishBlogger(target.payload, env);
    case 'tumblr': return publishTumblr(target.payload, env);
    case 'linkedin': return publishLinkedin(target.payload, env);
    case 'facebook': return publishFacebook(target.payload, env);
    case 'instagram': return publishInstagram(target.payload, env);
    case 'threads': return publishThreads(target.payload, env);
    case 'mastodon': return publishMastodon(target.payload, plan.digest, env);
    case 'bluesky': return publishBluesky(target.payload, env);
    case 'reddit': return publishReddit(target.payload, env);
    case 'pinterest': return publishPinterest(target.payload, env);
    case 'discord': return publishDiscord(target.payload, env);
    default: throw new Error('No existe un publicador automático oficial para este destino.');
  }
}

async function writeLabsBundle(plan) {
  const targetDirectory = join(outputRoot, plan.digestShort, 'solarispkn-labs', plan.content.slug);
  const locale = plan.content.language.split('-')[0];
  const metadata = {
    id: plan.content.slug,
    title: plan.content.title,
    description: plan.content.description,
    date: new Date().toISOString().slice(0, 10),
    tags: plan.content.tags,
    canonicalUrl: plan.content.canonicalUrl || undefined,
    coverImage: plan.content.coverImage || undefined,
    status: 'draft'
  };
  await atomicWrite(join(targetDirectory, `index-${locale}.mdx`), `${plan.content.markdown.trim()}\n`);
  await atomicWrite(join(targetDirectory, 'post.json'), `${JSON.stringify(metadata, null, 2)}\n`);
  return { status: 'bundle-created', location: targetDirectory, externalId: null, url: null };
}

async function publishDevto(payload, env) {
  const data = await requestJson('DEV', 'https://dev.to/api/articles', {
    method: 'POST',
    headers: { 'api-key': env.SOLARIS_DEVTO_API_KEY, Accept: 'application/vnd.forem.api-v1+json' },
    body: { article: compact(payload) }
  });
  return { status: 'published', externalId: String(data.id || ''), url: data.url || null };
}

async function publishBlogger(payload, env) {
  const blogId = encodeURIComponent(env.SOLARIS_BLOGGER_BLOG_ID);
  const data = await requestJson('Blogger', `https://www.googleapis.com/blogger/v3/blogs/${blogId}/posts?isDraft=false`, {
    method: 'POST', headers: bearer(env.SOLARIS_BLOGGER_ACCESS_TOKEN), body: payload
  });
  return { status: 'published', externalId: data.id || null, url: data.url || null };
}

async function publishTumblr(payload, env) {
  const blogId = encodeURIComponent(env.SOLARIS_TUMBLR_BLOG_ID);
  const data = await requestJson('Tumblr', `https://api.tumblr.com/v2/blog/${blogId}/posts`, {
    method: 'POST', headers: bearer(env.SOLARIS_TUMBLR_ACCESS_TOKEN), body: payload
  });
  const response = data.response || data;
  return { status: 'published', externalId: String(response.id || response.id_string || ''), url: response.post_url || null };
}

async function publishLinkedin(payload, env) {
  const body = {
    author: env.SOLARIS_LINKEDIN_AUTHOR_URN,
    commentary: payload.commentary,
    visibility: 'PUBLIC',
    distribution: { feedDistribution: 'MAIN_FEED', targetEntities: [], thirdPartyDistributionChannels: [] },
    lifecycleState: 'PUBLISHED',
    isReshareDisabledByAuthor: false,
    ...(payload.article ? { content: { article: payload.article } } : {})
  };
  const version = env.SOLARIS_LINKEDIN_VERSION || currentLinkedinVersion();
  const response = await request('LinkedIn', 'https://api.linkedin.com/rest/posts', {
    method: 'POST',
    headers: { ...bearer(env.SOLARIS_LINKEDIN_ACCESS_TOKEN), 'Linkedin-Version': version, 'X-Restli-Protocol-Version': '2.0.0' },
    body
  });
  return { status: 'published', externalId: response.headers.get('x-restli-id'), url: null };
}

async function publishFacebook(payload, env) {
  const version = env.SOLARIS_META_GRAPH_VERSION || 'v25.0';
  const pageId = encodeURIComponent(env.SOLARIS_FACEBOOK_PAGE_ID);
  const data = await requestForm('Facebook', `https://graph.facebook.com/${version}/${pageId}/feed`, { ...compact(payload), access_token: env.SOLARIS_FACEBOOK_PAGE_TOKEN });
  return { status: 'published', externalId: data.id || null, url: null };
}

async function publishInstagram(payload, env) {
  const version = env.SOLARIS_META_GRAPH_VERSION || 'v25.0';
  const userId = encodeURIComponent(env.SOLARIS_INSTAGRAM_USER_ID);
  const container = await requestForm('Instagram', `https://graph.facebook.com/${version}/${userId}/media`, { image_url: payload.image_url, caption: payload.caption, access_token: env.SOLARIS_INSTAGRAM_ACCESS_TOKEN });
  const published = await requestForm('Instagram', `https://graph.facebook.com/${version}/${userId}/media_publish`, { creation_id: container.id, access_token: env.SOLARIS_INSTAGRAM_ACCESS_TOKEN });
  return { status: 'published', externalId: published.id || null, url: null };
}

async function publishThreads(payload, env) {
  const userId = encodeURIComponent(env.SOLARIS_THREADS_USER_ID);
  const base = env.SOLARIS_THREADS_API_BASE || 'https://graph.threads.net/v1.0';
  const container = await requestForm('Threads', `${base}/${userId}/threads`, { ...payload, access_token: env.SOLARIS_THREADS_ACCESS_TOKEN });
  const published = await requestForm('Threads', `${base}/${userId}/threads_publish`, { creation_id: container.id, access_token: env.SOLARIS_THREADS_ACCESS_TOKEN });
  return { status: 'published', externalId: published.id || null, url: null };
}

async function publishMastodon(payload, digest, env) {
  const baseUrl = requireHttpsBase(env.SOLARIS_MASTODON_BASE_URL, 'SOLARIS_MASTODON_BASE_URL');
  const response = await requestJson('Mastodon', `${baseUrl}/api/v1/statuses`, {
    method: 'POST',
    headers: { ...bearer(env.SOLARIS_MASTODON_ACCESS_TOKEN), 'Idempotency-Key': digest },
    body: payload
  });
  return { status: 'published', externalId: response.id || null, url: response.url || response.uri || null };
}

async function publishBluesky(payload, env) {
  const baseUrl = requireHttpsBase(env.SOLARIS_BLUESKY_PDS || 'https://bsky.social', 'SOLARIS_BLUESKY_PDS');
  const session = await requestJson('Bluesky', `${baseUrl}/xrpc/com.atproto.server.createSession`, {
    method: 'POST', body: { identifier: env.SOLARIS_BLUESKY_HANDLE, password: env.SOLARIS_BLUESKY_APP_PASSWORD }
  });
  const record = { $type: 'app.bsky.feed.post', text: payload.text, langs: payload.langs, createdAt: new Date().toISOString() };
  const created = await requestJson('Bluesky', `${baseUrl}/xrpc/com.atproto.repo.createRecord`, {
    method: 'POST', headers: bearer(session.accessJwt), body: { repo: session.did, collection: 'app.bsky.feed.post', record }
  });
  return { status: 'published', externalId: created.uri || null, url: created.uri ? `https://bsky.app/profile/${session.did}/post/${created.uri.split('/').pop()}` : null };
}

async function publishReddit(payload, env) {
  const data = await requestForm('Reddit', 'https://oauth.reddit.com/api/submit', { api_type: 'json', ...payload }, {
    ...bearer(env.SOLARIS_REDDIT_ACCESS_TOKEN),
    'User-Agent': env.SOLARIS_REDDIT_USER_AGENT || 'windows:com.solarispkn.publisher:v0.1.0 (by /u/TecnOtakus)'
  });
  const errors = data.json?.errors || [];
  if (errors.length) throw new Error(`Reddit rechazó la publicación: ${errors.map((item) => item[1] || item[0]).join('; ')}`);
  return { status: 'published', externalId: data.json?.data?.id || null, url: data.json?.data?.url || null };
}

async function publishPinterest(payload, env) {
  const data = await requestJson('Pinterest', 'https://api.pinterest.com/v5/pins', {
    method: 'POST', headers: bearer(env.SOLARIS_PINTEREST_ACCESS_TOKEN), body: payload
  });
  return { status: 'published', externalId: data.id || null, url: data.link || null };
}

async function publishDiscord(payload, env) {
  const webhookUrl = new URL(env.SOLARIS_DISCORD_WEBHOOK_URL);
  if (webhookUrl.protocol !== 'https:' || !/(^|\.)discord(?:app)?\.com$/.test(webhookUrl.hostname)) {
    throw new Error('SOLARIS_DISCORD_WEBHOOK_URL no apunta a un host oficial de Discord.');
  }
  const response = await request('Discord', `${webhookUrl.href}${webhookUrl.search ? '&' : '?'}wait=true`, { method: 'POST', body: payload });
  const data = await response.json().catch(() => ({}));
  return { status: 'published', externalId: data.id || null, url: null };
}

async function requestForm(label, url, values, extraHeaders = {}) {
  const body = new URLSearchParams();
  for (const [key, value] of Object.entries(values)) {
    if (value !== undefined && value !== null && value !== '') body.set(key, typeof value === 'object' ? JSON.stringify(value) : String(value));
  }
  const response = await request(label, url, { method: 'POST', headers: extraHeaders, formBody: body });
  return response.json().catch(() => ({}));
}

async function requestJson(label, url, options) {
  const response = await request(label, url, options);
  return response.json().catch(() => ({}));
}

async function request(label, url, { method = 'GET', headers = {}, body, formBody } = {}) {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 30000);
  let response;
  try {
    response = await fetch(url, {
      method,
      headers: {
        Accept: 'application/json',
        ...(formBody ? { 'Content-Type': 'application/x-www-form-urlencoded' } : body ? { 'Content-Type': 'application/json' } : {}),
        ...headers
      },
      body: formBody ? formBody.toString() : body ? JSON.stringify(body) : undefined,
      signal: controller.signal,
      redirect: 'error'
    });
  } catch (error) {
    if (error.name === 'AbortError') throw new Error(`${label} no respondió dentro de 30 segundos.`);
    throw new Error(`${label} no pudo conectarse con su API oficial.`);
  } finally {
    clearTimeout(timeout);
  }
  if (!response.ok) {
    const detail = await response.text().catch(() => '');
    throw new Error(`${label} respondió HTTP ${response.status}${detail ? `: ${truncateError(detail)}` : '.'}`);
  }
  return response;
}

async function atomicWrite(targetPath, contents) {
  await mkdir(dirname(targetPath), { recursive: true });
  const temporaryPath = `${targetPath}.${process.pid}.${Date.now()}.tmp`;
  try {
    await writeFile(temporaryPath, contents, { encoding: 'utf8', flag: 'wx' });
    try {
      await rename(temporaryPath, targetPath);
    } catch (error) {
      if (!['EEXIST', 'EPERM'].includes(error.code)) throw error;
      await rm(targetPath, { force: true });
      await rename(temporaryPath, targetPath);
    }
  } catch (error) {
    await rm(temporaryPath, { force: true });
    throw error;
  }
}

function bearer(token) {
  return { Authorization: `Bearer ${token}` };
}

function compact(value) {
  return Object.fromEntries(Object.entries(value).filter(([, item]) => item !== undefined && item !== null && item !== ''));
}

function blockedReason(target) {
  if (target.missingCredentials.length) return `Faltan credenciales: ${target.missingCredentials.join(', ')}.`;
  if (target.warnings.length) return target.warnings.join(' ');
  return target.limitation || 'Este destino sólo admite preparación asistida.';
}

function failure(id, error, name = id) {
  return { id, name, ok: false, error };
}

function safeError(error) {
  return String(error?.message || 'Error de publicación.').replace(/Bearer\s+[A-Za-z0-9._~-]+/gi, 'Bearer [redactado]').slice(0, 1000);
}

function truncateError(value) {
  return String(value).replace(/[\r\n]+/g, ' ').slice(0, 500);
}

function requireHttpsBase(value, name) {
  const url = new URL(value);
  if (url.protocol !== 'https:') throw new Error(`${name} debe usar HTTPS.`);
  return url.href.replace(/\/$/, '');
}

function currentLinkedinVersion() {
  const now = new Date();
  return `${now.getUTCFullYear()}${String(now.getUTCMonth() + 1).padStart(2, '0')}`;
}

