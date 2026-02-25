import express from 'express';
import { marked } from 'marked';
import fs from 'fs';
import path from 'path';
import puppeteer from 'puppeteer';

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
  poweredBy?: boolean;
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

    /* ── PDF BUTTON ── */
    .sidebar-pdf {
      padding: 0 10px 14px;
    }

    .pdf-btn {
      display: block;
      padding: 11px 14px;
      background: ${accentColor};
      color: #ffffff;
      border-radius: 8px;
      text-align: center;
      text-decoration: none;
      font-size: 13px;
      font-weight: 600;
      letter-spacing: 0.01em;
      transition: opacity 0.15s;
    }
    .pdf-btn:hover { opacity: 0.85; }
    .pdf-btn.loading { opacity: 0.6; pointer-events: none; }

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
    <div class="sidebar-pdf">
      <a href="/pdf" class="pdf-btn" id="pdfBtn">Descargar PDF</a>
    </div>
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
      // Botón PDF: feedback visual mientras genera
      var pdfBtn = document.getElementById('pdfBtn');
      if (pdfBtn) {
        pdfBtn.addEventListener('click', function () {
          pdfBtn.textContent = 'Generando…';
          pdfBtn.classList.add('loading');
          setTimeout(function () {
            pdfBtn.textContent = 'Descargar PDF';
            pdfBtn.classList.remove('loading');
          }, 30000);
        });
      }

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

// ── Página de impresión (portada + índice + todos los docs) ──────────────────

