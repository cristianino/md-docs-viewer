package mdocs

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cristianino/mdocs/assets"
	"github.com/cristianino/mdocs/internal/brand"
	"github.com/cristianino/mdocs/internal/docs"
)

type initOptions struct {
	company       string
	title         string
	description   string
	year          string
	poweredBy     bool
	poweredByText string
	poweredByURL  string
	yes           bool
	force         bool
	noSample      bool
	noAgents      bool
}

func NewInitCmd() *cobra.Command {
	o := initOptions{}
	def := brand.Defaults()

	cmd := &cobra.Command{
		Use:   "init [carpeta]",
		Short: "Crea un proyecto de documentación nuevo",
		Long: `Crea brand.config.json, docs.config.json, logo.svg, AGENTS.md, .gitignore y un
documento de ejemplo. Los archivos que ya existen (salvo las dos configs con
--force) no se tocan.

Pregunta los datos fijos (empresa, título, descripción, año y "powered by")
salvo los que pases por flag. Con --yes no pregunta nada y usa flags o
valores por defecto, útil para scripts e IAs. "powered by" está desactivado
por defecto.`,
		Example: `  mdocs init
  mdocs init mi-portal
  mdocs init --yes --company "Acme" --title "Manual técnico" --description "API interna" --year 2026
  mdocs init --yes --powered-by --powered-by-text "Acme Labs" --powered-by-url https://acme.dev`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			changed := func(name string) bool { return cmd.Flags().Changed(name) }
			return runInit(dir, o, changed, os.Stdin, cmd.OutOrStdout())
		},
	}

	f := cmd.Flags()
	f.StringVar(&o.company, "company", def.CompanyName, "Nombre de la empresa")
	f.StringVar(&o.title, "title", def.DocsTitle, "Título de la documentación")
	f.StringVar(&o.description, "description", "", "Descripción corta (portada del PDF y metadatos)")
	f.StringVar(&o.year, "year", def.DocsYear, "Año mostrado en el footer")
	f.BoolVar(&o.poweredBy, "powered-by", false, `Muestra "Powered by" en el visor y el PDF`)
	f.StringVar(&o.poweredByText, "powered-by-text", "", `Texto de "Powered by" (default "mdocs")`)
	f.StringVar(&o.poweredByURL, "powered-by-url", "", `Enlace de "Powered by"`)
	f.BoolVarP(&o.yes, "yes", "y", false, "No preguntar; usar flags y valores por defecto")
	f.BoolVar(&o.force, "force", false, "Sobrescribir brand.config.json y docs.config.json si existen")
	f.BoolVar(&o.noSample, "no-sample", false, "No crear el documento de ejemplo")
	f.BoolVar(&o.noAgents, "no-agents", false, "No crear AGENTS.md (guía para agentes de IA)")
	return cmd
}

