import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { test } from 'node:test';
import { buildCanonicalContent, buildSocialText, contentDigest, stripExecutableMdx } from '../src/core/content.js';
import { platformRegistry, preparePlan } from '../src/core/platforms.js';

const source = `---
title: Publicar con criterio
description: Una prueba completa del adaptador.
tags:
  - SolarisPKN
  - Open Source
---

# Publicar con criterio

Un texto con **formato**, una lista y un enlace.

- Uno
- Dos

![Portada](https://example.com/cover.png)
`;

test('construye contenido canónico sin ejecutar MDX y conserva datos útiles', () => {
  const content = buildCanonicalContent(source, { canonicalUrl: 'https://labs.solarispkn.com.ar/es/blog/publicar', language: 'es-AR' });
  assert.equal(content.title, 'Publicar con criterio');
  assert.equal(content.canonicalUrl, 'https://labs.solarispkn.com.ar/es/blog/publicar');
  assert.equal(content.coverImage, 'https://example.com/cover.png');
  assert.deepEqual(content.tags, ['SolarisPKN','OpenSource']);
  assert.equal(content.language, 'es-AR');
  assert.match(content.html, /<strong>formato<\/strong>/);
  assert.ok(content.stats.words > 5);
});

test('elimina imports, exports, JSX y expresiones del Markdown distribuible', () => {
  const cleaned = stripExecutableMdx(`import Secret from './x'\n# Hola\n<Card foo={bar}>oculto</Card>\n{danger()}`);
  assert.doesNotMatch(cleaned, /import|danger|<Card/);
  assert.match(cleaned, /# Hola/);
});

test('adapta el social copy al límite sin perder la URL canónica', () => {
  const content = buildCanonicalContent(source, { canonicalUrl: 'https://example.com/post' });
  const value = buildSocialText(content, 180, { hashtagLimit: 2 });
  assert.ok(value.length <= 180);
  assert.match(value, /https:\/\/example\.com\/post/);
  assert.match(value, /#SolarisPKN/);
});

test('el digest cambia con contenido o destinos y es estable con el orden', () => {
  const content = buildCanonicalContent(source);
  assert.equal(contentDigest(content,['devto','mastodon']), contentDigest(content,['mastodon','devto']));
  assert.notEqual(contentDigest(content,['devto']), contentDigest(content,['devto','mastodon']));
});

test('prepara los 19 contratos y bloquea APIs sin credenciales o media', () => {
  const content = buildCanonicalContent('# Hola\n\nTexto sin imagen.');
  const plan = preparePlan(content, platformRegistry.map((item) => item.id), {});
  assert.equal(plan.targets.length, 19);
  assert.equal(plan.targets.find((item) => item.id === 'solarispkn-blog').canPublish, true);
  assert.equal(plan.targets.find((item) => item.id === 'devto').canPublish, false);
  assert.match(plan.targets.find((item) => item.id === 'instagram').warnings.join(' '), /coverImage/);
  assert.equal(plan.targets.find((item) => item.id === 'x').mode, 'assisted');
});

test('habilita una API sólo con toda su configuración mínima', () => {
  const content = buildCanonicalContent(source);
  const env = { SOLARIS_DEVTO_API_KEY: 'test-only' };
  const plan = preparePlan(content,['devto'],env);
  assert.equal(plan.targets[0].canPublish,true);
  assert.deepEqual(plan.targets[0].missingCredentials,[]);
});

test('el inventario público contiene exactamente 109 perfiles y URLs HTTPS', async () => {
  const inventory = JSON.parse(await readFile(new URL('../config/profiles.json',import.meta.url),'utf8'));
  const profiles = inventory.categories.flatMap((category)=>category.profiles);
  assert.equal(profiles.length,109);
  assert.ok(profiles.every(([,url])=>url.startsWith('https://')));
  assert.equal(new Set(profiles.map(([,url])=>url)).size,109);
});

