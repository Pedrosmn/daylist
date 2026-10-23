// Package cmd define os comandos do CLI com Cobra.
package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"daylist/internal/vault"
)

// v é carregado antes de qualquer comando rodar.
var v *vault.Vault

var rootCmd = &cobra.Command{
	Use:          "daylist",
	Short:        "Daylist, weeklist, feedback e timer do vault Obsidian",
	Long:         "Sem subcomando, abre a daylist de hoje no Obsidian (ou cria, se ainda não existir).",
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		v, err = vault.Load()
		return err
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		today := time.Now()
		if fileExists(v.DaylistFile(today)) {
			return v.OpenNote(today)
		}

		fmt.Println("Daylist de hoje ainda não existe. Criando...")
		if err := createDaylist(today, false); err != nil {
			return err
		}
		if !fileExists(v.DaylistFile(today)) {
			return nil // usuário cancelou
		}
		return v.OpenNote(today)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
