package brand

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	PrimaryColor  string `json:"primaryColor"`
	AccentColor   string `json:"accentColor"`
	CompanyName   string `json:"companyName"`
	DocsTitle     string `json:"docsTitle"`
	Description   string `json:"description"`
	DocsYear      string `json:"docsYear"`
	PoweredBy     bool   `json:"poweredBy"`
	PoweredByText string `json:"poweredByText,omitempty"`
	PoweredByURL  string `json:"poweredByUrl,omitempty"`
}

// Defaults devuelve la configuración por defecto. poweredBy es false.
func Defaults() Config {
	return Config{
		PrimaryColor: "#1a202c",
		AccentColor:  "#3182ce",
		CompanyName:  "Mi Empresa",
		DocsTitle:    "Documentación",
		Description:  "",
		DocsYear:     strconv.Itoa(time.Now().Year()),
		PoweredBy:    false,
	}
}

// PoweredByLabel devuelve el texto de atribución (por defecto "mdocs").
func (c *Config) PoweredByLabel() string {
	if strings.TrimSpace(c.PoweredByText) != "" {
		return c.PoweredByText
	}
	return "mdocs"
}

// PoweredByLink devuelve la URL de atribución.
func (c *Config) PoweredByLink() string {
	if strings.TrimSpace(c.PoweredByURL) != "" {
		return c.PoweredByURL
	}
	if c.PoweredByText == "" {
		return "https://github.com/cristianino/md-docs-viewer"
	}
	return ""
}

func Load(path string) (*Config, error) {
	cfg := Defaults()
	data, err := os.ReadFile(path)
	if err != nil {
		return &cfg, nil
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return &cfg, fmt.Errorf("brand config: %w", err)
	}
	return &cfg, nil
}

func Save(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

// HexToRGBA convierte un color hex a formato CSS rgba(r, g, b, alpha).
func HexToRGBA(hex string, alpha float64) string {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	if len(hex) != 6 {
		return fmt.Sprintf("rgba(0,0,0,%.2f)", alpha)
	}
	r, _ := strconv.ParseInt(hex[0:2], 16, 64)
	g, _ := strconv.ParseInt(hex[2:4], 16, 64)
	b, _ := strconv.ParseInt(hex[4:6], 16, 64)
	return fmt.Sprintf("rgba(%d,%d,%d,%.2f)", r, g, b, alpha)
}
