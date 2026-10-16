package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/chromedp/chromedp"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/cristianino/mdocs/internal/brand"
	"github.com/cristianino/mdocs/internal/docs"
)

var reMermaidBlock = regexp.MustCompile("(?s)```mermaid\r?\n(.*?)```")

type mermaidBlock struct {
	DocID    string
	DocTitle string
	Index    int // posición dentro del documento (1-based)
	Code     string
}

// ExtractMermaidBlocks devuelve los bloques ```mermaid de todos los documentos.
func ExtractMermaidBlocks(docList []*docs.Doc) []mermaidBlock {
	var blocks []mermaidBlock
	for _, doc := range docList {
		raw, err := os.ReadFile(doc.File)
		if err != nil {
			continue
		}
		matches := reMermaidBlock.FindAllStringSubmatch(string(raw), -1)
		for i, m := range matches {
			blocks = append(blocks, mermaidBlock{
				DocID:    doc.ID,
				DocTitle: doc.Title,
				Index:    i + 1,
				Code:     strings.TrimSpace(m[1]),
			})
		}
	}
	return blocks
}

// --- Página de exportación Mermaid ---

type mermaidExportBlock struct {
	Index int
	Code  string
}

type mermaidExportData struct {
	Blocks []mermaidExportBlock
}

func renderMermaidExport(docList []*docs.Doc) (string, error) {
	tmpl, err := loadTemplate("mermaid-export.gohtml")
	if err != nil {
		return "", err
	}
	data := mermaidExportData{}
	for i, b := range ExtractMermaidBlocks(docList) {
		data.Blocks = append(data.Blocks, mermaidExportBlock{Index: i + 1, Code: b.Code})
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func handleMermaidExport(w http.ResponseWriter, r *http.Request) {
	_, docList, err := LoadState()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	htmlStr, err := renderMermaidExport(docList)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, htmlStr)
}

// --- Captura en Chrome: geometría (para formas nativas) + PNG (respaldo) ---

type capture struct {
	exports []diagramExport
	pngs    [][]byte
}

func captureDiagrams(base string, count int) (*capture, error) {
	ctx, cancel := newChromeContext(context.Background())
	defer cancel()
	ctx, cancelTimeout := context.WithTimeout(ctx, 120*time.Second)
	defer cancelTimeout()

	c := &capture{pngs: make([][]byte, count)}
	var raw string
	tasks := chromedp.Tasks{
		chromedp.Navigate(base + "/mermaid-export"),
		chromedp.WaitReady("body"),
		waitFlag("__diagramsDone", 30*time.Second),
		chromedp.Evaluate(`JSON.stringify(window.__exports || [])`, &raw),
	}
	for i := 0; i < count; i++ {
		i := i
		selector := fmt.Sprintf("#diag-%d", i+1)
		tasks = append(tasks, chromedp.ActionFunc(func(ctx context.Context) error {
			var buf []byte
			if err := chromedp.Screenshot(selector, &buf, chromedp.ByQuery).Do(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "drawio: sin captura del diagrama %d: %v\n", i+1, err)
				return nil // no abortar por un error individual
			}
			c.pngs[i] = buf
			return nil
		}))
	}
	if err := chromedp.Run(ctx, tasks); err != nil {
		return nil, fmt.Errorf("capturando diagramas: %w", err)
	}
	if err := json.Unmarshal([]byte(raw), &c.exports); err != nil {
		return nil, fmt.Errorf("leyendo geometría de diagramas: %w", err)
	}
	return c, nil
}

// DrawioMode controla cómo se exporta cada diagrama.
type DrawioMode string

const (
	// DrawioAuto usa formas nativas cuando el tipo de diagrama es
	// compatible (flowchart, sequence, state, class) e imagen en el resto.
	DrawioAuto DrawioMode = "auto"
	// DrawioImage exporta todos los diagramas como imagen (comportamiento anterior).
	DrawioImage DrawioMode = "image"
)

// DiagramReport resume cómo se exportó un diagrama.
type DiagramReport struct {
	Doc    string `json:"doc"`
	Index  int    `json:"index"`
	Type   string `json:"type"`
	Output string `json:"output"` // native | image | code
	Error  string `json:"error,omitempty"`
}

// GenerateDrawio genera el archivo .drawio de todos los diagramas Mermaid.
func GenerateDrawio(base string, docList []*docs.Doc, mode DrawioMode) ([]byte, []DiagramReport, error) {
	blocks := ExtractMermaidBlocks(docList)
	if len(blocks) == 0 {
		return nil, nil, errNoDiagrams
	}
	c, err := captureDiagrams(base, len(blocks))
	if err != nil {
		return nil, nil, err
	}
	data, report := buildDrawio(blocks, c, mode)
	return data, report, nil
}

var errNoDiagrams = fmt.Errorf("no se encontraron diagramas Mermaid en los documentos")

// --- Handler ---

func handleDrawio(w http.ResponseWriter, r *http.Request) {
	cfg, docList, err := LoadState()
	if err != nil {
		http.Error(w, "Error de configuración: "+err.Error(), http.StatusInternalServerError)
		return
	}
	mode := DrawioMode(r.URL.Query().Get("mode"))
	if mode == "" {
		mode = DrawioAuto
	}
	data, _, err := GenerateDrawio(baseURL(), docList, mode)
	if err == errNoDiagrams {
		http.Error(w, "No se encontraron diagramas Mermaid en los documentos.", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Error generando .drawio: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, DrawioFilename(cfg)))
	w.Write(data)
}

// DrawioFilename genera el nombre normalizado del archivo .drawio.
func DrawioFilename(cfg *brand.Config) string {
	return slugFilename(fmt.Sprintf("%s-%s-diagramas-%s", cfg.CompanyName, cfg.DocsTitle, time.Now().Format("2006-01-02"))) + ".drawio"
}

// slugFilename quita tildes, pasa a minúsculas y deja solo [a-z0-9-].
func slugFilename(raw string) string {
	t := transform.Chain(norm.NFD, transform.RemoveFunc(func(r rune) bool {
		return unicode.Is(unicode.Mn, r)
	}), norm.NFC)
	cleaned, _, _ := transform.String(t, raw)
	cleaned = strings.ToLower(cleaned)
	cleaned = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(cleaned, "-")
	return strings.Trim(cleaned, "-")
}
