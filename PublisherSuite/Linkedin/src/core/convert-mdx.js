import { unified } from 'unified';
import remarkParse from 'remark-parse';
import remarkMdx from 'remark-mdx';
import remarkFrontmatter from 'remark-frontmatter';
import remarkGfm from 'remark-gfm';
import { parse as parseYaml } from 'yaml';

export const MAX_SOURCE_BYTES = 2 * 1024 * 1024;

const parser = unified()
  .use(remarkParse)
  .use(remarkMdx)
  .use(remarkFrontmatter, ['yaml'])
  .use(remarkGfm);

const blockTypes = new Set([
  'root',
  'paragraph',
  'heading',
  'blockquote',
  'list',
  'listItem',
  'code',
  'thematicBreak',
  'table',
  'tableRow',
  'tableCell',
  'mdxJsxFlowElement',
  'mdxFlowExpression',
  'mdxjsEsm',
  'yaml'
]);

const blockedMdxElements = new Set([
  'script',
  'style',
  'iframe',
  'object',
  'embed',
  'link',
  'meta',
  'base'
]);

export function convertMdx(source, options = {}) {
  assertSource(source);

  const warnings = [];
  const assets = [];
  let tree;

  try {
    tree = parser.parse(source);
  } catch (error) {
    const detail = error?.position?.start?.line
      ? ` (línea ${error.position.start.line})`
      : '';
    throw new Error(`No se pudo interpretar el MDX${detail}: ${error.message}`);
  }

  const frontmatter = extractFrontmatter(tree, warnings);
  const suppliedMetadata = isPlainObject(options.metadata) ? options.metadata : {};
  const metadata = { ...frontmatter, ...suppliedMetadata };
  const heading = findFirst(tree, (node) => node.type === 'heading');
  const h1 = findFirst(tree, (node) => node.type === 'heading' && node.depth === 1);
  const firstContentNode = (tree.children || []).find((node) => !['yaml', 'toml', 'mdxjsEsm'].includes(node.type));
  const titleHeadingNode = firstContentNode?.type === 'heading' && firstContentNode.depth === 1
    ? firstContentNode
    : null;
  const context = {
    assets,
    baseUrl: normalizeBaseUrl(options.baseUrl, warnings),
    warnings,
    tableCount: 0,
    titleHeadingNode
  };

  const articleHtml = renderHtml(tree, context);
  const plainText = normalizePlainText(renderPlain(tree, { depth: 0, titleHeadingNode }));
  const title = cleanInlineText(
    options.title || metadata.title || inlineText(h1 || heading) || 'Artículo sin título'
  );
  const description = cleanInlineText(
    options.description || metadata.description || findDescription(tree) || ''
  );
  const slug = slugify(options.slug || metadata.id || title);
  const postText = buildPostText({
    title,
    description,
    url: options.articleUrl || '',
    tags: options.tags || metadata.tags || []
  });

  return {
    schemaVersion: 1,
    title,
    description,
    slug,
    metadata,
    articleHtml,
    plainText,
    postText,
    assets,
    warnings: deduplicateWarnings(warnings),
    stats: {
      sourceCharacters: source.length,
      plainTextCharacters: plainText.length,
      words: countWords(plainText),
      headings: countNodes(tree, (node) => node.type === 'heading'),
      links: countNodes(tree, (node) => node.type === 'link'),
      images: assets.length,
      tables: context.tableCount
    }
  };
}

export function buildPostText({ title = '', description = '', url = '', tags = [] } = {}) {
  const normalizedTags = normalizeTags(tags);
  const sections = [cleanInlineText(title), cleanInlineText(description), String(url).trim()]
    .filter(Boolean);

  if (normalizedTags.length > 0) {
    sections.push(normalizedTags.map((tag) => `#${tag}`).join(' '));
  }

  return sections.join('\n\n');
}

