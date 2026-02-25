import express from 'express';
import { marked } from 'marked';
import fs from 'fs';
import path from 'path';

const app = express();
const PORT = process.env.PORT ? parseInt(process.env.PORT) : 3000;
const ROOT = path.join(__dirname, '..');

// ── Interfaces ────────────────────────────────────────────────────────────────

interface BrandConfig {
  primaryColor: string;
  accentColor: string;
  companyName: string;
  docsTitle: string;
  docsYear: string;
}

interface Doc {
  id: string;
  file: string;
  title: string;
  subtitle: string;
}

// ── Carga de configuración ────────────────────────────────────────────────────

function loadConfig<T>(filePath: string, fallback: T): T {
  try {
    return JSON.parse(fs.readFileSync(filePath, 'utf-8')) as T;
  } catch {
    console.warn(`  Advertencia: no se pudo leer ${path.basename(filePath)}. Usando valores por defecto.`);
    return fallback;
  }
}

const brand = loadConfig<BrandConfig>(path.join(ROOT, 'brand.config.json'), {
  primaryColor: '#1a202c',
  accentColor: '#3182ce',
  companyName: 'Mi Empresa',
  docsTitle: 'Documentación',
  docsYear: new Date().getFullYear().toString(),
});

const docs = loadConfig<Doc[]>(path.join(ROOT, 'docs.config.json'), []);

// ── Helpers de color ──────────────────────────────────────────────────────────

function hexToRgba(hex: string, alpha: number): string {
  const clean = hex.replace('#', '');
  const r = parseInt(clean.slice(0, 2), 16);
  const g = parseInt(clean.slice(2, 4), 16);
  const b = parseInt(clean.slice(4, 6), 16);
  return `rgba(${r}, ${g}, ${b}, ${alpha})`;
}

// ── Renderizado de página ─────────────────────────────────────────────────────

