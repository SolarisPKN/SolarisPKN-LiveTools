# SolarisPKN LinkedIn Publisher

Conversor local de Markdown y MDX a contenido listo para revisar y pegar en el editor de artículos de LinkedIn.

La herramienta conserva los elementos que el editor de LinkedIn puede representar razonablemente —títulos, listas, enlaces, negrita, cursiva, citas, separadores y código— y muestra advertencias para tablas, enlaces relativos, HTML crudo, expresiones y componentes MDX.

## Qué hace

- Procesa el archivo íntegramente en el navegador.
- No sube el MDX a ningún servidor.
- No solicita usuario, contraseña, cookies ni tokens de LinkedIn.
- No ejecuta imports, exports, JSX ni expresiones JavaScript del MDX.
- Acepta un JSON opcional con `title`, `description`, `tags` e `id`.
- Genera una vista previa, un cuerpo enriquecido para el portapapeles y texto plano.
- Prepara un post corto de difusión.
- Enumera imágenes con su referencia y texto alternativo.
- Exporta HTML, TXT y un manifiesto JSON.
- Incluye un conversor por línea de comandos para flujos repetibles.

## Qué no hace

- No inicia sesión en LinkedIn.
- No automatiza el navegador ni pulsa el botón de publicación.
- No ejecuta componentes MDX personalizados.
- No garantiza que LinkedIn conserve tablas o estilos no documentados.
- No publica todavía posts mediante OAuth/API; esa integración queda separada del conversor.

## Requisitos

- Node.js 20 o posterior.
- npm.
- Un navegador moderno con acceso al portapapeles en `localhost`.

## Abrir con doble clic en Windows

La forma recomendada es ejecutar:

```text
abrir-linkedin-publisher.cmd
```

El launcher:

1. Comprueba que Node.js y npm estén instalados.
2. Instala las dependencias locales solamente durante la primera ejecución.
3. Compila la interfaz.
4. Inicia el servidor en loopback.
5. Abre automáticamente el navegador.

La ventana permanece abierta mientras el publicador está funcionando. Cerrala o presioná `Ctrl+C` para terminar la sesión.

Si el puerto `4179` está ocupado, el launcher selecciona automáticamente otro puerto libre. Si definís `SOLARIS_LINKEDIN_PORT` manualmente, ese puerto se considera obligatorio y no se reemplaza.

## Iniciar desde la terminal

Desde esta carpeta:

```powershell
npm install
npm start
```

Después abrí, si el navegador no se abrió automáticamente:

```text
http://127.0.0.1:4179
```

La sesión escucha solamente en `127.0.0.1`. Para cambiar el puerto:

```powershell
$env:SOLARIS_LINKEDIN_PORT = '4180'
npm start
```

El servidor no recibe el contenido del MDX. Solo entrega la aplicación estática; el navegador lee y convierte el archivo localmente.

## Flujo recomendado

1. Cargá el `.mdx` o `.md`.
2. Opcionalmente cargá el JSON de traducción o metadatos.
3. Completá la URL base si existen enlaces relativos.
4. Revisá la vista previa y el diagnóstico.
5. Copiá el título.
6. Copiá el cuerpo enriquecido.
7. Abrí el editor oficial de LinkedIn y pegá ambos campos.
8. Insertá las imágenes enumeradas en los marcadores correspondientes.
9. Usá la vista previa de LinkedIn antes de publicar.

## Convertir desde la terminal

```powershell
npm run convert -- "D:\ruta\index-es.mdx" `
  --metadata "D:\ruta\es\posts\articulo.json" `
  --base-url "https://solarispkn.dev/es/blog/articulo/" `
  --article-url "https://solarispkn.dev/es/blog/articulo/"
```

Las salidas se escriben en `linkedin-output` junto al MDX, salvo que se indique otra carpeta:

```powershell
npm run convert -- "D:\ruta\index-es.mdx" --out "D:\salida"
```

Archivos generados:

- `<slug>.linkedin.html`: documento autónomo para revisar o copiar.
- `<slug>.linkedin.txt`: representación en texto plano.
- `<slug>.post.txt`: post corto de difusión.
- `<slug>.manifest.json`: activos, advertencias y estadísticas.

## Validación

```powershell
npm run check
```

Esto ejecuta las pruebas del conversor y compila la interfaz.

## Límites de compatibilidad

### Tablas

Se mantienen como tablas en la vista previa y en el HTML exportado, pero LinkedIn puede simplificarlas o eliminarlas al pegar. La herramienta siempre genera una advertencia para revisarlas.

### Imágenes

Una referencia Markdown se convierte en un marcador numerado. Las imágenes deben insertarse manualmente en LinkedIn; el manifiesto conserva la ruta y el texto alternativo.

### MDX y JSX

Los componentes se analizan, pero nunca se ejecutan. Si tienen contenido textual, se conserva ese contenido; si no, se muestra un marcador. Imports, exports y expresiones JavaScript se omiten y quedan registrados en el diagnóstico.

### Metadatos de SolarisPKN-Labs

Los artículos de Labs separan el cuerpo MDX, `post.json` y los textos localizados. Se puede cargar como metadatos el JSON localizado que contiene `title` y `description`. Si además hacen falta `tags` o `id`, pueden agregarse en un único JSON combinado o dejar que el conversor use los valores disponibles.

## Seguridad

- El servidor se enlaza exclusivamente a loopback.
- Rechaza encabezados `Host` ajenos a `localhost` y `127.0.0.1`.
- Sirve únicamente archivos estáticos generados dentro de `dist`.
- Aplica Content Security Policy y encabezados defensivos.
- El renderizador escapa texto y atributos.
- Solo permite enlaces `http`, `https`, `mailto`, anclas y rutas relativas.
- El tamaño máximo de un MDX es 2 MiB.

El clic de publicación permanece bajo control humano.
