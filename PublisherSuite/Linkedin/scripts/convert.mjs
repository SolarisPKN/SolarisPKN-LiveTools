import { mkdir, readFile, rename, rm, writeFile } from 'node:fs/promises';
import { basename, dirname, join, resolve } from 'node:path';
import { convertMdx, createArticleDocument } from '../src/core/convert-mdx.js';

const args = parseArgs(process.argv.slice(2));

if (!args.input) {
  printHelp();
  process.exit(1);
}

const inputPath = resolve(args.input);
const outputDirectory = resolve(args.output || join(dirname(inputPath), 'linkedin-output'));
const metadata = args.metadata
  ? JSON.parse(await readFile(resolve(args.metadata), 'utf8'))
  : {};
const source = await readFile(inputPath, 'utf8');
const result = convertMdx(source, {
  metadata,
  title: args.title,
  baseUrl: args.baseUrl,
  articleUrl: args.articleUrl
});

await mkdir(outputDirectory, { recursive: true });

const files = [
  [`${result.slug}.linkedin.html`, createArticleDocument(result)],
  [`${result.slug}.linkedin.txt`, result.plainText + '\n'],
  [`${result.slug}.post.txt`, result.postText + '\n'],
  [`${result.slug}.manifest.json`, JSON.stringify({
    schemaVersion: result.schemaVersion,
    source: basename(inputPath),
    title: result.title,
    description: result.description,
    assets: result.assets,
    warnings: result.warnings,
    stats: result.stats
  }, null, 2) + '\n']
];

for (const [name, contents] of files) {
  await atomicWrite(join(outputDirectory, name), contents);
}

console.log(`Conversión terminada: ${outputDirectory}`);
console.log(`Título: ${result.title}`);
console.log(`Palabras: ${result.stats.words} | Imágenes: ${result.stats.images} | Advertencias: ${result.warnings.length}`);
for (const warning of result.warnings) {
  console.warn(`- ${warning.line ? `Línea ${warning.line}: ` : ''}${warning.message}`);
}

async function atomicWrite(targetPath, contents) {
  const temporaryPath = `${targetPath}.${process.pid}.${Date.now()}.tmp`;
  try {
    await writeFile(temporaryPath, contents, { encoding: 'utf8', flag: 'wx' });
    try {
      await rename(temporaryPath, targetPath);
    } catch (error) {
      if (!['EEXIST', 'EPERM'].includes(error.code)) {
        throw error;
      }
      await rm(targetPath, { force: true });
      await rename(temporaryPath, targetPath);
    }
  } catch (error) {
    await rm(temporaryPath, { force: true });
    throw error;
  }
}

function parseArgs(values) {
  const result = {};
  const positional = [];
  const mapping = {
    '--out': 'output',
    '--metadata': 'metadata',
    '--title': 'title',
    '--base-url': 'baseUrl',
    '--article-url': 'articleUrl'
  };

  for (let index = 0; index < values.length; index += 1) {
    const value = values[index];
    if (value === '--help' || value === '-h') {
      printHelp();
      process.exit(0);
    }
    if (mapping[value]) {
      const next = values[index + 1];
      if (!next || next.startsWith('--')) {
        throw new Error(`Falta el valor de ${value}.`);
      }
      result[mapping[value]] = next;
      index += 1;
      continue;
    }
    if (value.startsWith('--')) {
      throw new Error(`Opción desconocida: ${value}`);
    }
    positional.push(value);
  }

  result.input = positional[0];
  return result;
}

function printHelp() {
  console.log(`Uso:
  npm run convert -- archivo.mdx [opciones]

Opciones:
  --out carpeta          Carpeta de salida
  --metadata archivo     JSON opcional con title, description, tags e id
  --title texto          Sobrescribe el título inferido
  --base-url url         Resuelve enlaces e imágenes relativos
  --article-url url      URL para el post corto de difusión
  -h, --help             Muestra esta ayuda`);
}
