package mdocs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cristianino/mdocs/internal/server"
)

type buildResult struct {
	Target   string                 `json:"target"`
	File     string                 `json:"file,omitempty"`
	Bytes    int                    `json:"bytes,omitempty"`
	Diagrams []server.DiagramReport `json:"diagrams,omitempty"`
	Skipped  string                 `json:"skipped,omitempty"`
}

func NewBuildCmd() *cobra.Command {
	var out, mode string
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "build [pdf|drawio|all]",
		Short: "Genera el PDF y/o el .drawio sin abrir el navegador",
		Long: `Arranca un servidor temporal en 127.0.0.1, genera los archivos con Chrome
headless y lo detiene. No necesita "mdocs serve".

  pdf     PDF A4 con portada, índice enlazado con páginas, marcadores y pie.
  drawio  Un .drawio con una página por diagrama Mermaid.
  all     Ambos (por defecto).

--out acepta una carpeta (default "dist") o, con un solo objetivo, la ruta
del archivo. Los nombres por defecto son {empresa}-{titulo}-{fecha}.pdf y
{empresa}-{titulo}-diagramas-{fecha}.drawio.

--mode auto convierte flowchart, sequence, state y class a formas nativas
y deja el resto como imagen; --mode image exporta todo como imagen.`,
		Example: `  mdocs build
  mdocs build pdf -o manual.pdf
  mdocs build drawio --mode image
  mdocs build all --json`,
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: []string{"pdf", "drawio", "all"},
		RunE: func(cmd *cobra.Command, args []string) error {
			target := "all"
			if len(args) == 1 {
				target = args[0]
			}
			if target != "pdf" && target != "drawio" && target != "all" {
				return fmt.Errorf("objetivo inválido %q (usa pdf, drawio o all)", target)
			}
			if mode != string(server.DrawioAuto) && mode != string(server.DrawioImage) {
				return fmt.Errorf("--mode inválido %q (usa auto o image)", mode)
			}
			cfg, docList, err := server.LoadState()
			if err != nil {
				return err
			}
			base, stop, err := server.Ephemeral()
			if err != nil {
				return err
			}
			defer stop()

			pathFor := func(def string) string {
				if target != "all" && (strings.HasSuffix(out, ".pdf") || strings.HasSuffix(out, ".drawio")) {
					return out
				}
				return filepath.Join(out, def)
			}
			write := func(p string, data []byte) error {
				if dir := filepath.Dir(p); dir != "." {
					if err := os.MkdirAll(dir, 0755); err != nil {
						return err
					}
				}
				return os.WriteFile(p, data, 0644)
			}

			var results []buildResult
			if target == "pdf" || target == "all" {
				logf(asJSON, "Generando PDF…\n")
				data, err := server.GeneratePDF(base, cfg)
				if err != nil {
					return err
				}
				p := pathFor(server.PDFFilename(cfg))
				if err := write(p, data); err != nil {
					return err
				}
				results = append(results, buildResult{Target: "pdf", File: p, Bytes: len(data)})
				logf(asJSON, "  ✓ %s (%d KB)\n", p, len(data)/1024)
			}
			if target == "drawio" || target == "all" {
				logf(asJSON, "Generando .drawio…\n")
				data, report, err := server.GenerateDrawio(base, docList, server.DrawioMode(mode))
				if err != nil && target == "all" && strings.Contains(err.Error(), "no se encontraron") {
					results = append(results, buildResult{Target: "drawio", Skipped: err.Error()})
					logf(asJSON, "  – omitido: %v\n", err)
				} else if err != nil {
					return err
				} else {
					p := pathFor(server.DrawioFilename(cfg))
					if err := write(p, data); err != nil {
						return err
					}
					results = append(results, buildResult{Target: "drawio", File: p, Bytes: len(data), Diagrams: report})
					logf(asJSON, "  ✓ %s\n", p)
					for _, r := range report {
						logf(asJSON, "    %s #%d %-14s → %s\n", r.Doc, r.Index, r.Type, r.Output)
					}
				}
			}
			if asJSON {
				b, _ := json.MarshalIndent(results, "", "  ")
				fmt.Println(string(b))
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&out, "out", "o", "dist", "Carpeta o archivo de salida")
	cmd.Flags().StringVar(&mode, "mode", string(server.DrawioAuto), "Exportación draw.io: auto | image")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Salida JSON (para scripts e IAs)")
	return cmd
}

// logf escribe progreso en stderr (stdout queda limpio para --json).
func logf(quiet bool, format string, args ...any) {
	if quiet {
		return
	}
	fmt.Fprintf(os.Stderr, format, args...)
}
