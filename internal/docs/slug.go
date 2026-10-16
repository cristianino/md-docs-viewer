package docs

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/yuin/goldmark/ast"
)

// ids genera ids de títulos estilo GitHub (minúsculas, letras con tilde
// incluidas, espacios como guiones) con un prefijo opcional.
type ids struct {
	prefix string
	used   map[string]bool
}

func newIDs(prefix string) *ids {
	return &ids{prefix: prefix, used: map[string]bool{}}
}

// Slug convierte un texto en un ancla estilo GitHub.
func Slug(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-':
			b.WriteRune(r)
		case unicode.IsSpace(r):
			b.WriteRune('-')
		}
	}
	return b.String()
}

func (s *ids) Generate(value []byte, kind ast.NodeKind) []byte {
	base := Slug(string(value))
	if base == "" {
		base = "section"
	}
	id := base
	for i := 1; s.used[id]; i++ {
		id = base + "-" + strconv.Itoa(i)
	}
	s.used[id] = true
	return []byte(s.prefix + id)
}

func (s *ids) Put(value []byte) {
	s.used[strings.TrimPrefix(string(value), s.prefix)] = true
}
