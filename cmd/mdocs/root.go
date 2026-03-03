package mdocs

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mdocs",
	Short: "Visor de documentación Markdown con branding corporativo",
	Long: `mdocs sirve documentación Markdown como una web con branding dinámico.

Personaliza logo.svg, brand.config.json y docs.config.json
antes de iniciar el servidor.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func Execute() {
	rootCmd.AddCommand(NewServeCmd(), NewStylesCmd())
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
