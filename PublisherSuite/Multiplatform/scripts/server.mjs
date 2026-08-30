import { randomBytes } from 'node:crypto';
import { spawn } from 'node:child_process';
import { access, readFile, stat } from 'node:fs/promises';
import { constants } from 'node:fs';
import { dirname, extname, join, normalize, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { buildCanonicalContent } from '../src/core/content.js';
import { platformRegistry, preparePlan } from '../src/core/platforms.js';
import { publishTargets } from '../src/core/publishers.js';

const scriptDirectory = dirname(fileURLToPath(import.meta.url));
const projectRoot = dirname(scriptDirectory);
const publicRoot = resolve(projectRoot, 'dist');
const host = '127.0.0.1';
const portWasExplicitlyConfigured = Boolean(process.env.SOLARIS_PUBLISHER_PORT);
const requestedPort = parsePort(process.env.SOLARIS_PUBLISHER_PORT || '4180');
const shouldOpenBrowser = process.argv.includes('--open') && process.env.SOLARIS_PUBLISHER_NO_OPEN !== '1';
const sessionToken = randomBytes(32).toString('base64url');
let fallbackAttempted = false;

await ensureBuildExists();

const server = (await import('node:http')).createServer(async (request, response) => {
  try {
    if (!isAllowedHost(request.headers.host)) return send(response, 400, { error: 'Host no permitido.' });
    const url = new URL(request.url || '/', `http://${request.headers.host}`);

    if (url.pathname.startsWith('/api/')) {
      await handleApi(request, response, url);
      return;
    }

    if (!['GET', 'HEAD'].includes(request.method || '')) {
      response.setHeader('Allow', 'GET, HEAD');
      return send(response, 405, 'Método no permitido.', 'text/plain; charset=utf-8');
    }
    await serveStatic(request, response, url);
  } catch (error) {
    console.error(error);
    send(response, 500, { error: safeMessage(error) });
  }
});

server.on('error', (error) => {
  if (error.code === 'EADDRINUSE' && !portWasExplicitlyConfigured && !fallbackAttempted) {
    fallbackAttempted = true;
    console.warn(`El puerto ${requestedPort} está ocupado; se elegirá otro puerto local.`);
    server.listen(0, host);
    return;
  }
  if (error.code === 'EADDRINUSE') {
    console.error(`El puerto configurado ${requestedPort} ya está ocupado.`);
    process.exitCode = 1;
    return;
  }
  throw error;
});

server.listen(requestedPort, host, () => {
  const activePort = server.address().port;
  const localUrl = `http://${host}:${activePort}`;
  console.log('');
  console.log('SolarisPKN Multiplatform Publisher está listo.');
  console.log(`Abrí: ${localUrl}`);
  console.log('Los tokens sólo se leen desde variables de entorno y nunca se envían al navegador.');
  console.log('Presioná Ctrl+C para cerrar la sesión local.');
  console.log('');
  if (shouldOpenBrowser) openBrowser(localUrl);
});

async function handleApi(request, response, url) {
  if (request.method === 'GET' && url.pathname === '/api/session') {
    return send(response, 200, {
      sessionToken,
      platforms: platformRegistry.map((item) => ({
        ...item,
        configured: item.credentials.every((name) => Boolean(process.env[name])),
        missingCredentials: item.credentials.filter((name) => !process.env[name])
      }))
    });
  }
  if (request.method !== 'POST' || !['/api/plan', '/api/publish'].includes(url.pathname)) {
    response.setHeader('Allow', 'GET, POST');
    return send(response, 404, { error: 'Endpoint no encontrado.' });
  }
  if (!isSameOrigin(request) || request.headers['x-solaris-session'] !== sessionToken) {
    return send(response, 403, { error: 'Solicitud local no autorizada.' });
  }
  if (!String(request.headers['content-type'] || '').toLowerCase().startsWith('application/json')) {
    return send(response, 415, { error: 'Se requiere application/json.' });
  }

  const body = await readJsonBody(request, 2.5 * 1024 * 1024);
  const input = validateInput(body.input);
  const targets = validateTargets(body.targets);
  const content = buildCanonicalContent(input.source, input.options);
  const plan = preparePlan(content, targets);

  if (url.pathname === '/api/plan') {
    return send(response, 200, publicPlan(plan));
  }
  if (body.confirmation !== 'PUBLICAR') return send(response, 409, { error: 'La confirmación literal PUBLICAR es obligatoria.' });
  if (body.digest !== plan.digest) return send(response, 409, { error: 'El contenido o los destinos cambiaron desde la vista previa. Prepará el plan otra vez.' });
  const publishSelection = validateTargets(body.publishTargets);
  if (publishSelection.some((id) => !targets.includes(id))) return send(response, 409, { error: 'La publicación incluye un destino ajeno al plan confirmado.' });
  const result = await publishTargets(plan, publishSelection);
  return send(response, result.results.every((item) => item.ok) ? 200 : 207, result);
}

async function serveStatic(request, response, url) {
  const pathname = decodeURIComponent(url.pathname);
  const requestedPath = pathname === '/' ? 'index.html' : pathname.replace(/^\/+/, '');
  const filePath = resolve(publicRoot, normalize(requestedPath));
  if (relative(publicRoot, filePath).startsWith('..')) return send(response, 403, 'Ruta no permitida.', 'text/plain; charset=utf-8');
  const fileStat = await stat(filePath).catch(() => null);
  if (!fileStat?.isFile()) return send(response, 404, 'Archivo no encontrado.', 'text/plain; charset=utf-8');
  const data = await readFile(filePath);
  setSecurityHeaders(response);
  response.statusCode = 200;
  response.setHeader('Content-Type', mimeType(filePath));
  response.setHeader('Content-Length', data.byteLength);
  response.setHeader('Cache-Control', pathname === '/' ? 'no-store' : 'no-cache');
  response.end(request.method === 'HEAD' ? undefined : data);
}

function publicPlan(plan) {
  return {
    ...plan,
    content: { ...plan.content, source: undefined, markdown: undefined },
    targets: plan.targets.map((target) => ({ ...target, credentials: undefined }))
  };
}

function validateInput(value) {
  if (!value || typeof value !== 'object') throw new Error('Falta input.');
  if (typeof value.source !== 'string' || !value.source.trim()) throw new Error('El Markdown/MDX está vacío.');
  if (Buffer.byteLength(value.source, 'utf8') > 2 * 1024 * 1024) throw new Error('El Markdown/MDX supera 2 MiB.');
  const options = value.options && typeof value.options === 'object' && !Array.isArray(value.options) ? value.options : {};
  return { source: value.source, options };
}

function validateTargets(value) {
  if (!Array.isArray(value) || value.length === 0 || value.length > platformRegistry.length) throw new Error('Elegí entre 1 y 19 destinos.');
  const known = new Set(platformRegistry.map((item) => item.id));
  const targets = [...new Set(value.map(String))];
  if (targets.some((id) => !known.has(id))) throw new Error('La selección incluye una plataforma desconocida.');
  return targets;
}

async function readJsonBody(request, limit) {
  const chunks = [];
  let size = 0;
  for await (const chunk of request) {
    size += chunk.length;
    if (size > limit) throw new Error('La solicitud supera el límite permitido.');
    chunks.push(chunk);
  }
  try { return JSON.parse(Buffer.concat(chunks).toString('utf8')); }
  catch { throw new Error('El cuerpo JSON no es válido.'); }
}

function isSameOrigin(request) {
  const origin = String(request.headers.origin || '');
  if (!origin) return false;
  try {
    const url = new URL(origin);
    return url.protocol === 'http:' && ['127.0.0.1', 'localhost'].includes(url.hostname) && url.host === request.headers.host;
  } catch { return false; }
}

function openBrowser(url) {
  const options = { detached: true, stdio: 'ignore', windowsHide: true };
  const child = process.platform === 'win32'
    ? spawn('cmd.exe', ['/d', '/s', '/c', 'start', '', url], options)
    : process.platform === 'darwin' ? spawn('open', [url], options) : spawn('xdg-open', [url], options);
  child.unref();
}

function isAllowedHost(value = '') {
  const hostname = value.split(':')[0].toLowerCase();
  return hostname === host || hostname === 'localhost';
}

function setSecurityHeaders(response) {
  response.setHeader('Content-Security-Policy', "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data: blob: https:; connect-src 'self'; object-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'");
  response.setHeader('Referrer-Policy', 'no-referrer');
  response.setHeader('X-Content-Type-Options', 'nosniff');
  response.setHeader('X-Frame-Options', 'DENY');
  response.setHeader('Permissions-Policy', 'camera=(), microphone=(), geolocation=(), payment=()');
  response.setHeader('Cross-Origin-Opener-Policy', 'same-origin');
}

function send(response, statusCode, body, contentType = 'application/json; charset=utf-8') {
  setSecurityHeaders(response);
  const value = typeof body === 'string' ? body : `${JSON.stringify(body)}\n`;
  response.statusCode = statusCode;
  response.setHeader('Content-Type', contentType);
  response.setHeader('Content-Length', Buffer.byteLength(value));
  response.end(value);
}

function mimeType(filePath) {
  return ({ '.html':'text/html; charset=utf-8', '.js':'text/javascript; charset=utf-8', '.css':'text/css; charset=utf-8', '.map':'application/json; charset=utf-8', '.json':'application/json; charset=utf-8', '.svg':'image/svg+xml' })[extname(filePath).toLowerCase()] || 'application/octet-stream';
}

function parsePort(value) {
  const port = Number.parseInt(value, 10);
  if (!Number.isInteger(port) || port < 1024 || port > 65535) throw new Error('SOLARIS_PUBLISHER_PORT debe ser un número entre 1024 y 65535.');
  return port;
}

function safeMessage(error) {
  return String(error?.message || 'Error interno.').replace(/Bearer\s+[A-Za-z0-9._~-]+/gi, 'Bearer [redactado]').slice(0, 1000);
}

async function ensureBuildExists() {
  try { await access(join(publicRoot, 'index.html'), constants.R_OK); }
  catch { console.error('No existe dist/index.html. Ejecutá primero: npm run build'); process.exit(1); }
}