function renderPrintPage(): string {
  const { primaryColor, accentColor, companyName, docsTitle, docsYear, poweredBy = true } = brand;
  const accentFaint = hexToRgba(accentColor, 0.08);
  const date = new Date().toLocaleDateString('es-ES', {
    year: 'numeric', month: 'long', day: 'numeric',
  });

  const tocItems = docs.map((doc, i) => `
    <div class="toc-item">
      <span class="toc-num">${String(i + 1).padStart(2, '0')}</span>
      <div class="toc-info">
        <span class="toc-title">${doc.title}</span>
        <span class="toc-subtitle">${doc.subtitle}</span>
      </div>
    </div>`).join('');

  const sections = docs.map((doc) => {
    let htmlContent = '<p><em>Error al cargar el documento.</em></p>';
    try {
      const md = fs.readFileSync(path.join(ROOT, doc.file), 'utf-8');
      htmlContent = marked.parse(md) as string;
    } catch { /* noop */ }
    return `
    <section class="doc-section">
      <div class="doc-header">
        <div class="doc-header-title">${doc.title}</div>
        <div class="doc-header-subtitle">${doc.subtitle}</div>
      </div>
      <div class="content">${htmlContent}</div>
    </section>`;
  }).join('');

  return `<!DOCTYPE html>
<html lang="es">
<head>
  <meta charset="UTF-8" />
  <title>${companyName} — ${docsTitle}</title>
  <script src="https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.min.js"></script>
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Arial, sans-serif;
      color: #2d3748;
      background: #ffffff;
      font-size: 13.5px;
      line-height: 1.65;
    }

    /* ── PORTADA ── */
    .cover {
      height: 247mm; /* A4 (297mm) - márgenes top/bottom (25mm c/u) */
      display: flex;
      flex-direction: column;
      justify-content: center;
      align-items: center;
      text-align: center;
      break-after: page;
      padding: 40px;
    }
    .cover-logo { margin-bottom: 44px; }
    .cover-logo img { max-height: 90px; width: auto; }
    .cover-company {
      font-size: 2rem;
      font-weight: 800;
      color: ${primaryColor};
      margin-bottom: 10px;
    }
    .cover-title {
      font-size: 1.2rem;
      font-weight: 400;
      color: #4a5568;
      margin-bottom: 36px;
    }
    .cover-line {
      width: 72px;
      height: 4px;
      background: ${accentColor};
      border-radius: 2px;
      margin: 0 auto 36px;
    }
    .cover-date { font-size: 0.85rem; color: #718096; margin-bottom: 12px; }
    .cover-powered { font-size: 0.8rem; color: #a0aec0; }
    .cover-powered a { color: ${accentColor}; text-decoration: none; }

    /* ── ÍNDICE ── */
    .toc-page {
      break-before: page;
      break-after: page;
      padding: 10px 0;
    }
    .toc-heading {
      font-size: 1.5rem;
      font-weight: 800;
      color: ${primaryColor};
      border-bottom: 3px solid ${accentColor};
      padding-bottom: 12px;
      margin-bottom: 28px;
    }
    .toc-item {
      display: flex;
      align-items: flex-start;
      gap: 18px;
      padding: 14px 0;
      border-bottom: 1px solid #edf2f7;
    }
    .toc-item:last-child { border-bottom: none; }
    .toc-num {
      font-size: 1.4rem;
      font-weight: 800;
      color: ${accentColor};
      min-width: 34px;
      line-height: 1;
      padding-top: 2px;
    }
    .toc-info { display: flex; flex-direction: column; gap: 3px; }
    .toc-title { font-size: 0.95rem; font-weight: 700; color: ${primaryColor}; }
    .toc-subtitle { font-size: 0.82rem; color: #718096; }

    /* ── SECCIONES DE DOCUMENTO ── */
    .doc-section { break-before: page; }
    .doc-header {
      background: ${primaryColor};
      color: #ffffff;
      padding: 20px 28px;
      margin-bottom: 30px;
    }
    .doc-header-title { font-size: 1.3rem; font-weight: 800; margin-bottom: 4px; }
    .doc-header-subtitle { font-size: 0.88rem; opacity: 0.7; }

    /* ── TIPOGRAFÍA DE CONTENIDO ── */
    .content h1 {
      font-size: 1.6rem; font-weight: 800; color: ${primaryColor};
      padding-bottom: 11px; border-bottom: 3px solid ${accentColor};
      margin: 24px 0 10px; line-height: 1.25;
    }
    .content h1:first-child { margin-top: 0; }
    .content h2 {
      font-size: 1.15rem; font-weight: 700; color: ${primaryColor};
      margin: 30px 0 11px; padding-bottom: 7px; border-bottom: 1px solid #e2e8f0;
    }
    .content h3 {
      font-size: 1.02rem; font-weight: 700; color: #2d3748;
      margin: 20px 0 7px;
    }
    .content h4 {
      font-size: 0.95rem; font-weight: 600; color: #4a5568;
      margin: 14px 0 5px;
    }
    .content p { line-height: 1.75; color: #4a5568; margin-bottom: 11px; }
    .content strong { color: #2d3748; font-weight: 700; }
    .content em { color: #718096; }
    .content a { color: ${accentColor}; text-decoration: none; }
    .content ul, .content ol { padding-left: 24px; margin-bottom: 11px; }
    .content li { line-height: 1.7; color: #4a5568; margin-bottom: 3px; }
    .content li p { margin-bottom: 3px; }
    .content hr { border: none; border-top: 2px solid #e2e8f0; margin: 26px 0; }

    .content blockquote {
      border-left: 4px solid ${accentColor};
      background: ${accentFaint};
      padding: 11px 15px;
      border-radius: 0 6px 6px 0;
      margin: 14px 0;
      color: #4a5568;
      font-size: 13px;
      break-inside: avoid;
    }
    .content blockquote p { margin-bottom: 0; }

    .content table {
      width: 100%; border-collapse: collapse;
      margin: 16px 0; font-size: 12px;
      break-inside: avoid;
    }
    .content th {
      background: ${primaryColor}; color: #ffffff;
      padding: 8px 11px; text-align: left;
      font-size: 10px; font-weight: 700;
      text-transform: uppercase; letter-spacing: 0.05em;
    }
    .content td {
      padding: 8px 11px; border-bottom: 1px solid #edf2f7;
      color: #4a5568; vertical-align: top;
    }
    .content tr:last-child td { border-bottom: none; }
    .content tr:nth-child(even) td { background: #f7fafc; }

    .content code {
      font-family: 'Cascadia Code', 'Fira Code', Consolas, monospace;
      font-size: 11.5px; background: #edf2f7;
      color: ${accentColor}; padding: 2px 5px; border-radius: 3px;
    }
    .content pre {
      background: #1a202c; border-radius: 7px;
      padding: 17px; overflow-x: auto;
      margin: 12px 0; break-inside: avoid;
    }
    .content pre code {
      background: none; color: #e2e8f0;
      padding: 0; font-size: 12px; line-height: 1.6;
    }

    .mermaid {
      background: #ffffff; border: 1px solid #e2e8f0;
      border-radius: 8px; padding: 22px 14px;
      margin: 14px 0; text-align: center; break-inside: avoid;
    }
    .content input[type="checkbox"] { margin-right: 5px; }
  </style>
</head>
<body>

  <!-- PORTADA -->
  <div class="cover">
    <div class="cover-logo"><img src="/logo.svg" alt="${companyName}" /></div>
    <div class="cover-company">${companyName}</div>
    <div class="cover-title">${docsTitle}</div>
    <div class="cover-line"></div>
    <div class="cover-date">${date}</div>
    ${poweredBy !== false ? `<div class="cover-powered">Powered by <a href="https://laratec.co/">Laratec SAS</a></div>` : ''}
  </div>

  <!-- ÍNDICE -->
  <div class="toc-page">
    <h1 class="toc-heading">Índice</h1>
    ${tocItems}
  </div>

  <!-- DOCUMENTOS -->
  ${sections}

  <script>
    mermaid.initialize({
      startOnLoad: false, theme: 'neutral',
      securityLevel: 'loose',
      flowchart: { useMaxWidth: true, htmlLabels: true },
      sequence: { useMaxWidth: true }
    });
    document.addEventListener('DOMContentLoaded', async function () {
      document.querySelectorAll('pre code.language-mermaid').forEach(function (block) {
        var pre = block.parentElement;
        var div = document.createElement('div');
        div.className = 'mermaid';
        div.textContent = block.textContent || '';
        if (pre && pre.parentNode) pre.parentNode.replaceChild(div, pre);
      });
      await mermaid.run();
      window.__mermaidDone = true;
    });
  </script>
</body>
</html>`;
}

