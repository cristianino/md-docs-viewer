package server

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"time"

	"github.com/cristianino/mdocs/internal/brand"
	"github.com/cristianino/mdocs/internal/docs"
)

var funcMap = template.FuncMap{
	"add": func(a, b int) int { return a + b },
}

func loadTemplate(name string) (*template.Template, error) {
	path := "templates/" + name
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("leyendo template %s: %w", path, err)
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
	ActiveID    string
	ActiveDoc   *docs.Doc
	Content     template.HTML
	AccentFaint string
}

type printSection struct {
	Doc     *docs.Doc
	Content template.HTML
}

type printData struct {
	Brand       *brand.Config
	Docs        []*docs.Doc
	Sections    []printSection
	Date        string
	AccentFaint string
}

func renderPage(cfg *brand.Config, docList []*docs.Doc, activeID string, content string) (string, error) {
	tmpl, err := loadTemplate("page.gohtml")
	if err != nil {
		return "", err
	}
	var activeDoc *docs.Doc
	for _, d := range docList {
		if d.ID == activeID {
			activeDoc = d
			break
		}
	}
	data := pageData{
		Brand:       cfg,
		Docs:        docList,
		ActiveID:    activeID,
		ActiveDoc:   activeDoc,
		Content:     template.HTML(content),
		AccentFaint: brand.HexToRGBA(cfg.AccentColor, 0.08),
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func renderPrint(cfg *brand.Config, docList []*docs.Doc) (string, error) {
	tmpl, err := loadTemplate("print.gohtml")
	if err != nil {
		return "", err
	}
	now := time.Now()
	months := []string{
		"enero", "febrero", "marzo", "abril", "mayo", "junio",
		"julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre",
	}
	date := fmt.Sprintf("%d de %s de %d", now.Day(), months[now.Month()-1], now.Year())

	var sections []printSection
	for _, d := range docList {
		raw, err := os.ReadFile(d.File)
		var html string
		if err == nil {
			html, err = docs.Render(raw)
		}
		if err != nil {
			html = "<p><em>Error al cargar el documento.</em></p>"
		}
		sections = append(sections, printSection{Doc: d, Content: template.HTML(html)})
	}

	data := printData{
		Brand:       cfg,
		Docs:        docList,
		Sections:    sections,
		Date:        date,
		AccentFaint: brand.HexToRGBA(cfg.AccentColor, 0.08),
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
