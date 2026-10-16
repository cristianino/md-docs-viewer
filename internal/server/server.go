package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/cristianino/mdocs/assets"
	"github.com/cristianino/mdocs/internal/brand"
	"github.com/cristianino/mdocs/internal/docs"
)

// ExportsMode decide cómo se sirven /pdf y /drawio.
type ExportsMode string

const (
	// ExportsLive genera los archivos al pedirlos (requiere Chrome). Se
	// cachean hasta que cambie algún archivo del proyecto.
	ExportsLive ExportsMode = "live"
	// ExportsStatic sirve los archivos ya generados en Dist (mdocs build).
	// No requiere Chrome en el servidor.
	ExportsStatic ExportsMode = "static"
	// ExportsOff oculta los botones y responde 404.
	ExportsOff ExportsMode = "off"
)

// Options es la configuración del servidor.
type Options struct {
	Host     string      // interfaz; "" = todas
	Port     int         // puerto
	BasePath string      // prefijo de URLs, p. ej. "/docs"; "" = raíz
	Exports  ExportsMode // live | static | off
	Dist     string      // carpeta de archivos generados (modo static)
}

var opts = Options{Port: 3000, Exports: ExportsLive, Dist: "dist"}

// NormalizeBasePath deja el prefijo como "/algo" (sin barra final) o "".
func NormalizeBasePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" || p == "/" {
		return ""
	}
	return "/" + strings.Trim(path.Clean("/"+p), "/")
}

// Validate revisa y normaliza las opciones.
func (o *Options) Validate() error {
	o.BasePath = NormalizeBasePath(o.BasePath)
	switch o.Exports {
	case ExportsLive, ExportsStatic, ExportsOff:
	case "":
		o.Exports = ExportsLive
	default:
		return fmt.Errorf("exports inválido %q (usa live, static u off)", o.Exports)
	}
	if o.Port < 0 || o.Port > 65535 {
		return fmt.Errorf("puerto inválido %d", o.Port)
	}
	if o.Dist == "" {
		o.Dist = "dist"
	}
	return nil
}

func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/healthz", handleHealth)
	mux.HandleFunc("/logo.svg", handleLogo)
	mux.HandleFunc("/assets/mermaid.min.js", handleMermaidJS)
	mux.HandleFunc("/doc/", handleDoc)
	mux.HandleFunc("/indice", handleIndex)
	mux.HandleFunc("/print", handlePrint)
	mux.HandleFunc("/pdf", handlePDF)
	mux.HandleFunc("/drawio", handleDrawio)
	mux.HandleFunc("/mermaid-export", handleMermaidExport)
	mux.HandleFunc("/files/", handleFiles)
	return mux
}

// handler monta el mux bajo BasePath y agrega cabeceras de seguridad.
func handler() http.Handler {
	var h http.Handler = newMux()
	if opts.BasePath != "" {
		inner := http.StripPrefix(opts.BasePath, h)
		h = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == opts.BasePath:
				http.Redirect(w, r, opts.BasePath+"/", http.StatusMovedPermanently)
			case r.URL.Path == "/healthz":
				handleHealth(w, r) // fuera del prefijo, para balanceadores
			case strings.HasPrefix(r.URL.Path, opts.BasePath+"/"):
				inner.ServeHTTP(w, r)
			default:
				http.NotFound(w, r)
			}
		})
	}
	return securityHeaders(h)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("X-Frame-Options", "SAMEORIGIN")
		next.ServeHTTP(w, r)
	})
}

