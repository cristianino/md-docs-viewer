package pdfnav

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Options controla Finalize.
type Options struct {
	// Cover es un PDF cuyas páginas se insertan al inicio (p. ej. la portada,
	// impresa sin pie de página). Puede ser nil.
	Cover []byte
	// CoverLabel es la etiqueta de página que muestran los visores para las
	// páginas de Cover (por defecto "Portada"). El resto se numera 1, 2, 3…
	// para coincidir con el número impreso en el pie.
	CoverLabel string
	// Attachments son archivos que se incrustan en el PDF (panel de
	// adjuntos del visor), p. ej. una colección de Postman enlazada desde
	// la documentación.
	Attachments []Attachment
}

// Attachment es un archivo incrustado en el PDF.
type Attachment struct {
	Name        string // nombre que muestra el visor (único)
	Description string
	MIME        string // p. ej. application/json; vacío = sin tipo
	Data        []byte
}

// rawObject es un objeto copiado tal cual: diccionario + stream opcional.
type rawObject struct {
	dict   []byte
	stream []byte // incluye "stream\n...endstream" o nil
}

// rawObj extrae el objeto num separando el diccionario del stream, para
// poder reescribir referencias sin tocar datos binarios.
func (d *doc) rawObj(num int) (*rawObject, error) {
	off, ok := d.offsets[num]
	if !ok {
		return nil, fmt.Errorf("pdfnav: objeto %d no está en xref", num)
	}
	b := d.data[off:]
	start := bytes.Index(b, []byte("obj"))
	if start < 0 {
		return nil, fmt.Errorf("pdfnav: objeto %d inválido", num)
	}
	b = b[start+3:]
	endobj := bytes.Index(b, []byte("endobj"))
	if endobj < 0 {
		return nil, fmt.Errorf("pdfnav: objeto %d sin endobj", num)
	}
	head := b[:endobj]
	si := streamKeyword(head)
	if si < 0 {
		return &rawObject{dict: bytes.TrimSpace(head)}, nil
	}
	dict := bytes.TrimSpace(head[:si])
	length := 0
	lv := key(dict, "Length")
	if n := refNum(lv); n != 0 {
		length, _ = strconv.Atoi(string(bytes.TrimSpace(d.object(n))))
	} else {
		length, _ = strconv.Atoi(string(lv))
	}
	dataStart := si + len("stream")
	if dataStart < len(b) && b[dataStart] == '\r' {
		dataStart++
	}
	if dataStart < len(b) && b[dataStart] == '\n' {
		dataStart++
	}
	if length <= 0 || dataStart+length > len(b) {
		return nil, fmt.Errorf("pdfnav: stream del objeto %d con Length inválido", num)
	}
	payload := b[dataStart : dataStart+length]
	stream := append([]byte("stream\n"), payload...)
	stream = append(stream, []byte("\nendstream")...)
	return &rawObject{dict: dict, stream: stream}, nil
}

// streamKeyword ubica "stream" fuera de strings del diccionario.
func streamKeyword(b []byte) int {
	depth := 0
	inStr := 0
	for i := 0; i < len(b); i++ {
		c := b[i]
		switch {
		case inStr > 0:
			if c == '\\' {
				i++
			} else if c == '(' {
				inStr++
			} else if c == ')' {
				inStr--
			}
		case c == '(':
			inStr = 1
		case c == '<' && i+1 < len(b) && b[i+1] == '<':
			depth++
			i++
		case c == '>' && i+1 < len(b) && b[i+1] == '>':
			depth--
			i++
		case depth == 0 && bytes.HasPrefix(b[i:], []byte("stream")):
			return i
		}
	}
	return -1
}

var reRefAll = regexp.MustCompile(`(\d+)\s+(\d+)\s+R\b`)

func renumber(dict []byte, offset int) []byte {
	return reRefAll.ReplaceAllFunc(dict, func(m []byte) []byte {
		sm := reRefAll.FindSubmatch(m)
		n, _ := strconv.Atoi(string(sm[1]))
		return []byte(fmt.Sprintf("%d 0 R", n+offset))
	})
}

func setKey(dict []byte, name, value string) []byte {
	inner := bytes.TrimSpace(bytes.TrimSuffix(bytes.TrimPrefix(bytes.TrimSpace(dict), []byte("<<")), []byte(">>")))
	if old := key(inner, name); old != nil {
		re := regexp.MustCompile(`/` + name + `\b\s*` + regexp.QuoteMeta(string(old)))
		inner = re.ReplaceAll(inner, nil)
	}
	return []byte(fmt.Sprintf("<<%s\n/%s %s>>", bytes.TrimSpace(inner), name, value))
}

