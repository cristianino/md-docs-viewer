# Introducción

> Reemplaza este archivo con el contenido real de tu documentación. Los archivos `.md` en la carpeta `docs/` son el contenido del visor.

Bienvenido al visor de documentación. Este proyecto convierte archivos **Markdown** en páginas web con identidad corporativa, un PDF navegable y un archivo draw.io con los diagramas editables.

## ¿Cómo funciona?

El servidor lee los archivos `.md` registrados en `docs.config.json` y los renderiza como HTML aplicando los colores definidos en `brand.config.json`.

```
logo.svg  →  mdocs generate-styles  →  brand.config.json  →  UI con tus colores
```

## Cómo usar este template

1. **Compila** el binario con `go build -o mdocs .`
2. **Inicializa** con `mdocs init` (empresa, título, descripción, año y "powered by")
3. **Reemplaza** `logo.svg` y ejecuta `mdocs generate-styles` para tomar sus colores
4. **Registra** documentos con `mdocs add docs/02-guia.md --title "Guía"`
5. **Valida** con `mdocs check`
6. **Inicia** el visor con `mdocs serve` y abre `http://localhost:3000`
7. **Genera** el PDF y el `.drawio` con `mdocs build`

## Configuración rápida

| Archivo | Propósito |
|---|---|
| `logo.svg` | Logo de la empresa (reemplazar) |
| `brand.config.json` | Colores, nombre, título, descripción, año y "powered by" |
| `docs.config.json` | Lista y orden de documentos |
| `docs/*.md` | Contenido de la documentación |
| `AGENTS.md` | Reglas para agentes de IA que editen el proyecto |

## Características soportadas

- Títulos H1 – H4 con jerarquía visual clara
- Tablas con cabecera de color corporativo
- Bloques de código con resaltado oscuro
- Diagramas Mermaid (flujos, secuencias, estados, clases y más)
- Blockquotes con borde de acento
- Listas de tareas con checkboxes
- Enlaces entre documentos que funcionan en la web y en el PDF
- Separadores horizontales
