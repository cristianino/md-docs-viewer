package mdocs

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

func NewHealthcheckCmd() *cobra.Command {
	var url string
	cmd := &cobra.Command{
		Use:   "healthcheck",
		Short: "Consulta /healthz (para HEALTHCHECK de Docker en imágenes sin curl)",
		Long: `Hace GET a /healthz y sale con 0 si responde 200, o 1 si no.
Por defecto usa http://127.0.0.1:$PORT/healthz (PORT=3000 si no existe).`,
		Example: `  mdocs healthcheck
  mdocs healthcheck --url http://127.0.0.1:8080/healthz`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if url == "" {
				port := os.Getenv("PORT")
				if port == "" {
					port = "3000"
				}
				url = "http://127.0.0.1:" + port + "/healthz"
			}
			client := http.Client{Timeout: 3 * time.Second}
			resp, err := client.Get(url)
			if err != nil || resp.StatusCode != http.StatusOK {
				fmt.Fprintln(os.Stderr, "unhealthy:", url, err)
				os.Exit(1)
			}
			fmt.Println("ok")
			return nil
		},
	}
	cmd.Flags().StringVar(&url, "url", "", "URL a consultar (default http://127.0.0.1:$PORT/healthz)")
	return cmd
}
