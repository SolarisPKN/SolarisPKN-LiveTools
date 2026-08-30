import { createHash } from 'node:crypto';
import { convertMdx } from '@solarispkn/linkedin-publisher/src/core/convert-mdx.js';

const DEFAULT_TAGS = ['SolarisPKN'];

export function buildCanonicalContent(source, options = {}) {
  const metadata = isObject(options.metadata) ? options.metadata : {};
  const suppliedTags = options.tags ?? metadata.tags;
  const canonicalUrl = cleanUrl(options.canonicalUrl ?? metadata.canonicalUrl ?? metadata.url ?? '');
  const coverImage = cleanUrl(options.coverImage ?? metadata.coverImage ?? metadata.image ?? '');
  const language = normalizeLanguage(options.language ?? metadata.language ?? metadata.lang ?? 'es');
  const converted = convertMdx(source, {
    metadata,
    title: options.title,
    description: options.description,
    baseUrl: options.baseUrl,
    articleUrl: canonicalUrl,
    tags: suppliedTags
  });
  const tags = normalizeTags(suppliedTags ?? converted.metadata.tags ?? DEFAULT_TAGS);
  const firstRemoteAsset = converted.assets.find((asset) => /^https:\/\//i.test(asset.source || ''))?.source || '';

  return {
    schemaVersion: 1,
    source,
    slug: converted.slug,
    title: converted.title,
    description: converted.description,
    markdown: stripExecutableMdx(source),
    html: converted.articleHtml,
    plainText: converted.plainText,
    canonicalUrl,
    coverImage: coverImage || firstRemoteAsset,
    language,
    tags,
    assets: converted.assets,
    warnings: converted.warnings,
    stats: converted.stats
  };
}

export function contentDigest(content, targetIds) {
  const stable = JSON.stringify({
    schemaVersion: content.schemaVersion,
    title: content.title,
    markdown: content.markdown,
    canonicalUrl: content.canonicalUrl,
    coverImage: content.coverImage,
    language: content.language,
    tags: content.tags,
    targets: [...targetIds].sort()
  });
  return createHash('sha256').update(stable).digest('hex');
}

export function buildSocialText(content, limit, options = {}) {
  const includeDescription = options.includeDescription !== false;
  const hashtagLimit = Number.isInteger(options.hashtagLimit) ? options.hashtagLimit : 5;
  const tags = content.tags.slice(0, hashtagLimit).map((tag) => `#${tag}`);
  const fixedSections = [content.canonicalUrl, tags.join(' ')].filter(Boolean);
  const narrative = [content.title, includeDescription ? content.description : ''].filter(Boolean).join('\n\n');
  const fixedLength = fixedSections.join('\n\n').length;
  const narrativeLimit = Math.max(0, limit - fixedLength - (fixedSections.length ? 2 : 0));
  const shortened = truncateAtBoundary(narrative, narrativeLimit);
  return [shortened, ...fixedSections].filter(Boolean).join('\n\n').slice(0, limit);
}

export function truncateAtBoundary(value, limit) {
  const text = String(value || '').trim();
  if (text.length <= limit) return text;
  if (limit <= 1) return text.slice(0, Math.max(0, limit));
  const candidate = text.slice(0, limit - 1);
  const boundary = Math.max(candidate.lastIndexOf(' '), candidate.lastIndexOf('\n'));
  return `${candidate.slice(0, boundary > limit * 0.65 ? boundary : candidate.length).trimEnd()}…`;
}

export function stripExecutableMdx(source) {
  return String(source)
    .replace(/^\s*(?:import|export)\s+.*$/gm, '')
    .replace(/<([A-Z][\w.]*)\b[^>]*\/?>(?:[\s\S]*?<\/\1>)?/g, '')
    .replace(/\{[^{}]*\}/g, '')
    .trim();
}

function normalizeTags(value) {
  const entries = Array.isArray(value) ? value : String(value || '').split(',');
  return [...new Set(entries.map((tag) => String(tag).trim().replace(/^#/, '').replace(/\s+/g, '')).filter(Boolean))].slice(0, 8);
}

function normalizeLanguage(value) {
  const language = String(value || 'es').trim();
  return /^[a-z]{2,3}(?:-[A-Za-z0-9]{2,8})?$/.test(language) ? language : 'es';
}

function cleanUrl(value) {
  const input = String(value || '').trim();
  if (!input) return '';
  try {
    const url = new URL(input);
    return ['http:', 'https:'].includes(url.protocol) ? url.href : '';
  } catch {
    return '';
  }
}

function isObject(value) {
  return value !== null && typeof value === 'object' && !Array.isArray(value);
}