func runInit(dir string, o initOptions, changed func(string) bool, in io.Reader, out io.Writer) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	brandPath := filepath.Join(dir, "brand.config.json")
	docsPath := filepath.Join(dir, "docs.config.json")
	if !o.force {
		for _, p := range []string{brandPath, docsPath} {
			if _, err := os.Stat(p); err == nil {
				return fmt.Errorf("%s ya existe (usa --force para sobrescribir)", p)
			}
		}
	}

	if !o.yes {
		r := bufio.NewReader(in)
		ask := func(flag, label string, val *string) {
			if !changed(flag) {
				*val = prompt(r, out, label, *val)
			}
		}
		ask("company", "Empresa", &o.company)
		ask("title", "Título", &o.title)
		ask("description", "Descripción", &o.description)
		ask("year", "Año", &o.year)
		if !changed("powered-by") {
			o.poweredBy = promptBool(r, out, `¿Mostrar "Powered by"?`, false)
		}
		if o.poweredBy {
			ask("powered-by-text", `Texto de "Powered by"`, &o.poweredByText)
			ask("powered-by-url", `Enlace de "Powered by"`, &o.poweredByURL)
		}
	}

	cfg := brand.Defaults()
	cfg.CompanyName = o.company
	cfg.DocsTitle = o.title
	cfg.Description = o.description
	cfg.DocsYear = o.year
	cfg.PoweredBy = o.poweredBy
	if o.poweredBy {
		cfg.PoweredByText = o.poweredByText
		cfg.PoweredByURL = o.poweredByURL
	}

	logoPath := filepath.Join(dir, "logo.svg")
	if data, err := os.ReadFile(logoPath); err == nil {
		cfg.PrimaryColor, cfg.AccentColor = brand.ColorsFromSVG(data)
		fmt.Fprintf(out, "  colores tomados de logo.svg: %s / %s\n", cfg.PrimaryColor, cfg.AccentColor)
	} else {
		writeIfMissing(out, logoPath, assets.DefaultLogo)
		cfg.PrimaryColor, cfg.AccentColor = brand.ColorsFromSVG(assets.DefaultLogo)
	}

	if err := brand.Save(brandPath, &cfg); err != nil {
		return err
	}
	fmt.Fprintf(out, "  creado  %s\n", brandPath)

	list := []*docs.Doc{}
	if !o.noSample {
		list = append(list, &docs.Doc{
			ID: "introduccion", File: "docs/01-introduccion.md",
			Title: "Introducción", Subtitle: "Descripción general",
		})
		writeIfMissing(out, filepath.Join(dir, "docs", "01-introduccion.md"), []byte(sampleDoc(&cfg)))
	} else {
		os.MkdirAll(filepath.Join(dir, "docs"), 0755)
	}
	if err := docs.Save(docsPath, list); err != nil {
		return err
	}
	fmt.Fprintf(out, "  creado  %s\n", docsPath)
	if !o.noAgents {
		writeIfMissing(out, filepath.Join(dir, "AGENTS.md"), assets.AgentsGuide)
	}
	writeIfMissing(out, filepath.Join(dir, ".gitignore"), []byte("dist/\nsite/\n"))

	fmt.Fprintf(out, "\nListo. Siguiente paso:\n")
	if dir != "." {
		fmt.Fprintf(out, "  cd %s\n", dir)
	}
	fmt.Fprintf(out, "  mdocs serve\n")
	return nil
}

func writeIfMissing(out io.Writer, path string, data []byte) {
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(out, "  existe  %s (sin cambios)\n", path)
		return
	}
	os.MkdirAll(filepath.Dir(path), 0755)
	if err := os.WriteFile(path, data, 0644); err != nil {
		fmt.Fprintf(out, "  error   %s: %v\n", path, err)
		return
	}
	fmt.Fprintf(out, "  creado  %s\n", path)
}

func prompt(r *bufio.Reader, out io.Writer, label, def string) string {
	if def != "" {
		fmt.Fprintf(out, "%s [%s]: ", label, def)
	} else {
		fmt.Fprintf(out, "%s: ", label)
	}
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

func promptBool(r *bufio.Reader, out io.Writer, label string, def bool) bool {
	hint := "s/N"
	if def {
		hint = "S/n"
	}
	fmt.Fprintf(out, "%s [%s]: ", label, hint)
	line, _ := r.ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "s", "si", "sí", "y", "yes":
		return true
	case "n", "no":
		return false
	}
	return def
}

func sampleDoc(cfg *brand.Config) string {
	desc := cfg.Description
	if desc == "" {
		desc = "Escribe aquí la descripción de tu documentación."
	}
	return "# Introducción\n\n" + desc + "\n\n" +
		"## Primeros pasos\n\n" +
		"- Edita este archivo en `docs/01-introduccion.md`.\n" +
		"- Registra más documentos con `mdocs add docs/02-guia.md --title \"Guía\"` o editando `docs.config.json`.\n" +
		"- Reemplaza `logo.svg` y ejecuta `mdocs generate-styles` para tomar sus colores.\n\n" +
		"## Diagrama de ejemplo\n\n" +
		"```mermaid\nflowchart LR\n    A[Markdown] --> B(mdocs)\n    B --> C{Salida}\n    C -->|web| D[Visor]\n    C -->|pdf| E[PDF]\n    C -->|drawio| F[draw.io]\n```\n"
}
