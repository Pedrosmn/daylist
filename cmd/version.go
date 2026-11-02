package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"daylist/internal/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Mostra a versão do daylist",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("daylist version %s\n", version.String())
	},
}

func init() {
	rootCmd.Version = version.String()
	rootCmd.AddCommand(versionCmd)
}
