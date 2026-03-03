package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"html"
	"image"
	_ "image/png"
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
	docTitle string
	index    int
	code     string
}

func extractMermaidBlocks(docList []*docs.Doc) []mermaidBlock {
	var blocks []mermaidBlock
	for _, doc := range docList {
		raw, err := os.ReadFile(doc.File)
		if err != nil {
			continue
		}
		matches := reMermaidBlock.FindAllStringSubmatch(string(raw), -1)
		for i, m := range matches {
			blocks = append(blocks, mermaidBlock{
				docTitle: doc.Title,
				index:    i + 1,
				code:     strings.TrimSpace(m[1]),
			})
		}
	}
	return blocks
}

// xmlAttr escapa una cadena para usarla como valor de atributo XML.
func xmlAttr(s string) string {
	var buf strings.Builder
	xml.EscapeText(&buf, []byte(s))
	return buf.String()
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
	blocks := extractMermaidBlocks(docList)
	data := mermaidExportData{}
	for i, b := range blocks {
		data.Blocks = append(data.Blocks, mermaidExportBlock{
			Index: i + 1,
			Code:  b.code,
		})
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func handleMermaidExport(w http.ResponseWriter, r *http.Request) {
	_, docList, err := loadState()
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

// --- Captura de PNGs con chromedp (screenshot por elemento) ---

func captureMermaidPNGs(p int, count int) ([][]byte, error) {
	url := fmt.Sprintf("http://localhost:%d/mermaid-export", p)

	ctx, cancel := chromedp.NewContext(context.Background(),
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

	waitDone := chromedp.ActionFunc(func(ctx context.Context) error {
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			var done bool
			if err := chromedp.Evaluate(`window.__diagramsDone === true`, &done).Do(ctx); err == nil && done {
				return nil
			}
			time.Sleep(200 * time.Millisecond)
		}
		return nil
	})

	pngs := make([][]byte, count)

	// Navegar y esperar que los diagramas estén listos
	tasks := chromedp.Tasks{
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
		waitDone,
	}

	// Un screenshot por cada elemento #diag-N
	for i := 0; i < count; i++ {
		i := i // capturar variable de bucle para closure
		selector := fmt.Sprintf("#diag-%d", i+1)
		tasks = append(tasks, chromedp.ActionFunc(func(ctx context.Context) error {
			var buf []byte
			if err := chromedp.Screenshot(selector, &buf).Do(ctx); err != nil {
				fmt.Printf("screenshot error diagrama %d: %v\n", i+1, err)
				return nil // no abortar por un error individual
			}
			pngs[i] = buf
			return nil
		}))
	}

	if err := chromedp.Run(ctx, tasks); err != nil {
		return nil, fmt.Errorf("capturando imágenes: %w", err)
	}
	return pngs, nil
}

// --- Generación del XML .drawio con imágenes PNG ---

func generateDrawioWithImages(blocks []mermaidBlock, pngs [][]byte) []byte {
	now := time.Now().Format("2006-01-02T15:04:05.000Z")
	var sb strings.Builder

	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<mxfile host="mdocs" modified="` + xmlAttr(now) + `" agent="mdocs" version="24.0.0" type="device">` + "\n")

	titleStyle := "text;html=0;strokeColor=none;fillColor=none;align=left;verticalAlign=top;fontStyle=1;fontSize=13;"

	for i, b := range blocks {
		pageID := fmt.Sprintf("mermaid-%d", i+1)
		pageName := fmt.Sprintf("%s #%d", b.docTitle, b.index)
		pageNameAttr := xmlAttr(pageName)

		sb.WriteString(`  <diagram name="` + pageNameAttr + `" id="` + pageID + `">` + "\n")
		sb.WriteString(`    <mxGraphModel>` + "\n")
		sb.WriteString(`      <root>` + "\n")
		sb.WriteString(`        <mxCell id="0"/>` + "\n")
		sb.WriteString(`        <mxCell id="1" parent="0"/>` + "\n")

		// Título de la página
		sb.WriteString(`        <mxCell id="2" value="` + pageNameAttr + `" style="` + titleStyle + `" vertex="1" parent="1">` + "\n")
		sb.WriteString(`          <mxGeometry x="20" y="20" width="700" height="28" as="geometry"/>` + "\n")
		sb.WriteString(`        </mxCell>` + "\n")

		// Diagrama: PNG embebido como <img> en value HTML.
		// No se usa image= en el style porque el `;` de "data:image/png;base64,"
		// hace que el parser de estilos de drawio trunque el valor.
		if i < len(pngs) && len(pngs[i]) > 0 {
			b64 := base64.StdEncoding.EncodeToString(pngs[i])
			imgHTML := fmt.Sprintf(`<img src="data:image/png;base64,%s" width="100%%"/>`, b64)
			valueAttr := xmlAttr(imgHTML) // < > " → &lt; &gt; &#34;
			cellStyle := "text;html=1;align=center;verticalAlign=middle;strokeColor=none;fillColor=none;overflow=hidden;"

			const displayW = 700.0
			displayH := 460.0
			if cfg, _, err := image.DecodeConfig(bytes.NewReader(pngs[i])); err == nil && cfg.Width > 0 {
				displayH = displayW * float64(cfg.Height) / float64(cfg.Width)
			}

			sb.WriteString(`        <mxCell id="3" value="` + valueAttr + `" style="` + cellStyle + `" vertex="1" parent="1">` + "\n")
			sb.WriteString(fmt.Sprintf(`          <mxGeometry x="20" y="56" width="%.0f" height="%.0f" as="geometry"/>`, displayW, displayH) + "\n")
			sb.WriteString(`        </mxCell>` + "\n")
		} else {
			// Fallback: código Mermaid como texto
			htmlCode := "<pre>" + html.EscapeString(b.code) + "</pre>"
			codeAttr := xmlAttr(htmlCode)
			codeStyle := "text;html=1;strokeColor=#d6b656;fillColor=#fffacd;align=left;verticalAlign=top;" +
				"spacingTop=4;spacingLeft=8;whiteSpace=pre;overflow=hidden;rounded=1;"
			sb.WriteString(`        <mxCell id="3" value="` + codeAttr + `" style="` + codeStyle + `" vertex="1" parent="1">` + "\n")
			sb.WriteString(`          <mxGeometry x="20" y="56" width="700" height="400" as="geometry"/>` + "\n")
			sb.WriteString(`        </mxCell>` + "\n")
		}

		sb.WriteString(`      </root>` + "\n")
		sb.WriteString(`    </mxGraphModel>` + "\n")
		sb.WriteString(`  </diagram>` + "\n")
	}

	sb.WriteString(`</mxfile>` + "\n")
	return []byte(sb.String())
}

// --- Handler principal ---

func handleDrawio(w http.ResponseWriter, r *http.Request) {
	cfg, docList, err := loadState()
	if err != nil {
		http.Error(w, "Error de configuración: "+err.Error(), http.StatusInternalServerError)
		return
	}

	blocks := extractMermaidBlocks(docList)
	if len(blocks) == 0 {
		http.Error(w, "No se encontraron diagramas Mermaid en los documentos.", http.StatusNotFound)
		return
	}

	pngs, err := captureMermaidPNGs(port, len(blocks))
	if err != nil {
		http.Error(w, "Error capturando imágenes: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := generateDrawioWithImages(blocks, pngs)
	filename := drawioFilename(cfg)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(data)
}

func drawioFilename(cfg *brand.Config) string {
	date := time.Now().Format("2006-01-02")
	raw := fmt.Sprintf("%s-%s-diagramas-%s", cfg.CompanyName, cfg.DocsTitle, date)
	t := transform.Chain(norm.NFD, transform.RemoveFunc(func(r rune) bool {
		return unicode.Is(unicode.Mn, r)
	}), norm.NFC)
	cleaned, _, _ := transform.String(t, raw)
	cleaned = strings.ToLower(cleaned)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	cleaned = re.ReplaceAllString(cleaned, "-")
	cleaned = strings.Trim(cleaned, "-")
	return cleaned + ".drawio"
}
