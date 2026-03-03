# Introducción

> Reemplaza este archivo con el contenido real de tu documentación. Los archivos `.md` en la carpeta `docs/` son el contenido del visor.

Bienvenido al visor de documentación. Este proyecto convierte archivos **Markdown** en páginas web con identidad corporativa, navegación lateral y soporte para diagramas.

## ¿Cómo funciona?

El servidor lee los archivos `.md` registrados en `docs.config.json` y los renderiza como HTML aplicando los colores definidos en `brand.config.json`.

```
logo.svg  →  ./mdocs generate-styles  →  brand.config.json  →  UI con tus colores
```

## Cómo usar este template

1. **Compila** el binario con `go build -o mdocs .`
2. **Reemplaza** `logo.svg` con el logo de tu empresa (formato SVG)
3. **Ejecuta** `./mdocs generate-styles` para extraer los colores automáticamente
4. **Ajusta** `brand.config.json` si quieres afinar nombre, título o año
5. **Registra** tus documentos en `docs.config.json`
6. **Crea** tus archivos `.md` en la carpeta `docs/`
7. **Inicia** el servidor con `./mdocs serve` y abre `http://localhost:3000`

## Configuración rápida

| Archivo | Propósito |
|---|---|
| `logo.svg` | Logo de la empresa (reemplazar) |
| `brand.config.json` | Colores, nombre y título |
| `docs.config.json` | Lista y orden de documentos |
| `docs/*.md` | Contenido de la documentación |

## Características soportadas

- Títulos H1 – H4 con jerarquía visual clara
- Tablas con cabecera de color corporativo
- Bloques de código con resaltado oscuro
- Diagramas Mermaid (flujos, secuencias, etc.)
- Blockquotes con borde de acento
- Listas de tareas con checkboxes
- Links con subrayado al pasar el cursor
- Separadores horizontales
