package docs

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"golang.org/x/text/unicode/norm"
)

// SearchSection es un bloque de un documento: un título y el texto que hay
// hasta el siguiente título. La primera sección puede no tener título
// (texto antes del primer encabezado).
type SearchSection struct {
	Heading string `json:"h,omitempty"` // texto del título
	Anchor  string `json:"a,omitempty"` // id del título (#ancla)
	Level   int    `json:"l,omitempty"` // nivel del título (1-6)
	Text    string `json:"t,omitempty"` // texto plano de la sección
}

// SearchDoc es un documento en el índice de búsqueda.
type SearchDoc struct {
	ID       string          `json:"id"`
	Title    string          `json:"title"`
	Subtitle string          `json:"subtitle,omitempty"`
	Sections []SearchSection `json:"sections"`
}

// IndexDoc extrae las secciones de un documento para el buscador. Las
// anclas coinciden con las del visor (ModeWeb). Los bloques Mermaid y el
// HTML crudo no se indexan.
func IndexDoc(source []byte, d *Doc) SearchDoc {
	out := SearchDoc{ID: d.ID, Title: d.Title, Subtitle: d.Subtitle}
	ctx := parser.NewContext(parser.WithIDs(newIDs("")))
	root := md.Parser().Parse(text.NewReader(source), parser.WithContext(ctx))

	cur := SearchSection{}
	var body strings.Builder
	flush := func() {
		cur.Text = collapseSpaces(body.String())
		if cur.Heading != "" || cur.Text != "" {
			out.Sections = append(out.Sections, cur)
		}
		body.Reset()
	}
	for n := root.FirstChild(); n != nil; n = n.NextSibling() {
		if h, ok := n.(*ast.Heading); ok {
			flush()
			var hb strings.Builder
			plainText(h, source, &hb)
			cur = SearchSection{Heading: collapseSpaces(hb.String()), Level: h.Level}
			if v, ok := h.AttributeString("id"); ok {
				if b, ok := v.([]byte); ok {
					cur.Anchor = string(b)
				}
			}
			continue
		}
		var nb strings.Builder
		plainText(n, source, &nb)
		if t := strings.TrimSpace(nb.String()); reTOCOnly.MatchString(t) {
			continue // marcador [TOC]
		}
		body.WriteString(nb.String())
		body.WriteByte(' ')
	}
	flush()
	if out.Sections == nil {
		out.Sections = []SearchSection{}
	}
	return out
}

var reTOCOnly = regexp.MustCompile(`(?i)^\[\[?toc\]?\]$`)

// plainText escribe el texto visible de un nodo.
func plainText(n ast.Node, src []byte, b *strings.Builder) {
	switch t := n.(type) {
	case *ast.Text:
		b.Write(t.Segment.Value(src))
		if t.SoftLineBreak() || t.HardLineBreak() {
			b.WriteByte(' ')
		}
		return
	case *ast.String:
		b.Write(t.Value)
		return
	case *ast.AutoLink:
		b.Write(t.Label(src))
		return
	case *ast.RawHTML, *ast.HTMLBlock:
		return
	case *ast.FencedCodeBlock:
		if strings.EqualFold(string(t.Language(src)), "mermaid") {
			return
		}
		writeLines(t.Lines(), src, b)
		return
	case *ast.CodeBlock:
		writeLines(t.Lines(), src, b)
		return
	}
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		plainText(c, src, b)
		if c.Type() == ast.TypeBlock {
			b.WriteByte(' ')
		}
	}
}

func writeLines(lines *text.Segments, src []byte, b *strings.Builder) {
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		b.Write(seg.Value(src))
		b.WriteByte(' ')
	}
}

func collapseSpaces(s string) string { return strings.Join(strings.Fields(s), " ") }

// ---------------------------------------------------------------- búsqueda

// Niveles de la jerarquía de resultados.
const (
	TierTitle   = 1 // título o subtítulo del documento
	TierHeading = 2 // títulos internos (##, ###…)
	TierContent = 3 // texto de las secciones
)

// SearchHit es un resultado de Search.
type SearchHit struct {
	Tier     int    `json:"tier"`
	Kind     string `json:"kind"` // title | heading | content
	DocID    string `json:"docId"`
	DocTitle string `json:"docTitle"`
	Heading  string `json:"heading,omitempty"`
	Anchor   string `json:"anchor,omitempty"`
	Level    int    `json:"level,omitempty"`
	Snippet  string `json:"snippet,omitempty"`
	score    int
	order    int
}

