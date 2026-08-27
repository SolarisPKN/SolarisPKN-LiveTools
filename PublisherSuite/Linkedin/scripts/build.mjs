import { build } from 'esbuild';
import { cp, mkdir, rm } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const scriptDirectory = dirname(fileURLToPath(import.meta.url));
const projectRoot = dirname(scriptDirectory);
const outputDirectory = join(projectRoot, 'dist');

await rm(outputDirectory, { recursive: true, force: true });
await mkdir(outputDirectory, { recursive: true });

await build({
  entryPoints: [join(projectRoot, 'src', 'web', 'app.js')],
  bundle: true,
  format: 'esm',
  minify: true,
  sourcemap: true,
  outfile: join(outputDirectory, 'app.js'),
  target: ['chrome105', 'edge105', 'firefox102', 'safari15.4'],
  logLevel: 'info'
});

await Promise.all([
  cp(join(projectRoot, 'src', 'web', 'index.html'), join(outputDirectory, 'index.html')),
  cp(join(projectRoot, 'src', 'web', 'styles.css'), join(outputDirectory, 'styles.css'))
]);

console.log(`Interfaz compilada en ${outputDirectory}`);
