package docs

import (
	"strings"
	"testing"
)

var list = []*Doc{
	{ID: "intro", File: "docs/01-intro.md", Title: "Introducción"},
	{ID: "guia", File: "docs/02-guia.md", Title: "Guía"},
}

const src = "# Introducción\n\n## Cómo funciona\n\n" +
	"[a](02-guia.md) [b](./02-guia.md#paso-1) [c](#cómo-funciona) [d](/doc/guia) [e](https://x.dev)\n\n" +
	"![img](img/a.png)\n"

func TestRenderWebLinks(t *testing.T) {
	res, err := Render([]byte(src), Options{Mode: ModeWeb, Doc: list[0], All: list})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`href="/doc/guia"`, `href="/doc/guia#paso-1"`, `href="#c%C3%B3mo-funciona"`,
		`href="https://x.dev"`, `src="/files/docs/img/a.png"`, `<h2 id="cómo-funciona">`,
	} {
		if !strings.Contains(res.HTML, want) {
			t.Errorf("falta %s en:\n%s", want, res.HTML)
		}
	}
}

func TestRenderPrint(t *testing.T) {
	res, err := Render([]byte(src), Options{Mode: ModePrint, Doc: list[0], All: list, DropTitle: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`href="#doc-guia"`, `href="#guia--paso-1"`, `href="#intro--c%C3%B3mo-funciona"`,
		`<h3 id="intro--cómo-funciona" class="md-h2">`,
	} {
		if !strings.Contains(res.HTML, want) {
			t.Errorf("falta %s en:\n%s", want, res.HTML)
		}
	}
	if strings.Contains(res.HTML, "<h2 id=\"intro--introducción\"") || len(res.Headings) != 1 {
		t.Errorf("el H1 igual al título debía eliminarse: %+v", res.Headings)
	}
}

func TestSlugDedup(t *testing.T) {
	g := newIDs("")
	a := string(g.Generate([]byte("Paso 1"), 0))
	b := string(g.Generate([]byte("Paso 1"), 0))
	if a != "paso-1" || b != "paso-1-1" {
		t.Errorf("got %q %q", a, b)
	}
}

func TestTOCMarker(t *testing.T) {
	src := "# Título\n\n[TOC]\n\n## Uno\n\n### Uno.a\n\n## Dos\n"
	res, err := Render([]byte(src), Options{Mode: ModeWeb, Doc: list[0], All: list})
	if err != nil {
		t.Fatal(err)
	}
	want := `<nav class="md-toc" aria-label="Tabla de contenido"><ul><li><a href="#uno">Uno</a><ul><li><a href="#unoa">Uno.a</a></li></ul></li><li><a href="#dos">Dos</a></li></ul></nav>`
	if !strings.Contains(res.HTML, want) {
		t.Errorf("TOC inesperado:\n%s", res.HTML)
	}
	if strings.Contains(res.HTML, "[TOC]") {
		t.Error("el marcador no se reemplazó")
	}
}

func TestRenderBasePath(t *testing.T) {
	res, err := Render([]byte(src), Options{Mode: ModeWeb, Doc: list[0], All: list, Base: "/docs"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`href="/docs/doc/guia"`, `src="/docs/files/docs/img/a.png"`, `href="#c%C3%B3mo-funciona"`} {
		if !strings.Contains(res.HTML, want) {
			t.Errorf("falta %s en:\n%s", want, res.HTML)
		}
	}
}

func TestTOCMarkerInsideCodeIsKept(t *testing.T) {
	res, err := Render([]byte("# A\n\n```\n[TOC]\n```\n\n## B\n"), Options{Mode: ModeWeb, Doc: list[0], All: list})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(res.HTML, "md-toc") || !strings.Contains(res.HTML, "[TOC]") {
		t.Errorf("[TOC] dentro de código no debe reemplazarse:\n%s", res.HTML)
	}
}
