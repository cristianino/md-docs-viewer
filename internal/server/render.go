package server

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"time"

	"github.com/cristianino/mdocs/internal/brand"
	"github.com/cristianino/mdocs/internal/docs"
	"github.com/cristianino/mdocs/templates"
)

var funcMap = template.FuncMap{
	"add": func(a, b int) int { return a + b },
}

// loadTemplate usa templates/<name> del proyecto si existe (edición en
// caliente); si no, la versión embebida en el binario.
func loadTemplate(name string) (*template.Template, error) {
	path := "templates/" + name
	data, err := os.ReadFile(path)
	if err != nil {
		data, err = templates.FS.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("leyendo template %s: %w", name, err)
		}
	}
	tmpl, err := template.New(name).Funcs(funcMap).Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("parseando template %s: %w", name, err)
	}
	return tmpl, nil
}

type pageData struct {
	Brand       *brand.Config
	Docs        []*docs.Doc
	PageTitle   string
	ActiveID    string
	ActiveDoc   *docs.Doc
	Content     template.HTML
	Headings    []docs.Heading
	OnPage      []docs.Heading // títulos ## y ### para «En esta página»
	IsIndex     bool
	Index       []indexEntry
	AccentFaint string
}

// indexEntry es un documento en la página de índice general.
type indexEntry struct {
	Doc      *docs.Doc
	Headings []docs.Heading
}

// onPage filtra los títulos que van en «En esta página» (## y ###).
// Si hay menos de dos, no se muestra el panel.
func onPage(hs []docs.Heading) []docs.Heading {
	var out []docs.Heading
	for _, h := range hs {
		if h.Level >= 2 && h.Level <= 3 && h.ID != "" {
			out = append(out, h)
		}
	}
	if len(out) < 2 {
		return nil
	}
	return out
}

// tocEntry es una línea del índice del PDF.
type tocEntry struct {
	Level  int // 1 = documento, 2.. = títulos internos
	Text   string
	Anchor string
}

type printSection struct {
	Doc     *docs.Doc
	Anchor  string
	Content template.HTML
	TOC     []tocEntry
}

type printData struct {
	Brand       *brand.Config
	Docs        []*docs.Doc
	Sections    []printSection
	Date        string
	AccentFaint string
	ShowCover   bool
	ShowBody    bool
}

// PrintPart elige qué parte del HTML de impresión se genera.
type PrintPart string

const (
	PartAll   PrintPart = ""
	PartCover PrintPart = "cover"
	PartBody  PrintPart = "body"
)

// tocDepth es el nivel máximo (original del Markdown) que entra al índice.
const tocDepth = 2

func renderPage(cfg *brand.Config, docList []*docs.Doc, active *docs.Doc, res *docs.Result) (string, error) {
	tmpl, err := loadTemplate("page.gohtml")
	if err != nil {
		return "", err
	}
	data := pageData{
		Brand:       cfg,
		Docs:        docList,
		PageTitle:   active.Title,
		ActiveID:    active.ID,
		ActiveDoc:   active,
		Content:     template.HTML(res.HTML),
		Headings:    res.Headings,
		OnPage:      onPage(res.Headings),
		AccentFaint: brand.HexToRGBA(cfg.AccentColor, 0.08),
	}
	return executePage(tmpl, data)
}

// renderIndex genera la página de índice general: todos los documentos con
// sus títulos ## y ###.
func renderIndex(cfg *brand.Config, docList []*docs.Doc) (string, error) {
	tmpl, err := loadTemplate("page.gohtml")
	if err != nil {
		return "", err
	}
	var entries []indexEntry
	for _, d := range docList {
		e := indexEntry{Doc: d}
		if raw, err := os.ReadFile(d.File); err == nil {
			if res, err := docs.Render(raw, docs.Options{Mode: docs.ModeWeb, Doc: d, All: docList}); err == nil {
				for _, h := range res.Headings {
					if h.Level >= 2 && h.Level <= 3 && h.ID != "" {
						e.Headings = append(e.Headings, h)
					}
				}
			}
		}
		entries = append(entries, e)
	}
	data := pageData{
		Brand:       cfg,
		Docs:        docList,
		PageTitle:   "Índice general",
		IsIndex:     true,
		Index:       entries,
		AccentFaint: brand.HexToRGBA(cfg.AccentColor, 0.08),
	}
	return executePage(tmpl, data)
}

func executePage(tmpl *template.Template, data pageData) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func spanishDate(t time.Time) string {
	months := []string{
		"enero", "febrero", "marzo", "abril", "mayo", "junio",
		"julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre",
	}
	return fmt.Sprintf("%d de %s de %d", t.Day(), months[t.Month()-1], t.Year())
}

func renderPrint(cfg *brand.Config, docList []*docs.Doc, part PrintPart) (string, error) {
	tmpl, err := loadTemplate("print.gohtml")
	if err != nil {
		return "", err
	}

	var sections []printSection
	for _, d := range docList {
		sec := printSection{Doc: d, Anchor: docs.SectionAnchor(d.ID)}
		raw, err := os.ReadFile(d.File)
		var res *docs.Result
		if err == nil {
			res, err = docs.Render(raw, docs.Options{Mode: docs.ModePrint, Doc: d, All: docList, DropTitle: true})
		}
		if err != nil {
			sec.Content = "<p><em>Error al cargar el documento.</em></p>"
		} else {
			sec.Content = template.HTML(res.HTML)
			for _, h := range res.Headings {
				if h.Level <= tocDepth && h.ID != "" {
					sec.TOC = append(sec.TOC, tocEntry{Level: h.Level + 1, Text: h.Text, Anchor: h.ID})
				}
			}
		}
		sections = append(sections, sec)
	}

	data := printData{
		Brand:       cfg,
		Docs:        docList,
		Sections:    sections,
		Date:        spanishDate(time.Now()),
		AccentFaint: brand.HexToRGBA(cfg.AccentColor, 0.08),
		ShowCover:   part != PartBody,
		ShowBody:    part != PartCover,
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
