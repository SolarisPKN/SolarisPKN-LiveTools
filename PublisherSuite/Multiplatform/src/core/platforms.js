import { buildSocialText, contentDigest, truncateAtBoundary } from './content.js';

export const platformRegistry = [
  platform('solarispkn-blog', 'SolarisPKN Labs Blog', 'bundle', ['full article', 'Labs-compatible bundle']),
  platform('devto', 'DEV Community', 'api', ['Markdown', 'canonical URL'], ['SOLARIS_DEVTO_API_KEY']),
  platform('medium', 'Medium', 'assisted', ['Markdown', 'canonical URL'], [], 'La API oficial no acepta integraciones nuevas y está sin soporte.'),
  platform('hashnode', 'Hashnode', 'assisted', ['Markdown'], [], 'La mutación publishPost requiere una publicación Pro.'),
  platform('blogger', 'Blogger', 'api', ['HTML', 'draft/public'], ['SOLARIS_BLOGGER_ACCESS_TOKEN', 'SOLARIS_BLOGGER_BLOG_ID']),
  platform('tumblr', 'Tumblr', 'api', ['NPF text blocks'], ['SOLARIS_TUMBLR_ACCESS_TOKEN', 'SOLARIS_TUMBLR_BLOG_ID']),
  platform('linkedin', 'LinkedIn', 'api', ['text post', 'link card'], ['SOLARIS_LINKEDIN_ACCESS_TOKEN', 'SOLARIS_LINKEDIN_AUTHOR_URN']),
  platform('facebook', 'Facebook Page', 'api', ['Page feed post'], ['SOLARIS_FACEBOOK_PAGE_TOKEN', 'SOLARIS_FACEBOOK_PAGE_ID']),
  platform('instagram', 'Instagram', 'api-media', ['caption', 'image'], ['SOLARIS_INSTAGRAM_ACCESS_TOKEN', 'SOLARIS_INSTAGRAM_USER_ID'], 'Requiere cuenta profesional y una imagen HTTPS pública.'),
  platform('threads', 'Threads', 'api', ['text post'], ['SOLARIS_THREADS_ACCESS_TOKEN', 'SOLARIS_THREADS_USER_ID']),
  platform('mastodon', 'Mastodon', 'api', ['status', 'idempotency'], ['SOLARIS_MASTODON_ACCESS_TOKEN', 'SOLARIS_MASTODON_BASE_URL']),
  platform('bluesky', 'Bluesky', 'api', ['post', 'language'], ['SOLARIS_BLUESKY_HANDLE', 'SOLARIS_BLUESKY_APP_PASSWORD']),
  platform('reddit', 'Reddit', 'api', ['self post'], ['SOLARIS_REDDIT_ACCESS_TOKEN', 'SOLARIS_REDDIT_SUBREDDIT']),
  platform('pinterest', 'Pinterest', 'api-media', ['pin', 'image'], ['SOLARIS_PINTEREST_ACCESS_TOKEN', 'SOLARIS_PINTEREST_BOARD_ID'], 'Requiere una imagen HTTPS pública.'),
  platform('discord', 'Discord', 'api', ['webhook message'], ['SOLARIS_DISCORD_WEBHOOK_URL']),
  platform('x', 'X', 'assisted', ['text'], [], 'La API oficial de escritura es paga por uso.'),
  platform('youtube', 'YouTube', 'media-only', ['video metadata'], [], 'El Markdown sólo puede preparar título y descripción; falta un archivo de video y OAuth auditado.'),
  platform('tiktok', 'TikTok', 'media-only', ['caption'], [], 'La publicación requiere foto/video y una aplicación auditada para contenido público.'),
  platform('vimeo', 'Vimeo', 'media-only', ['video metadata'], [], 'La publicación requiere video; las cuentas Free necesitan aprobación de upload API.')
];

export function preparePlan(content, requestedIds, env = process.env) {
  const ids = requestedIds?.length ? [...new Set(requestedIds)] : platformRegistry.map((item) => item.id);
  const targets = ids.map((id) => {
    const definition = platformRegistry.find((item) => item.id === id);
    if (!definition) throw new Error(`Plataforma desconocida: ${id}`);
    return prepareTarget(definition, content, env);
  });
  const digest = contentDigest(content, ids);
  return { schemaVersion: 1, digest, digestShort: digest.slice(0, 12), content, targets };
}

