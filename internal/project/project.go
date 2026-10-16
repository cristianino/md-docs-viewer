// Package project analiza un proyecto mdocs: valida la configuración
// (check) y resume su contenido para personas o agentes de IA (context).
package project

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/cristianino/mdocs/internal/brand"
	"github.com/cristianino/mdocs/internal/docs"
)

const (
	BrandFile = "brand.config.json"
	DocsFile  = "docs.config.json"
)

// Issue es un problema encontrado por Check.
type Issue struct {
	Level   string `json:"level"` // error | warning
	File    string `json:"file,omitempty"`
	Doc     string `json:"doc,omitempty"`
	Message string `json:"message"`
}

// Diagram describe un bloque Mermaid.
type Diagram struct {
	Doc    string `json:"doc"`
	Index  int    `json:"index"`
	Type   string `json:"type"`
	Native bool   `json:"native"` // se exporta a draw.io con formas nativas
}

// DocInfo resume un documento.
type DocInfo struct {
	*docs.Doc
	Exists   bool           `json:"exists"`
	Words    int            `json:"words"`
	Headings []docs.Heading `json:"headings"`
	Diagrams []Diagram      `json:"diagrams,omitempty"`
}

// Context es el resumen completo del proyecto.
type Context struct {
	Brand        *brand.Config `json:"brand"`
	Docs         []DocInfo     `json:"docs"`
	Unregistered []string      `json:"unregistered,omitempty"`
	Issues       []Issue       `json:"issues"`
}

