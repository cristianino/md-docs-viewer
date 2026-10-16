package server

import (
	"encoding/xml"
	"strings"
	"testing"
)

func sampleCapture() ([]mermaidBlock, *capture) {
	blocks := []mermaidBlock{
		{DocID: "a", DocTitle: "Doc A", Index: 1, Code: "flowchart LR\nA-->B"},
		{DocID: "a", DocTitle: "Doc A", Index: 2, Code: "pie\n\"x\": 1"},
	}
	c := &capture{
		exports: []diagramExport{
			{Kind: "graph", Type: "flowchart-v2",
				Nodes: []exportNode{{ID: "A", Label: "Inicio & fin", W: 80, H: 40, Shape: "diamond"}, {ID: "B", Label: "B", X: 200, W: 80, H: 40}},
				Edges: []exportEdge{{Source: "A", Target: "B", Label: "sí", EndArrow: "block", Points: [][2]float64{{80, 20}, {140, 20}, {200, 20}}}}},
			{Kind: "image", Type: "pie"},
		},
		pngs: make([][]byte, 2),
	}
	return blocks, c
}

func TestBuildDrawioIsValidXML(t *testing.T) {
	blocks, c := sampleCapture()
	data, report := buildDrawio(blocks, c, DrawioAuto)
	var v struct {
		Diagrams []struct {
			Name string `xml:"name,attr"`
		} `xml:"diagram"`
	}
	if err := xml.Unmarshal(data, &v); err != nil {
		t.Fatalf("XML inválido: %v", err)
	}
	if len(v.Diagrams) != 2 {
		t.Fatalf("se esperaban 2 páginas, hay %d", len(v.Diagrams))
	}
	if report[0].Output != "native" || report[1].Output != "code" {
		t.Errorf("reporte inesperado: %+v", report)
	}
	s := string(data)
	if !strings.Contains(s, `source="c2" target="c3"`) || !strings.Contains(s, "shape=rhombus") {
		t.Errorf("faltan formas/aristas nativas:\n%s", s)
	}
}

func TestBuildDrawioImageMode(t *testing.T) {
	blocks, c := sampleCapture()
	_, report := buildDrawio(blocks, c, DrawioImage)
	if report[0].Output == "native" {
		t.Error("mode=image no debe generar formas nativas")
	}
}

func TestCSSToHex(t *testing.T) {
	if got := cssToHex("rgb(236, 236, 255)"); got != "#ececff" {
		t.Errorf("got %s", got)
	}
}
