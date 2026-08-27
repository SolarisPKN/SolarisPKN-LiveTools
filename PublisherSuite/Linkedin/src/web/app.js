import { buildPostText, convertMdx, createArticleDocument, MAX_SOURCE_BYTES } from '../core/convert-mdx.js';

const elements = Object.fromEntries([
  'dropZone', 'mdxInput', 'chooseMdxButton', 'loadDemoButton', 'sourceName', 'sourceSize',
  'metadataInput', 'metadataStatus', 'titleInput', 'descriptionInput', 'baseUrlInput',
  'articleUrlInput', 'tagsInput', 'convertButton', 'emptyState', 'resultState', 'resultTitle',
  'stats', 'copyTitleButton', 'copyRichButton', 'previewTitle', 'articlePreview', 'assetCount',
  'warningCount', 'postOutput', 'postCharacterCount', 'copyPostButton', 'assetList', 'warningList',
  'downloadHtmlButton', 'downloadTextButton', 'downloadManifestButton', 'toast'
].map((id) => [id, document.getElementById(id)]));

const state = {
  source: '',
  sourceName: '',
  metadata: {},
  result: null,
  toastTimer: null
};

const demoSource = `# Automatizar sin delegar el criterio

> Una herramienta útil no reemplaza la revisión: la vuelve más clara.

## Qué conserva el conversor

- **Negrita** y *cursiva*.
- Enlaces a [SolarisPKN](https://solarispkn.dev/).
- Listas ordenadas y desordenadas.
- Citas, separadores y bloques de código.

1. Cargá el MDX.
2. Revisá las advertencias.
3. Copiá el cuerpo enriquecido.

---

| Etapa | Responsable |
| --- | --- |
| Conversión | PublisherSuite |
| Revisión final | Humano |

\`\`\`text
MDX → vista previa → LinkedIn
\`\`\`

![Diagrama del flujo](./images/flujo.png)

<ProjectCard id="demo" />
`;

elements.chooseMdxButton.addEventListener('click', (event) => {
  elements.mdxInput.click();
});
elements.mdxInput.addEventListener('change', () => loadMdxFile(elements.mdxInput.files?.[0]));
elements.metadataInput.addEventListener('change', () => loadMetadataFile(elements.metadataInput.files?.[0]));
elements.loadDemoButton.addEventListener('click', loadDemo);
elements.convertButton.addEventListener('click', () => processSource(false));
elements.copyTitleButton.addEventListener('click', () => copyText(state.result?.title || '', 'Título copiado.'));
elements.copyRichButton.addEventListener('click', copyRichArticle);
elements.copyPostButton.addEventListener('click', () => copyText(elements.postOutput.value, 'Post corto copiado.'));
elements.postOutput.addEventListener('input', updatePostCharacterCount);
elements.downloadHtmlButton.addEventListener('click', downloadHtml);
elements.downloadTextButton.addEventListener('click', downloadText);
elements.downloadManifestButton.addEventListener('click', downloadManifest);
elements.articlePreview.addEventListener('click', (event) => {
  const link = event.target.closest('a');
  if (link) {
    event.preventDefault();
    showToast('Los enlaces no se abren desde la vista previa. Revisá su destino en el artículo final.');
  }
});

for (const eventName of ['dragenter', 'dragover']) {
  elements.dropZone.addEventListener(eventName, (event) => {
    event.preventDefault();
    elements.dropZone.classList.add('dragging');
  });
}
for (const eventName of ['dragleave', 'drop']) {
  elements.dropZone.addEventListener(eventName, (event) => {
    event.preventDefault();
    elements.dropZone.classList.remove('dragging');
  });
}
elements.dropZone.addEventListener('drop', async (event) => {
  const files = [...(event.dataTransfer?.files || [])];
  const mdx = files.find((file) => /\.(md|mdx)$/i.test(file.name));
  const metadata = files.find((file) => /\.json$/i.test(file.name));
  if (metadata) {
    await loadMetadataFile(metadata);
  }
  if (mdx) {
    await loadMdxFile(mdx);
  } else {
    showToast('No encontré un archivo .md o .mdx en lo que soltaste.', true);
  }
});

document.querySelectorAll('.tab').forEach((button) => {
  button.addEventListener('click', () => activateTab(button.dataset.tab));
});

async function loadMdxFile(file) {
  if (!file) {
    return;
  }
  if (!/\.(md|mdx)$/i.test(file.name)) {
    showToast('Elegí un archivo con extensión .md o .mdx.', true);
    return;
  }
  if (file.size > MAX_SOURCE_BYTES) {
    showToast(`El archivo supera el límite de ${formatBytes(MAX_SOURCE_BYTES)}.`, true);
    return;
  }

  try {
    state.source = await file.text();
    state.sourceName = file.name;
    elements.sourceName.textContent = file.name;
    elements.sourceSize.textContent = formatBytes(file.size);
    elements.convertButton.disabled = false;
    processSource(true);
  } catch (error) {
    showToast(`No pude leer el archivo: ${error.message}`, true);
  }
}

