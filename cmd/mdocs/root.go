package mdocs

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/cristianino/mdocs/internal/server"
)

// Version se puede fijar al compilar: go build -ldflags "-X github.com/cristianino/mdocs/cmd/mdocs.Version=1.2.0"
var Version = "dev"

var (
	dirFlag    string
	chromeFlag string
)

var rootCmd = &cobra.Command{
	Use:           "mdocs",
	Short:         "Documentación Markdown con branding, PDF navegable y exportación a draw.io",
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       Version,
	Long: `mdocs convierte archivos .md en un sitio de documentación con la identidad
de tu empresa, un PDF con marcadores e índice enlazado, y un .drawio con los
diagramas Mermaid como formas editables.

Flujo típico:
  mdocs init                 crea la configuración
  mdocs new "Guía"           crea docs/NN-guia.md y lo registra
  mdocs delete guia          lo quita del índice y borra el .md
  mdocs serve                visor web en http://localhost:3000
  mdocs build all            genera dist/*.pdf y dist/*.drawio
  mdocs check                valida enlaces, archivos y diagramas

Para agentes de IA: mdocs context --json, mdocs check --json, mdocs agents.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if dirFlag != "" {
			if err := os.Chdir(dirFlag); err != nil {
				return fmt.Errorf("--dir: %w", err)
			}
		}
		if chromeFlag != "" {
			server.ChromePath = chromeFlag
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func Execute() {
	rootCmd.PersistentFlags().StringVarP(&dirFlag, "dir", "C", "", "Carpeta del proyecto (por defecto la actual)")
	rootCmd.PersistentFlags().StringVar(&chromeFlag, "chrome", "", "Ruta a Chrome/Chromium (o variable MDOCS_CHROME)")
	rootCmd.AddCommand(
		NewInitCmd(), NewServeCmd(), NewBuildCmd(), NewExportCmd(),
		NewNewCmd(), NewAddCmd(), NewIndexCmd(), NewDeleteCmd(), NewRemoveCmd(), NewListCmd(),
		NewCheckCmd(), NewContextCmd(), NewAgentsCmd(),
		NewStylesCmd(), NewHealthcheckCmd(),
	)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}
}
