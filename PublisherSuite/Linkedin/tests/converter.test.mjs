import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { buildPostText, convertMdx, createArticleDocument, MAX_SOURCE_BYTES } from '../src/core/convert-mdx.js';

const testDirectory = dirname(fileURLToPath(import.meta.url));
const fixture = await readFile(join(testDirectory, 'fixtures', 'complete.mdx'), 'utf8');

test('converts representative MDX without executing JSX', () => {
  const result = convertMdx(fixture, { baseUrl: 'https://solarispkn.dev/blog/post/' });

  assert.equal(result.title, 'Artículo de prueba');
  assert.equal(result.description, 'Una descripción breve para el post.');
  assert.doesNotMatch(result.articleHtml, /Título dentro del documento/);
  assert.match(result.articleHtml, /<strong>negrita<\/strong>/);
  assert.match(result.articleHtml, /<ul>/);
  assert.match(result.articleHtml, /<blockquote>/);
  assert.match(result.articleHtml, /<table>/);
  assert.match(result.articleHtml, /https:\/\/solarispkn\.dev\/blog\/post\/guia/);
  assert.doesNotMatch(result.articleHtml, /<script>/i);
  assert.doesNotMatch(result.articleHtml, /alert\(/i);
  assert.equal(result.assets.length, 1);
  assert.equal(result.assets[0].alt, 'Arquitectura');
  assert.ok(result.warnings.some((warning) => warning.code === 'table'));
  assert.ok(result.warnings.some((warning) => warning.code === 'mdx-component'));
});

test('blocks unsafe URL protocols', () => {
  const result = convertMdx('[No tocar](javascript:alert(1))');

  assert.match(result.articleHtml, /href="#"/);
  assert.ok(result.warnings.some((warning) => warning.code === 'unsafe-link'));
});

test('warns when a relative link has no base URL', () => {
  const result = convertMdx('[Guía](./guia)');

  assert.ok(result.warnings.some((warning) => warning.code === 'relative-link'));
});

test('external metadata overrides frontmatter and builds a post', () => {
  const result = convertMdx(fixture, {
    metadata: {
      id: 'identificador-externo',
      title: 'Título externo',
      description: 'Descripción externa',
      tags: ['IA aplicada', '#SolarisPKN']
    },
    articleUrl: 'https://solarispkn.dev/articulo'
  });

  assert.equal(result.title, 'Título externo');
  assert.equal(result.slug, 'identificador-externo');
  assert.match(result.postText, /^Título externo/);
  assert.match(result.postText, /#IAAplicada #SolarisPKN/);
  assert.match(result.postText, /https:\/\/solarispkn\.dev\/articulo/);
});

test('creates a standalone HTML document', () => {
  const result = convertMdx('# Documento\n\nContenido.');
  const document = createArticleDocument(result);

  assert.match(document, /<!doctype html>/i);
  assert.match(document, /<title>Documento<\/title>/);
  assert.match(document, /<body>/);
  assert.equal((document.match(/<h1>/g) || []).length, 1);
});

test('post builder normalizes and limits duplicate tags', () => {
  const post = buildPostText({
    title: 'Título',
    description: 'Descripción',
    tags: ['#Uno', 'Uno', 'Dos palabras']
  });

  assert.equal(post, 'Título\n\nDescripción\n\n#Uno #DosPalabras');
});

test('rejects inputs larger than the local safety limit', () => {
  assert.throws(
    () => convertMdx('a'.repeat(MAX_SOURCE_BYTES + 1)),
    /supera el límite/
  );
});
