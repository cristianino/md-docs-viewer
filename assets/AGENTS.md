# Guía para agentes de IA — proyecto mdocs

Este proyecto genera documentación a partir de archivos Markdown con `mdocs`.
Edita los `.md` y la configuración; no edites archivos generados en `dist/`.

## Archivos

| Archivo | Qué contiene |
|---|---|
| `brand.config.json` | Empresa, título, descripción, año, colores, `poweredBy` (false por defecto) |
| `docs.config.json` | Lista ordenada de documentos: `id`, `file`, `title`, `subtitle` |
| `docs/*.md` | Contenido (GFM + bloques ```` ```mermaid ````) |
| `logo.svg` | Logo; `mdocs generate-styles` toma sus colores |
| `templates/*.gohtml` | Opcional: sobrescribe las plantillas embebidas |

## Comandos recomendados

```bash
mdocs context --json        # entender el proyecto: docs, títulos/anclas, diagramas, problemas
mdocs check --json          # validar después de cada cambio (exit 1 si hay errores)
mdocs add docs/x.md --title "X" [--subtitle ".."] [--position N]   # registrar/crear doc
mdocs remove <id>           # quitar del índice (no borra el .md)
mdocs list --json           # documentos registrados
mdocs build all --json      # generar dist/*.pdf y dist/*.drawio (no requiere serve)
mdocs build drawio --mode image   # forzar diagramas como imagen
mdocs export -o site --base-path /docs --json   # sitio estático para publicar
```

Todos aceptan `-C <carpeta>` para trabajar en otro directorio y `--chrome <ruta>`
(o `MDOCS_CHROME`) si Chrome no se detecta solo. Con `--json` el progreso va a
stderr y stdout queda solo con JSON.

## Reglas al escribir Markdown

- Un `# Título` por archivo, igual al `title` de `docs.config.json` (en el PDF se omite porque la cabecera ya lo muestra).
- Los `##` entran al índice del PDF y al índice general del visor (`/indice`); `###` también aparece en «En esta página»; todo queda en los marcadores del PDF.
- Escribe `[TOC]` en una línea sola para insertar la tabla de contenido del documento (web y PDF).
- Enlaces entre documentos: usa la ruta relativa al `.md` (`[ver](02-guia.md#paso-1)`); funcionan en la web y en el PDF.
- Las anclas siguen el estilo GitHub: minúsculas, espacios → `-`, se quitan signos (`¿Cómo funciona?` → `#cómo-funciona`). Consulta las reales con `mdocs context`.
- Imágenes relativas al `.md` (`![x](img/diagrama.png)`).
- Diagramas: `flowchart`/`graph`, `sequenceDiagram`, `stateDiagram-v2` y `classDiagram` se exportan a draw.io como formas editables; el resto (pie, gantt, er, journey, mindmap…) como imagen.
