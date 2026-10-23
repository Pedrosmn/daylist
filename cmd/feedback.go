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

var feedbackCmd = &cobra.Command{
	Use:     "feedback",
	Aliases: []string{"f"},
	Short:   "Gera o feedback da semana (QuickAdd), substitui o antigo -f",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		prev := vault.ISOWeek(time.Now().AddDate(0, 0, -7))
		target := "" // vazio = semana atual

		_, statErr := os.Stat(v.FeedbackFile(prev))
		if errors.Is(statErr, os.ErrNotExist) {
			optCurrent := "Semana atual"
			optPrev := fmt.Sprintf("Semana passada (%s)", prev)

			choice, err := ui.Pick("Gerar feedback de qual semana? ", []string{optCurrent, optPrev})
			if err != nil {
				return err
			}
			if choice == "" {
				fmt.Println("Cancelado.")
				return nil
			}
			if choice == optPrev {
				target = prev
			}
		}

		if target != "" {
			if err := vault.WriteTarget(v.FeedbackTargetFile(), target); err != nil {
				return err
			}
		} else if err := vault.ClearTarget(v.FeedbackTargetFile()); err != nil {
			return err
		}

		return v.QuickAdd("feedback")
	},
}

func init() {
	rootCmd.AddCommand(feedbackCmd)
}
