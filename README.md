# md-docs-viewer

Visor de documentación Markdown con identidad corporativa dinámica. Convierte archivos `.md` en un sitio web navegable con los colores y el logo de tu empresa, y permite exportar todo el contenido a un PDF bien estructurado.

## Características

- Renderizado de Markdown con estilos corporativos
- Sidebar de navegación con lista de documentos
- Identidad visual extraída automáticamente del logo SVG
- Soporte para diagramas Mermaid (flujos, secuencias, etc.)
- Tablas, bloques de código, blockquotes y listas de tareas
- Generación de PDF en un clic (portada + índice + documentos)
- Configuración centralizada en dos archivos JSON

## Inicio rápido

### 1. Instalar dependencias

```bash
npm install
```

### 2. Reemplazar el logo

Coloca el logo de tu empresa en la raíz del proyecto como `logo.svg`.

### 3. Extraer colores del logo

```bash
npm run generate:styles
```

Detecta automáticamente los colores de `logo.svg` y actualiza `brand.config.json`.

### 4. Ajustar la identidad corporativa

Edita `brand.config.json` para afinar nombre, título, año y atribución:

```json
{
  "primaryColor": "#1a2035",
  "accentColor": "#0066cc",
  "companyName": "Mi Empresa S.A.",
  "docsTitle": "Portal de Documentación",
  "docsYear": "2026",
  "poweredBy": true
}
```

### 5. Registrar documentos

Edita `docs.config.json` para listar tus archivos Markdown:

```json
[
  {
    "id": "overview",
    "file": "docs/overview.md",
    "title": "Visión general",
    "subtitle": "Arquitectura y conceptos clave"
  }
]
```

### 6. Crear los archivos de contenido

Agrega tus archivos `.md` en la carpeta `docs/`.

### 7. Iniciar el servidor

```bash
npm start
```

Abre `http://localhost:3000` en tu navegador.

---

## Configuración

### `brand.config.json`

| Campo | Tipo | Descripción |
|---|---|---|
| `primaryColor` | `string` | Color principal — sidebar, cabeceras, títulos |
| `accentColor` | `string` | Color de acento — links, botones, bordes activos |
| `companyName` | `string` | Nombre de la empresa |
| `docsTitle` | `string` | Subtítulo bajo el logo |
| `docsYear` | `string` | Año mostrado en el footer |
| `poweredBy` | `boolean` | Muestra "Powered by Laratec SAS" en el PDF (default: `true`) |

### `docs.config.json`

| Campo | Tipo | Descripción |
|---|---|---|
| `id` | `string` | Identificador único para la URL (`/doc/{id}`) |
| `file` | `string` | Ruta al archivo `.md` relativa a la raíz del proyecto |
| `title` | `string` | Título mostrado en la barra lateral |
| `subtitle` | `string` | Subtítulo descriptivo en la barra lateral |

---

## Scripts

| Comando | Descripción |
|---|---|
| `npm start` | Inicia el servidor en `http://localhost:3000` |
| `npm run generate:styles` | Extrae colores de `logo.svg` y actualiza `brand.config.json` |

Para cambiar el puerto:

```bash
PORT=8080 npm start
```

---

## Generación de PDF

El botón **Descargar PDF** en la barra lateral genera un PDF A4 con:

- **Portada** — logo, nombre de empresa, título y fecha
- **Índice** — listado numerado de todos los documentos
- **Secciones** — cada documento con encabezado corporativo y salto de página

El archivo se descarga con el nombre `{empresa}-{titulo}-{fecha}.pdf`.

> El servidor re-lee los archivos `.md` en cada request, por lo que puedes editar el contenido en caliente sin reiniciar.

---

## Estructura del proyecto

```
proyecto/
├── src/
│   └── server.ts              ← Servidor Express + renderizado HTML
├── scripts/
│   └── generate-styles.ts     ← Extracción de colores del logo
├── docs/
│   └── *.md                   ← Archivos de contenido
├── brand.config.json          ← Colores e identidad corporativa
├── docs.config.json           ← Registro de documentos
├── logo.svg                   ← Logo de la empresa (reemplazar)
├── package.json
└── tsconfig.json
```

---

## Stack técnico

- [Express.js](https://expressjs.com/) — servidor HTTP
- [marked](https://marked.js.org/) — parser de Markdown
- [Puppeteer](https://pptr.dev/) — generación de PDF
- [Mermaid](https://mermaid.js.org/) — diagramas en el navegador
- [TypeScript](https://www.typescriptlang.org/) + [tsx](https://github.com/privatenumber/tsx) — runtime sin compilación

---

## Licencia

MIT — ver [LICENSE](LICENSE)
