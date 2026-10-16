package mdocs

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/cristianino/mdocs/internal/server"
)

func NewExportCmd() *cobra.Command {
	var o server.SiteOptions
	var noPDF, noDrawio, asJSON bool
	var mode string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Genera un sitio estático listo para publicar (S3, Pages, Netlify, nginx)",
		Long: `Escribe el visor completo como HTML estático, con el PDF y el .drawio ya
generados en descargas/. El resultado no necesita mdocs, Go ni Chrome en el
servidor: sirve para S3 + CloudFront, GitHub Pages, Netlify, Cloudflare Pages
o un nginx en un VPS.

Estructura:
  index.html               redirige al primer documento
  doc/{id}/index.html      cada documento (URL /doc/{id}/)
  indice/index.html        índice general
  descargas/*.pdf|*.drawio archivos generados
  assets/, files/, logo.svg, 404.html

--base-path debe ser la ruta donde se publicará: "" para la raíz de un
dominio, "/md-docs-viewer" para GitHub Pages de proyecto, etc. También se
lee de MDOCS_BASE_PATH. Chrome solo se necesita aquí, al exportar.`,
		Example: `  mdocs export
  mdocs export -o public --base-path /md-docs-viewer
  mdocs export --no-pdf --no-drawio      # sin Chrome
  mdocs export --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("base-path") {
				o.BasePath = os.Getenv("MDOCS_BASE_PATH")
			}
			if mode != string(server.DrawioAuto) && mode != string(server.DrawioImage) {
				return fmt.Errorf("--mode inválido %q (usa auto o image)", mode)
			}
			o.Mode = server.DrawioMode(mode)
			o.PDF, o.Drawio = !noPDF, !noDrawio
			o.Log = func(format string, args ...any) { logf(asJSON, format, args...) }
			rep, err := server.ExportSite(o)
			if err != nil {
				return err
			}
			if asJSON {
				b, _ := json.MarshalIndent(rep, "", "  ")
				fmt.Println(string(b))
				return nil
			}
			fmt.Printf("  ✓ %s/ · %d páginas · %d archivos · base %q\n", rep.Out, rep.Pages, rep.Files, rep.BasePath)
			if rep.PDF != "" {
				fmt.Printf("    %s\n", rep.PDF)
			}
			if rep.Drawio != "" {
				fmt.Printf("    %s\n", rep.Drawio)
			}
			for _, w := range rep.Warnings {
				fmt.Printf("  aviso: %s\n", w)
			}
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVarP(&o.Out, "out", "o", "site", "Carpeta de salida")
	f.StringVar(&o.BasePath, "base-path", "", "Ruta donde se publicará (p. ej. /docs)")
	f.BoolVar(&noPDF, "no-pdf", false, "No generar el PDF")
	f.BoolVar(&noDrawio, "no-drawio", false, "No generar el .drawio")
	f.StringVar(&mode, "mode", string(server.DrawioAuto), "Exportación draw.io: auto | image")
	f.BoolVar(&asJSON, "json", false, "Salida JSON")
	return cmd
}
