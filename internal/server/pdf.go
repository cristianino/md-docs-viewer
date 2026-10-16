package server

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"

	"github.com/cristianino/mdocs/internal/brand"
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
//  4. Une portada + cuerpo y ajusta el visor (panel de marcadores abierto,
//     etiquetas de página). Si el paso 4 falla, devuelve el cuerpo solo.
func GeneratePDF(baseURL string, cfg *brand.Config) ([]byte, error) {
	ctx, cancel := newChromeContext(context.Background())
	defer cancel()
	ctx, cancelTimeout := context.WithTimeout(ctx, 120*time.Second)
	defer cancelTimeout()

	var headings []headingRef
	var first, body, cover []byte
	err := chromedp.Run(ctx,
		chromedp.Navigate(baseURL+"/print?part=body"),
		chromedp.WaitReady("body"),
		waitFlag("__mermaidDone", 20*time.Second),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('h1,h2,h3,h4,h5,h6'))
			.map(h => ({id: h.id || '', text: (h.textContent || '').trim()}))`, &headings),
		printTo(&first, bodyParams(cfg)),
		chromedp.ActionFunc(func(ctx context.Context) error {
			pages := pageNumbers(first, headings)
			if len(pages) == 0 {
				body = first
				return nil
			}
			js, _ := json.Marshal(pages)
			script := fmt.Sprintf(`(function(p){
				document.querySelectorAll('[data-page-of]').forEach(function(el){
					var n = p[el.getAttribute('data-page-of')];
					if (n) el.textContent = n;
				});
				return true;
			})(%s)`, js)
			var ok bool
			if err := chromedp.Evaluate(script, &ok).Do(ctx); err != nil {
				return err
			}
			return printTo(&body, bodyParams(cfg)).Do(ctx)
		}),
		chromedp.Navigate(baseURL+"/print?part=cover"),
		chromedp.WaitReady("body"),
		printTo(&cover, coverParams()),
	)
	if err != nil {
		return nil, fmt.Errorf("generando PDF: %w", err)
	}
	out, err := pdfnav.Finalize(body, pdfnav.Options{Cover: cover})
	if err != nil {
		fmt.Fprintln(os.Stderr, "pdf: sin portada separada ni ajustes de visor:", err)
		return body, nil
	}
	return out, nil
}

func waitFlag(name string, max time.Duration) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		deadline := time.Now().Add(max)
		for time.Now().Before(deadline) {
			var done bool
			if err := chromedp.Evaluate(`window.`+name+` === true`, &done).Do(ctx); err == nil && done {
				return nil
			}
			time.Sleep(200 * time.Millisecond)
		}
		return nil // continuar aunque no termine (p. ej. sin diagramas)
	}
}

func printTo(out *[]byte, params *page.PrintToPDFParams) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		buf, _, err := params.Do(ctx)
		*out = buf
		return err
	}
}

func a4() *page.PrintToPDFParams {
	return page.PrintToPDF().
		WithPrintBackground(true).
		WithPaperWidth(8.27).  // A4 ancho en pulgadas
		WithPaperHeight(11.69) // A4 alto en pulgadas
}

func bodyParams(cfg *brand.Config) *page.PrintToPDFParams {
	return a4().
		WithMarginTop(0.98). // 25mm
		WithMarginBottom(0.98).
		WithMarginLeft(0.79). // 20mm
		WithMarginRight(0.79).
		WithDisplayHeaderFooter(true).
		WithHeaderTemplate(`<span></span>`).
		WithFooterTemplate(footerTemplate(cfg)).
		WithGenerateTaggedPDF(true).
		WithGenerateDocumentOutline(true)
}

func coverParams() *page.PrintToPDFParams {
	return a4().
		WithMarginTop(0).
		WithMarginBottom(0).
		WithMarginLeft(0).
		WithMarginRight(0).
		WithPageRanges("1")
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
