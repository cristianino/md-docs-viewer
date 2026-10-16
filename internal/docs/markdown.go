package docs

import (
	"bytes"
	"net/url"
	"path"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
)

var md = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		extension.Table,
		extension.Strikethrough,
		extension.TaskList,
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
	goldmark.WithRendererOptions(
		html.WithUnsafe(),
	),
)

// Mode indica para qué salida se renderiza el Markdown.
type Mode int

const (
	// ModeWeb renderiza para el visor: los enlaces a otros .md van a /doc/{id}.
	ModeWeb Mode = iota
	// ModePrint renderiza para el PDF: todo vive en un único HTML, así que
	// los ids de los títulos llevan el prefijo del documento y los enlaces
	// internos apuntan a anclas (#doc-{id}, #{id}--{ancla}).
	ModePrint
)

// Options controla el renderizado de un documento.
type Options struct {
	Mode Mode
	// Doc es el documento que se renderiza.
	Doc *Doc
	// All es la lista completa, usada para resolver enlaces entre documentos.
	All []*Doc
	// DropTitle elimina el primer H1 si coincide con el título del documento
	// (en el PDF el título ya aparece en la cabecera de la sección).
	DropTitle bool
}

// Heading es un título encontrado en el documento.
type Heading struct {
	Level int    // nivel original en el Markdown (1-6)
	Text  string // texto plano
	ID    string // id final en el HTML
}

// Result es la salida del renderizado.
type Result struct {
	HTML     string
	Headings []Heading
}

// Render convierte Markdown a HTML aplicando las opciones.
func Render(source []byte, opt Options) (*Result, error) {
	prefix := ""
	if opt.Mode == ModePrint && opt.Doc != nil {
		prefix = AnchorPrefix(opt.Doc.ID)
	}
	ctx := parser.NewContext(parser.WithIDs(newIDs(prefix)))
	doc := md.Parser().Parse(text.NewReader(source), parser.WithContext(ctx))

	if opt.DropTitle && opt.Doc != nil {
		if h, ok := doc.FirstChild().(*ast.Heading); ok && h.Level == 1 &&
			normalize(string(h.Text(source))) == normalize(opt.Doc.Title) {
			doc.RemoveChild(doc, h)
		}
	}

	res := &Result{}
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node := n.(type) {
		case *ast.Heading:
			id := ""
			if v, ok := node.AttributeString("id"); ok {
				if b, ok := v.([]byte); ok {
					id = string(b)
				}
			}
			res.Headings = append(res.Headings, Heading{
				Level: node.Level,
				Text:  strings.TrimSpace(string(node.Text(source))),
				ID:    id,
			})
			if opt.Mode == ModePrint {
				// En el PDF cada documento es un H1 (cabecera de sección), así
				// que su contenido baja un nivel. La clase conserva el estilo.
				node.SetAttributeString("class", []byte("md-h"+itoa(node.Level)))
				if node.Level < 6 {
					node.Level++
				}
			}
		case *ast.Link:
			node.Destination = []byte(rewriteLink(string(node.Destination), opt))
		case *ast.Image:
			node.Destination = []byte(rewriteAsset(string(node.Destination), opt))
		}
		return ast.WalkContinue, nil
	})

	var buf bytes.Buffer
	if err := md.Renderer().Render(&buf, source, doc); err != nil {
		return nil, err
	}
	res.HTML = buf.String()
	return res, nil
}

// AnchorPrefix es el prefijo de los ids de un documento dentro del PDF.
func AnchorPrefix(docID string) string { return docID + "--" }

// SectionAnchor es el id de la cabecera de un documento dentro del PDF.
func SectionAnchor(docID string) string { return "doc-" + docID }

func isExternal(dest string) bool {
	if dest == "" {
		return true
	}
	u, err := url.Parse(dest)
	if err != nil {
		return true
	}
	return u.Scheme != "" || strings.HasPrefix(dest, "//")
}

// resolveTarget devuelve el documento destino de un enlace y su ancla.
// ok es false si el enlace no apunta a un documento registrado.
func resolveTarget(dest string, opt Options) (target *Doc, fragment string, ok bool) {
	if isExternal(dest) {
		return nil, "", false
	}
	p, frag, _ := strings.Cut(dest, "#")
	if f, err := url.PathUnescape(frag); err == nil {
		frag = f
	}
	if p == "" {
		return opt.Doc, frag, opt.Doc != nil
	}
	if strings.HasPrefix(p, "/doc/") {
		id := strings.TrimSuffix(strings.TrimPrefix(p, "/doc/"), "/")
		for _, d := range opt.All {
			if d.ID == id {
				return d, frag, true
			}
		}
		return nil, "", false
	}
	if !strings.HasSuffix(strings.ToLower(p), ".md") {
		return nil, "", false
	}
	base := "."
	if opt.Doc != nil {
		base = path.Dir(toSlash(opt.Doc.File))
	}
	want := path.Clean(path.Join(base, toSlash(p)))
	if strings.HasPrefix(p, "/") {
		want = path.Clean(strings.TrimPrefix(toSlash(p), "/"))
	}
	for _, d := range opt.All {
		if path.Clean(toSlash(d.File)) == want {
			return d, frag, true
		}
	}
	return nil, "", false
}

func rewriteLink(dest string, opt Options) string {
	target, frag, ok := resolveTarget(dest, opt)
	if !ok {
		return dest
	}
	if opt.Mode == ModePrint {
		if frag == "" {
			return "#" + SectionAnchor(target.ID)
		}
		return "#" + AnchorPrefix(target.ID) + frag
	}
	if opt.Doc != nil && target.ID == opt.Doc.ID && !strings.Contains(dest, ".md") && !strings.HasPrefix(dest, "/doc/") {
		return dest // ancla en el mismo documento
	}
	out := "/doc/" + target.ID
	if frag != "" {
		out += "#" + frag
	}
	return out
}

// rewriteAsset convierte rutas relativas de imágenes (relativas al .md)
// en rutas servidas por /files/.
func rewriteAsset(dest string, opt Options) string {
	if isExternal(dest) || strings.HasPrefix(dest, "/") || strings.HasPrefix(dest, "data:") {
		return dest
	}
	base := "."
	if opt.Doc != nil {
		base = path.Dir(toSlash(opt.Doc.File))
	}
	return "/files/" + path.Clean(path.Join(base, toSlash(dest)))
}

func toSlash(p string) string { return strings.ReplaceAll(p, "\\", "/") }

func normalize(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

func itoa(n int) string { return string(rune('0' + n)) }