var (
	reMermaid = regexp.MustCompile("(?s)```mermaid\r?\n(.*?)```")
	reLink    = regexp.MustCompile(`!?\[[^\]]*\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)
	reFence   = regexp.MustCompile("(?s)```.*?```")
)

// nativeTypes son los tipos que mdocs convierte a formas draw.io.
var nativeTypes = map[string]bool{
	"flowchart": true, "graph": true, "sequenceDiagram": true,
	"stateDiagram": true, "stateDiagram-v2": true, "classDiagram": true,
}

// DiagramType devuelve la primera palabra significativa de un bloque Mermaid.
func DiagramType(code string) string {
	for _, line := range strings.Split(code, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "%%") || line == "---" {
			continue
		}
		if f := strings.Fields(line); len(f) > 0 {
			return strings.TrimSuffix(f[0], ":")
		}
	}
	return ""
}

// IsNative indica si un tipo Mermaid se exporta con formas nativas.
func IsNative(t string) bool { return nativeTypes[t] }

// Load analiza el proyecto de la carpeta actual.
func Load() (*Context, error) {
	ctx := &Context{Issues: []Issue{}}
	add := func(level, file, doc, msg string, args ...any) {
		ctx.Issues = append(ctx.Issues, Issue{Level: level, File: file, Doc: doc, Message: fmt.Sprintf(msg, args...)})
	}

	cfg, err := brand.Load(BrandFile)
	if err != nil {
		add("error", BrandFile, "", "%v", err)
	}
	if _, err := os.Stat(BrandFile); err != nil {
		add("warning", BrandFile, "", "no existe; se usan valores por defecto (ejecuta mdocs init)")
	}
	ctx.Brand = cfg

	list, err := docs.Load(DocsFile)
	if err != nil {
		add("error", DocsFile, "", "%v", err)
		return ctx, nil
	}
	if len(list) == 0 {
		add("warning", DocsFile, "", "no hay documentos registrados (usa mdocs add)")
	}

	seen := map[string]bool{}
	registered := map[string]bool{}
	anchors := map[string]map[string]bool{}
	sources := map[string][]byte{}

	for _, d := range list {
		info := DocInfo{Doc: d}
		switch {
		case strings.TrimSpace(d.ID) == "":
			add("error", DocsFile, "", "documento sin id (%s)", d.File)
		case seen[d.ID]:
			add("error", DocsFile, d.ID, "id duplicado %q", d.ID)
		case docs.Slug(d.ID) != d.ID:
			add("warning", DocsFile, d.ID, "el id %q tiene caracteres que no son de URL; sugerido %q", d.ID, docs.Slug(d.ID))
		}
		seen[d.ID] = true
		if d.Title == "" {
			add("warning", DocsFile, d.ID, "documento sin título")
		}
		registered[path.Clean(toSlash(d.File))] = true

		raw, err := os.ReadFile(d.File)
		if err != nil {
			add("error", d.File, d.ID, "no se puede leer el archivo: %v", err)
			ctx.Docs = append(ctx.Docs, info)
			continue
		}
		info.Exists = true
		sources[d.ID] = raw
		info.Words = len(strings.Fields(reFence.ReplaceAllString(string(raw), "")))
		if res, err := docs.Render(raw, docs.Options{Mode: docs.ModeWeb, Doc: d, All: list}); err == nil {
			info.Headings = res.Headings
			anchors[d.ID] = map[string]bool{}
			for _, h := range res.Headings {
				anchors[d.ID][h.ID] = true
			}
		}
		for i, m := range reMermaid.FindAllStringSubmatch(string(raw), -1) {
			t := DiagramType(m[1])
			info.Diagrams = append(info.Diagrams, Diagram{Doc: d.ID, Index: i + 1, Type: t, Native: IsNative(t)})
		}
		ctx.Docs = append(ctx.Docs, info)
	}

	// enlaces e imágenes
	for _, d := range list {
		raw, ok := sources[d.ID]
		if !ok {
			continue
		}
		text := reFence.ReplaceAllString(string(raw), "")
		for _, m := range reLink.FindAllStringSubmatch(text, -1) {
			checkLink(m[0], m[1], d, list, anchors, add)
		}
	}

	// .md sin registrar en docs/
	filepath.WalkDir("docs", func(p string, e os.DirEntry, err error) error {
		if err == nil && !e.IsDir() && strings.HasSuffix(strings.ToLower(p), ".md") && !registered[path.Clean(toSlash(p))] {
			ctx.Unregistered = append(ctx.Unregistered, toSlash(p))
		}
		return nil
	})
	sort.Strings(ctx.Unregistered)
	for _, u := range ctx.Unregistered {
		add("warning", u, "", "archivo .md sin registrar en docs.config.json (mdocs add %s)", u)
	}
	return ctx, nil
}

func checkLink(full, dest string, d *docs.Doc, list []*docs.Doc, anchors map[string]map[string]bool, add func(level, file, doc, msg string, args ...any)) {
	if strings.Contains(dest, "://") || strings.HasPrefix(dest, "mailto:") || strings.HasPrefix(dest, "data:") {
		return
	}
	isImage := strings.HasPrefix(full, "!")
	p, frag, _ := strings.Cut(dest, "#")
	if f, err := url.PathUnescape(frag); err == nil {
		frag = f
	}
	base := path.Dir(toSlash(d.File))
	if isImage {
		if strings.HasPrefix(p, "/") {
			return
		}
		if _, err := os.Stat(filepath.FromSlash(path.Join(base, p))); err != nil {
			add("warning", d.File, d.ID, "imagen no encontrada: %s", dest)
		}
		return
	}
	targetID := ""
	switch {
	case p == "":
		targetID = d.ID
	case strings.HasPrefix(p, "/doc/"):
		targetID = strings.TrimPrefix(p, "/doc/")
	case strings.HasSuffix(strings.ToLower(p), ".md"):
		want := path.Clean(path.Join(base, toSlash(p)))
		for _, o := range list {
			if path.Clean(toSlash(o.File)) == want {
				targetID = o.ID
			}
		}
		if targetID == "" {
			add("warning", d.File, d.ID, "enlace a un .md no registrado: %s", dest)
			return
		}
	default:
		return
	}
	a, ok := anchors[targetID]
	if !ok {
		add("warning", d.File, d.ID, "enlace a un documento inexistente: %s", dest)
		return
	}
	if frag != "" && !a[frag] {
		add("warning", d.File, d.ID, "ancla no encontrada: %s", dest)
	}
}

// Errors cuenta los problemas de nivel error.
func (c *Context) Errors() int {
	n := 0
	for _, i := range c.Issues {
		if i.Level == "error" {
			n++
		}
	}
	return n
}

// JSON devuelve el contexto como JSON indentado.
func (c *Context) JSON() []byte {
	b, _ := json.MarshalIndent(c, "", "  ")
	return append(b, '\n')
}

// Markdown devuelve un resumen legible (útil para pegar en un prompt).
func (c *Context) Markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s — %s\n\n", c.Brand.CompanyName, c.Brand.DocsTitle)
	if c.Brand.Description != "" {
		fmt.Fprintf(&b, "%s\n\n", c.Brand.Description)
	}
	fmt.Fprintf(&b, "- Año: %s · poweredBy: %v · colores: %s / %s\n", c.Brand.DocsYear, c.Brand.PoweredBy, c.Brand.PrimaryColor, c.Brand.AccentColor)
	fmt.Fprintf(&b, "- Documentos: %d\n\n", len(c.Docs))
	for i, d := range c.Docs {
		fmt.Fprintf(&b, "## %d. %s (`%s`)\n\n", i+1, d.Title, d.ID)
		fmt.Fprintf(&b, "- Archivo: `%s`", d.File)
		if !d.Exists {
			b.WriteString(" (no existe)")
		}
		fmt.Fprintf(&b, " · %d palabras\n", d.Words)
		if d.Subtitle != "" {
			fmt.Fprintf(&b, "- Subtítulo: %s\n", d.Subtitle)
		}
		for _, h := range d.Headings {
			fmt.Fprintf(&b, "%s- %s `#%s`\n", strings.Repeat("  ", h.Level), h.Text, h.ID)
		}
		for _, dg := range d.Diagrams {
			out := "imagen"
			if dg.Native {
				out = "formas nativas"
			}
			fmt.Fprintf(&b, "- Diagrama %d: %s → draw.io %s\n", dg.Index, dg.Type, out)
		}
		b.WriteString("\n")
	}
	if len(c.Unregistered) > 0 {
		b.WriteString("## Sin registrar\n\n")
		for _, u := range c.Unregistered {
			fmt.Fprintf(&b, "- `%s`\n", u)
		}
		b.WriteString("\n")
	}
	if len(c.Issues) > 0 {
		b.WriteString("## Problemas\n\n")
		for _, i := range c.Issues {
			fmt.Fprintf(&b, "- **%s** %s: %s\n", i.Level, firstNonEmpty(i.File, i.Doc), i.Message)
		}
	}
	return b.String()
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