export function createArticleDocument(result) {
  const title = escapeHtml(result.title || 'Artículo para LinkedIn');
  return `<!doctype html>
<html lang="es">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>${title}</title>
  <style>
    body { max-width: 760px; margin: 48px auto; padding: 0 24px; color: #1f2328; font: 18px/1.65 Arial, sans-serif; }
    h1, h2, h3, h4, h5, h6 { line-height: 1.25; margin: 1.5em 0 .6em; }
    blockquote { border-left: 4px solid #0a66c2; margin: 1.5em 0; padding: .2em 1em; color: #475569; }
    pre { overflow: auto; padding: 16px; background: #f3f6f8; border-radius: 8px; }
    code { background: #f3f6f8; padding: .12em .3em; border-radius: 4px; }
    table { width: 100%; border-collapse: collapse; }
    th, td { border: 1px solid #cbd5e1; padding: 8px; text-align: left; vertical-align: top; }
    .linkedin-asset-placeholder { border: 2px dashed #0a66c2; border-radius: 8px; padding: 16px; color: #0a66c2; }
    .linkedin-unsupported { background: #fff7ed; border-left: 4px solid #f97316; padding: 8px 12px; }
  </style>
</head>
<body>
  <h1>${title}</h1>
  ${result.articleHtml}
</body>
</html>\n`;
}

function renderHtml(node, context) {
  const children = () => (node.children || []).map((child) => renderHtml(child, context)).join('');

  switch (node.type) {
    case 'root':
      return children();
    case 'text':
      return escapeHtml(node.value);
    case 'paragraph':
      return (node.children || []).every((child) => child.type === 'image')
        ? children()
        : `<p>${children()}</p>`;
    case 'heading': {
      if (node === context.titleHeadingNode) {
        return '';
      }
      const depth = Math.max(1, Math.min(6, node.depth || 2));
      return `<h${depth}>${children()}</h${depth}>`;
    }
    case 'strong':
      return `<strong>${children()}</strong>`;
    case 'emphasis':
      return `<em>${children()}</em>`;
    case 'delete':
      return `<s>${children()}</s>`;
    case 'inlineCode':
      return `<code>${escapeHtml(node.value)}</code>`;
    case 'code': {
      const language = node.lang ? ` data-language="${escapeAttribute(node.lang)}"` : '';
      return `<pre><code${language}>${escapeHtml(node.value)}</code></pre>`;
    }
    case 'blockquote':
      return `<blockquote>${children()}</blockquote>`;
    case 'list': {
      const tag = node.ordered ? 'ol' : 'ul';
      const start = node.ordered && Number.isInteger(node.start) && node.start !== 1
        ? ` start="${node.start}"`
        : '';
      return `<${tag}${start}>${children()}</${tag}>`;
    }
    case 'listItem': {
      const checked = typeof node.checked === 'boolean'
        ? `<span aria-label="${node.checked ? 'completado' : 'pendiente'}">${node.checked ? '☑' : '☐'} </span>`
        : '';
      return `<li>${checked}${children()}</li>`;
    }
    case 'link': {
      const href = safeUrl(node.url, context);
      const title = node.title ? ` title="${escapeAttribute(node.title)}"` : '';
      return `<a href="${escapeAttribute(href)}"${title}>${children()}</a>`;
    }
    case 'image': {
      const index = context.assets.length + 1;
      const src = resolveAssetUrl(node.url, context);
      const alt = cleanInlineText(node.alt || `Imagen ${index}`);
      context.assets.push({ index, source: node.url, resolvedSource: src, alt, title: node.title || '' });
      return `<figure class="linkedin-asset-placeholder" data-linkedin-asset="${index}"><strong>[INSERTAR IMAGEN ${index}]</strong><br>${escapeHtml(alt)}<br><small>${escapeHtml(node.url)}</small></figure>`;
    }
    case 'thematicBreak':
      return '<hr>';
    case 'break':
      return '<br>';
    case 'html':
      addWarning(context, 'raw-html', 'Se omitió HTML crudo para evitar ejecutar contenido inseguro.', node);
      return `<div class="linkedin-unsupported">[HTML omitido]</div>`;
    case 'table':
      context.tableCount += 1;
      addWarning(context, 'table', 'LinkedIn puede degradar las tablas al pegar. Revisá esta tabla en la vista previa final.', node);
      return `<table><tbody>${children()}</tbody></table>`;
    case 'tableRow':
      return `<tr>${children()}</tr>`;
    case 'tableCell':
      return `<td>${children()}</td>`;
    case 'footnoteDefinition':
      return `<section><p><strong>Nota ${escapeHtml(node.identifier || '')}</strong></p>${children()}</section>`;
    case 'footnoteReference':
      return `<sup>[${escapeHtml(node.label || node.identifier || '')}]</sup>`;
    case 'yaml':
    case 'toml':
      return '';
    case 'mdxjsEsm':
      addWarning(context, 'mdx-module', 'Se omitió un import/export de MDX. El conversor nunca ejecuta módulos.', node);
      return '';
    case 'mdxFlowExpression':
    case 'mdxTextExpression':
      addWarning(context, 'mdx-expression', 'Se omitió una expresión JavaScript de MDX.', node);
      return '<span class="linkedin-unsupported">[Expresión MDX omitida]</span>';
    case 'mdxJsxFlowElement':
    case 'mdxJsxTextElement': {
      const name = node.name || 'Fragment';
      if (name === 'br') {
        return '<br>';
      }
      if (blockedMdxElements.has(name.toLowerCase())) {
        addWarning(context, 'blocked-mdx-element', `Se bloqueó por completo el elemento <${name}> y su contenido.`, node);
        return '';
      }
      addWarning(context, 'mdx-component', `El componente <${name}> no se ejecutó; se conservó solamente su contenido textual.`, node);
      const renderedChildren = children();
      return renderedChildren || `<span class="linkedin-unsupported">[Componente MDX: ${escapeHtml(name)}]</span>`;
    }
    case 'linkReference':
    case 'imageReference':
      addWarning(context, 'reference', 'Se encontró una referencia Markdown sin resolver.', node);
      return children() || escapeHtml(node.alt || node.label || node.identifier || '');
    case 'definition':
      return '';
    default:
      addWarning(context, 'unknown-node', `El nodo “${node.type}” no tiene una conversión específica.`, node);
      return children();
  }
}

