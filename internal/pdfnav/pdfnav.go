// Package pdfnav lee el árbol de marcadores (outline) de un PDF generado
// por Chrome y le aplica ajustes de navegación mediante una actualización
// incremental. Soporta PDFs con tabla xref clásica (la que produce Chrome);
// si el PDF usa otro formato, devuelve un error y el llamador sigue sin
// los extras.
package pdfnav

import (
	"bytes"
	"errors"
	"regexp"
	"strconv"
	"unicode/utf16"
)

// Bookmark es una entrada del outline, aplanada en orden de documento.
type Bookmark struct {
	Title string
	Level int // 1 = raíz
	Page  int // 1-based; 0 si no se pudo resolver
}

type doc struct {
	data    []byte
	offsets map[int]int // objeto -> offset
	trailer []byte
	xrefPos int
}

var (
	reStartXref = regexp.MustCompile(`startxref\s+(\d+)\s+%%EOF\s*$`)
	reRef       = regexp.MustCompile(`(\d+)\s+(\d+)\s+R`)
)

func parse(data []byte) (*doc, error) {
	m := reStartXref.FindSubmatch(bytes.TrimRight(data, "\x00"))
	if m == nil {
		return nil, errors.New("pdfnav: startxref no encontrado")
	}
	pos, _ := strconv.Atoi(string(m[1]))
	d := &doc{data: data, offsets: map[int]int{}, xrefPos: pos}
	if pos >= len(data) || !bytes.HasPrefix(data[pos:], []byte("xref")) {
		return nil, errors.New("pdfnav: xref en formato no soportado")
	}
	// Recorre la sección xref: "xref\n0 N\n<offset> <gen> n \n..."
	lines := bytes.Split(data[pos:], []byte("\n"))
	i := 1
	for i < len(lines) {
		line := bytes.TrimSpace(lines[i])
		if bytes.HasPrefix(line, []byte("trailer")) {
			break
		}
		f := bytes.Fields(line)
		if len(f) != 2 {
			return nil, errors.New("pdfnav: subsección xref inválida")
		}
		start, _ := strconv.Atoi(string(f[0]))
		count, _ := strconv.Atoi(string(f[1]))
		for k := 0; k < count && i+1+k < len(lines); k++ {
			e := bytes.Fields(lines[i+1+k])
			if len(e) == 3 && string(e[2]) == "n" {
				off, _ := strconv.Atoi(string(e[0]))
				d.offsets[start+k] = off
			}
		}
		i += 1 + count
	}
	t := bytes.Index(data[pos:], []byte("trailer"))
	if t < 0 {
		return nil, errors.New("pdfnav: trailer no encontrado")
	}
	end := bytes.Index(data[pos+t:], []byte("startxref"))
	d.trailer = data[pos+t+len("trailer") : pos+t+end]
	return d, nil
}

// object devuelve el texto del objeto num (sin stream).
func (d *doc) object(num int) []byte {
	off, ok := d.offsets[num]
	if !ok || off >= len(d.data) {
		return nil
	}
	b := d.data[off:]
	start := bytes.Index(b, []byte("obj"))
	if start < 0 {
		return nil
	}
	b = b[start+3:]
	end := len(b)
	if i := bytes.Index(b, []byte("endobj")); i >= 0 {
		end = i
	}
	if i := bytes.Index(b[:end], []byte("stream")); i >= 0 {
		end = i
	}
	return bytes.TrimSpace(b[:end])
}

// key devuelve el valor crudo de /Name en un diccionario (primer nivel
// aproximado: referencia, nombre, número, string o arreglo).
func key(dict []byte, name string) []byte {
	re := regexp.MustCompile(`/` + name + `\b\s*`)
	loc := re.FindIndex(dict)
	if loc == nil {
		return nil
	}
	rest := dict[loc[1]:]
	if len(rest) == 0 {
		return nil
	}
	switch rest[0] {
	case '[':
		return balanced(rest, '[', ']')
	case '(':
		return literalString(rest)
	case '<':
		if len(rest) > 1 && rest[1] == '<' {
			return balanced(rest, '<', '>')
		}
		if i := bytes.IndexByte(rest, '>'); i >= 0 {
			return rest[:i+1]
		}
	case '/':
		j := 1
		for j < len(rest) && !bytes.ContainsRune([]byte(" \t\r\n/[]<>()"), rune(rest[j])) {
			j++
		}
		return rest[:j]
	}
	if m := regexp.MustCompile(`^\d+\s+\d+\s+R`).Find(rest); m != nil {
		return m
	}
	if m := regexp.MustCompile(`^[-+.\d]+`).Find(rest); m != nil {
		return m
	}
	return nil
}

