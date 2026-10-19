package docs

import (
	"bytes"
	"fmt"
	stdhtml "html"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
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
	// Base es el prefijo de las URLs absolutas (p. ej. "/docs" detrás de un
	// proxy o en un sitio estático). Vacío = raíz.
	Base string
	// Slash agrega "/" al final de /doc/{id} (sitio estático: doc/{id}/index.html).
	Slash bool
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
	// Files son los archivos descargables enlazados desde el documento
	// (rutas relativas al proyecto, sin repetir), p. ej. una colección de
	// Postman: [Colección](files/api.postman_collection.json).
	Files []string
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
			dest := string(node.Destination)
			if rel, ok := LocalFile(dest, opt.Doc); ok {
				if !contains(res.Files, rel) {
					res.Files = append(res.Files, rel)
				}
				if opt.Mode == ModePrint {
					// en el PDF el archivo va adjunto: el enlace se cambia por
					// una nota (ver replaceAttachments)
					node.Destination = []byte(AttachmentScheme + rel)
				} else {
					node.Destination = []byte(opt.Base + "/files/" + rel)
					node.SetAttributeString("download", []byte(path.Base(rel)))
					node.SetAttributeString("class", []byte("md-file"))
					node.SetAttributeString("data-ext", []byte(strings.TrimPrefix(strings.ToLower(path.Ext(rel)), ".")))
				}
				break
			}
			node.Destination = []byte(rewriteLink(dest, opt))
		case *ast.Image:
			node.Destination = []byte(rewriteAsset(string(node.Destination), opt))
		}
		return ast.WalkContinue, nil
	})

	var buf bytes.Buffer
	if err := md.Renderer().Render(&buf, source, doc); err != nil {
		return nil, err
	}
	out := reTOCMarker.ReplaceAllLiteralString(buf.String(), TOCHTML(res.Headings, 3))
	if opt.Mode == ModePrint && len(res.Files) > 0 {
		out = replaceAttachments(out)
	}
	res.HTML = out
	return res, nil
}

// AttachmentScheme marca, al renderizar para el PDF, los enlaces a archivos
// que se adjuntan dentro del PDF.
const AttachmentScheme = "mdocs-attachment:"

var reAttachmentLink = regexp.MustCompile(`<a href="` + AttachmentScheme + `([^"]*)"[^>]*>(.*?)</a>`)

// replaceAttachments cambia los enlaces a archivos por el texto del enlace
// y una nota con el nombre del adjunto (el PDF no puede enlazar a un
// archivo del servidor, así que el archivo viaja adjunto en el PDF).
func replaceAttachments(h string) string {
	return reAttachmentLink.ReplaceAllStringFunc(h, func(m string) string {
		sm := reAttachmentLink.FindStringSubmatch(m)
		rel := stdhtml.UnescapeString(sm[1])
		if u, err := url.PathUnescape(rel); err == nil {
			rel = u
		}
		if !Attachable(rel) {
			return `<span class="md-attachment">` + sm[2] + `</span>`
		}
		return `<span class="md-attachment">` + sm[2] + `</span> <span class="md-attachment-note">(adjunto: ` +
			stdhtml.EscapeString(AttachmentName(rel)) + `)</span>`
	})
}

// MaxAttachmentSize es el tamaño máximo de un archivo adjunto al PDF.
const MaxAttachmentSize = 25 << 20

// Attachable indica si el archivo existe y cabe como adjunto del PDF.
func Attachable(rel string) bool {
	st, err := os.Stat(filepath.FromSlash(rel))
	return err == nil && !st.IsDir() && st.Size() <= MaxAttachmentSize
}

// AttachmentName es el nombre con que un archivo se adjunta al PDF.
func AttachmentName(rel string) string { return path.Base(rel) }

// LocalFile indica si dest (un enlace dentro de d) apunta a un archivo
// descargable del proyecto: relativo, que no sea un .md, ni un ancla, ni
// salga de la carpeta del proyecto. Devuelve la ruta relativa al proyecto.
func LocalFile(dest string, d *Doc) (string, bool) {
	if isExternal(dest) || strings.HasPrefix(dest, "#") || strings.HasPrefix(dest, "/") || strings.HasPrefix(dest, "data:") {
		return "", false
	}
	p, _, _ := strings.Cut(dest, "#")
	p, _, _ = strings.Cut(p, "?")
	if u, err := url.PathUnescape(p); err == nil {
		p = u
	}
	p = toSlash(p)
	if p == "" || strings.HasSuffix(p, "/") || strings.HasSuffix(strings.ToLower(p), ".md") {
		return "", false
	}
	base := "."
	if d != nil {
		base = path.Dir(toSlash(d.File))
	}
	rel := path.Clean(path.Join(base, p))
	if !InsideProject(rel) || hiddenSegment(rel) {
		return "", false
	}
	return rel, true
}

// hiddenSegment indica si algún tramo de la ruta empieza con "." (.env, .git/…).
func hiddenSegment(rel string) bool {
	for _, s := range strings.Split(rel, "/") {
		if strings.HasPrefix(s, ".") {
			return true
		}
	}
	return false
}

// HiddenPath es hiddenSegment para otros paquetes.
func HiddenPath(rel string) bool { return hiddenSegment(toSlash(rel)) }

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// reTOCMarker encuentra un párrafo que solo contiene [TOC] (o [[TOC]],
// sin distinguir mayúsculas).
var reTOCMarker = regexp.MustCompile(`(?i)<p>\s*\[\[?toc\]?\]\s*</p>`)

// TOCHTML genera una tabla de contenido anidada con los títulos del
// documento hasta maxLevel. Omite el título principal (el primer H1).
func TOCHTML(headings []Heading, maxLevel int) string {
	var items []Heading
	for i, h := range headings {
		if i == 0 && h.Level == 1 {
			continue
		}
		if h.Level <= maxLevel && h.ID != "" {
			items = append(items, h)
		}
	}
	if len(items) == 0 {
		return ""
	}
	base := items[0].Level
	for _, h := range items {
		if h.Level < base {
			base = h.Level
		}
	}
	var b strings.Builder
	b.WriteString(`<nav class="md-toc" aria-label="Tabla de contenido"><ul>`)
	depth := 0
	for i, h := range items {
		lvl := h.Level - base
		if lvl < 0 {
			lvl = 0
		}
		switch {
		case i == 0:
		case lvl > depth:
			for ; depth < lvl; depth++ {
				b.WriteString("<ul>")
			}
		case lvl < depth:
			for ; depth > lvl; depth-- {
				b.WriteString("</li></ul>")
			}
			b.WriteString("</li>")
		default:
			b.WriteString("</li>")
		}
		depth = lvl
		fmt.Fprintf(&b, `<li><a href="#%s">%s</a>`, stdhtml.EscapeString(h.ID), stdhtml.EscapeString(h.Text))
	}
	for ; depth > 0; depth-- {
		b.WriteString("</li></ul>")
	}
	b.WriteString("</li></ul></nav>")
	return b.String()
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
	out := opt.Base + "/doc/" + target.ID
	if opt.Slash {
		out += "/"
	}
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
	return opt.Base + "/files/" + path.Clean(path.Join(base, toSlash(dest)))
}

func toSlash(p string) string { return strings.ReplaceAll(p, "\\", "/") }

func normalize(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

func itoa(n int) string { return string(rune('0' + n)) }