function renderPlain(node, state) {
  const childText = () => (node.children || []).map((child) => renderPlain(child, state)).join('');

  switch (node.type) {
    case 'root':
      return (node.children || []).map((child) => renderPlain(child, state)).join('\n');
    case 'text':
      return node.value;
    case 'paragraph':
      return `${childText()}\n`;
    case 'heading':
      return node === state.titleHeadingNode ? '' : `${childText()}\n`;
    case 'strong':
    case 'emphasis':
    case 'delete':
    case 'link':
    case 'mdxJsxFlowElement':
    case 'mdxJsxTextElement':
      return childText();
    case 'inlineCode':
      return node.value;
    case 'code':
      return `${node.value}\n`;
    case 'blockquote':
      return childText()
        .split('\n')
        .filter(Boolean)
        .map((line) => `“${line}”`)
        .join('\n') + '\n';
    case 'list':
      return (node.children || []).map((child, index) => {
        const marker = node.ordered ? `${(node.start || 1) + index}.` : '•';
        const value = normalizePlainText(renderPlain(child, { ...state, depth: state.depth + 1 }));
        const indent = '  '.repeat(state.depth);
        return `${indent}${marker} ${value}`;
      }).join('\n') + '\n';
    case 'listItem':
      return childText();
    case 'image':
      return `[IMAGEN: ${node.alt || node.url}]\n`;
    case 'thematicBreak':
      return '──────────\n';
    case 'break':
      return '\n';
    case 'table':
      return (node.children || []).map((row) => renderPlain(row, state)).join('\n') + '\n';
    case 'tableRow':
      return (node.children || []).map((cell) => normalizePlainText(renderPlain(cell, state))).join(' | ');
    case 'tableCell':
      return childText();
    case 'footnoteDefinition':
      return `[${node.identifier}] ${childText()}\n`;
    case 'footnoteReference':
      return `[${node.label || node.identifier}]`;
    case 'html':
    case 'yaml':
    case 'toml':
    case 'mdxjsEsm':
    case 'mdxFlowExpression':
    case 'mdxTextExpression':
    case 'definition':
      return '';
    default:
      return childText();
  }
}

function extractFrontmatter(tree, warnings) {
  const node = findFirst(tree, (candidate) => candidate.type === 'yaml');
  if (!node) {
    return {};
  }

  try {
    const value = parseYaml(node.value);
    return isPlainObject(value) ? value : {};
  } catch (error) {
    warnings.push({
      code: 'frontmatter',
      message: `No se pudo interpretar el frontmatter YAML: ${error.message}`,
      line: node.position?.start?.line || null
    });
    return {};
  }
}

function findDescription(tree) {
  let result = '';
  visit(tree, (node, parents) => {
    if (result || node.type !== 'paragraph') {
      return;
    }
    if (parents.some((parent) => parent.type === 'blockquote' || parent.type === 'listItem')) {
      return;
    }
    result = inlineText(node);
  });
  return result;
}