// Start arranca el servidor y lo detiene limpiamente con SIGINT/SIGTERM.
func Start(o Options) error {
	if err := o.Validate(); err != nil {
		return err
	}
	opts = o
	if opts.Exports == ExportsLive {
		warmCheckChrome()
	}
	srv := &http.Server{
		Addr:              net.JoinHostPort(opts.Host, fmt.Sprint(opts.Port)),
		Handler:           handler(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	cfg, _ := brand.Load("brand.config.json")
	printBanner(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		fmt.Println("\n  deteniendo…")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

// Ephemeral arranca el servidor en un puerto libre de 127.0.0.1 (para
// generar PDF/draw.io desde la CLI) y devuelve su URL base y cómo pararlo.
func Ephemeral() (string, func(), error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}
	opts = Options{Host: "127.0.0.1", Port: ln.Addr().(*net.TCPAddr).Port, Exports: ExportsLive, Dist: "dist"}
	srv := &http.Server{Handler: handler(), ReadHeaderTimeout: 10 * time.Second}
	go srv.Serve(ln)
	return baseURL(), func() { srv.Close() }, nil
}

// baseURL es la URL con la que Chrome (en este mismo equipo) llega al servidor.
func baseURL() string {
	host := opts.Host
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, fmt.Sprint(opts.Port)) + opts.BasePath
}

// pageLinks son las URLs que usan las plantillas.
type pageLinks struct {
	Base      string // prefijo de todas las URLs
	Slash     string // "/" al final de las páginas en el sitio estático
	PDFURL    string // vacío = sin botón
	DrawioURL string // vacío = sin botón
}

func serverLinks() pageLinks {
	l := pageLinks{Base: opts.BasePath}
	if opts.Exports != ExportsOff {
		l.PDFURL = opts.BasePath + "/pdf"
		l.DrawioURL = opts.BasePath + "/drawio"
	}
	return l
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	fmt.Fprintln(w, "ok")
}

// handleFiles sirve archivos del proyecto (imágenes referenciadas desde los
// .md). Solo rutas relativas dentro de la carpeta actual y nunca .json.
func handleFiles(w http.ResponseWriter, r *http.Request) {
	rel := path.Clean(strings.TrimPrefix(r.URL.Path, "/files/"))
	if rel == "." || strings.HasPrefix(rel, "..") || path.IsAbs(rel) ||
		strings.HasSuffix(strings.ToLower(rel), ".json") || strings.HasPrefix(path.Base(rel), ".") {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filepath.FromSlash(rel))
}

// LoadState lee brand.config.json y docs.config.json de la carpeta actual.
func LoadState() (*brand.Config, []*docs.Doc, error) {
	cfg, err := brand.Load("brand.config.json")
	if err != nil {
		return nil, nil, err
	}
	docList, err := docs.Load("docs.config.json")
	if err != nil {
		return nil, nil, err
	}
	return cfg, docList, nil
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	_, docList, err := LoadState()
	if err != nil || len(docList) == 0 {
		http.Error(w, "No hay documentos configurados.", http.StatusNotFound)
		return
	}
	http.Redirect(w, r, opts.BasePath+"/doc/"+docList[0].ID, http.StatusFound)
}

func handleLogo(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("logo.svg")
	if err != nil {
		data = assets.DefaultLogo
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Write(data)
}

func handleMermaidJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(assets.MermaidJS)
}

func writeHTML(w http.ResponseWriter, html string, err error) {
	if err != nil {
		http.Error(w, "Error de template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	cfg, docList, err := LoadState()
	if err != nil {
		http.Error(w, "Error de configuración: "+err.Error(), http.StatusInternalServerError)
		return
	}
	html, err := renderIndex(cfg, docList, serverLinks())
	writeHTML(w, html, err)
}

func handleDoc(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/doc/"), "/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	cfg, docList, err := LoadState()
	if err != nil {
		http.Error(w, "Error de configuración: "+err.Error(), http.StatusInternalServerError)
		return
	}
	var doc *docs.Doc
	for _, d := range docList {
		if d.ID == id {
			doc = d
			break
		}
	}
	if doc == nil {
		http.NotFound(w, r)
		return
	}
	html, err := renderDocPage(cfg, docList, doc, serverLinks())
	if errors.Is(err, os.ErrNotExist) {
		http.Error(w, "Documento no encontrado: "+doc.File, http.StatusNotFound)
		return
	}
	writeHTML(w, html, err)
}

func handlePrint(w http.ResponseWriter, r *http.Request) {
	cfg, docList, err := LoadState()
	if err != nil {
		http.Error(w, "Error de configuración: "+err.Error(), http.StatusInternalServerError)
		return
	}
	html, err := renderPrint(cfg, docList, PrintPart(r.URL.Query().Get("part")), opts.BasePath)
	writeHTML(w, html, err)
}

// latestFile devuelve el archivo más reciente de dir con la extensión dada.
func latestFile(dir, ext string) (string, error) {
	matches, _ := filepath.Glob(filepath.Join(dir, "*"+ext))
	if len(matches) == 0 {
		return "", fmt.Errorf("no hay archivos %s en %s (ejecuta mdocs build)", ext, dir)
	}
	sort.Slice(matches, func(i, j int) bool {
		a, _ := os.Stat(matches[i])
		b, _ := os.Stat(matches[j])
		return a.ModTime().After(b.ModTime())
	})
	return matches[0], nil
}

func serveDownload(w http.ResponseWriter, contentType, filename string, data []byte) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(data)
}

func handlePDF(w http.ResponseWriter, r *http.Request) {
	switch opts.Exports {
	case ExportsOff:
		http.NotFound(w, r)
		return
	case ExportsStatic:
		p, err := latestFile(opts.Dist, ".pdf")
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(p)))
		http.ServeFile(w, r, p)
		return
	}
	cfg, _, err := LoadState()
	if err != nil {
		http.Error(w, "Error de configuración: "+err.Error(), http.StatusInternalServerError)
		return
	}
	data, err := cachedExport("pdf", func() ([]byte, error) { return GeneratePDF(baseURL(), cfg) })
	if err != nil {
		http.Error(w, "Error generando PDF: "+err.Error(), http.StatusInternalServerError)
		return
	}
	serveDownload(w, "application/pdf", PDFFilename(cfg), data)
}

// PDFFilename genera el nombre de archivo normalizado para el PDF.
func PDFFilename(cfg *brand.Config) string {
	return slugFilename(fmt.Sprintf("%s-%s-%s", cfg.CompanyName, cfg.DocsTitle, time.Now().Format("2006-01-02"))) + ".pdf"
}

func printBanner(cfg *brand.Config) {
	host := opts.Host
	if host == "" {
		host = "localhost"
	}
	fmt.Printf("\n  %s — %s\n", cfg.CompanyName, cfg.DocsTitle)
	fmt.Printf("  http://%s%s/\n", net.JoinHostPort(host, fmt.Sprint(opts.Port)), opts.BasePath)
	fmt.Printf("  exports: %s", opts.Exports)
	if opts.Exports == ExportsStatic {
		fmt.Printf(" (desde %s/)", opts.Dist)
	}
	fmt.Printf(" · health: /healthz\n\n")
}
