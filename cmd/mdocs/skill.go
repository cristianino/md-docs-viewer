package mdocs

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/cristianino/mdocs/skills"
)

func NewSkillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Instala o empaqueta el skill de mdocs para Claude y otros agentes",
		Long: `El skill (skills/mdocs/SKILL.md, formato Agent Skills) enseña a un agente a
usar mdocs: encontrar el binario, entender el proyecto con context/check,
crear y borrar páginas, las reglas del Markdown y cómo generar PDF, draw.io
o el sitio estático.

  mdocs skill install            .claude/skills/mdocs/ del proyecto actual (Claude Code)
  mdocs skill install --global   ~/.claude/skills/mdocs/ (todos tus proyectos)
  mdocs skill zip                mdocs-skill.zip para subir en claude.ai
  mdocs skill print              muestra el SKILL.md`,
	}
	cmd.AddCommand(newSkillInstallCmd(), newSkillZipCmd(), newSkillPrintCmd())
	return cmd
}

func newSkillInstallCmd() *cobra.Command {
	var global, force, withBinary bool
	var dir string
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Copia el skill a .claude/skills/mdocs (o a ~/.claude/skills/mdocs con --global)",
		Long: `Escribe SKILL.md en la carpeta de skills de Claude Code:
  sin flags   ./.claude/skills/mdocs/SKILL.md   (solo este proyecto; se puede commitear)
  --global    ~/.claude/skills/mdocs/SKILL.md   (todos tus proyectos en este equipo)
  --dir       <dir>/mdocs/SKILL.md              (otra carpeta de skills)

--with-binary copia además este ejecutable a bin/ dentro del skill, para que el
agente lo encuentre aunque mdocs no esté en el PATH.`,
		Example: `  mdocs skill install
  mdocs skill install --global --with-binary`,
		RunE: func(cmd *cobra.Command, args []string) error {
			base := dir
			switch {
			case base != "":
			case global:
				home, err := os.UserHomeDir()
				if err != nil {
					return err
				}
				base = filepath.Join(home, ".claude", "skills")
			default:
				base = filepath.Join(".claude", "skills")
			}
			target := filepath.Join(base, skills.Name)
			file := filepath.Join(target, "SKILL.md")
			if _, err := os.Stat(file); err == nil && !force {
				return fmt.Errorf("%s ya existe (usa --force para reemplazarlo)", file)
			}
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			if err := os.WriteFile(file, skills.MdocsSkill, 0644); err != nil {
				return err
			}
			fmt.Printf("  instalado %s\n", file)
			if withBinary {
				p, err := copySelf(filepath.Join(target, "bin"))
				if err != nil {
					return err
				}
				fmt.Printf("  binario   %s\n", p)
			}
			fmt.Println("  Claude Code lo detecta en la próxima sesión; también se invoca con /mdocs")
			return nil
		},
	}
	f := cmd.Flags()
	f.BoolVar(&global, "global", false, "Instalar en ~/.claude/skills (todos los proyectos)")
	f.StringVar(&dir, "dir", "", "Carpeta de skills de destino")
	f.BoolVar(&force, "force", false, "Reemplazar si ya existe")
	f.BoolVar(&withBinary, "with-binary", false, "Copiar también este ejecutable a bin/")
	return cmd
}

func newSkillZipCmd() *cobra.Command {
	var out string
	var withBinary bool
	cmd := &cobra.Command{
		Use:   "zip",
		Short: "Crea mdocs-skill.zip para subirlo en claude.ai (Settings → Capabilities → Skills)",
		Long: `Empaqueta la carpeta mdocs/ con SKILL.md en un .zip, el formato que pide
claude.ai al subir un skill.

--with-binary agrega este ejecutable como mdocs/bin/mdocs. En claude.ai el
skill corre en un entorno Linux amd64, así que el binario debe ser de
linux/amd64 (el release publica mdocs-skill_<versión>.zip ya armado).`,
		Example: `  mdocs skill zip
  mdocs skill zip --with-binary -o mdocs-skill.zip`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if withBinary && (runtime.GOOS != "linux" || runtime.GOARCH != "amd64") {
				fmt.Fprintf(os.Stderr, "  aviso: este binario es %s/%s; claude.ai necesita linux/amd64\n", runtime.GOOS, runtime.GOARCH)
			}
			f, err := os.Create(out)
			if err != nil {
				return err
			}
			defer f.Close()
			zw := zip.NewWriter(f)
			sh := &zip.FileHeader{Name: skills.Name + "/SKILL.md", Method: zip.Deflate}
			sh.SetMode(0644)
			w, err := zw.CreateHeader(sh)
			if err != nil {
				return err
			}
			if _, err := w.Write(skills.MdocsSkill); err != nil {
				return err
			}
			if withBinary {
				exe, err := os.Executable()
				if err != nil {
					return err
				}
				src, err := os.Open(exe)
				if err != nil {
					return err
				}
				defer src.Close()
				name := "mdocs"
				if runtime.GOOS == "windows" {
					name = "mdocs.exe"
				}
				hdr := &zip.FileHeader{Name: skills.Name + "/bin/" + name, Method: zip.Deflate}
				hdr.SetMode(0755)
				bw, err := zw.CreateHeader(hdr)
				if err != nil {
					return err
				}
				if _, err := io.Copy(bw, src); err != nil {
					return err
				}
			}
			if err := zw.Close(); err != nil {
				return err
			}
			fmt.Printf("  creado %s\n", out)
			return nil
		},
	}
	cmd.Flags().StringVarP(&out, "out", "o", "mdocs-skill.zip", "Archivo de salida")
	cmd.Flags().BoolVar(&withBinary, "with-binary", false, "Incluir este ejecutable en bin/")
	return cmd
}

func newSkillPrintCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "print",
		Short: "Muestra el SKILL.md",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := os.Stdout.Write(skills.MdocsSkill)
			return err
		},
	}
}

// copySelf copia el ejecutable actual a dir y devuelve la ruta.
func copySelf(dir string) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	name := "mdocs"
	if runtime.GOOS == "windows" {
		name = "mdocs.exe"
	}
	dst := filepath.Join(dir, name)
	in, err := os.Open(exe)
	if err != nil {
		return "", err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return "", err
	}
	return dst, out.Close()
}
