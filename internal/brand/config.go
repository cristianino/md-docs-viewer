package brand

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	PrimaryColor string `json:"primaryColor"`
	AccentColor  string `json:"accentColor"`
	CompanyName  string `json:"companyName"`
	DocsTitle    string `json:"docsTitle"`
	DocsYear     string `json:"docsYear"`
	PoweredBy    bool   `json:"poweredBy"`
}

var defaults = Config{
	PrimaryColor: "#1a202c",
	AccentColor:  "#3182ce",
	CompanyName:  "Mi Empresa",
	DocsTitle:    "Documentación",
	DocsYear:     "2026",
	PoweredBy:    false,
}

func Load(path string) (*Config, error) {
	cfg := defaults
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
	return os.WriteFile(path, data, 0644)
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
