package server

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"mime"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"

	"github.com/cristianino/mdocs/internal/brand"
	"github.com/cristianino/mdocs/internal/docs"
	"github.com/cristianino/mdocs/internal/pdfnav"
)

type headingRef struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// GeneratePDF abre {baseURL}/print en Chrome headless y devuelve el PDF.
//
//  1. Imprime el cuerpo (índice + documentos) con pie de página y marcadores.
//  2. Lee los marcadores para saber en qué página cae cada título, escribe
//     esos números en el índice y vuelve a imprimir.
//  3. Imprime la portada aparte, sin márgenes ni pie.
//  4. Une portada + cuerpo, adjunta los archivos descargables enlazados
//     desde los documentos y ajusta el visor (panel de marcadores abierto,
//     etiquetas de página). Si el paso 4 falla, devuelve el cuerpo solo.
func GeneratePDF(baseURL string, cfg *brand.Config) ([]byte, error) {
	ctx, cancel := newChromeContext(context.Background())
	defer cancel()
	ctx, cancelTimeout := context.WithTimeout(ctx, 120*time.Second)
	defer cancelTimeout()

	fail := func(err error) ([]byte, error) { return nil, fmt.Errorf("generando PDF: %w", err) }

	// 1. cuerpo: índice + documentos, con pie y marcadores
	if err := chromedp.Do(ctx,
		chromedp.Navigate(baseURL+"/print?part=body"),
		chromedp.WaitReady(chromedp.CSS("body")),
		waitFlag("__mermaidDone", 20*time.Second),
	); err != nil {
		return fail(err)
	}
	headings, err := chromedp.Run(ctx, chromedp.Evaluate[[]headingRef](
		`Array.from(document.querySelectorAll('h1,h2,h3,h4,h5,h6'))
			.map(h => ({id: h.id || '', text: (h.textContent || '').trim()}))`))
	if err != nil {
		return fail(err)
	}
	body, err := chromedp.Run(ctx, chromedp.PrintToPDF(bodyOptions(cfg)...))
	if err != nil {
		return fail(err)
	}

	// 2. números de página en el índice y segunda impresión
	if pages := pageNumbers(body, headings); len(pages) > 0 {
		js, _ := json.Marshal(pages)
		script := fmt.Sprintf(`(function(p){
			document.querySelectorAll('[data-page-of]').forEach(function(el){
				var n = p[el.getAttribute('data-page-of')];
				if (n) el.textContent = n;
			});
			return true;
		})(%s)`, js)
		if _, err := chromedp.Run(ctx, chromedp.Evaluate[bool](script)); err != nil {
			return fail(err)
		}
		if body, err = chromedp.Run(ctx, chromedp.PrintToPDF(bodyOptions(cfg)...)); err != nil {
			return fail(err)
		}
	}

	// 3. portada sin márgenes ni pie
	if err := chromedp.Do(ctx,
		chromedp.Navigate(baseURL+"/print?part=cover"),
		chromedp.WaitReady(chromedp.CSS("body")),
	); err != nil {
		return fail(err)
	}
	cover, err := chromedp.Run(ctx, chromedp.PrintToPDF(coverOptions()...))
	if err != nil {
		return fail(err)
	}

	// 4. unir y ajustar el visor
	out, err := pdfnav.Finalize(body, pdfnav.Options{Cover: cover, Attachments: pdfAttachments()})
	if err != nil {
		fmt.Fprintln(os.Stderr, "pdf: sin portada separada ni ajustes de visor:", err)
		return body, nil
	}
	return out, nil
}

// waitFlag espera hasta max a que window.<name> sea true. No falla si no
// llega (p. ej. una página sin diagramas).
func waitFlag(name string, max time.Duration) chromedp.Action[chromedp.Void] {
	return chromedp.Func(func(ctx context.Context, t *chromedp.Target) error {
		deadline := time.Now().Add(max)
		for time.Now().Before(deadline) {
			if done, err := chromedp.Evaluate[bool](`window.`+name+` === true`)(ctx, t); err == nil && done {
				return nil
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(200 * time.Millisecond):
			}
		}
		return nil
	})
}

