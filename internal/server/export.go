package server

import (
	"fmt"
	"html"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cristianino/mdocs/assets"
	"github.com/cristianino/mdocs/internal/brand"
)

// SiteOptions controla la exportación estática.
type SiteOptions struct {
	Out      string     // carpeta de salida
	BasePath string     // prefijo de URLs donde se publicará ("" = raíz del dominio)
	PDF      bool       // incluir el PDF
	Drawio   bool       // incluir el .drawio
	Mode     DrawioMode // auto | image
	Log      func(format string, args ...any)
}

// SiteReport resume lo que se escribió.
type SiteReport struct {
	Out      string          `json:"out"`
	BasePath string          `json:"basePath"`
	Pages    int             `json:"pages"`
	Files    int             `json:"files"`
	PDF      string          `json:"pdf,omitempty"`
	Drawio   string          `json:"drawio,omitempty"`
	Diagrams []DiagramReport `json:"diagrams,omitempty"`
	Warnings []string        `json:"warnings,omitempty"`
}

var reFilesRef = regexp.MustCompile(`(?:src|href)="[^"]*?/files/([^"#?]+)"`)

// ExportSite genera un sitio estático: no necesita servidor ni Chrome para
// servirse (S3, CloudFront, Netlify, GitHub Pages, nginx…). Cada documento
// queda en doc/{id}/index.html para que /doc/{id} funcione en cualquier host.
func ExportSite(o SiteOptions) (*SiteReport, error) {
	if o.Log == nil {
		o.Log = func(string, ...any) {}
	}
	base := NormalizeBasePath(o.BasePath)
	cfg, docList, err := LoadState()
	if err != nil {
		return nil, err
	}
	if len(docList) == 0 {
		return nil, fmt.Errorf("no hay documentos registrados")
	}
	rep := &SiteReport{Out: o.Out, BasePath: base}
	if err := os.MkdirAll(o.Out, 0755); err != nil {
		return nil, err
	}
	write := func(rel string, data []byte) error {
		p := filepath.Join(o.Out, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			return err
		}
		rep.Files++
		return os.WriteFile(p, data, 0644)
	}

	links := pageLinks{Base: base, Slash: "/"}

	// 1. descargas (primero: los botones solo aparecen si existen)
	if o.PDF || o.Drawio {
		baseURL, stop, err := Ephemeral()
		if err != nil {
			return nil, err
		}
		if o.PDF {
			o.Log("Generando PDF…\n")
			data, err := GeneratePDF(baseURL, cfg)
			if err != nil {
				stop()
				return nil, err
			}
			name := PDFFilename(cfg)
			if err := write("descargas/"+name, data); err != nil {
				stop()
				return nil, err
			}
			rep.PDF = "descargas/" + name
			links.PDFURL = base + "/descargas/" + name
		}
		if o.Drawio {
			o.Log("Generando .drawio…\n")
			data, report, err := GenerateDrawio(baseURL, docList, o.Mode)
			switch {
			case err == errNoDiagrams:
				rep.Warnings = append(rep.Warnings, "sin diagramas Mermaid: no se generó .drawio")
			case err != nil:
				stop()
				return nil, err
			default:
				name := DrawioFilename(cfg)
				if err := write("descargas/"+name, data); err != nil {
					stop()
					return nil, err
				}
				rep.Drawio = "descargas/" + name
				rep.Diagrams = report
				links.DrawioURL = base + "/descargas/" + name
			}
		}
		stop()
	}

	// 2. páginas
	o.Log("Generando páginas…\n")
	used := map[string]bool{}
	collect := func(htmlStr string) {
		for _, m := range reFilesRef.FindAllStringSubmatch(htmlStr, -1) {
			used[m[1]] = true
		}
	}
	for _, d := range docList {
		page, err := renderDocPage(cfg, docList, d, links)
		if err != nil {
			rep.Warnings = append(rep.Warnings, fmt.Sprintf("%s: %v", d.File, err))
			continue
		}
		collect(page)
		if err := write("doc/"+d.ID+"/index.html", []byte(page)); err != nil {
			return nil, err
		}
		rep.Pages++
	}
	idx, err := renderIndex(cfg, docList, links)
	if err != nil {
		return nil, err
	}
	if err := write("indice/index.html", []byte(idx)); err != nil {
		return nil, err
	}
	rep.Pages++
	first := base + "/doc/" + docList[0].ID + "/"
	if err := write("index.html", []byte(redirectPage(cfg, first))); err != nil {
		return nil, err
	}
	if err := write("404.html", []byte(notFoundPage(cfg, base))); err != nil {
		return nil, err
	}

	// 3. recursos
	logo, err := os.ReadFile("logo.svg")
	if err != nil {
		logo = assets.DefaultLogo
	}
	if err := write("logo.svg", logo); err != nil {
		return nil, err
	}
	if err := write("assets/mermaid.min.js", assets.MermaidJS); err != nil {
		return nil, err
	}
	for rel := range used {
		clean := path.Clean(rel)
		if strings.HasPrefix(clean, "..") || strings.HasSuffix(strings.ToLower(clean), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.FromSlash(clean))
		if err != nil {
			rep.Warnings = append(rep.Warnings, "archivo referenciado no encontrado: "+clean)
			continue
		}
		if err := write("files/"+clean, data); err != nil {
			return nil, err
		}
	}
	return rep, nil
}

func redirectPage(cfg *brand.Config, to string) string {
	t := html.EscapeString(cfg.DocsTitle)
	u := html.EscapeString(to)
	return `<!DOCTYPE html><html lang="es"><head><meta charset="UTF-8"><title>` + t + `</title>` +
		`<meta http-equiv="refresh" content="0; url=` + u + `"><link rel="canonical" href="` + u + `"></head>` +
		`<body><a href="` + u + `">` + t + `</a></body></html>`
}

func notFoundPage(cfg *brand.Config, base string) string {
	return `<!DOCTYPE html><html lang="es"><head><meta charset="UTF-8"><title>No encontrado — ` + html.EscapeString(cfg.DocsTitle) + `</title>` +
		`<style>body{font-family:-apple-system,'Segoe UI',sans-serif;color:#4a5568;display:flex;min-height:100vh;align-items:center;justify-content:center;margin:0}a{color:` + html.EscapeString(cfg.AccentColor) + `}</style></head>` +
		`<body><p>Página no encontrada. <a href="` + html.EscapeString(base) + `/indice/">Ir al índice</a></p></body></html>`
}
