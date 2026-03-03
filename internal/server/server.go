package server

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

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
	mux.HandleFunc("/doc/", handleDoc)
	mux.HandleFunc("/print", handlePrint)
	mux.HandleFunc("/pdf", handlePDF)

	cfg, _ := brand.Load("brand.config.json")
	printBanner(cfg, p)

	return http.ListenAndServe(fmt.Sprintf(":%d", p), mux)
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
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Write(data)
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
	htmlContent, err := docs.Render(raw)
	if err != nil {
		http.Error(w, "Error al renderizar: "+err.Error(), http.StatusInternalServerError)
		return
	}

	html, err := renderPage(cfg, docList, id, htmlContent)
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
	html, err := renderPrint(cfg, docList)
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

	pdfBytes, err := GeneratePDF(port)
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
