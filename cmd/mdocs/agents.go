package mdocs

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/cristianino/mdocs/assets"
)

func NewAgentsCmd() *cobra.Command {
	var force, print bool
	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Escribe AGENTS.md: cómo debe trabajar un agente de IA en este proyecto",
		Long: `Crea AGENTS.md con la estructura del proyecto, los comandos recomendados
(context, check, add, build --json) y las reglas para escribir Markdown,
enlaces y diagramas. Herramientas como Claude Code, Codex o Cursor lo leen
automáticamente. Con --print lo muestra sin escribir el archivo.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if print {
				os.Stdout.Write(assets.AgentsGuide)
				return nil
			}
			if _, err := os.Stat("AGENTS.md"); err == nil && !force {
				return fmt.Errorf("AGENTS.md ya existe (usa --force para sobrescribir o --print para verlo)")
			}
			if err := os.WriteFile("AGENTS.md", assets.AgentsGuide, 0644); err != nil {
				return err
			}
			fmt.Println("  creado  AGENTS.md")
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Sobrescribir AGENTS.md si existe")
	cmd.Flags().BoolVar(&print, "print", false, "Mostrar la guía en stdout sin escribir")
	return cmd
}