func balanced(b []byte, open, close byte) []byte {
	depth := 0
	for i := 0; i < len(b); i++ {
		switch b[i] {
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return b[:i+1]
			}
		}
	}
	return b
}

func literalString(b []byte) []byte {
	depth := 0
	for i := 0; i < len(b); i++ {
		switch b[i] {
		case '\\':
			i++
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return b[:i+1]
			}
		}
	}
	return b
}

func refNum(v []byte) int {
	m := reRef.FindSubmatch(v)
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(string(m[1]))
	return n
}

func (d *doc) root() int { return refNum(key(d.trailer, "Root")) }

// pages devuelve los números de objeto de las páginas en orden.
func (d *doc) pages() []int {
	var out []int
	var walk func(n, depth int)
	walk = func(n, depth int) {
		if depth > 64 {
			return
		}
		obj := d.object(n)
		if bytes.Contains(key(obj, "Type"), []byte("/Pages")) {
			for _, m := range reRef.FindAllSubmatch(key(obj, "Kids"), -1) {
				k, _ := strconv.Atoi(string(m[1]))
				walk(k, depth+1)
			}
			return
		}
		out = append(out, n)
	}
	walk(refNum(key(d.object(d.root()), "Pages")), 0)
	return out
}

// Outline devuelve los marcadores del PDF aplanados en orden.
func Outline(data []byte) ([]Bookmark, error) {
	d, err := parse(data)
	if err != nil {
		return nil, err
	}
	pageIndex := map[int]int{}
	for i, p := range d.pages() {
		pageIndex[p] = i + 1
	}
	outlines := refNum(key(d.object(d.root()), "Outlines"))
	if outlines == 0 {
		return nil, errors.New("pdfnav: el PDF no tiene outline")
	}
	var out []Bookmark
	seen := map[int]bool{}
	var walk func(n, level int)
	walk = func(n, level int) {
		for n != 0 && !seen[n] {
			seen[n] = true
			obj := d.object(n)
			dest := key(obj, "Dest")
			if dest == nil {
				if a := key(obj, "A"); a != nil {
					dest = key(a, "D")
				}
			}
			out = append(out, Bookmark{
				Title: decodeString(key(obj, "Title")),
				Level: level,
				Page:  pageIndex[refNum(dest)],
			})
			if first := refNum(key(obj, "First")); first != 0 {
				walk(first, level+1)
			}
			n = refNum(key(obj, "Next"))
		}
	}
	walk(refNum(key(d.object(outlines), "First")), 1)
	return out, nil
}

// decodeString decodifica un string PDF literal o hexadecimal (UTF-16BE con BOM o PDFDoc).
func decodeString(v []byte) string {
	if len(v) == 0 {
		return ""
	}
	var raw []byte
	switch v[0] {
	case '<':
		h := bytes.Trim(v, "<> \n\r\t")
		h = bytes.Join(bytes.Fields(h), nil)
		if len(h)%2 == 1 {
			h = append(h, '0')
		}
		for i := 0; i+1 < len(h); i += 2 {
			n, _ := strconv.ParseUint(string(h[i:i+2]), 16, 8)
			raw = append(raw, byte(n))
		}
	case '(':
		s := v[1 : len(v)-1]
		for i := 0; i < len(s); i++ {
			c := s[i]
			if c != '\\' || i+1 >= len(s) {
				raw = append(raw, c)
				continue
			}
			i++
			switch s[i] {
			case 'n':
				raw = append(raw, '\n')
			case 'r':
				raw = append(raw, '\r')
			case 't':
				raw = append(raw, '\t')
			case 'b':
				raw = append(raw, '\b')
			case 'f':
				raw = append(raw, '\f')
			case '\n':
			case '0', '1', '2', '3', '4', '5', '6', '7':
				j := i
				for j < len(s) && j < i+3 && s[j] >= '0' && s[j] <= '7' {
					j++
				}
				n, _ := strconv.ParseUint(string(s[i:j]), 8, 8)
				raw = append(raw, byte(n))
				i = j - 1
			default:
				raw = append(raw, s[i])
			}
		}
	default:
		return string(v)
	}
	if len(raw) >= 2 && raw[0] == 0xFE && raw[1] == 0xFF {
		u := make([]uint16, 0, len(raw)/2)
		for i := 2; i+1 < len(raw); i += 2 {
			u = append(u, uint16(raw[i])<<8|uint16(raw[i+1]))
		}
		return string(utf16.Decode(u))
	}
	// PDFDocEncoding ≈ Latin-1 para el rango imprimible
	r := make([]rune, len(raw))
	for i, c := range raw {
		r[i] = rune(c)
	}
	return string(r)
}
