package cmd

import (
	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:     "new",
	Aliases: []string{"n"},
	Short:   "Cria a daylist de amanhã (QuickAdd)",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return v.QuickAdd("daylist")
	},
}

func init() {
	rootCmd.AddCommand(newCmd)
}
