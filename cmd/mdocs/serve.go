package mdocs

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cristianino/mdocs/internal/server"
)

func NewServeCmd() *cobra.Command {
	var portFlag int

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Inicia el servidor web de documentación",
		Example: `  mdocs serve
  mdocs serve --port 8080`,
		RunE: func(cmd *cobra.Command, args []string) error {
			p := portFlag
			// PORT env var tiene prioridad sobre el flag solo si el flag no fue cambiado
			if !cmd.Flags().Changed("port") {
				if envPort := os.Getenv("PORT"); envPort != "" {
					parsed, err := strconv.Atoi(envPort)
					if err != nil {
						return fmt.Errorf("PORT env inválido: %w", err)
					}
					p = parsed
				}
			}
			return server.Start(p)
		},
	}

	cmd.Flags().IntVarP(&portFlag, "port", "p", 3000, "Puerto en el que escucha el servidor")
	return cmd
}