function renderPage(activeDoc: Doc, htmlContent: string): string {
  const { primaryColor, accentColor, companyName, docsTitle, docsYear } = brand;

  const accentBg   = hexToRgba(accentColor, 0.15);
  const accentFaint = hexToRgba(accentColor, 0.06);

  const navItems = docs
    .map(
      (doc) => `
      <a href="/doc/${doc.id}" class="nav-item${doc.id === activeDoc.id ? ' active' : ''}">
        <span class="nav-title">${doc.title}</span>
        <span class="nav-subtitle">${doc.subtitle}</span>
      </a>`,
    )
    .join('');

  return `<!DOCTYPE html>
<html lang="es">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>${activeDoc.title} — ${companyName}</title>
  <script src="https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.min.js"></script>
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
      background: #f0f2f5;
      color: #2d3748;
      display: flex;
      min-height: 100vh;
    }

    /* ── SIDEBAR ── */
    .sidebar {
      width: 268px;
      min-height: 100vh;
      background: ${primaryColor};
      display: flex;
      flex-direction: column;
      position: fixed;
      top: 0; left: 0; bottom: 0;
      z-index: 10;
    }

    .sidebar-logo {
      background: #ffffff;
      padding: 18px 22px 14px;
    }

    .sidebar-logo img {
      display: block;
      height: 36px;
      width: auto;
    }

    .sidebar-logo-label {
      margin-top: 8px;
      font-size: 9.5px;
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: 0.12em;
      color: ${primaryColor};
    }

    .sidebar-nav {
      flex: 1;
      padding: 20px 10px;
      overflow-y: auto;
    }

    .nav-section-label {
      font-size: 9.5px;
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: 0.14em;
      color: rgba(255, 255, 255, 0.3);
      padding: 0 12px;
      margin-bottom: 8px;
    }

    .nav-item {
      display: flex;
      flex-direction: column;
      padding: 11px 14px;
      border-radius: 8px;
      text-decoration: none;
      color: rgba(255, 255, 255, 0.6);
      margin-bottom: 3px;
      transition: background 0.15s, color 0.15s, border-color 0.15s;
      border-left: 3px solid transparent;
    }

    .nav-item:hover {
      background: rgba(255, 255, 255, 0.07);
      color: rgba(255, 255, 255, 0.9);
    }

    .nav-item.active {
      background: ${accentBg};
      border-left-color: ${accentColor};
      color: #ffffff;
    }

    .nav-title {
      font-size: 13.5px;
      font-weight: 600;
      line-height: 1.3;
    }

    .nav-subtitle {
      font-size: 11px;
      opacity: 0.6;
      margin-top: 3px;
    }

    .sidebar-footer {
      padding: 14px 22px;
      font-size: 11px;
      color: rgba(255, 255, 255, 0.22);
      border-top: 1px solid rgba(255, 255, 255, 0.06);
    }

    .sidebar-footer a {
      color: rgba(255, 255, 255, 0.35);
      text-decoration: none;
      transition: color 0.15s;
    }
    .sidebar-footer a:hover { color: rgba(255, 255, 255, 0.65); }

    /* ── MAIN ── */
    .main {
      margin-left: 268px;
      flex: 1;
      padding: 48px 56px;
      min-height: 100vh;
    }

    .content {
      max-width: 900px;
      margin: 0 auto;
    }

    /* ── TYPOGRAPHY ── */
    .content h1 {
      font-size: 2rem;
      font-weight: 800;
      color: ${primaryColor};
      margin-bottom: 6px;
      padding-bottom: 14px;
      border-bottom: 3px solid ${accentColor};
      line-height: 1.25;
    }

    .content h2 {
      font-size: 1.35rem;
      font-weight: 700;
      color: ${primaryColor};
      margin-top: 44px;
      margin-bottom: 14px;
      padding-bottom: 9px;
      border-bottom: 1px solid #e2e8f0;
    }

    .content h3 {
      font-size: 1.1rem;
      font-weight: 700;
      color: #2d3748;
      margin-top: 26px;
      margin-bottom: 10px;
    }

    .content h4 {
      font-size: 1rem;
      font-weight: 600;
      color: #4a5568;
      margin-top: 18px;
      margin-bottom: 8px;
    }

    .content p {
      line-height: 1.8;
      color: #4a5568;
      margin-bottom: 14px;
    }

    .content strong { color: #2d3748; font-weight: 700; }
    .content em { color: #718096; }

    .content a {
      color: ${accentColor};
      text-decoration: none;
      border-bottom: 1px solid transparent;
      transition: border-color 0.15s;
    }
    .content a:hover { border-bottom-color: ${accentColor}; }

    /* ── LISTS ── */
    .content ul, .content ol {
      padding-left: 26px;
      margin-bottom: 14px;
    }
    .content li {
      line-height: 1.75;
      color: #4a5568;
      margin-bottom: 4px;
    }
    .content li p { margin-bottom: 4px; }

    /* ── HR ── */
    .content hr {
      border: none;
      border-top: 2px solid #e2e8f0;
      margin: 38px 0;
    }

    /* ── BLOCKQUOTE ── */
    .content blockquote {
      border-left: 4px solid ${accentColor};
      background: ${accentFaint};
      padding: 14px 18px;
      border-radius: 0 8px 8px 0;
      margin: 18px 0;
      color: #4a5568;
      font-size: 14.5px;
    }
    .content blockquote p { margin-bottom: 0; }

    /* ── TABLES ── */
    .content table {
      width: 100%;
      border-collapse: collapse;
      margin: 22px 0;
      font-size: 13.5px;
      background: #ffffff;
      border-radius: 10px;
      overflow: hidden;
      box-shadow: 0 1px 4px rgba(0,0,0,0.08);
    }
    .content th {
      background: ${primaryColor};
      color: #ffffff;
      padding: 10px 14px;
      text-align: left;
      font-size: 11px;
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: 0.06em;
    }
    .content td {
      padding: 10px 14px;
      border-bottom: 1px solid #edf2f7;
      color: #4a5568;
      vertical-align: top;
      line-height: 1.55;
    }
    .content tr:last-child td { border-bottom: none; }
    .content tr:nth-child(even) td { background: #f7fafc; }
    .content tbody tr:hover td { background: ${accentFaint}; }

    /* ── CODE ── */
    .content code {
      font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', Consolas, monospace;
      font-size: 13px;
      background: #edf2f7;
      color: ${accentColor};
      padding: 2px 6px;
      border-radius: 4px;
    }
    .content pre {
      background: #1a202c;
      border-radius: 10px;
      padding: 22px;
      overflow-x: auto;
      margin: 18px 0;
      box-shadow: 0 2px 8px rgba(0,0,0,0.15);
    }
    .content pre code {
      background: none;
      color: #e2e8f0;
      padding: 0;
      font-size: 13.5px;
      line-height: 1.65;
    }

    /* ── MERMAID ── */
    .mermaid {
      background: #ffffff;
      border-radius: 10px;
      padding: 28px 20px;
      margin: 22px 0;
      box-shadow: 0 1px 4px rgba(0,0,0,0.08);
      text-align: center;
      overflow-x: auto;
    }

    /* ── CHECKBOXES ── */
    .content input[type="checkbox"] {
      margin-right: 6px;
      accent-color: ${accentColor};
    }

    /* ── SCROLLBAR ── */
    ::-webkit-scrollbar { width: 6px; height: 6px; }
    ::-webkit-scrollbar-track { background: transparent; }
    ::-webkit-scrollbar-thumb { background: rgba(0,0,0,0.18); border-radius: 3px; }
  </style>
</head>
<body>
  <aside class="sidebar">
    <div class="sidebar-logo">
      <img src="/logo.svg" alt="${companyName}" />
      <div class="sidebar-logo-label">${docsTitle}</div>
    </div>
    <nav class="sidebar-nav">
      <div class="nav-section-label">Documentos</div>
      ${navItems}
    </nav>
    <div class="sidebar-footer">
      <div>${companyName} &mdash; ${docsYear}</div>
      <div style="margin-top:6px;">Powered by <a href="https://laratec.co/" target="_blank" rel="noopener">Laratec SAS</a></div>
    </div>
  </aside>

  <main class="main">
    <div class="content">
      ${htmlContent}
    </div>
  </main>

  <script>
    mermaid.initialize({
      startOnLoad: false,
      theme: 'neutral',
      securityLevel: 'loose',
      flowchart: { useMaxWidth: true, htmlLabels: true },
      sequence: { useMaxWidth: true }
    });

    document.addEventListener('DOMContentLoaded', function () {
      // Convierte bloques de código mermaid en divs renderizables
      document.querySelectorAll('pre code.language-mermaid').forEach(function (block) {
        var pre = block.parentElement;
        var div = document.createElement('div');
        div.className = 'mermaid';
        div.textContent = block.textContent || '';
        if (pre && pre.parentNode) {
          pre.parentNode.replaceChild(div, pre);
        }
      });
      mermaid.run();
    });
  </script>
</body>
</html>`;
}

