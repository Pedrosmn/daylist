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
	Long:         "Sem subcomando, abre o TUI da daylist de hoje quando houver terminal interativo. Sem TTY, abre a daylist de hoje no Obsidian (ou cria, se ainda não existir). Use `daylist open` para abrir no Obsidian em qualquer caso.",
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// version e --version não podem depender de um vault configurado.
		if cmd.Name() == "version" {
			return nil
		}
		var err error
		v, err = vault.Load()
		return err
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if isInteractive() {
			return runTUI(v)
		}
		return openToday(v)
	},
}

// openToday preserva o comportamento antigo: abre a daylist de hoje e, se ela
// ainda não existir, cria pelo wizard antes de abrir.
func openToday(v *vault.Vault) error {
	today := time.Now()
	if fileExists(v.DaylistFile(today)) {
		return v.OpenNote(today)
	}

	fmt.Println("Daylist de hoje ainda não existe. Criando...")
	if err := createDaylist(v, today, false); err != nil {
		return err
	}
	if !fileExists(v.DaylistFile(today)) {
		return nil // usuário cancelou
	}
	return v.OpenNote(today)
}

// isInteractive informa se stdin e stdout são terminais, para decidir entre o
// TUI e o comportamento de abertura no Obsidian.
func isInteractive() bool {
	return isCharDevice(os.Stdin) && isCharDevice(os.Stdout)
}

func isCharDevice(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
