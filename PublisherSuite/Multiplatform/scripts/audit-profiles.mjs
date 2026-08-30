import { mkdir, readFile, rename, rm, writeFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const scriptDirectory = dirname(fileURLToPath(import.meta.url));
const projectRoot = dirname(scriptDirectory);
const inventory = JSON.parse(await readFile(join(projectRoot, 'config', 'profiles.json'), 'utf8'));
const profiles = inventory.categories.flatMap((category) => category.profiles.map(([name, url]) => ({ category: category.name, name, url })));

if (profiles.length !== inventory.declaredCount) throw new Error(`El inventario contiene ${profiles.length} perfiles, pero declara ${inventory.declaredCount}.`);

const queue = [...profiles];
const results = [];
const workers = Array.from({ length: 10 }, async () => {
  while (queue.length) {
    const profile = queue.shift();
    results.push({ ...profile, ...(await probe(profile.url)) });
  }
});
await Promise.all(workers);
results.sort((a,b) => a.category.localeCompare(b.category) || a.name.localeCompare(b.name));

const summary = {
  checkedAt: new Date().toISOString(),
  source: inventory.source,
  total: results.length,
  reachable: results.filter((item) => item.classification === 'reachable').length,
  guarded: results.filter((item) => item.classification === 'guarded').length,
  broken: results.filter((item) => item.classification === 'broken').length,
  results
};
const target = join(projectRoot, 'audit-output', 'profiles-latest.json');
await atomicWrite(target, `${JSON.stringify(summary, null, 2)}\n`);
console.log(`Auditoría terminada: ${summary.total} perfiles`);
console.log(`Accesibles/redirigen: ${summary.reachable} | Protegidos (401/403/429): ${summary.guarded} | Rotos/no verificables: ${summary.broken}`);
console.log(`Informe: ${target}`);

async function probe(url) {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 15000);
  const started = Date.now();
  try {
    let response = await fetch(url, { method: 'HEAD', redirect: 'follow', signal: controller.signal, headers: { 'User-Agent': 'SolarisPKN-Profile-Audit/0.1 (+https://solarispkn.com.ar)' } });
    if ([400,404,405].includes(response.status)) response = await fetch(url, { method: 'GET', redirect: 'follow', signal: controller.signal, headers: { Range: 'bytes=0-1023', 'User-Agent': 'SolarisPKN-Profile-Audit/0.1 (+https://solarispkn.com.ar)' } });
    const protectedRedirect = /\/(?:captcha|splashui\/captcha)(?:[/?#]|$)/i.test(new URL(response.url).pathname);
    const classification = protectedRedirect || [401,403,429].includes(response.status) ? 'guarded' : response.status >= 400 ? 'broken' : 'reachable';
    return { status: response.status, finalUrl: response.url, classification, latencyMs: Date.now()-started };
  } catch (error) {
    return { status: null, finalUrl: null, classification: 'broken', latencyMs: Date.now()-started, error: error.name === 'AbortError' ? 'timeout' : String(error.cause?.code || error.message).slice(0,160) };
  } finally { clearTimeout(timeout); }
}

async function atomicWrite(targetPath, contents) {
  await mkdir(dirname(targetPath), { recursive: true });
  const temporaryPath=`${targetPath}.${process.pid}.${Date.now()}.tmp`;
  try { await writeFile(temporaryPath,contents,{encoding:'utf8',flag:'wx'}); try { await rename(temporaryPath,targetPath); } catch(error) { if(!['EEXIST','EPERM'].includes(error.code)) throw error; await rm(targetPath,{force:true}); await rename(temporaryPath,targetPath); } }
  catch(error) { await rm(temporaryPath,{force:true}); throw error; }
}
