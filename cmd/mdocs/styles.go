package mdocs

import (
	"github.com/spf13/cobra"

	"github.com/cristianino/mdocs/internal/brand"
)

func NewStylesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "generate-styles",
		Short: "Extrae colores del logo SVG y actualiza brand.config.json",
		Long: `Lee logo.svg, extrae los colores dominantes mediante análisis de
luminancia y saturación, y actualiza brand.config.json con los
colores primario y de acento detectados.`,
		Example: `  mdocs generate-styles`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return brand.ExtractAndSave("logo.svg", "brand.config.json")
		},
	}
}