// Search busca query en el índice con jerarquía: primero los documentos
// cuyo título (o subtítulo) coincide, luego los títulos internos y al final
// el contenido. No distingue mayúsculas ni tildes; con varias palabras,
// todas deben aparecer. limit <= 0 = sin límite.
func Search(index []SearchDoc, query string, limit int) []SearchHit {
	q := Fold(collapseSpaces(query))
	terms := strings.Fields(q)
	if len(terms) == 0 {
		return []SearchHit{}
	}
	var hits []SearchHit
	order := 0
	for _, d := range index {
		order++
		if s, ok := matchScore(d.Title+" "+d.Subtitle, q, terms); ok {
			if t, ok2 := matchScore(d.Title, q, terms); ok2 && t > s {
				s = t
			}
			hits = append(hits, SearchHit{Tier: TierTitle, Kind: "title", DocID: d.ID, DocTitle: d.Title, Snippet: d.Subtitle, score: s, order: order})
		}
		titleFold := Fold(d.Title)
		headingHit := map[int]bool{}
		for i, sec := range d.Sections {
			order++
			if sec.Heading == "" || (sec.Level == 1 && Fold(sec.Heading) == titleFold) {
				continue // el H1 del documento ya cuenta como título
			}
			if s, ok := matchScore(sec.Heading, q, terms); ok {
				headingHit[i] = true
				hits = append(hits, SearchHit{Tier: TierHeading, Kind: "heading", DocID: d.ID, DocTitle: d.Title,
					Heading: sec.Heading, Anchor: sec.Anchor, Level: sec.Level, Snippet: Snippet(sec.Text, terms, 140), score: s, order: order})
			}
		}
		for i, sec := range d.Sections {
			if headingHit[i] {
				continue
			}
			// el título cuenta como parte de su sección: "errores 401"
			// encuentra "### Errores" aunque el 401 esté en el texto
			if s, ok := matchScore(strings.TrimSpace(sec.Heading+" "+sec.Text), q, terms); ok {
				hits = append(hits, SearchHit{Tier: TierContent, Kind: "content", DocID: d.ID, DocTitle: d.Title,
					Heading: sec.Heading, Anchor: sec.Anchor, Level: sec.Level, Snippet: Snippet(sec.Text, terms, 140), score: s, order: order + i})
			}
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		a, b := hits[i], hits[j]
		if a.Tier != b.Tier {
			return a.Tier < b.Tier
		}
		if a.score != b.score {
			return a.score > b.score
		}
		return a.order < b.order
	})
	if limit > 0 && len(hits) > limit {
		hits = hits[:limit]
	}
	if hits == nil {
		hits = []SearchHit{}
	}
	return hits
}

// matchScore indica si todos los términos están en s y qué tan buena es la
// coincidencia: 3 = igual a la búsqueda, 2 = la frase empieza una palabra,
// 1 = la frase aparece, 0 = solo los términos sueltos.
func matchScore(s, q string, terms []string) (int, bool) {
	f := Fold(s)
	for _, t := range terms {
		if !strings.Contains(f, t) {
			return 0, false
		}
	}
	switch i := strings.Index(f, q); {
	case strings.TrimSpace(f) == q:
		return 3, true
	case i == 0 || (i > 0 && !isWordRune(lastRune(f[:i]))):
		return 2, true
	case i > 0:
		return 1, true
	}
	return 0, true
}

func isWordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

func lastRune(s string) rune {
	r, _ := utf8.DecodeLastRuneInString(s)
	return r
}

// Fold pasa a minúsculas y quita tildes conservando un rune por rune, para
// que las posiciones coincidan con el texto original.
func Fold(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		b.WriteRune(foldRune(r))
	}
	return b.String()
}

func foldRune(r rune) rune {
	if r < 0x80 {
		return unicode.ToLower(r)
	}
	d := norm.NFD.String(string(r))
	for _, c := range d {
		return unicode.ToLower(c) // primer rune: la letra base
	}
	return unicode.ToLower(r)
}

// Snippet devuelve un fragmento de text de hasta max runes alrededor de la
// primera aparición de alguno de los términos.
func Snippet(s string, terms []string, max int) string {
	rs := []rune(s)
	if len(rs) <= max {
		return s
	}
	fr := []rune(Fold(s))
	pos := -1
	for _, t := range terms {
		if i := runeIndex(fr, []rune(t)); i >= 0 && (pos < 0 || i < pos) {
			pos = i
		}
	}
	start := 0
	if pos > max/3 {
		start = pos - max/3
	}
	end := start + max
	if end > len(rs) {
		end = len(rs)
		if end-max > 0 {
			start = end - max
		}
	}
	// no cortar palabras
	for start > 0 && start < end && !unicode.IsSpace(rs[start-1]) {
		start++
	}
	for end < len(rs) && end > start && !unicode.IsSpace(rs[end]) {
		end--
	}
	out := strings.TrimSpace(string(rs[start:end]))
	if start > 0 {
		out = "…" + out
	}
	if end < len(rs) {
		out += "…"
	}
	return out
}

func runeIndex(s, sub []rune) int {
	if len(sub) == 0 {
		return 0
	}
outer:
	for i := 0; i+len(sub) <= len(s); i++ {
		for j := range sub {
			if s[i+j] != sub[j] {
				continue outer
			}
		}
		return i
	}
	return -1
}
