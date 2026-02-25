/**
 * generate-styles.ts
 *
 * Lee logo.svg, extrae los colores dominantes y actualiza brand.config.json
 * para que la identidad visual siempre sea acorde al logo.
 *
 * Uso: npm run generate:styles
 */

import fs from 'fs';
import path from 'path';

const ROOT = path.join(__dirname, '..');
const LOGO_PATH = path.join(ROOT, 'logo.svg');
const CONFIG_PATH = path.join(ROOT, 'brand.config.json');

// ── Tipos ─────────────────────────────────────────────────────────────────────

interface Color {
  hex: string;
  r: number;
  g: number;
  b: number;
  luminance: number;
  saturation: number;
}

interface BrandConfig {
  primaryColor: string;
  accentColor: string;
  companyName?: string;
  docsTitle?: string;
  docsYear?: string;
}

// ── Helpers de color ──────────────────────────────────────────────────────────

function hexToRgb(hex: string): { r: number; g: number; b: number } | null {
  let clean = hex.replace('#', '');
  if (clean.length === 3) {
    clean = clean[0] + clean[0] + clean[1] + clean[1] + clean[2] + clean[2];
  }
  if (clean.length !== 6) return null;
  return {
    r: parseInt(clean.slice(0, 2), 16),
    g: parseInt(clean.slice(2, 4), 16),
    b: parseInt(clean.slice(4, 6), 16),
  };
}

/** Luminancia perceptual (0 = negro, 255 = blanco) */
function getLuminance(r: number, g: number, b: number): number {
  return 0.299 * r + 0.587 * g + 0.114 * b;
}

/** Saturación HSL (0 = gris, 1 = color puro) */
function getSaturation(r: number, g: number, b: number): number {
  const rn = r / 255, gn = g / 255, bn = b / 255;
  const max = Math.max(rn, gn, bn);
  const min = Math.min(rn, gn, bn);
  const l = (max + min) / 2;
  if (max === min) return 0;
  const d = max - min;
  return l > 0.5 ? d / (2 - max - min) : d / (max + min);
}

// ── Extracción de colores del SVG ─────────────────────────────────────────────

interface ExtractedColors {
  main: Color[];      // fill/stroke directos (más fiables como colores de marca)
  gradient: Color[];  // stop-color de gradientes (colores de transición)
}

