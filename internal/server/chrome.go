package server

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
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
	execPath := ChromePath
	if execPath == "" {
		execPath = os.Getenv("MDOCS_CHROME")
	}
	if execPath != "" {
		opts = append(opts, chromedp.ExecPath(execPath))
	}
	// Chrome no arranca como root sin --no-sandbox (Docker, CI). También se
	// puede forzar con MDOCS_NO_SANDBOX=1 (contenedores sin user namespaces).
	if os.Getenv("MDOCS_NO_SANDBOX") == "1" || (runtime.GOOS == "linux" && os.Geteuid() == 0) {
		opts = append(opts, chromedp.NoSandbox)
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

// warmCheckChrome avisa al arrancar si en Linux no se encuentra Chrome, para
// no descubrirlo recién al pulsar "Descargar PDF".
func warmCheckChrome() {
	if ChromePath != "" || os.Getenv("MDOCS_CHROME") != "" || runtime.GOOS != "linux" {
		return
	}
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "headless-shell", "chrome"} {
		if _, err := exec.LookPath(name); err == nil {
			return
		}
	}
	fmt.Fprintln(os.Stderr, "  aviso: no se encontró Chrome/Chromium; /pdf y /drawio fallarán.")
	fmt.Fprintln(os.Stderr, "         Instálalo, usa --chrome/MDOCS_CHROME, o sirve archivos generados con --exports static.")
}
