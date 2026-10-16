package mdocs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cristianino/mdocs/internal/docs"
	"github.com/cristianino/mdocs/internal/project"
)

var reNumPrefix = regexp.MustCompile(`^\d+[-_. ]+`)

func NewAddCmd() *cobra.Command {
	var id, title, subtitle string
	var position int

	cmd := &cobra.Command{
		Use:   "add <archivo.md>",
		Short: "Registra un documento en docs.config.json (y lo crea si no existe)",
		Long: `Agrega el archivo a docs.config.json. Si el archivo no existe lo crea con
un título "# {title}".

Valores por defecto:
  --id        nombre del archivo sin número inicial ni extensión (02-guia.md → guia)
  --title     primer H1 del archivo; si no hay, el id con mayúscula inicial
  --position  al final (1 = primero)`,
		Example: `  mdocs add docs/02-guia.md
  mdocs add docs/api.md --title "API" --subtitle "Endpoints REST" --position 2`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			file := filepath.ToSlash(args[0])
			list, err := loadDocsOrEmpty()
			if err != nil {
				return err
			}
			for _, d := range list {
				if filepath.ToSlash(d.File) == file {
					return fmt.Errorf("%s ya está registrado como %q", file, d.ID)
				}
			}
			if id == "" {
				base := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
				id = docs.Slug(reNumPrefix.ReplaceAllString(base, ""))
			}
			for _, d := range list {
				if d.ID == id {
					return fmt.Errorf("el id %q ya existe (usa --id)", id)
				}
			}
			raw, readErr := os.ReadFile(file)
			if title == "" {
				if readErr == nil {
					for _, line := range strings.Split(string(raw), "\n") {
						if strings.HasPrefix(line, "# ") {
							title = strings.TrimSpace(line[2:])
							break
						}
					}
				}
				if title == "" {
					title = strings.ToUpper(id[:1]) + strings.ReplaceAll(id[1:], "-", " ")
				}
			}
			if readErr != nil {
				if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
					return err
				}
				if err := os.WriteFile(file, []byte("# "+title+"\n\n"), 0644); err != nil {
					return err
				}
				fmt.Printf("  creado  %s\n", file)
			}
			d := &docs.Doc{ID: id, File: file, Title: title, Subtitle: subtitle}
			if position < 1 || position > len(list) {
				list = append(list, d)
			} else {
				list = append(list[:position-1], append([]*docs.Doc{d}, list[position-1:]...)...)
			}
			if err := docs.Save(project.DocsFile, list); err != nil {
				return err
			}
			fmt.Printf("  registrado %q → %s (/doc/%s)\n", title, file, id)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Identificador (URL /doc/{id})")
	cmd.Flags().StringVar(&title, "title", "", "Título en la barra lateral y el PDF")
	cmd.Flags().StringVar(&subtitle, "subtitle", "", "Subtítulo")
	cmd.Flags().IntVar(&position, "position", 0, "Posición en la lista (1 = primero; 0 = al final)")
	return cmd
}

func NewRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove <id>",
		Aliases: []string{"rm"},
		Short:   "Quita un documento de docs.config.json (no borra el archivo .md)",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			list, err := docs.Load(project.DocsFile)
			if err != nil {
				return err
			}
			out := list[:0]
			var removed *docs.Doc
			for _, d := range list {
				if d.ID == args[0] {
					removed = d
					continue
				}
				out = append(out, d)
			}
			if removed == nil {
				return fmt.Errorf("no hay un documento con id %q", args[0])
			}
			if err := docs.Save(project.DocsFile, out); err != nil {
				return err
			}
			fmt.Printf("  quitado %q (%s sigue en disco)\n", removed.ID, removed.File)
			return nil
		},
	}
}

func NewListCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Lista los documentos registrados",
		RunE: func(cmd *cobra.Command, args []string) error {
			list, err := docs.Load(project.DocsFile)
			if err != nil {
				return err
			}
			if asJSON {
				b, _ := json.MarshalIndent(list, "", "  ")
				fmt.Println(string(b))
				return nil
			}
			for i, d := range list {
				mark := " "
				if _, err := os.Stat(d.File); err != nil {
					mark = "!"
				}
				fmt.Printf("%s %2d  %-18s %-28s %s\n", mark, i+1, d.ID, d.Title, d.File)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Salida JSON")
	return cmd
}

func NewCheckCmd() *cobra.Command {
	var asJSON, strict bool
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Valida configuración, archivos, enlaces, imágenes y diagramas",
		Long: `Revisa el proyecto y reporta:
  error    config inválida, ids vacíos o duplicados, archivos que no existen
  warning  enlaces o anclas rotas, imágenes faltantes, .md sin registrar en docs/

Sale con código 1 si hay errores (o warnings con --strict); útil en CI.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := project.Load()
			if err != nil {
				return err
			}
			warnings := len(ctx.Issues) - ctx.Errors()
			if asJSON {
				out := map[string]any{
					"ok":       ctx.Errors() == 0 && (!strict || warnings == 0),
					"errors":   ctx.Errors(),
					"warnings": warnings,
					"issues":   ctx.Issues,
				}
				b, _ := json.MarshalIndent(out, "", "  ")
				fmt.Println(string(b))
			} else {
				for _, i := range ctx.Issues {
					fmt.Printf("%-7s %s: %s\n", i.Level, firstNonEmpty(i.File, i.Doc), i.Message)
				}
				diagrams, native := 0, 0
				for _, d := range ctx.Docs {
					for _, dg := range d.Diagrams {
						diagrams++
						if dg.Native {
							native++
						}
					}
				}
				fmt.Printf("\n%d documentos · %d diagramas (%d nativos en draw.io) · %d errores · %d warnings\n",
					len(ctx.Docs), diagrams, native, ctx.Errors(), warnings)
			}
			if ctx.Errors() > 0 || (strict && warnings > 0) {
				os.Exit(1)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Salida JSON")
	cmd.Flags().BoolVar(&strict, "strict", false, "Tratar warnings como errores")
	return cmd
}

func NewContextCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Resume el proyecto (config, documentos, títulos, diagramas) para IAs",
		Long: `Imprime todo lo que un agente necesita para trabajar en el proyecto sin
leer cada archivo: datos de brand.config.json, cada documento con su archivo,
títulos y anclas (#id), diagramas Mermaid y cómo se exportan, archivos sin
registrar y problemas detectados. Markdown por defecto, JSON con --json.`,
		Example: `  mdocs context
  mdocs context --json > contexto.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := project.Load()
			if err != nil {
				return err
			}
			if asJSON {
				os.Stdout.Write(ctx.JSON())
			} else {
				fmt.Print(ctx.Markdown())
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Salida JSON")
	return cmd
}

func loadDocsOrEmpty() ([]*docs.Doc, error) {
	if _, err := os.Stat(project.DocsFile); os.IsNotExist(err) {
		return []*docs.Doc{}, nil
	}
	return docs.Load(project.DocsFile)
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}
