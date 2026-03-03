package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// GeneratePDF navega a /print en el servidor local, espera Mermaid y devuelve bytes del PDF.
func GeneratePDF(port int) ([]byte, error) {
	url := fmt.Sprintf("http://localhost:%d/print", port)

	ctx, cancel := chromedp.NewContext(context.Background(),
		// Silenciar errores de enum desconocidos en CDP (e.g. IPAddressSpace: Loopback)
		// que aparecen cuando Chrome es más nuevo que cdproto pero son inofensivos.
		chromedp.WithErrorf(func(format string, args ...any) {
			msg := fmt.Sprintf(format, args...)
			if !strings.Contains(msg, "could not unmarshal event") {
				fmt.Println("chromedp:", msg)
			}
		}),
	)
	defer cancel()

	ctx, cancelTimeout := context.WithTimeout(ctx, 90*time.Second)
	defer cancelTimeout()

	// Esperar que window.__mermaidDone sea true (poll manual)
	waitMermaid := chromedp.ActionFunc(func(ctx context.Context) error {
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			var done bool
			if err := chromedp.Evaluate(`window.__mermaidDone === true`, &done).Do(ctx); err == nil && done {
				return nil
			}
			time.Sleep(200 * time.Millisecond)
		}
		// Si no termina en 15s continuamos igual (puede no haber diagramas)
		return nil
	})

	var pdfBuf []byte
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
		waitMermaid,
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdfBuf, _, err = page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(8.27).   // A4 ancho en pulgadas
				WithPaperHeight(11.69). // A4 alto en pulgadas
				WithMarginTop(0.98).    // 25mm
				WithMarginBottom(0.98).
				WithMarginLeft(0.79). // 20mm
				WithMarginRight(0.79).
				Do(ctx)
			return err
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("generando PDF: %w", err)
	}
	return pdfBuf, nil
}