// ── Rutas ─────────────────────────────────────────────────────────────────────

app.get('/', (_req, res) => {
  const first = docs[0];
  if (first) {
    res.redirect(`/doc/${first.id}`);
  } else {
    res.status(404).send('<h1>No hay documentos configurados en docs.config.json</h1>');
  }
});

app.get('/logo.svg', (_req, res) => {
  res.sendFile(path.join(ROOT, 'logo.svg'));
});

app.get('/doc/:id', (req, res) => {
  const doc = docs.find((d) => d.id === req.params.id);
  if (!doc) {
    res.status(404).send('<h1>Documento no encontrado</h1>');
    return;
  }
  try {
    const filePath = path.join(ROOT, doc.file);
    const mdContent = fs.readFileSync(filePath, 'utf-8');
    const htmlContent = marked.parse(mdContent) as string;
    res.send(renderPage(doc, htmlContent));
  } catch {
    res.status(500).send('<h1>Error al leer el documento</h1>');
  }
});

// ── Inicio ────────────────────────────────────────────────────────────────────

app.listen(PORT, () => {
  console.log('\n  ┌─────────────────────────────────────────┐');
  console.log(`  │   ${brand.companyName.padEnd(33)}   │`);
  console.log(`  │   ${brand.docsTitle.padEnd(33)}   │`);
  console.log('  └─────────────────────────────────────────┘');
  console.log(`\n  http://localhost:${PORT}\n`);
});
