import { copyFile, mkdir, rm } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { build } from 'esbuild';

const scriptDirectory = dirname(fileURLToPath(import.meta.url));
const projectRoot = dirname(scriptDirectory);
const dist = join(projectRoot, 'dist');

await rm(dist, { recursive: true, force: true });
await mkdir(dist, { recursive: true });
await Promise.all([
  copyFile(join(projectRoot, 'src', 'web', 'index.html'), join(dist, 'index.html')),
  copyFile(join(projectRoot, 'src', 'web', 'styles.css'), join(dist, 'styles.css')),
  build({
    entryPoints: [join(projectRoot, 'src', 'web', 'app.js')],
    bundle: true,
    format: 'esm',
    target: ['es2022'],
    outfile: join(dist, 'app.js'),
    sourcemap: true,
    logLevel: 'info'
  })
]);

console.log(`Build listo en ${dist}`);