// A4 con fondo: 25 mm arriba y abajo, 20 mm a los lados.
func bodyOptions(cfg *brand.Config) []chromedp.PDFOption {
	return []chromedp.PDFOption{
		chromedp.PDFPaper(chromedp.PaperA4),
		chromedp.PDFPrintBackground(),
		chromedp.PDFMargins(0.98, 0.79, 0.98, 0.79), // arriba, derecha, abajo, izquierda (pulgadas)
		chromedp.PDFHeaderTemplate(`<span></span>`),
		chromedp.PDFFooterTemplate(footerTemplate(cfg)),
		chromedp.PDFOutlineAndTagged(),
	}
}

func coverOptions() []chromedp.PDFOption {
	return []chromedp.PDFOption{
		chromedp.PDFPaper(chromedp.PaperA4),
		chromedp.PDFPrintBackground(),
		chromedp.PDFMargins(0, 0, 0, 0),
		chromedp.PDFPageRanges("1"),
	}
}

func footerTemplate(cfg *brand.Config) string {
	left := html.EscapeString(cfg.CompanyName + " · " + cfg.DocsTitle)
	return `<div style="width:100%;font-family:-apple-system,'Segoe UI',Roboto,Arial,sans-serif;` +
		`font-size:8px;color:#a0aec0;padding:0 20mm;display:flex;justify-content:space-between;">` +
		`<span>` + left + `</span>` +
		`<span><span class="pageNumber"></span> / <span class="totalPages"></span></span></div>`
}

// pageNumbers relaciona los títulos del HTML con los marcadores del PDF
// (ambos en orden de documento) y devuelve id -> página.
func pageNumbers(pdf []byte, headings []headingRef) map[string]int {
	marks, err := pdfnav.Outline(pdf)
	if err != nil {
		fmt.Fprintln(os.Stderr, "pdf: sin números de página en el índice:", err)
		return nil
	}
	norm := func(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }
	out := map[string]int{}
	j := 0
	for _, h := range headings {
		// avanzar hasta el marcador con el mismo texto (Chrome omite títulos vacíos)
		k := j
		for k < len(marks) && norm(marks[k].Title) != norm(h.Text) {
			k++
		}
		if k == len(marks) {
			continue
		}
		if h.ID != "" && marks[k].Page > 0 {
			out[h.ID] = marks[k].Page
		}
		j = k + 1
	}
	return out
}

// pdfAttachments lee los archivos descargables enlazados desde los
// documentos para incrustarlos en el PDF. Omite (con aviso) los que no
// existen o pasan de docs.MaxAttachmentSize.
func pdfAttachments() []pdfnav.Attachment {
	_, docList, err := LoadState()
	if err != nil {
		return nil
	}
	var out []pdfnav.Attachment
	used := map[string]bool{}
	for _, rel := range LinkedFiles(docList) {
		st, err := os.Stat(filepath.FromSlash(rel))
		if err != nil || st.IsDir() {
			fmt.Fprintln(os.Stderr, "pdf: adjunto no encontrado:", rel)
			continue
		}
		if st.Size() > docs.MaxAttachmentSize {
			fmt.Fprintf(os.Stderr, "pdf: %s pesa más de %d MB; no se adjunta\n", rel, docs.MaxAttachmentSize>>20)
			continue
		}
		data, err := os.ReadFile(filepath.FromSlash(rel))
		if err != nil {
			continue
		}
		name := docs.AttachmentName(rel)
		if used[name] {
			ext := path.Ext(name)
			for n := 2; used[name]; n++ {
				name = fmt.Sprintf("%s-%d%s", strings.TrimSuffix(docs.AttachmentName(rel), ext), n, ext)
			}
		}
		used[name] = true
		mt, _, _ := mime.ParseMediaType(mime.TypeByExtension(path.Ext(rel)))
		if mt == "" && strings.EqualFold(path.Ext(rel), ".json") {
			mt = "application/json"
		}
		out = append(out, pdfnav.Attachment{Name: name, Description: rel, MIME: mt, Data: data})
	}
	return out
}
