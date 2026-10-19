package docs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const searchSrc = "# Guía de la API\n\nIntro con **Postman**.\n\n[TOC]\n\n## Autenticación\n\nUsa un token Bearer.\n\n### Errores de autenticación\n\nCódigos 401.\n\n```mermaid\nflowchart LR\n  secreto --> x\n```\n\n## Colección Postman\n\nDescarga la [colección](files/api.json).\n"

func TestIndexDoc(t *testing.T) {
	d := IndexDoc([]byte(searchSrc), &Doc{ID: "api", File: "docs/api.md", Title: "Guía de la API"})
	if len(d.Sections) != 4 {
		t.Fatalf("secciones = %d, quiero 4: %+v", len(d.Sections), d.Sections)
	}
	if s := d.Sections[1]; s.Heading != "Autenticación" || s.Anchor != "autenticación" || s.Level != 2 {
		t.Errorf("sección 1 = %+v", s)
	}
	if strings.Contains(d.Sections[0].Text, "TOC") {
		t.Errorf("el marcador [TOC] no debe indexarse: %q", d.Sections[0].Text)
	}
	for _, s := range d.Sections {
		if strings.Contains(s.Text, "secreto") {
			t.Errorf("los bloques mermaid no deben indexarse: %q", s.Text)
		}
	}
}

func TestSearchHierarchy(t *testing.T) {
	idx := []SearchDoc{
		IndexDoc([]byte(searchSrc), &Doc{ID: "api", File: "docs/api.md", Title: "Guía de la API"}),
		{ID: "postman", Title: "Postman", Subtitle: "Colecciones", Sections: []SearchSection{{Heading: "Postman", Level: 1, Anchor: "postman", Text: "Cómo importar."}}},
	}
	hits := Search(idx, "postman", 0)
	if len(hits) < 3 {
		t.Fatalf("hits = %+v", hits)
	}
	if hits[0].Tier != TierTitle || hits[0].DocID != "postman" {
		t.Errorf("primero debe ir el título: %+v", hits[0])
	}
	if hits[1].Tier != TierHeading || hits[1].Anchor != "colección-postman" {
		t.Errorf("luego el subtítulo: %+v", hits[1])
	}
	if hits[len(hits)-1].Tier != TierContent {
		t.Errorf("al final el contenido: %+v", hits[len(hits)-1])
	}
	for i := 1; i < len(hits); i++ {
		if hits[i].Tier < hits[i-1].Tier {
			t.Fatalf("jerarquía rota: %+v", hits)
		}
	}
}

func TestSearchAccentsAndTerms(t *testing.T) {
	idx := []SearchDoc{IndexDoc([]byte(searchSrc), &Doc{ID: "api", File: "docs/api.md", Title: "Guía de la API"})}
	if h := Search(idx, "AUTENTICACION", 0); len(h) == 0 || h[0].Heading != "Autenticación" {
		t.Errorf("sin tildes ni mayúsculas: %+v", h)
	}
	if h := Search(idx, "errores 401", 0); len(h) != 1 || h[0].Tier != TierContent || h[0].Anchor != "errores-de-autenticación" {
		t.Errorf("todas las palabras: %+v", h)
	}
	if h := Search(idx, "inexistente", 0); len(h) != 0 {
		t.Errorf("sin resultados: %+v", h)
	}
	if h := Search(idx, "a", 2); len(h) != 2 {
		t.Errorf("limit: %d", len(h))
	}
}

func TestSnippet(t *testing.T) {
	long := strings.Repeat("palabra ", 40) + "Configuración final " + strings.Repeat("resto ", 40)
	s := Snippet(long, []string{"configuracion"}, 60)
	if !strings.Contains(s, "Configuración") || !strings.HasPrefix(s, "…") || !strings.HasSuffix(s, "…") {
		t.Errorf("snippet = %q", s)
	}
}

func TestLocalFile(t *testing.T) {
	d := &Doc{ID: "api", File: "docs/api.md"}
	cases := map[string]string{
		"files/api.postman_collection.json": "docs/files/api.postman_collection.json",
		"../assets/schema%20v2.sql":         "assets/schema v2.sql",
		"files/a.json#x":                    "docs/files/a.json",
	}
	for in, want := range cases {
		if got, ok := LocalFile(in, d); !ok || got != want {
			t.Errorf("LocalFile(%q) = %q, %v; quiero %q", in, got, ok, want)
		}
	}
	for _, in := range []string{"02-guia.md", "#ancla", "https://x.com/a.json", "/doc/x", "../../fuera.json", ".env", "files/.secret/a.txt", "mailto:a@b.c", ""} {
		if got, ok := LocalFile(in, d); ok {
			t.Errorf("LocalFile(%q) = %q; no debía ser descargable", in, got)
		}
	}
}

func TestRenderFileLinks(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	os.MkdirAll(filepath.Join("docs", "files"), 0755)
	os.WriteFile(filepath.Join("docs", "files", "api.json"), []byte("{}"), 0644)
	d := &Doc{ID: "api", File: "docs/api.md", Title: "API"}
	src := []byte("# API\n\n[Colección](files/api.json) y [guía](02-guia.md). [Falta](files/no.zip)\n")
	web, err := Render(src, Options{Mode: ModeWeb, Doc: d, All: []*Doc{d}, Base: "/docs"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(web.HTML, `href="/docs/files/docs/files/api.json"`) || !strings.Contains(web.HTML, `download="api.json"`) || !strings.Contains(web.HTML, `class="md-file"`) {
		t.Errorf("web: %s", web.HTML)
	}
	if len(web.Files) != 2 || web.Files[0] != "docs/files/api.json" {
		t.Errorf("files = %v", web.Files)
	}
	pr, err := Render(src, Options{Mode: ModePrint, Doc: d, All: []*Doc{d}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(pr.HTML, AttachmentScheme) || !strings.Contains(pr.HTML, `<span class="md-attachment">Colección</span>`) || !strings.Contains(pr.HTML, "(adjunto: api.json)") {
		t.Errorf("print: %s", pr.HTML)
	}
	if strings.Contains(pr.HTML, "adjunto: no.zip") || !strings.Contains(pr.HTML, `<span class="md-attachment">Falta</span>`) {
		t.Errorf("un archivo inexistente no se anuncia como adjunto: %s", pr.HTML)
	}
}
