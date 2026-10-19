---
name: mdocs
description: Create and maintain documentation sites from Markdown with the mdocs CLI (with search and downloadable files such as Postman collections), and export them as a navigable PDF (bookmarks, linked TOC with page numbers), a draw.io file with Mermaid diagrams as editable shapes, or a static website. Use it when the user wants to write, organize, check, publish or export project documentation, manuals, guides or runbooks as .md files, add or delete documentation pages, or turn Markdown docs into PDF or draw.io.
license: MIT
---

# mdocs: documentación desde Markdown

`mdocs` es un único binario que convierte una carpeta de `.md` en un sitio de documentación con marca propia, un PDF navegable, un `.drawio` editable o un sitio estático. Todo lo que hagas pasa por la CLI; con `--json` cada comando devuelve datos que puedes leer sin interpretar texto.

## 1. Encontrar el binario

Prueba en este orden y usa el primero que responda a `--version`:

1. `bin/mdocs` dentro de la carpeta de este skill (el paquete `mdocs-skill.zip` lo trae para Linux amd64).
2. `mdocs` en el PATH.
3. `./mdocs` o `./mdocs.exe` en la carpeta del proyecto.

Si no existe, descárgalo del último release de https://github.com/cristianino/md-docs-viewer/releases (`mdocs_<versión>_<os>_<arch>.tar.gz`, o `.zip` en Windows) o compílalo con Go 1.26+: `go build -o mdocs .` en el repositorio. No inventes comandos: `mdocs --help` y `mdocs <comando> --help` listan los flags reales.

En los ejemplos `mdocs` es la ruta que encontraste. Si el proyecto no está en la carpeta actual, agrega `-C <carpeta>` en vez de hacer `cd`.

## 2. Entender el proyecto antes de editar

```bash
mdocs context --json     # marca, documentos, títulos con su ancla real, diagramas, problemas
mdocs check --json       # {ok, errors, warnings, issues[]}
```

Si no hay `brand.config.json`, el proyecto no existe todavía:

```bash
mdocs init <carpeta> --yes --company "<empresa>" --title "<título>" --description "<descripción>"
```

`init` no muestra "powered by" salvo que se pida `--powered-by`. Si el usuario no dio empresa o título, pregúntale o usa los valores por defecto y díselo.

## 3. Páginas

| Tarea | Comando |
|---|---|
| Crear página (archivo + índice) | `mdocs new "<Título>" [--subtitle ".."] [--position N] --json` |
| Registrar un `.md` que escribiste a mano | `mdocs add docs/x.md --title "X"` o `mdocs index --json` |
| Quitar del índice sin borrar | `mdocs remove <id>` |
| Borrar página | `mdocs delete <id> --yes --json` → revisa `brokenLinks` y corrige esos enlaces |
| Ver el orden | `mdocs list --json` |
| Buscar dónde se documenta algo | `mdocs search "<texto>" --json` → `hits[]` en jerarquía título → subtítulo → contenido, con `docId` y `anchor` |
| Ofrecer un archivo para descargar (colección Postman, OpenAPI, SQL…) | `mdocs attach <archivo> --doc <id> --title ".." --json` |

No edites `docs.config.json` a mano si un comando lo hace. Después de cada cambio corre `mdocs check --json` y corrige todo lo que tenga `"level": "error"`; los `warning` (enlaces o anclas rotas, imágenes faltantes) también conviene resolverlos.

## 4. Reglas para escribir el Markdown

- Un solo `# Título` por archivo, igual al `title` del documento (en el PDF se omite porque la cabecera ya lo muestra).
- `##` entra al índice del PDF y al índice general del visor; `###` aparece en «En esta página»; todo queda en los marcadores del PDF.
- `[TOC]` en una línea sola inserta la tabla de contenido del documento.
- Enlaces entre documentos con la ruta relativa al `.md`: `[ver](02-guia.md#paso-1)`. Funcionan en la web y dentro del PDF. Usa las anclas que reporta `mdocs context` (estilo GitHub: minúsculas, espacios → `-`, sin signos).
- Imágenes relativas al `.md`: `![x](img/diagrama.png)`.
- Archivos descargables: enlace relativo a un archivo que no sea `.md`, p. ej. `[Colección Postman](files/api.postman_collection.json)`. El visor lo muestra como botón de descarga, `mdocs export` lo copia y el PDF lo lleva adjunto. Usa `mdocs attach`; no pongas el contenido del JSON pegado en el `.md` si el usuario pidió que se pueda descargar.
- Antes de escribir una sección nueva, `mdocs search` para no duplicar lo que ya existe y enlazarlo.
- Diagramas en bloques ```` ```mermaid ````. `flowchart`/`graph`, `sequenceDiagram`, `stateDiagram-v2` y `classDiagram` se exportan a draw.io como formas editables; pie, gantt, ER y el resto, como imagen. Si el usuario quiere editar el diagrama en draw.io, prefiere los tipos nativos.

## 5. Entregables

| El usuario pide | Comando | Requiere Chrome |
|---|---|---|
| PDF | `mdocs build pdf -o <archivo>.pdf --json` | sí |
| draw.io | `mdocs build drawio [--mode image] --json` | sí |
| Ambos | `mdocs build all --json` (deja todo en `dist/`) | sí |
| Sitio estático (S3, Pages, Netlify, nginx) | `mdocs export -o site --base-path /<ruta> --json` | solo para incluir PDF y draw.io; si no hay Chrome usa `--no-pdf --no-drawio` |
| Verlo en el navegador | `mdocs serve` → http://localhost:3000 | no |

Si Chrome no se encuentra, prueba `--chrome <ruta>` o la variable `MDOCS_CHROME`; si corres como root o en un contenedor, `MDOCS_NO_SANDBOX=1`. Si no hay Chrome disponible, dilo y entrega el sitio estático o los `.md`.

Después de `build`, revisa en el JSON `diagrams[].output`: `native` es editable, `image` es una imagen y `code` significa que el Mermaid tiene un error de sintaxis que hay que corregir.

## 6. Al terminar

Reporta qué páginas creaste, cambiaste o borraste, el resultado de `mdocs check` y la ruta de cada archivo generado. No dejes enlaces rotos sin mencionarlos.