// Finalize mejora la navegación del PDF con una actualización incremental:
//   - inserta las páginas de opt.Cover al inicio,
//   - añade etiquetas de página (portada + numeración desde 1),
//   - /PageMode /UseOutlines: el visor abre el panel de marcadores,
//   - /ViewerPreferences <</DisplayDocTitle true>>: muestra el título.
func Finalize(body []byte, opt Options) ([]byte, error) {
	d, err := parse(body)
	if err != nil {
		return nil, err
	}
	rootNum := d.root()
	cat := d.object(rootNum)
	if cat == nil || !bytes.HasPrefix(cat, []byte("<<")) {
		return nil, errors.New("pdfnav: catálogo no encontrado")
	}
	size, _ := strconv.Atoi(string(key(d.trailer, "Size")))
	if size == 0 {
		return nil, errors.New("pdfnav: trailer sin /Size")
	}

	updates := map[int][]byte{} // objeto -> contenido completo (dict + stream)
	coverPages := 0
	newSize := size

	if len(opt.Cover) > 0 {
		c, err := parse(opt.Cover)
		if err != nil {
			return nil, fmt.Errorf("portada: %w", err)
		}
		pagesNum := refNum(key(d.object(rootNum), "Pages"))
		pagesDict := d.object(pagesNum)
		if pagesNum == 0 || pagesDict == nil {
			return nil, errors.New("pdfnav: árbol de páginas no encontrado")
		}
		offset := size
		maxNum := 0
		for n := range c.offsets {
			if n > maxNum {
				maxNum = n
			}
		}
		coverRoot := c.root()
		coverPagesRoot := refNum(key(c.object(coverRoot), "Pages"))
		cp := c.pages()
		coverPages = len(cp)
		isCoverPage := map[int]bool{}
		for _, n := range cp {
			isCoverPage[n] = true
		}
		for n := range c.offsets {
			if n == 0 || n == coverRoot || n == coverPagesRoot {
				continue // catálogo y árbol de la portada no se necesitan
			}
			ro, err := c.rawObj(n)
			if err != nil {
				return nil, fmt.Errorf("portada: %w", err)
			}
			dict := renumber(ro.dict, offset)
			if isCoverPage[n] {
				dict = setKey(dict, "Parent", fmt.Sprintf("%d 0 R", pagesNum))
			}
			full := append([]byte{}, dict...)
			if ro.stream != nil {
				full = append(full, '\n')
				full = append(full, ro.stream...)
			}
			updates[n+offset] = full
		}
		newSize = offset + maxNum + 1

		// Kids: portada primero
		var kids bytes.Buffer
		kids.WriteString("[")
		for _, n := range cp {
			fmt.Fprintf(&kids, "%d 0 R ", n+offset)
		}
		oldKids := bytes.Trim(key(pagesDict, "Kids"), "[]")
		kids.Write(bytes.TrimSpace(oldKids))
		kids.WriteString("]")
		count, _ := strconv.Atoi(string(key(pagesDict, "Count")))
		pd := setKey(pagesDict, "Kids", kids.String())
		pd = setKey(pd, "Count", strconv.Itoa(count+coverPages))
		updates[pagesNum] = pd
	}

	newCat := setKey(cat, "PageMode", "/UseOutlines")
	if key(newCat, "ViewerPreferences") == nil {
		newCat = setKey(newCat, "ViewerPreferences", "<</DisplayDocTitle true>>")
	}
	if coverPages > 0 {
		label := opt.CoverLabel
		if label == "" {
			label = "Portada"
		}
		newCat = setKey(newCat, "PageLabels", fmt.Sprintf(
			"<</Nums [0 <</P %s>> %d <</S /D /St 1>>]>>", pdfString(label), coverPages))
	}
	if len(opt.Attachments) > 0 {
		var arr string
		arr, newSize = embedFiles(updates, newSize, opt.Attachments)
		ef := "<</Names " + arr + ">>"
		switch nv := bytes.TrimSpace(key(cat, "Names")); {
		case nv == nil:
			newCat = setKey(newCat, "Names", "<</EmbeddedFiles "+ef+">>")
		case bytes.HasPrefix(nv, []byte("<<")):
			newCat = setKey(newCat, "Names", string(setKey(nv, "EmbeddedFiles", ef)))
		default:
			n := refNum(nv)
			nd := d.object(n)
			if n == 0 || nd == nil || !bytes.HasPrefix(bytes.TrimSpace(nd), []byte("<<")) {
				return nil, errors.New("pdfnav: diccionario /Names inválido")
			}
			updates[n] = setKey(nd, "EmbeddedFiles", ef)
		}
	}
	updates[rootNum] = newCat

	return writeIncremental(body, d, updates, newSize, rootNum), nil
}