async function loadMetadataFile(file) {
  if (!file) {
    return;
  }
  if (file.size > 1024 * 1024) {
    showToast('El JSON de metadatos supera 1 MiB.', true);
    return;
  }

  try {
    const parsed = JSON.parse(await file.text());
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      throw new Error('la raíz debe ser un objeto JSON');
    }
    state.metadata = parsed;
    elements.metadataStatus.textContent = `${file.name} cargado.`;
    if (state.source) {
      processSource(true);
    }
  } catch (error) {
    state.metadata = {};
    elements.metadataStatus.textContent = '';
    showToast(`Metadatos inválidos: ${error.message}`, true);
  }
}

function loadDemo() {
  state.source = demoSource;
  state.sourceName = 'ejemplo-linkedin.mdx';
  state.metadata = {
    id: 'automatizar-sin-delegar-el-criterio',
    description: 'Una demostración del flujo local de conversión y revisión.',
    tags: ['MDX', 'LinkedIn', 'SolarisPKN']
  };
  elements.sourceName.textContent = state.sourceName;
  elements.sourceSize.textContent = formatBytes(new TextEncoder().encode(demoSource).byteLength);
  elements.metadataStatus.textContent = 'Metadatos de demostración cargados.';
  elements.convertButton.disabled = false;
  processSource(true);
}

function processSource(hydrateFields) {
  if (!state.source) {
    showToast('Primero cargá un MDX.', true);
    return;
  }

  try {
    const result = convertMdx(state.source, {
      metadata: state.metadata,
      title: hydrateFields ? '' : elements.titleInput.value,
      description: hydrateFields ? '' : elements.descriptionInput.value,
      baseUrl: elements.baseUrlInput.value,
      articleUrl: elements.articleUrlInput.value,
      tags: hydrateFields ? undefined : parseTags(elements.tagsInput.value)
    });

    if (hydrateFields) {
      elements.titleInput.value = result.title;
      elements.descriptionInput.value = result.description;
      elements.tagsInput.value = normalizeMetadataTags(result.metadata.tags).join(', ');
      result.postText = buildPostText({
        title: result.title,
        description: result.description,
        url: elements.articleUrlInput.value,
        tags: normalizeMetadataTags(result.metadata.tags)
      });
    }

    state.result = result;
    renderResult();
    showToast('Conversión terminada. Revisá las advertencias antes de pegar.');
  } catch (error) {
    showToast(error.message, true);
  }
}

function renderResult() {
  const result = state.result;
  elements.emptyState.hidden = true;
  elements.resultState.hidden = false;
  elements.resultTitle.textContent = result.title;
  elements.previewTitle.textContent = result.title;
  elements.articlePreview.innerHTML = result.articleHtml;
  elements.postOutput.value = result.postText;
  elements.assetCount.textContent = String(result.assets.length);
  elements.warningCount.textContent = String(result.warnings.length);
  elements.stats.replaceChildren(
    statPill(`${result.stats.words} palabras`),
    statPill(`${result.stats.headings} títulos`),
    statPill(`${result.stats.links} enlaces`),
    statPill(`${result.stats.images} imágenes`),
    statPill(`${result.warnings.length} advertencias`)
  );
  renderAssets(result.assets);
  renderWarnings(result.warnings);
  updatePostCharacterCount();
}

function renderAssets(assets) {
  elements.assetList.replaceChildren();
  if (assets.length === 0) {
    const item = document.createElement('li');
    item.textContent = 'El artículo no contiene referencias a imágenes.';
    elements.assetList.append(item);
    return;
  }

  for (const asset of assets) {
    const item = document.createElement('li');
    const title = document.createElement('strong');
    const source = document.createElement('span');
    const alt = document.createElement('span');
    const copyButton = document.createElement('button');
    title.textContent = `Imagen ${asset.index}: ${asset.alt}`;
    source.textContent = asset.source;
    alt.textContent = `Texto alternativo: ${asset.alt}`;
    copyButton.className = 'text-button';
    copyButton.type = 'button';
    copyButton.textContent = 'Copiar referencia';
    copyButton.addEventListener('click', () => copyText(asset.source, 'Referencia de imagen copiada.'));
    item.append(title, source, alt, copyButton);
    elements.assetList.append(item);
  }
}

function renderWarnings(warnings) {
  elements.warningList.replaceChildren();
  if (warnings.length === 0) {
    const item = document.createElement('li');
    const icon = document.createElement('span');
    const content = document.createElement('div');
    icon.className = 'success-icon';
    icon.textContent = '●';
    content.textContent = 'No se detectaron incompatibilidades conocidas.';
    item.append(icon, content);
    elements.warningList.append(item);
    return;
  }

  for (const warning of warnings) {
    const item = document.createElement('li');
    const icon = document.createElement('span');
    const content = document.createElement('div');
    icon.className = 'warning-icon';
    icon.textContent = '▲';
    content.textContent = warning.message;
    if (warning.line) {
      const line = document.createElement('small');
      line.textContent = `Línea ${warning.line} · ${warning.code}`;
      content.append(line);
    }
    item.append(icon, content);
    elements.warningList.append(item);
  }
}

