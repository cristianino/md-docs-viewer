package docs

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var reNumPrefix = regexp.MustCompile(`^(\d+)[-_. ]+`)

// IDFromFile deriva un id del nombre de archivo: 02-guia-rapida.md → guia-rapida.
func IDFromFile(file string) string {
	base := strings.TrimSuffix(path.Base(toSlash(file)), path.Ext(file))
	return FileSlug(reNumPrefix.ReplaceAllString(base, ""))
}

// TitleFromFile devuelve el primer "# " del archivo, o "" si no hay.
func TitleFromFile(file string) string {
	raw, err := os.ReadFile(file)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(line[2:])
		}
	}
	return ""
}

// TitleFromID convierte un id en un título legible: guia-rapida → Guia rapida.
func TitleFromID(id string) string {
	if id == "" {
		return ""
	}
	r := []rune(strings.ReplaceAll(id, "-", " "))
	return strings.ToUpper(string(r[0])) + string(r[1:])
}

// FindID devuelve el índice del documento con ese id, o -1.
func FindID(list []*Doc, id string) int {
	for i, d := range list {
		if d.ID == id {
			return i
		}
	}
	return -1
}

// FindFile devuelve el índice del documento con ese archivo, o -1.
func FindFile(list []*Doc, file string) int {
	want := path.Clean(toSlash(file))
	for i, d := range list {
		if path.Clean(toSlash(d.File)) == want {
			return i
		}
	}
	return -1
}

// UniqueID agrega -2, -3… si el id ya existe.
func UniqueID(list []*Doc, id string) string {
	if id == "" {
		id = "doc"
	}
	out := id
	for n := 2; FindID(list, out) >= 0; n++ {
		out = id + "-" + strconv.Itoa(n)
	}
	return out
}

// Insert agrega d en la posición (1 = primero; fuera de rango = al final).
func Insert(list []*Doc, d *Doc, position int) []*Doc {
	if position < 1 || position > len(list) {
		return append(list, d)
	}
	out := append([]*Doc{}, list[:position-1]...)
	out = append(out, d)
	return append(out, list[position-1:]...)
}

// NextNumberedFile propone dir/NN-slug.md con NN = mayor prefijo numérico de
// los .md de dir + 1, con dos dígitos.
func NextNumberedFile(dir, slug string) string {
	max := 0
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if m := reNumPrefix.FindStringSubmatch(e.Name()); m != nil && strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			if n, err := strconv.Atoi(m[1]); err == nil && n > max {
				max = n
			}
		}
	}
	return toSlash(filepath.Join(dir, fmt.Sprintf("%02d-%s.md", max+1, slug)))
}

// InsideProject indica si la ruta es relativa y no sale de la carpeta actual.
func InsideProject(file string) bool {
	p := path.Clean(toSlash(file))
	return p != "." && !path.IsAbs(p) && !strings.HasPrefix(p, "../") && p != ".." && !filepath.IsAbs(file)
}

// FileSlug es Slug sin tildes ni caracteres fuera de [a-z0-9-], para nombres
// de archivo e ids que viajan en URLs ("Guía rápida" → "guia-rapida").
func FileSlug(s string) string {
	t := transform.Chain(norm.NFD, transform.RemoveFunc(func(r rune) bool {
		return unicode.Is(unicode.Mn, r)
	}), norm.NFC)
	clean, _, _ := transform.String(t, s)
	clean = strings.ToLower(clean)
	clean = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(clean, "-")
	return strings.Trim(clean, "-")
}
