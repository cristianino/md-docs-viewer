package mdocs

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cristianino/mdocs/internal/docs"
	"github.com/cristianino/mdocs/internal/project"
	"github.com/cristianino/mdocs/internal/server"
)

// ---------------------------------------------------------------- search

func NewSearchCmd() *cobra.Command {
	var limit int
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "search <texto…>",
		Short: "Busca en la documentación: títulos, luego subtítulos y luego contenido",
		Long: `Busca igual que el buscador del visor. Los resultados salen en jerarquía:

  1 title    el título o subtítulo del documento coincide
  2 heading  un título interno (##, ###…) coincide
  3 content  el texto de una sección coincide

No distingue mayúsculas ni tildes; con varias palabras, todas deben aparecer.
Cada resultado trae el id del documento y el ancla (#id) para enlazarlo.`,
		Example: `  mdocs search instalación
  mdocs search "variables de entorno" --json
  mdocs search postman --limit 5`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			list, err := docs.Load(project.DocsFile)
			if err != nil {
				return err
			}
			q := strings.Join(args, " ")
			hits := docs.Search(server.BuildSearchIndex(list).Docs, q, limit)
			if asJSON {
				printJSON(map[string]any{"query": q, "count": len(hits), "hits": hits})
				return nil
			}
			if len(hits) == 0 {
				fmt.Printf("Sin resultados para %q.\n", q)
				return nil
			}
			labels := map[int]string{docs.TierTitle: "Documentos", docs.TierHeading: "Secciones", docs.TierContent: "Contenido"}
			tier := 0
			for _, h := range hits {
				if h.Tier != tier {
					tier = h.Tier
					fmt.Printf("\n%s\n", labels[tier])
				}
				link := "/doc/" + h.DocID
				if h.Anchor != "" && h.Tier != docs.TierTitle {
					link += "#" + h.Anchor
				}
				title := h.DocTitle
				if h.Heading != "" && h.Tier != docs.TierTitle {
					title += " › " + h.Heading
				}
				fmt.Printf("  %-48s %s\n", title, link)
				if h.Snippet != "" && h.Tier != docs.TierTitle {
					fmt.Printf("      %s\n", h.Snippet)
				}
			}
			fmt.Println()
			return nil
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "Máximo de resultados (0 = todos)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Salida JSON")
	return cmd
}

// ---------------------------------------------------------------- attach

