package server

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/chromedp/chromedp"
)

// ChromePath permite fijar el ejecutable de Chrome/Chromium. Si está vacío
// se usa la variable de entorno MDOCS_CHROME y, si tampoco existe, la
// detección automática de chromedp.
var ChromePath string

// newChromeContext crea un contexto de Chrome headless.
func newChromeContext(parent context.Context) (context.Context, context.CancelFunc) {
	opts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	exec := ChromePath
	if exec == "" {
		exec = os.Getenv("MDOCS_CHROME")
	}
	if exec != "" {
		opts = append(opts, chromedp.ExecPath(exec))
	}
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(parent, opts...)
	ctx, cancel := chromedp.NewContext(allocCtx,
		// Silenciar errores de enum desconocidos en CDP (e.g. IPAddressSpace: Loopback)
		// que aparecen cuando Chrome es más nuevo que cdproto pero son inofensivos.
		chromedp.WithErrorf(func(format string, args ...any) {
			msg := fmt.Sprintf(format, args...)
			if !strings.Contains(msg, "could not unmarshal event") {
				fmt.Fprintln(os.Stderr, "chromedp:", msg)
			}
		}),
	)
	return ctx, func() { cancel(); cancelAlloc() }
}
