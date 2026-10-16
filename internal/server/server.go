package server

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/cristianino/mdocs/assets"
	"github.com/cristianino/mdocs/internal/brand"
	"github.com/cristianino/mdocs/internal/docs"
)

var port int

// Start arranca el servidor HTTP en el puerto indicado.
func Start(p int) error {
	port = p

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/logo.svg", handleLogo)
	mux.HandleFunc("/assets/mermaid.min.js", handleMermaidJS)
	mux.HandleFunc("/doc/", handleDoc)
	mux.HandleFunc("/print", handlePrint)
	mux.HandleFunc("/pdf", handlePDF)
	mux.HandleFunc("/drawio", handleDrawio)
	mux.HandleFunc("/mermaid-export", handleMermaidExport)
	mux.HandleFunc("/files/", handleFiles)

	cfg, _ := brand.Load("brand.config.json")
	printBanner(cfg, p)

	return http.ListenAndServe(fmt.Sprintf(":%d", p), mux)
}

func baseURL() string { return fmt.Sprintf("http://localhost:%d", port) }

// handleFiles sirve archivos del proyecto (imágenes referenciadas desde los
// .md). Solo rutas relativas dentro de la carpeta actual.
func handleFiles(w http.ResponseWriter, r *http.Request) {
	rel := path.Clean(strings.TrimPrefix(r.URL.Path, "/files/"))
	if rel == "." || strings.HasPrefix(rel, "..") || path.IsAbs(rel) || strings.HasSuffix(rel, ".json") {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filepath.FromSlash(rel))
}

func loadState() (*brand.Config, []*docs.Doc, error) {
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
	_, docList, err := loadState()
	if err != nil || len(docList) == 0 {
		http.Error(w, "No hay documentos configurados.", http.StatusNotFound)
		return
	}
	http.Redirect(w, r, "/doc/"+docList[0].ID, http.StatusFound)
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

func handleDoc(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/doc/")
	if id == "" {
		http.NotFound(w, r)
		return
	}

	cfg, docList, err := loadState()
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

	raw, err := os.ReadFile(doc.File)
	if err != nil {
		http.Error(w, "Documento no encontrado: "+doc.File, http.StatusNotFound)
		return
	}
	res, err := docs.Render(raw, docs.Options{Mode: docs.ModeWeb, Doc: doc, All: docList})
	if err != nil {
		http.Error(w, "Error al renderizar: "+err.Error(), http.StatusInternalServerError)
		return
	}

	html, err := renderPage(cfg, docList, doc, res)
	if err != nil {
		http.Error(w, "Error de template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}

func handlePrint(w http.ResponseWriter, r *http.Request) {
	cfg, docList, err := loadState()
	if err != nil {
		http.Error(w, "Error de configuración: "+err.Error(), http.StatusInternalServerError)
		return
	}
	html, err := renderPrint(cfg, docList, PrintPart(r.URL.Query().Get("part")))
	if err != nil {
		http.Error(w, "Error de template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}

func handlePDF(w http.ResponseWriter, r *http.Request) {
	cfg, _, err := loadState()
	if err != nil {
		http.Error(w, "Error de configuración: "+err.Error(), http.StatusInternalServerError)
		return
	}

	pdfBytes, err := GeneratePDF(baseURL(), cfg)
	if err != nil {
		http.Error(w, "Error generando PDF: "+err.Error(), http.StatusInternalServerError)
		return
	}

	filename := pdfFilename(cfg)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(pdfBytes)
}

// pdfFilename genera el nombre de archivo normalizado para el PDF.
func pdfFilename(cfg *brand.Config) string {
	date := time.Now().Format("2006-01-02")
	raw := fmt.Sprintf("%s-%s-%s", cfg.CompanyName, cfg.DocsTitle, date)
	// Normalizar: quitar tildes, minúsculas, solo alfanumérico y guiones
	t := transform.Chain(norm.NFD, transform.RemoveFunc(func(r rune) bool {
		return unicode.Is(unicode.Mn, r) // marcas diacríticas
	}), norm.NFC)
	cleaned, _, _ := transform.String(t, raw)
	cleaned = strings.ToLower(cleaned)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	cleaned = re.ReplaceAllString(cleaned, "-")
	cleaned = strings.Trim(cleaned, "-")
	return cleaned + ".pdf"
}

func printBanner(cfg *brand.Config, p int) {
	fmt.Printf("\n  %s — %s\n", cfg.CompanyName, cfg.DocsTitle)
	fmt.Printf("  http://localhost:%d\n\n", p)
}
