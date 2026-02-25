# Guía de uso

> Reemplaza este archivo con tu guía de implementación, tutorial o proceso.

Esta sección muestra cómo estructurar una guía paso a paso con instrucciones, código y listas de verificación.

## Paso 1 — Configurar el entorno

Instala las dependencias del proyecto:

```bash
npm install
```

Verifica que tienes Node.js 18 o superior:

```bash
node --version
```

## Paso 2 — Personalizar la identidad corporativa

Reemplaza el archivo `logo.svg` con el logo de tu empresa y ejecuta el script de generación de estilos:

```bash
npm run generate:styles
```

El script detectará automáticamente los colores del logo y actualizará `brand.config.json`.

### Ajuste manual de colores

Si el resultado automático no es exactamente lo que buscas, puedes editar `brand.config.json` directamente:

```json
{
  "primaryColor": "#1a2035",
  "accentColor": "#0066cc",
  "companyName": "Mi Empresa S.A.",
  "docsTitle": "Portal de Documentación",
  "docsYear": "2025"
}
```

## Paso 3 — Registrar documentos

Edita `docs.config.json` para listar tus documentos en el orden que quieres que aparezcan en la barra lateral:

```json
[
  {
    "id": "overview",
    "file": "docs/overview.md",
    "title": "Visión general",
    "subtitle": "Arquitectura y conceptos clave"
  },
  {
    "id": "setup",
    "file": "docs/setup.md",
    "title": "Instalación",
    "subtitle": "Requisitos y configuración inicial"
  }
]
```

## Paso 4 — Iniciar el servidor

```bash
npm start
```

Abre tu navegador en `http://localhost:3000`.

---

## Lista de verificación

- [ ] Logo reemplazado (`logo.svg`)
- [ ] Colores generados (`npm run generate:styles`)
- [ ] `brand.config.json` revisado
- [ ] Documentos registrados en `docs.config.json`
- [ ] Archivos `.md` creados en `docs/`
- [ ] Servidor iniciado correctamente

## Notas importantes

> **Tip:** Para cambiar el puerto del servidor, define la variable de entorno `PORT` antes de iniciar:
> ```bash
> PORT=8080 npm start
> ```

El servidor re-lee los archivos `.md` en cada request, por lo que puedes editar el contenido en caliente sin reiniciar.
