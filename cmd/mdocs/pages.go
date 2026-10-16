package mdocs

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cristianino/mdocs/internal/docs"
	"github.com/cristianino/mdocs/internal/project"
)

func printJSON(v any) {
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}

// ---------------------------------------------------------------- new

func NewNewCmd() *cobra.Command {
	var id, subtitle, file, dir string
	var position int
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "new <título>",
		Short: "Crea una página: escribe el .md y la registra en docs.config.json",
		Long: `Crea el archivo Markdown con "# {título}" y lo agrega al índice.

Valores por defecto:
  --file      {dir}/NN-{slug}.md, con NN = mayor número de los .md de --dir + 1
              ("Guía rápida" → docs/04-guia-rapida.md)
  --dir       docs
  --id        slug del título sin tildes ("Guía rápida" → guia-rapida); si existe, se agrega -2, -3…
  --position  al final (1 = primero)

Falla si el archivo ya existe (para registrar un .md existente usa mdocs add).`,
		Example: `  mdocs new "Guía de instalación"
  mdocs new "API" --subtitle "Endpoints REST" --position 2
  mdocs new "Anexo" --file docs/anexos/a.md --id anexo-a --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			title := strings.TrimSpace(args[0])
			if title == "" {
				return fmt.Errorf("el título no puede estar vacío")
			}
			list, err := loadDocsOrEmpty()
			if err != nil {
				return err
			}
			slug := docs.FileSlug(title)
			if slug == "" {
				slug = "pagina"
			}
			if id == "" {
				id = docs.UniqueID(list, slug)
			} else if docs.FindID(list, id) >= 0 {
				return fmt.Errorf("el id %q ya existe", id)
			}
			if file == "" {
				file = docs.NextNumberedFile(dir, slug)
			}
			file = filepath.ToSlash(file)
			if !docs.InsideProject(file) {
				return fmt.Errorf("%s está fuera de la carpeta del proyecto", file)
			}
			if !strings.HasSuffix(strings.ToLower(file), ".md") {
				return fmt.Errorf("%s debe terminar en .md", file)
			}
			if _, err := os.Stat(file); err == nil {
				return fmt.Errorf("%s ya existe (para registrarlo usa: mdocs add %s)", file, file)
			}
			if docs.FindFile(list, file) >= 0 {
				return fmt.Errorf("%s ya está registrado", file)
			}

			content := "# " + title + "\n\n"
			if subtitle != "" {
				content += subtitle + "\n\n"
			}
			content += "## Contenido\n\nEscribe aquí el contenido.\n"
			if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(file, []byte(content), 0644); err != nil {
				return err
			}
			d := &docs.Doc{ID: id, File: file, Title: title, Subtitle: subtitle}
			list = docs.Insert(list, d, position)
			if err := docs.Save(project.DocsFile, list); err != nil {
				os.Remove(file) // no dejar un .md huérfano
				return err
			}
			if asJSON {
				printJSON(map[string]any{"action": "created", "doc": d, "position": docs.FindID(list, id) + 1})
				return nil
			}
			fmt.Printf("  creado     %s\n  registrado %q (/doc/%s, posición %d)\n", file, title, id, docs.FindID(list, id)+1)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&id, "id", "", "Identificador (URL /doc/{id})")
	f.StringVar(&subtitle, "subtitle", "", "Subtítulo (también va como primer párrafo)")
	f.StringVar(&file, "file", "", "Ruta del .md (default {dir}/NN-{slug}.md)")
	f.StringVar(&dir, "dir", "docs", "Carpeta donde se crea el .md si no se da --file")
	f.IntVar(&position, "position", 0, "Posición en la lista (1 = primero; 0 = al final)")
	f.BoolVar(&asJSON, "json", false, "Salida JSON")
	return cmd
}

// ---------------------------------------------------------------- delete

func NewDeleteCmd() *cobra.Command {
	var yes, keepFile, asJSON bool

	cmd := &cobra.Command{
		Use:     "delete <id>",
		Aliases: []string{"del"},
		Short:   "Elimina una página: la quita de docs.config.json y borra su .md",
		Long: `Quita el documento del índice y borra su archivo .md.

Antes de borrar muestra el archivo y los enlaces que otros documentos tienen
hacia él (quedarían rotos) y pide confirmación. Con --yes no pregunta (scripts
e IAs). Con --keep-file solo lo quita del índice, igual que mdocs remove.

Solo borra archivos dentro de la carpeta del proyecto. Las imágenes que use el
documento no se borran.`,
		Example: `  mdocs delete guia
  mdocs delete guia --yes --json
  mdocs delete guia --keep-file`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			list, err := docs.Load(project.DocsFile)
			if err != nil {
				return err
			}
			i := docs.FindID(list, args[0])
			if i < 0 {
				return fmt.Errorf("no hay un documento con id %q (mira mdocs list)", args[0])
			}
			d := list[i]
			refs := project.References(d, list)
			_, statErr := os.Stat(d.File)
			deleteFile := !keepFile && statErr == nil
			if deleteFile && !docs.InsideProject(d.File) {
				return fmt.Errorf("%s está fuera de la carpeta del proyecto; usa --keep-file y bórralo a mano", d.File)
			}

			if !yes {
				out := cmd.ErrOrStderr()
				fmt.Fprintf(out, "Documento %q (%s)\n", d.Title, d.ID)
				if deleteFile {
					fmt.Fprintf(out, "  se borrará  %s\n", d.File)
				} else if keepFile {
					fmt.Fprintf(out, "  se conserva %s (solo se quita del índice)\n", d.File)
				} else {
					fmt.Fprintf(out, "  %s no existe; solo se quita del índice\n", d.File)
				}
				for _, r := range refs {
					fmt.Fprintf(out, "  aviso: %s enlaza aquí (%s) y quedará roto\n", r.File, r.Link)
				}
				fmt.Fprint(out, "¿Continuar? [s/N]: ")
				line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
				switch strings.ToLower(strings.TrimSpace(line)) {
				case "s", "si", "sí", "y", "yes":
				default:
					return fmt.Errorf("cancelado (usa --yes para no preguntar)")
				}
			}

			list = append(list[:i], list[i+1:]...)
			if err := docs.Save(project.DocsFile, list); err != nil {
				return err
			}
			if deleteFile {
				if err := os.Remove(d.File); err != nil {
					return fmt.Errorf("quitado del índice, pero no se pudo borrar %s: %w", d.File, err)
				}
			}
			if asJSON {
				printJSON(map[string]any{"action": "deleted", "doc": d, "fileDeleted": deleteFile, "brokenLinks": refs})
				return nil
			}
			if deleteFile {
				fmt.Printf("  borrado  %s\n", d.File)
			}
			fmt.Printf("  quitado  %q del índice\n", d.ID)
			if len(refs) > 0 {
				fmt.Printf("  %d enlace(s) quedaron rotos; revísalos con mdocs check\n", len(refs))
			}
			return nil
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&yes, "yes", "y", false, "No pedir confirmación")
	f.BoolVar(&keepFile, "keep-file", false, "No borrar el .md (solo quitarlo del índice)")
	f.BoolVar(&asJSON, "json", false, "Salida JSON")
	return cmd
}

// ---------------------------------------------------------------- index

func NewIndexCmd() *cobra.Command {
	var dir string
	var prune, dryRun, asJSON bool

	cmd := &cobra.Command{
		Use:   "index",
		Short: "Indexa los .md sin registrar de docs/ (y con --prune quita los que ya no existen)",
		Long: `Busca archivos .md en --dir (recursivo, en orden alfabético) que no estén en
docs.config.json y los agrega al final. id y título se toman como en
mdocs add: id del nombre de archivo (sin número inicial) y título del primer
"# " del archivo.

--prune quita del índice las entradas cuyo archivo ya no existe.
--dry-run muestra los cambios sin escribir nada.`,
		Example: `  mdocs index
  mdocs index --prune --dry-run
  mdocs index --dir manuales --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			list, err := loadDocsOrEmpty()
			if err != nil {
				return err
			}
			var files []string
			filepath.WalkDir(dir, func(p string, e fs.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if e.IsDir() && p != dir && strings.HasPrefix(e.Name(), ".") {
					return filepath.SkipDir
				}
				if !e.IsDir() && strings.HasSuffix(strings.ToLower(p), ".md") && docs.FindFile(list, p) < 0 {
					files = append(files, filepath.ToSlash(p))
				}
				return nil
			})
			sort.Strings(files)

			var added, removed []*docs.Doc
			for _, f := range files {
				id := docs.UniqueID(list, docs.IDFromFile(f))
				title := docs.TitleFromFile(f)
				if title == "" {
					title = docs.TitleFromID(id)
				}
				d := &docs.Doc{ID: id, File: f, Title: title}
				list = append(list, d)
				added = append(added, d)
			}
			if prune {
				kept := list[:0]
				for _, d := range list {
					if _, err := os.Stat(d.File); err != nil {
						removed = append(removed, d)
						continue
					}
					kept = append(kept, d)
				}
				list = kept
			}
			if !dryRun && (len(added) > 0 || len(removed) > 0) {
				if err := docs.Save(project.DocsFile, list); err != nil {
					return err
				}
			}
			if asJSON {
				printJSON(map[string]any{"added": orEmpty(added), "removed": orEmpty(removed), "dryRun": dryRun})
				return nil
			}
			verb := ""
			if dryRun {
				verb = " (dry-run)"
			}
			for _, d := range added {
				fmt.Printf("  + %-18s %-28s %s%s\n", d.ID, d.Title, d.File, verb)
			}
			for _, d := range removed {
				fmt.Printf("  - %-18s %-28s %s (no existe)%s\n", d.ID, d.Title, d.File, verb)
			}
			if len(added) == 0 && len(removed) == 0 {
				fmt.Println("  índice al día: no hay cambios")
			}
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&dir, "dir", "docs", "Carpeta donde buscar .md")
	f.BoolVar(&prune, "prune", false, "Quitar entradas cuyo archivo no existe")
	f.BoolVar(&dryRun, "dry-run", false, "Mostrar cambios sin escribir")
	f.BoolVar(&asJSON, "json", false, "Salida JSON")
	return cmd
}

func orEmpty(v []*docs.Doc) []*docs.Doc {
	if v == nil {
		return []*docs.Doc{}
	}
	return v
}