// ── Rutas: print y PDF ────────────────────────────────────────────────────────

app.get('/print', (_req, res) => {
  res.send(renderPrintPage());
});

app.get('/pdf', async (req, res) => {
  let browser;
  try {
    browser = await puppeteer.launch({
      headless: true,
      args: ['--no-sandbox', '--disable-setuid-sandbox'],
    });

    const page = await browser.newPage();
    await page.goto(`http://localhost:${PORT}/print`, {
      waitUntil: 'networkidle0',
      timeout: 30000,
    });

    // Esperar a que Mermaid termine de renderizar
    await page.waitForFunction('window.__mermaidDone === true', { timeout: 10000 })
      .catch(() => { /* si no hay diagramas, ignorar timeout */ });

    const slug = `${brand.companyName} ${brand.docsTitle}`
      .toLowerCase()
      .normalize('NFD').replace(/[\u0300-\u036f]/g, '')
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-+|-+$/g, '');
    const dateStr = new Date().toISOString().slice(0, 10);
    const filename = `${slug}-${dateStr}.pdf`;

    const pdfBuffer = await page.pdf({
      format: 'A4',
      printBackground: true,
      margin: { top: '25mm', right: '20mm', bottom: '25mm', left: '20mm' },
      displayHeaderFooter: true,
      headerTemplate: `<div style="font-family:Arial,sans-serif;font-size:8px;color:#a0aec0;width:100%;padding:0 20mm;text-align:right;box-sizing:border-box;">${brand.companyName} &mdash; ${brand.docsTitle}</div>`,
      footerTemplate: brand.poweredBy !== false
        ? `<div style="font-family:Arial,sans-serif;font-size:8px;color:#a0aec0;width:100%;padding:0 20mm;display:flex;justify-content:space-between;box-sizing:border-box;"><span>Powered by Laratec SAS</span><span><span class="pageNumber"></span> / <span class="totalPages"></span></span></div>`
        : `<div style="font-family:Arial,sans-serif;font-size:8px;color:#a0aec0;width:100%;padding:0 20mm;text-align:right;box-sizing:border-box;"><span class="pageNumber"></span> / <span class="totalPages"></span></div>`,
    });

    await browser.close();

    res.setHeader('Content-Type', 'application/pdf');
    res.setHeader('Content-Disposition', `attachment; filename="${filename}"`);
    res.send(Buffer.from(pdfBuffer));
  } catch (err) {
    if (browser) await browser.close().catch(() => {});
    console.error('  Error al generar PDF:', err);
    res.status(500).send('<h1>Error al generar el PDF</h1><p>Revisa la consola del servidor.</p>');
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
