package cmd

import (
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open",
	Short: "Abre a daylist de hoje no Obsidian",
	Long: `Abre a daylist de hoje no Obsidian. Se a nota ainda não existir, inicia o
wizard e abre a nota depois da criação, preservando o comportamento antigo do
comando sem subcomando.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return openToday(v)
	},
}

func init() {
	rootCmd.AddCommand(openCmd)
}