export function prepareTarget(definition, content, env = process.env) {
  const missingCredentials = definition.credentials.filter((name) => !env[name]);
  const warnings = [];
  let payload;

  switch (definition.id) {
    case 'solarispkn-blog':
      payload = {
        files: [`src/content/blog/${content.slug}/index-${content.language.split('-')[0]}.mdx`, `src/content/blog/${content.slug}/post.json`],
        markdown: content.markdown
      };
      break;
    case 'devto':
      payload = { title: content.title, body_markdown: content.markdown, published: true, canonical_url: content.canonicalUrl || undefined, description: content.description, tags: content.tags.slice(0, 4), main_image: content.coverImage || undefined };
      break;
    case 'medium':
    case 'hashnode':
      payload = { title: content.title, markdown: content.markdown, canonicalUrl: content.canonicalUrl, tags: content.tags };
      break;
    case 'blogger':
      payload = { title: content.title, content: content.html, labels: content.tags };
      break;
    case 'tumblr':
      payload = { state: 'published', tags: content.tags, content: [{ type: 'text', subtype: 'heading1', text: content.title }, { type: 'text', text: truncateAtBoundary(content.plainText, 65000) }] };
      break;
    case 'linkedin':
      payload = { commentary: buildSocialText(content, 3000), article: content.canonicalUrl ? { source: content.canonicalUrl, title: content.title, description: truncateAtBoundary(content.description, 256) } : undefined };
      if (!content.canonicalUrl) warnings.push('Sin URL canónica: se preparará un post de texto, no una tarjeta de enlace.');
      break;
    case 'facebook':
      payload = { message: buildSocialText(content, 5000), link: content.canonicalUrl || undefined };
      break;
    case 'instagram':
      payload = { caption: buildSocialText(content, 2200), image_url: content.coverImage || '' };
      if (!content.coverImage) warnings.push('Falta coverImage HTTPS pública; Instagram no permite publicar sólo texto.');
      break;
    case 'threads':
      payload = { media_type: 'TEXT', text: buildSocialText(content, 500) };
      break;
    case 'mastodon': {
      const configuredLimit = Number.parseInt(env.SOLARIS_MASTODON_CHARACTER_LIMIT || '500', 10);
      payload = { status: buildSocialText(content, Number.isInteger(configuredLimit) ? configuredLimit : 500), visibility: 'public', language: content.language.split('-')[0] };
      break;
    }
    case 'bluesky':
      payload = { text: buildSocialText(content, 300, { hashtagLimit: 3 }), langs: [content.language] };
      break;
    case 'reddit':
      payload = { sr: env.SOLARIS_REDDIT_SUBREDDIT || '', kind: 'self', title: truncateAtBoundary(content.title, 300), text: [content.markdown, content.canonicalUrl ? `\n\nFuente original: ${content.canonicalUrl}` : ''].join('') };
      break;
    case 'pinterest':
      payload = { board_id: env.SOLARIS_PINTEREST_BOARD_ID || '', title: truncateAtBoundary(content.title, 100), description: truncateAtBoundary(content.description || content.plainText, 500), link: content.canonicalUrl || undefined, media_source: { source_type: 'image_url', url: content.coverImage || '' } };
      if (!content.coverImage) warnings.push('Falta coverImage HTTPS pública; Pinterest necesita media para crear el Pin.');
      break;
    case 'discord':
      payload = { content: buildSocialText(content, 2000) };
      break;
    case 'x':
      payload = { text: buildSocialText(content, 280, { hashtagLimit: 2 }) };
      break;
    default:
      payload = { title: truncateAtBoundary(content.title, 100), description: truncateAtBoundary(content.description || content.plainText, 500), mediaRequired: true };
  }

  const mediaBlocked = definition.mode === 'api-media' && !content.coverImage;
  const canPublish = definition.mode === 'api' || definition.mode === 'api-media'
    ? missingCredentials.length === 0 && !mediaBlocked
    : definition.mode === 'bundle';

  return {
    ...definition,
    canPublish,
    missingCredentials,
    warnings,
    payload,
    characterCount: typeof payload.text === 'string' ? payload.text.length : typeof payload.commentary === 'string' ? payload.commentary.length : typeof payload.caption === 'string' ? payload.caption.length : typeof payload.status === 'string' ? payload.status.length : null
  };
}

function platform(id, name, mode, capabilities, credentials = [], limitation = '') {
  return { id, name, mode, capabilities, credentials, limitation };
}
