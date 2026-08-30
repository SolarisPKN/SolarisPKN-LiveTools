import { mkdir, readFile, rename, rm, writeFile } from 'node:fs/promises';
import { basename, dirname, join, resolve } from 'node:path';
import { buildCanonicalContent } from '../src/core/content.js';
import { preparePlan } from '../src/core/platforms.js';

const args = parseArgs(process.argv.slice(2));
if (!args.input) { printHelp(); process.exit(1); }

const inputPath = resolve(args.input);
const source = await readFile(inputPath, 'utf8');
const metadata = args.metadata ? JSON.parse(await readFile(resolve(args.metadata), 'utf8')) : {};
const targets = args.targets ? args.targets.split(',').map((item) => item.trim()).filter(Boolean) : undefined;
const content = buildCanonicalContent(source, { metadata, canonicalUrl: args.canonicalUrl, coverImage: args.coverImage, language: args.language, baseUrl: args.baseUrl });
const plan = preparePlan(content, targets);
const outputDirectory = resolve(args.output || join(dirname(inputPath), 'publisher-output', content.slug));
await mkdir(outputDirectory, { recursive: true });
await atomicWrite(join(outputDirectory, `${content.slug}.plan.json`), `${JSON.stringify(plan, null, 2)}\n`);
for (const target of plan.targets) {
  await atomicWrite(join(outputDirectory, `${content.slug}.${target.id}.json`), `${JSON.stringify(target.payload, null, 2)}\n`);
}
console.log(`Plan preparado: ${outputDirectory}`);
console.log(`SHA-256: ${plan.digest}`);
console.log(`Fuente: ${basename(inputPath)} | Destinos: ${plan.targets.length} | Advertencias MDX: ${content.warnings.length}`);

async function atomicWrite(targetPath, contents) {
  const temporaryPath = `${targetPath}.${process.pid}.${Date.now()}.tmp`;
  try {
    await writeFile(temporaryPath, contents, { encoding: 'utf8', flag: 'wx' });
    try { await rename(temporaryPath, targetPath); }
    catch (error) { if (!['EEXIST','EPERM'].includes(error.code)) throw error; await rm(targetPath, { force: true }); await rename(temporaryPath, targetPath); }
  } catch (error) { await rm(temporaryPath, { force: true }); throw error; }
}

function parseArgs(values) {
  const result = {};
  const map = { '--out':'output', '--metadata':'metadata', '--targets':'targets', '--canonical-url':'canonicalUrl', '--cover-image':'coverImage', '--language':'language', '--base-url':'baseUrl' };
  const positional = [];
  for (let index=0; index<values.length; index+=1) {
    const value=values[index];
    if (['--help','-h'].includes(value)) { printHelp(); process.exit(0); }
    if (map[value]) { if (!values[index+1] || values[index+1].startsWith('--')) throw new Error(`Falta el valor de ${value}.`); result[map[value]]=values[++index]; }
    else if (value.startsWith('--')) throw new Error(`Opción desconocida: ${value}`);
    else positional.push(value);
  }
  result.input=positional[0]; return result;
}

function printHelp() {
  console.log(`Uso:\n  npm run plan -- articulo.mdx [opciones]\n\nOpciones:\n  --out carpeta\n  --metadata archivo.json\n  --targets devto,linkedin,mastodon\n  --canonical-url https://...\n  --cover-image https://...\n  --language es\n  --base-url https://...`);
}