func toSlash(p string) string { return strings.ReplaceAll(p, "\\", "/") }

// Reference es un enlace desde un documento hacia otro.
type Reference struct {
	From string `json:"from"` // id del documento que enlaza
	File string `json:"file"`
	Link string `json:"link"`
}

// References busca enlaces en los demás documentos que apunten a target
// (por ruta relativa al .md o por /doc/{id}).
func References(target *docs.Doc, list []*docs.Doc) []Reference {
	var out []Reference
	want := path.Clean(toSlash(target.File))
	for _, d := range list {
		if d.ID == target.ID {
			continue
		}
		raw, err := os.ReadFile(d.File)
		if err != nil {
			continue
		}
		text := reFence.ReplaceAllString(string(raw), "")
		base := path.Dir(toSlash(d.File))
		for _, m := range reLink.FindAllStringSubmatch(text, -1) {
			if strings.HasPrefix(m[0], "!") {
				continue
			}
			p, _, _ := strings.Cut(m[1], "#")
			hit := false
			switch {
			case strings.HasPrefix(p, "/doc/"):
				hit = strings.TrimSuffix(strings.TrimPrefix(p, "/doc/"), "/") == target.ID
			case strings.HasSuffix(strings.ToLower(p), ".md") && !strings.Contains(p, "://"):
				hit = path.Clean(path.Join(base, toSlash(p))) == want
			}
			if hit {
				out = append(out, Reference{From: d.ID, File: d.File, Link: m[1]})
			}
		}
	}
	return out
}