func NewAttachCmd() *cobra.Command {
	var docID, title, dir string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "attach <archivo>",
		Short: "Agrega un archivo descargable (p. ej. una colección de Postman) y da su enlace",
		Long: `Copia el archivo a la carpeta de descargas del proyecto (por defecto
docs/files/) y muestra el enlace Markdown para ponerlo en un documento.
Si el archivo ya está dentro del proyecto no se copia.

Con --doc el enlace se agrega al final de ese documento, en una sección
"## Descargas" (se crea si no existe).

En el visor el enlace se muestra como botón de descarga; en el PDF el archivo
va adjunto (panel de adjuntos del lector de PDF); en mdocs export se copia al
sitio estático.`,
		Example: `  mdocs attach ~/Descargas/api.postman_collection.json --doc api --title "Colección Postman"
  mdocs attach docs/files/esquema.sql
  mdocs attach openapi.yaml --dir docs/descargas --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			src := args[0]
			st, err := os.Stat(src)
			if err != nil {
				return err
			}
			if st.IsDir() {
				return fmt.Errorf("%s es una carpeta; adjunta archivos (o un .zip)", src)
			}
			rel := filepath.ToSlash(filepath.Clean(src))
			copied := false
			if filepath.IsAbs(src) || !docs.InsideProject(rel) {
				rel = filepath.ToSlash(filepath.Join(dir, filepath.Base(src)))
				if _, err := os.Stat(filepath.FromSlash(rel)); err == nil {
					if !sameContent(src, filepath.FromSlash(rel)) {
						return fmt.Errorf("ya existe %s con otro contenido (renómbralo o usa --dir)", rel)
					}
				} else {
					if err := copyFile(src, filepath.FromSlash(rel)); err != nil {
						return err
					}
					copied = true
				}
			}
			if docs.HiddenPath(rel) {
				return fmt.Errorf("%s: las rutas con carpetas o archivos ocultos (.algo) no se publican", rel)
			}
			if title == "" {
				title = path.Base(rel)
			}

			out := map[string]any{"file": rel, "copied": copied, "size": st.Size()}
			linkFrom := "."
			var target *docs.Doc
			if docID != "" {
				list, err := docs.Load(project.DocsFile)
				if err != nil {
					return err
				}
				i := docs.FindID(list, docID)
				if i < 0 {
					return fmt.Errorf("no hay un documento con id %q", docID)
				}
				target = list[i]
				linkFrom = path.Dir(filepath.ToSlash(target.File))
			}
			link, err := filepath.Rel(filepath.FromSlash(linkFrom), filepath.FromSlash(rel))
			if err != nil {
				return err
			}
			link = filepath.ToSlash(link)
			md := fmt.Sprintf("[%s](%s)", title, mdLinkDest(link))
			out["markdown"] = md
			if target != nil {
				added, err := appendDownload(target.File, md)
				if err != nil {
					return err
				}
				out["doc"] = target.ID
				out["linked"] = added
			}

			if asJSON {
				printJSON(out)
				return nil
			}
			if copied {
				fmt.Printf("  copiado  %s → %s\n", src, rel)
			}
			if target != nil && out["linked"] == true {
				fmt.Printf("  enlazado en %s (sección Descargas)\n", target.File)
			} else if target != nil {
				fmt.Printf("  %s ya enlazaba ese archivo\n", target.File)
			} else {
				fmt.Printf("  enlace (relativo a la raíz; ajústalo a la carpeta del .md):\n  %s\n", md)
			}
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&docID, "doc", "", "Id del documento donde agregar el enlace (sección Descargas)")
	f.StringVar(&title, "title", "", "Texto del enlace (por defecto el nombre del archivo)")
	f.StringVar(&dir, "dir", "docs/files", "Carpeta donde se copian los archivos de fuera del proyecto")
	f.BoolVar(&asJSON, "json", false, "Salida JSON")
	return cmd
}

// mdLinkDest protege rutas con espacios o paréntesis: [x](<a b.json>).
func mdLinkDest(p string) string {
	if strings.ContainsAny(p, " ()<>") {
		return "<" + p + ">"
	}
	return p
}

// appendDownload agrega "- link" al final del documento, bajo "## Descargas".
// Si el documento ya enlaza ese archivo no cambia nada y devuelve false.
func appendDownload(file, link string) (bool, error) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return false, err
	}
	text := strings.TrimRight(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	if i := strings.LastIndex(link, "]("); i >= 0 && strings.Contains(text, link[i:]) {
		return false, nil
	}
	lines := strings.Split(text, "\n")
	section := -1
	for i, l := range lines {
		if strings.TrimSpace(l) == "## Descargas" {
			section = i
		}
	}
	item := "- " + link
	if section < 0 {
		text += "\n\n## Descargas\n\n" + item
	} else {
		// después del último elemento de la sección (antes del siguiente título)
		end := len(lines)
		for i := section + 1; i < len(lines); i++ {
			if strings.HasPrefix(lines[i], "#") {
				end = i
				break
			}
		}
		insert := end
		for insert > section+1 && strings.TrimSpace(lines[insert-1]) == "" {
			insert--
		}
		lines = append(lines[:insert], append([]string{item}, lines[insert:]...)...)
		text = strings.Join(lines, "\n")
	}
	return true, os.WriteFile(file, []byte(text+"\n"), 0644)
}

func sameContent(a, b string) bool {
	x, err1 := os.ReadFile(a)
	y, err2 := os.ReadFile(b)
	return err1 == nil && err2 == nil && string(x) == string(y)
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