function extractColors(svgContent: string): ExtractedColors {
  const mainSet     = new Set<string>();
  const gradientSet = new Set<string>();

  // Patrones de colores directos (atributos y CSS de clases)
  // Nota: se excluye el patrón genérico "color=" para evitar matches falsos en stop-color
  const mainPatterns = [
    /\bfill="(#[0-9a-fA-F]{3,6})"/g,
    /\bstroke="(#[0-9a-fA-F]{3,6})"/g,
    /\bfill:\s*(#[0-9a-fA-F]{3,6})/g,
    /\bstroke:\s*(#[0-9a-fA-F]{3,6})/g,
  ];

  // Patrones exclusivos de gradientes
  const gradientPatterns = [
    /\bstop-color="(#[0-9a-fA-F]{3,6})"/g,
    /\bstop-color:\s*(#[0-9a-fA-F]{3,6})/g,
  ];

  for (const pattern of mainPatterns) {
    let match: RegExpExecArray | null;
    while ((match = pattern.exec(svgContent)) !== null) {
      mainSet.add(match[1].toLowerCase());
    }
  }

  for (const pattern of gradientPatterns) {
    let match: RegExpExecArray | null;
    while ((match = pattern.exec(svgContent)) !== null) {
      const hex = match[1].toLowerCase();
      // Solo añadir a gradient si no apareció ya en los directos
      if (!mainSet.has(hex)) gradientSet.add(hex);
    }
  }

  function toColorList(set: Set<string>): Color[] {
    const list: Color[] = [];
    for (const hex of set) {
      const rgb = hexToRgb(hex);
      if (!rgb) continue;
      const luminance  = getLuminance(rgb.r, rgb.g, rgb.b);
      const saturation = getSaturation(rgb.r, rgb.g, rgb.b);
      // Filtrar blancos puros (> 245) y negros puros (< 8)
      if (luminance > 245 || luminance < 8) continue;
      list.push({ hex, ...rgb, luminance, saturation });
    }
    return list;
  }

  return { main: toColorList(mainSet), gradient: toColorList(gradientSet) };
}

// ── Selección de colores primario y acento ────────────────────────────────────

function pickColors(extracted: ExtractedColors): { primary: string; accent: string } {
  // Combinamos main + gradient para el display, pero priorizamos main para selección
  const allColors = [...extracted.main, ...extracted.gradient];

  if (allColors.length === 0) {
    console.warn('\n  ⚠  No se encontraron colores aprovechables. Usando colores por defecto.\n');
    return { primary: '#1a202c', accent: '#3182ce' };
  }

  // Primario: el color más oscuro de los directos (fill/stroke); si no hay, usar todos
  const primaryPool = extracted.main.length > 0 ? extracted.main : allColors;
  const primary = [...primaryPool].sort((a, b) => a.luminance - b.luminance)[0];

  // Acento: el color con mayor saturación distinto del primario (de todos los colores)
  const accentCandidates = allColors.filter((c) => c.hex !== primary.hex);

  if (accentCandidates.length === 0) {
    console.warn('  ⚠  Solo se encontró un color. Se usará como primario y acento.');
    return { primary: primary.hex, accent: primary.hex };
  }

  const accent = [...accentCandidates].sort((a, b) => b.saturation - a.saturation)[0];

  return { primary: primary.hex, accent: accent.hex };
}

// ── Main ──────────────────────────────────────────────────────────────────────

console.log('\n  ┌─────────────────────────────────────────┐');
console.log('  │       generate:styles — logo.svg        │');
console.log('  └─────────────────────────────────────────┘\n');

if (!fs.existsSync(LOGO_PATH)) {
  console.error('  ✗  No se encontró logo.svg en la raíz del proyecto.\n');
  process.exit(1);
}

const svgContent = fs.readFileSync(LOGO_PATH, 'utf-8');
const extracted  = extractColors(svgContent);
const allColors  = [...extracted.main, ...extracted.gradient];

if (allColors.length > 0) {
  if (extracted.main.length > 0) {
    console.log('  Colores principales (fill / stroke):\n');
    for (const c of extracted.main) {
      console.log(
        `    ███  ${c.hex}  ` +
        `luminance: ${Math.round(c.luminance).toString().padStart(3)}  ` +
        `saturation: ${c.saturation.toFixed(2)}`
      );
    }
    console.log('');
  }
  if (extracted.gradient.length > 0) {
    console.log('  Colores de gradiente (stop-color):\n');
    for (const c of extracted.gradient) {
      console.log(
        `    ░░░  ${c.hex}  ` +
        `luminance: ${Math.round(c.luminance).toString().padStart(3)}  ` +
        `saturation: ${c.saturation.toFixed(2)}`
      );
    }
    console.log('');
  }
} else {
  console.log('  No se encontraron colores hex en el SVG.\n');
}

const { primary, accent } = pickColors(extracted);

// Leer config existente para conservar otros campos
const existingConfig: BrandConfig = fs.existsSync(CONFIG_PATH)
  ? JSON.parse(fs.readFileSync(CONFIG_PATH, 'utf-8'))
  : {};

const updatedConfig: BrandConfig = {
  ...existingConfig,
  primaryColor: primary,
  accentColor: accent,
};

fs.writeFileSync(CONFIG_PATH, JSON.stringify(updatedConfig, null, 2) + '\n');

console.log('  ✔  brand.config.json actualizado:\n');
console.log(`    primaryColor  →  ${primary}`);
console.log(`    accentColor   →  ${accent}`);
console.log('\n  Reinicia el servidor (npm start) para ver los cambios.\n');
