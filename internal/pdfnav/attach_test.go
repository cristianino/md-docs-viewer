package pdfnav

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// minimalPDF arma un PDF de una página con xref clásica.
func minimalPDF(catalogExtra string) []byte {
	objs := []string{
		"<</Type /Catalog /Pages 2 0 R" + catalogExtra + ">>",
		"<</Type /Pages /Kids [3 0 R] /Count 1>>",
		"<</Type /Page /Parent 2 0 R /MediaBox [0 0 200 200]>>",
		"<</Dests <<>>>>",
	}
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offs := []int{}
	for i, o := range objs {
		offs = append(offs, b.Len())
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	x := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, o := range offs {
		fmt.Fprintf(&b, "%010d 00000 n \n", o)
	}
	fmt.Fprintf(&b, "trailer\n<</Size %d /Root 1 0 R>>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, x)
	return b.Bytes()
}

func TestFinalizeAttachments(t *testing.T) {
	files := []Attachment{
		{Name: "z.json", MIME: "application/json", Data: []byte(`{"a":1}`)},
		{Name: "colección.json", Description: "docs/files/colección.json", Data: []byte("{}")},
	}
	for name, extra := range map[string]string{"sin Names": "", "Names por referencia": " /Names 4 0 R", "Names en línea": " /Names <</Dests 4 0 R>>"} {
		out, err := Finalize(minimalPDF(extra), Options{Attachments: files})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		d, err := parse(out)
		if err != nil {
			t.Fatalf("%s: el resultado no se puede leer: %v", name, err)
		}
		s := string(out)
		if strings.Count(s, "/Type /EmbeddedFile ") != 2 || !strings.Contains(s, "/Subtype /application#2Fjson") {
			t.Errorf("%s: faltan los EmbeddedFile:\n%s", name, s)
		}
		if !strings.Contains(s, "/EmbeddedFiles <</Names [") {
			t.Errorf("%s: falta /EmbeddedFiles", name)
		}
		// z.json va después de colección.json (orden por nombre)
		if i, j := strings.Index(s, pdfString("colección.json")+" "), strings.Index(s, "(z.json) "); i < 0 || j < 0 || i > j {
			t.Errorf("%s: árbol de nombres sin ordenar", name)
		}
		if strings.Contains(extra, "Dests") && !strings.Contains(s, "/Dests 4 0 R") {
			t.Errorf("%s: se perdieron las entradas previas de /Names", name)
		}
		// la xref incremental trae el catálogo y los 4 objetos nuevos
		for _, n := range []int{1, 5, 6, 7, 8} {
			if _, ok := d.offsets[n]; !ok {
				t.Errorf("%s: el objeto %d no está en la xref", name, n)
			}
		}
	}
}

func TestPDFName(t *testing.T) {
	if got := pdfName("application/json"); got != "/application#2Fjson" {
		t.Errorf("pdfName = %s", got)
	}
}