function inlineText(node) {
  if (!node) {
    return '';
  }
  if (typeof node.value === 'string' && ['text', 'inlineCode'].includes(node.type)) {
    return node.value;
  }
  if (node.type === 'image') {
    return node.alt || '';
  }
  return (node.children || []).map(inlineText).join('');
}

function safeUrl(rawUrl, context) {
  const value = String(rawUrl || '').trim();
  if (!value) {
    return '#';
  }
  if (value.startsWith('#')) {
    return value;
  }

  try {
    if (/^[a-z][a-z\d+.-]*:/i.test(value)) {
      const url = new URL(value);
      if (!['http:', 'https:', 'mailto:'].includes(url.protocol)) {
        throw new Error('protocolo no permitido');
      }
      return url.href;
    }
    if (context.baseUrl) {
      return new URL(value, context.baseUrl).href;
    }
    addWarning(context, 'relative-link', `El enlace relativo “${value}” necesita una URL base.`, null);
    return value;
  } catch {
    addWarning(context, 'unsafe-link', `Se bloqueó el enlace inseguro o inválido “${value}”.`, null);
    return '#';
  }
}

function resolveAssetUrl(rawUrl, context) {
  const value = String(rawUrl || '').trim();
  if (!context.baseUrl || /^[a-z][a-z\d+.-]*:/i.test(value)) {
    return value;
  }
  try {
    return new URL(value, context.baseUrl).href;
  } catch {
    return value;
  }
}

function normalizeBaseUrl(baseUrl, warnings) {
  if (!baseUrl) {
    return '';
  }
  try {
    const url = new URL(baseUrl);
    if (!['http:', 'https:'].includes(url.protocol)) {
      throw new Error('protocolo inválido');
    }
    return url.href;
  } catch {
    warnings.push({ code: 'base-url', message: 'La URL base no es válida; no se resolverán enlaces relativos.', line: null });
    return '';
  }
}

function normalizeTags(tags) {
  const values = Array.isArray(tags)
    ? tags
    : String(tags || '').split(',');
  return [...new Set(values
    .map((tag) => String(tag).trim().replace(/^#+/, ''))
    .map((tag) => tag
      .split(/\s+/)
      .filter(Boolean)
      .map((part, index) => index === 0 ? part : part.charAt(0).toUpperCase() + part.slice(1))
      .join(''))
    .filter(Boolean))]
    .slice(0, 8);
}

function addWarning(context, code, message, node) {
  context.warnings.push({ code, message, line: node?.position?.start?.line || null });
}

function deduplicateWarnings(warnings) {
  const seen = new Set();
  return warnings.filter((warning) => {
    const key = `${warning.code}|${warning.message}|${warning.line || ''}`;
    if (seen.has(key)) {
      return false;
    }
    seen.add(key);
    return true;
  });
}

function countWords(text) {
  return (text.match(/[\p{L}\p{N}]+(?:['’_-][\p{L}\p{N}]+)*/gu) || []).length;
}

function countNodes(tree, predicate) {
  let count = 0;
  visit(tree, (node) => {
    if (predicate(node)) {
      count += 1;
    }
  });
  return count;
}

function findFirst(tree, predicate) {
  let match = null;
  visit(tree, (node) => {
    if (!match && predicate(node)) {
      match = node;
    }
  });
  return match;
}

function visit(node, callback, parents = []) {
  callback(node, parents);
  for (const child of node.children || []) {
    visit(child, callback, [...parents, node]);
  }
}

function normalizePlainText(value) {
  return String(value || '')
    .replace(/[ \t]+\n/g, '\n')
    .replace(/\n{3,}/g, '\n\n')
    .trim();
}

function cleanInlineText(value) {
  return String(value || '').replace(/\s+/g, ' ').trim();
}

function slugify(value) {
  return String(value || 'articulo')
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 100) || 'articulo';
}

function assertSource(source) {
  if (typeof source !== 'string') {
    throw new TypeError('El contenido MDX debe ser texto.');
  }
  const size = new TextEncoder().encode(source).byteLength;
  if (size > MAX_SOURCE_BYTES) {
    throw new Error(`El archivo supera el límite de ${MAX_SOURCE_BYTES / 1024 / 1024} MiB.`);
  }
}

function isPlainObject(value) {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value);
}

function escapeHtml(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

function escapeAttribute(value) {
  return escapeHtml(value).replaceAll('`', '&#96;');
}

export const __testing = { blockTypes, escapeHtml, normalizePlainText, slugify };
