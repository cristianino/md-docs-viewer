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

## Requisitos

- [Go 1.22+](https://go.dev/dl/)
- [Chrome o Chromium](https://www.chromium.org/getting-involved/download-chromium/) (solo para generación de PDF)

## Inicio rápido

### 1. Compilar el binario

```bash
go build -o mdocs .
```

### 2. Reemplazar el logo

Coloca el logo de tu empresa en la raíz del proyecto como `logo.svg`.

### 3. Extraer colores del logo

```bash
./mdocs generate-styles
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
  "poweredBy": false
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
./mdocs serve
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
| `poweredBy` | `boolean` | Muestra "Powered by Laratec SAS" en el PDF (default: `false`) |

### `docs.config.json`

| Campo | Tipo | Descripción |
|---|---|---|
| `id` | `string` | Identificador único para la URL (`/doc/{id}`) |
| `file` | `string` | Ruta al archivo `.md` relativa a la raíz del proyecto |
| `title` | `string` | Título mostrado en la barra lateral |
| `subtitle` | `string` | Subtítulo descriptivo en la barra lateral |

---

## Comandos

| Comando | Descripción |
|---|---|
| `./mdocs serve` | Inicia el servidor en `http://localhost:3000` |
| `./mdocs serve --port 8080` | Inicia en un puerto específico |
| `./mdocs generate-styles` | Extrae colores de `logo.svg` y actualiza `brand.config.json` |
| `./mdocs --help` | Muestra la ayuda |

También puedes usar la variable de entorno `PORT`:

```bash
PORT=8080 ./mdocs serve
```

---

## Generación de PDF

El botón **Descargar PDF** en la barra lateral genera un PDF A4 con:

- **Portada** — logo, nombre de empresa, título y fecha
- **Índice** — listado numerado de todos los documentos
- **Secciones** — cada documento con encabezado corporativo y salto de página

El archivo se descarga con el nombre `{empresa}-{titulo}-{fecha}.pdf`.

> El servidor re-lee los archivos `.md` y las templates en cada request, por lo que puedes editar el contenido en caliente sin reiniciar.

---

## Estructura del proyecto

```
proyecto/
├── main.go                    ← Entry point
├── go.mod / go.sum
├── cmd/mdocs/
│   ├── root.go                ← CLI Cobra
│   ├── serve.go               ← Subcomando serve
│   └── styles.go              ← Subcomando generate-styles
├── internal/
│   ├── brand/                 ← Configuración y extracción de colores
│   ├── docs/                  ← Carga de docs y renderizado Markdown
│   └── server/                ← Servidor HTTP y generación de PDF
├── templates/
│   ├── page.gohtml            ← Template HTML del visor
│   └── print.gohtml           ← Template HTML para PDF
├── docs/
│   └── *.md                   ← Archivos de contenido
├── brand.config.json          ← Colores e identidad corporativa
├── docs.config.json           ← Registro de documentos
└── logo.svg                   ← Logo de la empresa (reemplazar)
```

---

## Stack técnico

- [Go](https://go.dev/) — lenguaje y servidor HTTP (`net/http`)
- [Cobra](https://github.com/spf13/cobra) — framework CLI
- [goldmark](https://github.com/yuin/goldmark) — parser de Markdown (GFM)
- [chromedp](https://github.com/chromedp/chromedp) — generación de PDF via Chrome headless
- [Mermaid](https://mermaid.js.org/) — diagramas en el navegador (CDN)

---

## Licencia

MIT — ver [LICENSE](LICENSE)
