package server

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"html"
	"image"
	_ "image/png"
	"math"
	"strings"
	"time"
)

// diagramExport es la geometría que extrae mermaid-export.gohtml.
type diagramExport struct {
	Kind     string       `json:"kind"` // graph | sequence | image | error
	Type     string       `json:"type"` // tipo Mermaid (flowchart-v2, sequence, ...)
	Error    string       `json:"error"`
	Width    float64      `json:"width"`
	Height   float64      `json:"height"`
	Nodes    []exportNode `json:"nodes"`
	Clusters []exportNode `json:"clusters"`
	Edges    []exportEdge `json:"edges"`
	Rects    []exportRect `json:"rects"`
	Lines    []exportEdge `json:"lines"`
	Texts    []exportText `json:"texts"`
}

type exportNode struct {
	ID        string  `json:"id"`
	Label     string  `json:"label"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	W         float64 `json:"w"`
	H         float64 `json:"h"`
	Shape     string  `json:"shape"`
	Fill      string  `json:"fill"`
	Stroke    string  `json:"stroke"`
	ClassLike bool    `json:"classLike"`
	FontSize  float64 `json:"fontSize"`
}

type exportEdge struct {
	Source     string       `json:"source"`
	Target     string       `json:"target"`
	Label      string       `json:"label"`
	Points     [][2]float64 `json:"points"`
	StartArrow string       `json:"startArrow"`
	EndArrow   string       `json:"endArrow"`
	Dashed     bool         `json:"dashed"`
	Thick      bool         `json:"thick"`
	Stroke     string       `json:"stroke"`
	Width      float64      `json:"width"`
	Cls        string       `json:"cls"`
}

type exportRect struct {
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	W       float64 `json:"w"`
	H       float64 `json:"h"`
	Label   string  `json:"label"`
	Cls     string  `json:"cls"`
	Fill    string  `json:"fill"`
	Stroke  string  `json:"stroke"`
	Rounded bool    `json:"rounded"`
}

type exportText struct {
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	W        float64 `json:"w"`
	H        float64 `json:"h"`
	Text     string  `json:"text"`
	Cls      string  `json:"cls"`
	FontSize float64 `json:"fontSize"`
}

// offsetY deja espacio para el título de cada página.
const (
	offsetX = 20.0
	offsetY = 64.0
)

// cellWriter escribe celdas mxCell con ids consecutivos.
type cellWriter struct {
	sb   strings.Builder
	next int
}

func (c *cellWriter) id() string {
	c.next++
	return fmt.Sprintf("c%d", c.next)
}

func xmlAttr(s string) string {
	var buf strings.Builder
	xml.EscapeText(&buf, []byte(s))
	return strings.ReplaceAll(buf.String(), "\n", "&#xa;")
}

// htmlLabel convierte texto plano (con \n) en una etiqueta HTML de draw.io.
func htmlLabel(s string) string {
	return strings.ReplaceAll(html.EscapeString(s), "\n", "<br>")
}

func f(v float64) string { return fmt.Sprintf("%.1f", v) }

func (c *cellWriter) vertex(id, value, style string, x, y, w, h float64) {
	fmt.Fprintf(&c.sb, `        <mxCell id="%s" value="%s" style="%s" vertex="1" parent="1">`+"\n", id, xmlAttr(value), xmlAttr(style))
	fmt.Fprintf(&c.sb, `          <mxGeometry x="%s" y="%s" width="%s" height="%s" as="geometry"/>`+"\n", f(x), f(y), f(w), f(h))
	c.sb.WriteString("        </mxCell>\n")
}

// edge escribe una arista. Si source/target están vacíos usa los puntos
// extremos como sourcePoint/targetPoint (líneas libres).
func (c *cellWriter) edge(value, style, source, target string, pts [][2]float64, waypoints [][2]float64) {
	attrs := ""
	if source != "" {
		attrs += fmt.Sprintf(` source="%s"`, source)
	}
	if target != "" {
		attrs += fmt.Sprintf(` target="%s"`, target)
	}
	fmt.Fprintf(&c.sb, `        <mxCell id="%s" value="%s" style="%s" edge="1" parent="1"%s>`+"\n", c.id(), xmlAttr(value), xmlAttr(style), attrs)
	c.sb.WriteString(`          <mxGeometry relative="1" as="geometry">` + "\n")
	if len(pts) >= 2 {
		if source == "" {
			p := pts[0]
			fmt.Fprintf(&c.sb, `            <mxPoint x="%s" y="%s" as="sourcePoint"/>`+"\n", f(p[0]+offsetX), f(p[1]+offsetY))
		}
		if target == "" {
			p := pts[len(pts)-1]
			fmt.Fprintf(&c.sb, `            <mxPoint x="%s" y="%s" as="targetPoint"/>`+"\n", f(p[0]+offsetX), f(p[1]+offsetY))
		}
	}
	if len(waypoints) > 0 {
		c.sb.WriteString(`            <Array as="points">` + "\n")
		for _, p := range waypoints {
			fmt.Fprintf(&c.sb, `              <mxPoint x="%s" y="%s"/>`+"\n", f(p[0]+offsetX), f(p[1]+offsetY))
		}
		c.sb.WriteString("            </Array>\n")
	}
	c.sb.WriteString("          </mxGeometry>\n        </mxCell>\n")
}

func colorOr(css, def string) string {
	if hex := cssToHex(css); hex != "" {
		return hex
	}
	return def
}

// cssToHex convierte "rgb(r, g, b)" o "#rrggbb" en "#rrggbb".
func cssToHex(css string) string {
	css = strings.TrimSpace(css)
	if strings.HasPrefix(css, "#") {
		return css
	}
	var r, g, b int
	var a float64 = 1
	if n, _ := fmt.Sscanf(css, "rgb(%d, %d, %d)", &r, &g, &b); n == 3 {
		return fmt.Sprintf("#%02x%02x%02x", r, g, b)
	}
	if n, _ := fmt.Sscanf(css, "rgba(%d, %d, %d, %g)", &r, &g, &b, &a); n == 4 {
		if a == 0 {
			return "none"
		}
		return fmt.Sprintf("#%02x%02x%02x", r, g, b)
	}
	return ""
}

// shapeStyle traduce la forma Mermaid a estilo draw.io.
func shapeStyle(shape string) string {
	switch shape {
	case "round":
		return "rounded=1;arcSize=20;"
	case "stadium":
		return "rounded=1;arcSize=50;"
	case "subroutine":
		return "shape=process;backgroundOutline=1;"
	case "cylinder":
		return "shape=cylinder3;boundedLbl=1;backgroundOutline=1;size=8;"
	case "circle":
		return "shape=ellipse;perimeter=ellipsePerimeter;aspect=fixed;"
	case "doublecircle":
		return "shape=doubleEllipse;perimeter=ellipsePerimeter;aspect=fixed;"
	case "diamond":
		return "shape=rhombus;perimeter=rhombusPerimeter;"
	case "hexagon":
		return "shape=hexagon;perimeter=hexagonPerimeter2;fixedSize=1;size=12;"
	case "odd":
		return "shape=mxgraph.arrows2.arrow;dy=0;dx=0;notch=12;direction=west;"
	case "trapezoid":
		return "shape=trapezoid;perimeter=trapezoidPerimeter;fixedSize=1;size=12;"
	case "inv_trapezoid":
		return "shape=trapezoid;perimeter=trapezoidPerimeter;fixedSize=1;size=12;flipV=1;"
	case "lean_right":
		return "shape=parallelogram;perimeter=parallelogramPerimeter;fixedSize=1;size=12;"
	case "lean_left":
		return "shape=parallelogram;perimeter=parallelogramPerimeter;fixedSize=1;size=12;flipH=1;"
	}
	return "rounded=0;"
}

func arrowStyle(prefix, kind string) string {
	switch kind {
	case "block":
		return prefix + "Arrow=block;" + prefix + "Fill=1;"
	case "oval":
		return prefix + "Arrow=oval;" + prefix + "Fill=1;"
	case "cross":
		return prefix + "Arrow=cross;"
	}
	return prefix + "Arrow=none;"
}

func edgeStyle(e exportEdge) string {
	s := "html=1;rounded=1;labelBackgroundColor=#ffffff;fontSize=12;" +
		arrowStyle("end", e.EndArrow) + arrowStyle("start", e.StartArrow)
	s += "strokeColor=" + colorOr(e.Stroke, "#333333") + ";"
	if e.Dashed {
		s += "dashed=1;"
	}
	if e.Thick {
		s += "strokeWidth=3;"
	} else if e.Width > 1.5 {
		s += fmt.Sprintf("strokeWidth=%.0f;", e.Width)
	}
	return s
}

// waypoints toma algunos puntos interiores del trazado; si la arista es casi
// recta no devuelve ninguno (draw.io la traza directamente).
func waypoints(pts [][2]float64) [][2]float64 {
	if len(pts) < 3 {
		return nil
	}
	a, b := pts[0], pts[len(pts)-1]
	dx, dy := b[0]-a[0], b[1]-a[1]
	length := math.Hypot(dx, dy)
	maxDev := 0.0
	for _, p := range pts[1 : len(pts)-1] {
		var d float64
		if length < 1e-6 {
			d = math.Hypot(p[0]-a[0], p[1]-a[1])
		} else {
			d = math.Abs(dy*p[0]-dx*p[1]+b[0]*a[1]-b[1]*a[0]) / length
		}
		maxDev = math.Max(maxDev, d)
	}
	if maxDev < 4 {
		return nil
	}
	var out [][2]float64
	for i := 2; i < len(pts)-1; i += 2 {
		out = append(out, pts[i])
	}
	return out
}

func writeGraph(c *cellWriter, e *diagramExport) {
	for _, cl := range e.Clusters {
		style := "rounded=0;whiteSpace=wrap;html=1;verticalAlign=top;align=center;fontStyle=1;" +
			"fillColor=" + colorOr(cl.Fill, "#ffffde") + ";strokeColor=" + colorOr(cl.Stroke, "#aaaa33") + ";"
		c.vertex(c.id(), htmlLabel(cl.Label), style, cl.X+offsetX, cl.Y+offsetY, cl.W, cl.H)
	}
	ids := map[string]string{}
	for _, n := range e.Nodes {
		id := c.id()
		ids[n.ID] = id
		style := shapeStyle(n.Shape) + "whiteSpace=wrap;html=1;" +
			"fillColor=" + colorOr(n.Fill, "#ECECFF") + ";strokeColor=" + colorOr(n.Stroke, "#9370DB") + ";"
		if n.FontSize > 0 {
			style += fmt.Sprintf("fontSize=%.0f;", n.FontSize)
		}
		label := htmlLabel(n.Label)
		if n.ClassLike {
			// clase UML: primera línea en negrita y separada
			lines := strings.SplitN(n.Label, "\n", 2)
			label = "<b>" + html.EscapeString(lines[0]) + "</b>"
			if len(lines) > 1 {
				label += "<hr>" + htmlLabel(lines[1])
			}
			style += "align=left;verticalAlign=top;spacingLeft=6;"
		}
		c.vertex(id, label, style, n.X+offsetX, n.Y+offsetY, n.W, n.H)
	}
	for _, ed := range e.Edges {
		src, dst := ids[ed.Source], ids[ed.Target]
		wp := waypoints(ed.Points)
		style := edgeStyle(ed)
		if len(wp) > 0 {
			style += "curved=1;"
		}
		c.edge(htmlLabel(ed.Label), style, src, dst, ed.Points, wp)
	}
}

func writeSequence(c *cellWriter, e *diagramExport) {
	for _, r := range e.Rects {
		fill := colorOr(r.Fill, "#ffffff")
		stroke := colorOr(r.Stroke, "#666666")
		style := "whiteSpace=wrap;html=1;fillColor=" + fill + ";strokeColor=" + stroke + ";"
		switch {
		case strings.Contains(r.Cls, "note"):
			style = "shape=note;size=10;whiteSpace=wrap;html=1;fillColor=" + colorOr(r.Fill, "#fff5ad") + ";strokeColor=" + colorOr(r.Stroke, "#aaaa33") + ";"
		case strings.Contains(r.Cls, "labelBox"):
			style = "shape=card;size=8;flipV=1;whiteSpace=wrap;html=1;fontStyle=1;fillColor=" + fill + ";strokeColor=" + stroke + ";"
		case strings.Contains(r.Cls, "rect") && r.Label == "":
			style += "opacity=40;" // bloques rect de fondo
		}
		if r.Rounded {
			style += "rounded=1;"
		}
		c.vertex(c.id(), htmlLabel(r.Label), style, r.X+offsetX, r.Y+offsetY, r.W, r.H)
	}
	for _, l := range e.Lines {
		if len(l.Points) < 2 {
			continue
		}
		style := edgeStyle(l)
		var wp [][2]float64
		if len(l.Points) > 2 {
			wp = l.Points[1 : len(l.Points)-1]
			style += "curved=1;"
		}
		c.edge("", style, "", "", l.Points, wp)
	}
	for _, t := range e.Texts {
		size := t.FontSize
		if size == 0 {
			size = 12
		}
		pad := 4.0
		style := fmt.Sprintf("text;html=1;strokeColor=none;fillColor=none;align=center;verticalAlign=middle;whiteSpace=nowrap;fontSize=%.0f;", size)
		c.vertex(c.id(), htmlLabel(t.Text), style, t.X+offsetX-pad, t.Y+offsetY-pad/2, t.W+2*pad, t.H+pad)
	}
}

func writeImage(c *cellWriter, png []byte) (ok bool) {
	if len(png) == 0 {
		return false
	}
	w, h := 700.0, 460.0
	if cfg, _, err := image.DecodeConfig(bytes.NewReader(png)); err == nil && cfg.Width > 0 {
		// captura a escala 1: usar el tamaño real
		w, h = float64(cfg.Width), float64(cfg.Height)
	}
	// draw.io guarda imágenes como "data:image/png,<base64>" (sin ";base64",
	// porque el ";" corta el valor dentro del estilo).
	style := "shape=image;imageAspect=1;aspect=fixed;verticalLabelPosition=bottom;verticalAlign=top;" +
		"image=data:image/png," + base64.StdEncoding.EncodeToString(png) + ";"
	c.vertex(c.id(), "", style, offsetX, offsetY, w, h)
	return true
}

func writeCode(c *cellWriter, code string) {
	style := "text;html=1;strokeColor=#d6b656;fillColor=#fffacd;align=left;verticalAlign=top;" +
		"spacingTop=4;spacingLeft=8;whiteSpace=pre;overflow=hidden;rounded=1;fontFamily=Courier New;"
	c.vertex(c.id(), "<pre>"+html.EscapeString(code)+"</pre>", style, offsetX, offsetY, 700, 400)
}

// buildDrawio arma el XML .drawio: una página por diagrama.
func buildDrawio(blocks []mermaidBlock, cap *capture, mode DrawioMode) ([]byte, []DiagramReport) {
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	var sb strings.Builder
	var report []DiagramReport

	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<mxfile host="mdocs" modified="` + xmlAttr(now) + `" agent="mdocs" version="24.0.0" type="device">` + "\n")

	titleStyle := "text;html=1;strokeColor=none;fillColor=none;align=left;verticalAlign=top;fontStyle=1;fontSize=14;"
	noteStyle := "text;html=1;strokeColor=none;fillColor=none;align=left;verticalAlign=top;fontSize=10;fontColor=#718096;"

	for i, b := range blocks {
		var exp diagramExport
		if i < len(cap.exports) {
			exp = cap.exports[i]
		}
		var png []byte
		if i < len(cap.pngs) {
			png = cap.pngs[i]
		}
		rep := DiagramReport{Doc: b.DocID, Index: b.Index, Type: exp.Type, Error: exp.Error}

		pageName := fmt.Sprintf("%s #%d", b.DocTitle, b.Index)
		c := &cellWriter{}
		c.vertex(c.id(), html.EscapeString(pageName), titleStyle, offsetX, 16, 700, 24)

		native := mode != DrawioImage && (exp.Kind == "graph" && len(exp.Nodes) > 0 ||
			exp.Kind == "sequence" && len(exp.Rects)+len(exp.Lines) > 0)
		switch {
		case native:
			if exp.Kind == "graph" {
				writeGraph(c, &exp)
			} else {
				writeSequence(c, &exp)
			}
			rep.Output = "native"
		case writeImage(c, png):
			rep.Output = "image"
			if mode != DrawioImage {
				c.vertex(c.id(), html.EscapeString(fmt.Sprintf("Tipo «%s»: exportado como imagen (sin conversión a formas).", exp.Type)),
					noteStyle, offsetX, 40, 700, 18)
			}
		default:
			writeCode(c, b.Code)
			rep.Output = "code"
		}
		report = append(report, rep)

		sb.WriteString(fmt.Sprintf(`  <diagram name="%s" id="mermaid-%d">`+"\n", xmlAttr(pageName), i+1))
		sb.WriteString(`    <mxGraphModel grid="1" gridSize="10" guides="1" tooltips="1" connect="1" arrows="1" fold="1" page="0">` + "\n")
		sb.WriteString("      <root>\n")
		sb.WriteString(`        <mxCell id="0"/>` + "\n")
		sb.WriteString(`        <mxCell id="1" parent="0"/>` + "\n")
		sb.WriteString(c.sb.String())
		sb.WriteString("      </root>\n    </mxGraphModel>\n  </diagram>\n")
	}
	sb.WriteString("</mxfile>\n")
	return []byte(sb.String()), report
}
