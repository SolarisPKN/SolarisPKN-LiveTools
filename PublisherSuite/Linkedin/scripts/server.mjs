import { createServer } from 'node:http';
import { spawn } from 'node:child_process';
import { access, readFile, stat } from 'node:fs/promises';
import { constants } from 'node:fs';
import { dirname, extname, join, normalize, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const scriptDirectory = dirname(fileURLToPath(import.meta.url));
const projectRoot = dirname(scriptDirectory);
const publicRoot = resolve(projectRoot, 'dist');
const host = '127.0.0.1';
const portWasExplicitlyConfigured = Boolean(process.env.SOLARIS_LINKEDIN_PORT);
const requestedPort = parsePort(process.env.SOLARIS_LINKEDIN_PORT || '4179');
const shouldOpenBrowser = process.argv.includes('--open') && process.env.SOLARIS_LINKEDIN_NO_OPEN !== '1';
let fallbackAttempted = false;

await ensureBuildExists();

const server = createServer(async (request, response) => {
  try {
    if (!isAllowedHost(request.headers.host)) {
      send(response, 400, 'Host no permitido.', 'text/plain; charset=utf-8');
      return;
    }

    if (!['GET', 'HEAD'].includes(request.method || '')) {
      response.setHeader('Allow', 'GET, HEAD');
      send(response, 405, 'Método no permitido.', 'text/plain; charset=utf-8');
      return;
    }

    const url = new URL(request.url || '/', `http://${request.headers.host}`);
    const pathname = decodeURIComponent(url.pathname);
    const requestedPath = pathname === '/' ? 'index.html' : pathname.replace(/^\/+/, '');
    const filePath = resolve(publicRoot, normalize(requestedPath));

    if (relative(publicRoot, filePath).startsWith('..')) {
      send(response, 403, 'Ruta no permitida.', 'text/plain; charset=utf-8');
      return;
    }

    const fileStat = await stat(filePath).catch(() => null);
    if (!fileStat?.isFile()) {
      send(response, 404, 'Archivo no encontrado.', 'text/plain; charset=utf-8');
      return;
    }

    const data = await readFile(filePath);
    setSecurityHeaders(response);
    response.statusCode = 200;
    response.setHeader('Content-Type', mimeType(filePath));
    response.setHeader('Content-Length', data.byteLength);
    response.setHeader('Cache-Control', pathname === '/' ? 'no-store' : 'no-cache');
    response.end(request.method === 'HEAD' ? undefined : data);
  } catch (error) {
    console.error(error);
    send(response, 500, 'Error interno del servidor local.', 'text/plain; charset=utf-8');
  }
});

server.on('error', (error) => {
  if (error.code === 'EADDRINUSE' && !portWasExplicitlyConfigured && !fallbackAttempted) {
    fallbackAttempted = true;
    console.warn(`El puerto ${requestedPort} está ocupado; se elegirá automáticamente otro puerto local.`);
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
  console.log('SolarisPKN LinkedIn Publisher está listo.');
  console.log(`Abrí: ${localUrl}`);
  console.log('La sesión escucha únicamente en esta computadora. Presioná Ctrl+C para cerrarla.');
  console.log('');
  if (shouldOpenBrowser) {
    openBrowser(localUrl);
  }
});

function openBrowser(url) {
  const options = { detached: true, stdio: 'ignore', windowsHide: true };
  const child = process.platform === 'win32'
    ? spawn('cmd.exe', ['/d', '/s', '/c', 'start', '', url], options)
    : process.platform === 'darwin'
      ? spawn('open', [url], options)
      : spawn('xdg-open', [url], options);
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

function send(response, statusCode, body, contentType) {
  setSecurityHeaders(response);
  response.statusCode = statusCode;
  response.setHeader('Content-Type', contentType);
  response.setHeader('Content-Length', Buffer.byteLength(body));
  response.end(body);
}

function mimeType(filePath) {
  return ({
    '.html': 'text/html; charset=utf-8',
    '.js': 'text/javascript; charset=utf-8',
    '.css': 'text/css; charset=utf-8',
    '.map': 'application/json; charset=utf-8',
    '.json': 'application/json; charset=utf-8',
    '.svg': 'image/svg+xml'
  })[extname(filePath).toLowerCase()] || 'application/octet-stream';
}

function parsePort(value) {
  const port = Number.parseInt(value, 10);
  if (!Number.isInteger(port) || port < 1024 || port > 65535) {
    throw new Error('SOLARIS_LINKEDIN_PORT debe ser un número entre 1024 y 65535.');
  }
  return port;
}

async function ensureBuildExists() {
  try {
    await access(join(publicRoot, 'index.html'), constants.R_OK);
  } catch {
    console.error('No existe dist/index.html. Ejecutá primero: npm run build');
    process.exit(1);
  }
}
