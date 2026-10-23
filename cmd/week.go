package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"daylist/internal/ui"
	"daylist/internal/vault"
)

var weekCmd = &cobra.Command{
	Use:     "week",
	Aliases: []string{"w"},
	Short:   "Cria a weeklist (QuickAdd), substitui o antigo -w",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		current := vault.ISOWeek(time.Now())
		target := "" // vazio = comportamento padrão (semana que vem)

		_, statErr := os.Stat(v.WeeklistFile(current))
		if errors.Is(statErr, os.ErrNotExist) {
			optNext := "Semana que vem (padrão)"
			optCurrent := fmt.Sprintf("Semana atual (recuperar, %s)", current)

			choice, err := ui.Pick("Criar weeklist de qual semana? ", []string{optNext, optCurrent})
			if err != nil {
				return err
			}
			if choice == "" {
				fmt.Println("Cancelado.")
				return nil
			}
			if choice == optCurrent {
				target = current
			}
		}

		if target != "" {
			if err := vault.WriteTarget(v.WeeklistTargetFile(), target); err != nil {
				return err
			}
		} else if err := vault.ClearTarget(v.WeeklistTargetFile()); err != nil {
			return err
		}

		return v.QuickAdd("weeklist")
	},
}

func init() {
	rootCmd.AddCommand(weekCmd)
}
