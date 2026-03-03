package brand

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"strings"
)

type color struct {
	hex        string
	r, g, b    float64
	luminance  float64
	saturation float64
}

var (
	reFillAttr    = regexp.MustCompile(`\bfill="(#[0-9a-fA-F]{3,6})"`)
	reStrokeAttr  = regexp.MustCompile(`\bstroke="(#[0-9a-fA-F]{3,6})"`)
	reFillCSS     = regexp.MustCompile(`\bfill:\s*(#[0-9a-fA-F]{3,6})`)
	reStrokeCSS   = regexp.MustCompile(`\bstroke:\s*(#[0-9a-fA-F]{3,6})`)
	reStopAttr    = regexp.MustCompile(`\bstop-color="(#[0-9a-fA-F]{3,6})"`)
	reStopCSS     = regexp.MustCompile(`\bstop-color:\s*(#[0-9a-fA-F]{3,6})`)
)

func hexToRGB(hex string) (r, g, b float64, ok bool) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	if len(hex) != 6 {
		return 0, 0, 0, false
	}
	var ri, gi, bi int64
	fmt.Sscanf(hex[0:2], "%x", &ri)
	fmt.Sscanf(hex[2:4], "%x", &gi)
	fmt.Sscanf(hex[4:6], "%x", &bi)
	return float64(ri), float64(gi), float64(bi), true
}

func getLuminance(r, g, b float64) float64 {
	return 0.299*r + 0.587*g + 0.114*b
}

func getSaturation(r, g, b float64) float64 {
	rn, gn, bn := r/255, g/255, b/255
	mx := math.Max(rn, math.Max(gn, bn))
	mn := math.Min(rn, math.Min(gn, bn))
	if mx == mn {
		return 0
	}
	l := (mx + mn) / 2
	d := mx - mn
	if l > 0.5 {
		return d / (2 - mx - mn)
	}
	return d / (mx + mn)
}

func newColor(hex string) (color, bool) {
	r, g, b, ok := hexToRGB(hex)
	if !ok {
		return color{}, false
	}
	lum := getLuminance(r, g, b)
	// Filtrar blanco puro y negro puro
	if lum > 245 || lum < 8 {
		return color{}, false
	}
	return color{
		hex:        strings.ToLower(hex),
		r:          r,
		g:          g,
		b:          b,
		luminance:  lum,
		saturation: getSaturation(r, g, b),
	}, true
}

func extractColors(svg string) (main []color, gradient []color) {
	seen := map[string]bool{}

	addMain := func(hex string) {
		h := strings.ToLower(hex)
		if seen[h] {
			return
		}
		if c, ok := newColor(h); ok {
			seen[h] = true
			main = append(main, c)
		}
	}
	addGradient := func(hex string) {
		h := strings.ToLower(hex)
		if seen[h] {
			return
		}
		if c, ok := newColor(h); ok {
			seen[h] = true
			gradient = append(gradient, c)
		}
	}

	for _, re := range []*regexp.Regexp{reFillAttr, reStrokeAttr, reFillCSS, reStrokeCSS} {
		for _, m := range re.FindAllStringSubmatch(svg, -1) {
			addMain(m[1])
		}
	}
	for _, re := range []*regexp.Regexp{reStopAttr, reStopCSS} {
		for _, m := range re.FindAllStringSubmatch(svg, -1) {
			addGradient(m[1])
		}
	}
	return
}

func pickColors(main, gradient []color) (primary, accent string) {
	all := append(main, gradient...)
	if len(all) == 0 {
		return "#1a202c", "#3182ce"
	}

	// Primary: color más oscuro de los colores principales
	candidates := main
	if len(candidates) == 0 {
		candidates = all
	}
	primary = candidates[0].hex
	for _, c := range candidates[1:] {
		if c.luminance < candidates[0].luminance {
			primary = c.hex
			candidates[0] = c
		}
	}

	// Accent: color más saturado (excluido el primario)
	accent = primary
	bestSat := -1.0
	for _, c := range all {
		if c.hex == primary {
			continue
		}
		if c.saturation > bestSat {
			bestSat = c.saturation
			accent = c.hex
		}
	}
	return
}

// ExtractAndSave lee logo.svg, extrae colores y actualiza brand.config.json.
func ExtractAndSave(logoPath, configPath string) error {
	data, err := os.ReadFile(logoPath)
	if err != nil {
		return fmt.Errorf("no se encontró %s: %w", logoPath, err)
	}

	main, gradient := extractColors(string(data))

	fmt.Printf("\nColores principales (%d):\n", len(main))
	for _, c := range main {
		fmt.Printf("  %s  lum=%.0f  sat=%.2f\n", c.hex, c.luminance, c.saturation)
	}
	fmt.Printf("Colores de gradiente (%d):\n", len(gradient))
	for _, c := range gradient {
		fmt.Printf("  %s  lum=%.0f  sat=%.2f\n", c.hex, c.luminance, c.saturation)
	}

	primary, accent := pickColors(main, gradient)
	fmt.Printf("\nPrimario: %s\n", primary)
	fmt.Printf("Acento:   %s\n", accent)

	cfg, err := Load(configPath)
	if err != nil {
		return err
	}
	cfg.PrimaryColor = primary
	cfg.AccentColor = accent

	if err := Save(configPath, cfg); err != nil {
		return fmt.Errorf("guardando config: %w", err)
	}
	fmt.Printf("\n✓ brand.config.json actualizado\n")
	return nil
}
