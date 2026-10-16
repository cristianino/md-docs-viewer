package mdocs

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cristianino/mdocs/internal/server"
)

func NewServeCmd() *cobra.Command {
	o := server.Options{}
	var exports string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Inicia el servidor web de documentación",
		Long: `Sirve el visor. Cada opción se puede dar por flag o por variable de entorno
(el flag gana):

  --host       MDOCS_HOST       interfaz ("" = todas; 127.0.0.1 detrás de un proxy)
  --port, -p   PORT             puerto (default 3000)
  --base-path  MDOCS_BASE_PATH  prefijo de URLs, p. ej. /docs (proxy, API Gateway)
  --exports    MDOCS_EXPORTS    live | static | off
  --dist       MDOCS_DIST       carpeta de archivos generados para --exports static

--exports:
  live    /pdf y /drawio se generan con Chrome al pedirlos (uno a la vez, con
          caché hasta que cambie un archivo del proyecto).
  static  sirve el .pdf y el .drawio más recientes de --dist (generados con
          mdocs build). No necesita Chrome en el servidor.
  off     sin botones de descarga; /pdf y /drawio responden 404.

/healthz responde "ok" (también fuera de --base-path) para balanceadores y
Lambda Web Adapter. SIGINT/SIGTERM detienen el servidor limpiamente.`,
		Example: `  mdocs serve
  mdocs serve --port 8080
  mdocs serve --host 127.0.0.1 --base-path /docs --exports static
  MDOCS_EXPORTS=off PORT=8080 mdocs serve`,
		RunE: func(cmd *cobra.Command, args []string) error {
			f := cmd.Flags()
			env := func(flag, name string, dst *string) {
				if !f.Changed(flag) {
					if v, ok := os.LookupEnv(name); ok {
						*dst = v
					}
				}
			}
			env("host", "MDOCS_HOST", &o.Host)
			env("base-path", "MDOCS_BASE_PATH", &o.BasePath)
			env("exports", "MDOCS_EXPORTS", &exports)
			env("dist", "MDOCS_DIST", &o.Dist)
			if !f.Changed("port") {
				if v := os.Getenv("PORT"); v != "" {
					p, err := strconv.Atoi(v)
					if err != nil {
						return fmt.Errorf("PORT inválido: %w", err)
					}
					o.Port = p
				}
			}
			o.Exports = server.ExportsMode(exports)
			return server.Start(o)
		},
	}

	f := cmd.Flags()
	f.StringVar(&o.Host, "host", "", `Interfaz de escucha ("" = todas)`)
	f.IntVarP(&o.Port, "port", "p", 3000, "Puerto en el que escucha el servidor")
	f.StringVar(&o.BasePath, "base-path", "", "Prefijo de URLs (p. ej. /docs)")
	f.StringVar(&exports, "exports", string(server.ExportsLive), "PDF y draw.io: live | static | off")
	f.StringVar(&o.Dist, "dist", "dist", "Carpeta de archivos generados (--exports static)")
	return cmd
}