function activateTab(name) {
  document.querySelectorAll('.tab').forEach((button) => {
    const active = button.dataset.tab === name;
    button.classList.toggle('active', active);
    button.setAttribute('aria-selected', String(active));
  });
  document.querySelectorAll('.tab-panel').forEach((panel) => {
    const active = panel.id === `${name}Tab`;
    panel.classList.toggle('active', active);
    panel.hidden = !active;
  });
}

async function copyRichArticle() {
  if (!state.result) {
    return;
  }

  try {
    if (window.ClipboardItem && navigator.clipboard?.write) {
      const item = new ClipboardItem({
        'text/html': new Blob([state.result.articleHtml], { type: 'text/html' }),
        'text/plain': new Blob([state.result.plainText], { type: 'text/plain' })
      });
      await navigator.clipboard.write([item]);
    } else {
      fallbackRichCopy(state.result.articleHtml);
    }
    showToast('Cuerpo enriquecido copiado. Pegalo en el campo “Escribe aquí” de LinkedIn.');
  } catch (error) {
    try {
      fallbackRichCopy(state.result.articleHtml);
      showToast('Cuerpo enriquecido copiado mediante el modo compatible.');
    } catch {
      showToast(`El navegador no permitió copiar: ${error.message}`, true);
    }
  }
}

function fallbackRichCopy(html) {
  const container = document.createElement('div');
  container.contentEditable = 'true';
  container.className = 'visually-hidden';
  container.innerHTML = html;
  document.body.append(container);
  const selection = window.getSelection();
  const range = document.createRange();
  range.selectNodeContents(container);
  selection.removeAllRanges();
  selection.addRange(range);
  const copied = document.execCommand('copy');
  selection.removeAllRanges();
  container.remove();
  if (!copied) {
    throw new Error('No se pudo copiar el contenido enriquecido.');
  }
}

async function copyText(value, successMessage) {
  if (!value) {
    showToast('No hay contenido para copiar.', true);
    return;
  }
  try {
    await navigator.clipboard.writeText(value);
    showToast(successMessage);
  } catch {
    const textarea = document.createElement('textarea');
    textarea.value = value;
    textarea.className = 'visually-hidden';
    document.body.append(textarea);
    textarea.select();
    const copied = document.execCommand('copy');
    textarea.remove();
    showToast(copied ? successMessage : 'El navegador bloqueó el portapapeles.', !copied);
  }
}

function downloadHtml() {
  if (!state.result) return;
  downloadBlob(`${state.result.slug}.linkedin.html`, createArticleDocument(state.result), 'text/html;charset=utf-8');
}

function downloadText() {
  if (!state.result) return;
  downloadBlob(`${state.result.slug}.linkedin.txt`, `${state.result.plainText}\n`, 'text/plain;charset=utf-8');
}

function downloadManifest() {
  if (!state.result) return;
  const manifest = {
    schemaVersion: state.result.schemaVersion,
    source: state.sourceName,
    title: state.result.title,
    description: state.result.description,
    postText: elements.postOutput.value,
    assets: state.result.assets,
    warnings: state.result.warnings,
    stats: state.result.stats
  };
  downloadBlob(`${state.result.slug}.manifest.json`, `${JSON.stringify(manifest, null, 2)}\n`, 'application/json;charset=utf-8');
}

function downloadBlob(name, contents, type) {
  const url = URL.createObjectURL(new Blob([contents], { type }));
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = name;
  document.body.append(anchor);
  anchor.click();
  anchor.remove();
  setTimeout(() => URL.revokeObjectURL(url), 0);
  showToast(`${name} descargado.`);
}

function updatePostCharacterCount() {
  const count = [...elements.postOutput.value].length;
  elements.postCharacterCount.textContent = `${count.toLocaleString('es-AR')} caracteres`;
}

function statPill(text) {
  const element = document.createElement('span');
  element.textContent = text;
  return element;
}

function parseTags(value) {
  return String(value || '').split(',').map((tag) => tag.trim()).filter(Boolean);
}

function normalizeMetadataTags(value) {
  return Array.isArray(value) ? value.map(String) : parseTags(value);
}

function formatBytes(bytes) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KiB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MiB`;
}

function showToast(message, isError = false) {
  clearTimeout(state.toastTimer);
  elements.toast.textContent = message;
  elements.toast.classList.toggle('error', isError);
  elements.toast.hidden = false;
  state.toastTimer = setTimeout(() => {
    elements.toast.hidden = true;
  }, 4200);
}