func writeIncremental(body []byte, d *doc, updates map[int][]byte, size, rootNum int) []byte {
	var buf bytes.Buffer
	buf.Write(body)
	if !bytes.HasSuffix(body, []byte("\n")) {
		buf.WriteByte('\n')
	}
	nums := make([]int, 0, len(updates))
	for n := range updates {
		nums = append(nums, n)
	}
	sort.Ints(nums)
	offs := map[int]int{}
	for _, n := range nums {
		offs[n] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n", n)
		buf.Write(updates[n])
		buf.WriteString("\nendobj\n")
	}
	xref := buf.Len()
	buf.WriteString("xref\n")
	// subsecciones de números consecutivos
	for i := 0; i < len(nums); {
		j := i
		for j+1 < len(nums) && nums[j+1] == nums[j]+1 {
			j++
		}
		fmt.Fprintf(&buf, "%d %d\n", nums[i], j-i+1)
		for k := i; k <= j; k++ {
			fmt.Fprintf(&buf, "%010d 00000 n \n", offs[nums[k]])
		}
		i = j + 1
	}
	trailer := fmt.Sprintf("trailer\n<</Size %d /Root %d 0 R", size, rootNum)
	if info := key(d.trailer, "Info"); info != nil {
		trailer += " /Info " + string(info)
	}
	if id := key(d.trailer, "ID"); id != nil {
		trailer += " /ID " + string(id)
	}
	trailer += fmt.Sprintf(" /Prev %d>>\n", d.xrefPos)
	buf.WriteString(trailer)
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xref)
	return buf.Bytes()
}

// pdfString codifica un texto como string PDF: literal si es ASCII
// imprimible, hexadecimal UTF-16BE si no.
func pdfString(s string) string {
	ascii := true
	for _, r := range s {
		if r < 0x20 || r > 0x7E {
			ascii = false
			break
		}
	}
	if ascii {
		r := strings.NewReplacer(`\`, `\\`, "(", `\(`, ")", `\)`)
		return "(" + r.Replace(s) + ")"
	}
	var b bytes.Buffer
	b.WriteString("<FEFF")
	for _, r := range s {
		if r > 0xFFFF {
			r1, r2 := (r-0x10000)>>10+0xD800, (r-0x10000)&0x3FF+0xDC00
			fmt.Fprintf(&b, "%04X%04X", r1, r2)
			continue
		}
		fmt.Fprintf(&b, "%04X", r)
	}
	b.WriteString(">")
	return b.String()
}

// embedFiles agrega un /EmbeddedFile y un /Filespec por adjunto a partir
// del objeto next. Devuelve el arreglo del árbol de nombres (ordenado por
// nombre, como exige la especificación) y el siguiente número libre.
func embedFiles(updates map[int][]byte, next int, files []Attachment) (string, int) {
	sorted := append([]Attachment{}, files...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	var arr bytes.Buffer
	arr.WriteString("[")
	for _, f := range sorted {
		fileNum, specNum := next, next+1
		next += 2
		var ef bytes.Buffer
		ef.WriteString("<</Type /EmbeddedFile")
		if f.MIME != "" {
			ef.WriteString(" /Subtype " + pdfName(f.MIME))
		}
		fmt.Fprintf(&ef, " /Length %d /Params <</Size %d>>>>\nstream\n", len(f.Data), len(f.Data))
		ef.Write(f.Data)
		ef.WriteString("\nendstream")
		updates[fileNum] = ef.Bytes()

		name := pdfString(f.Name)
		spec := fmt.Sprintf("<</Type /Filespec /F %s /UF %s /EF <</F %d 0 R /UF %d 0 R>> /AFRelationship /Supplement", name, name, fileNum, fileNum)
		if f.Description != "" {
			spec += " /Desc " + pdfString(f.Description)
		}
		updates[specNum] = []byte(spec + ">>")
		fmt.Fprintf(&arr, "%s %d 0 R ", name, specNum)
	}
	return strings.TrimSpace(arr.String()) + "]", next
}

// pdfName codifica s como nombre PDF (/application#2Fjson).
func pdfName(s string) string {
	var b strings.Builder
	b.WriteByte('/')
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < 0x21 || c > 0x7E || strings.IndexByte("#()<>[]{}/%", c) >= 0 {
			fmt.Fprintf(&b, "#%02X", c)
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}
