package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"daylist/internal/timer"
	"daylist/internal/ui"
	"daylist/internal/vault"
)

var timerCmd = &cobra.Command{
	Use:     "timer",
	Aliases: []string{"t"},
	Short:   "Controla o timer de um item da daylist",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return statusTimer(v, false)
	},
}

var timerStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Escolhe um item pendente e inicia o timer",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		statePath := v.StateFile()

		if s, err := timer.Load(statePath); err == nil {
			fmt.Printf("Já tem um timer rodando: %s\n", s.Desc)
			return nil
		} else if !errors.Is(err, timer.ErrNoTimer) {
			return err
		}

		today := time.Now()
		file := v.DaylistFile(today)
		if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
			fmt.Println("Daylist de hoje não encontrada.")
			return nil
		}

		items, err := timer.PendingItems(file)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			fmt.Println("Nenhum item pendente no daylist de hoje.")
			return nil
		}

		chosen, err := ui.Pick("Qual item começar? ", items)
		if err != nil {
			return err
		}
		if chosen == "" {
			fmt.Println("Nenhum item selecionado.")
			return nil
		}

		s := timer.New(chosen, v.DaylistRel(today))
		if err := s.Save(statePath); err != nil {
			return err
		}
		fmt.Printf("Timer iniciado: %s\n", s.Desc)
		return nil
	},
}

var timerPauseCmd = &cobra.Command{
	Use:   "pause",
	Short: "Pausa ou retoma o timer",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		statePath := v.StateFile()

		s, err := timer.Load(statePath)
		if errors.Is(err, timer.ErrNoTimer) {
			fmt.Println("Nenhum timer rodando.")
			return nil
		}
		if err != nil {
			return err
		}

		resumed := s.Toggle()
		if err := s.Save(statePath); err != nil {
			return err
		}

		if resumed {
			fmt.Printf("Timer retomado: %s\n", s.Desc)
		} else {
			fmt.Printf("Timer pausado: %s — %dmin acumulados\n", s.Desc, s.Elapsed/60000)
		}
		return nil
	},
}

var statusPrompt bool

var timerStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Mostra o timer atual",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return statusTimer(v, statusPrompt)
	},
}

var timerStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Para o timer e grava o tempo na daylist",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return stopTimer(v)
	},
}

// promptText devolve a saída curta para RPROMPT.
func promptText(s *timer.State) string {
	text := fmt.Sprintf("⏱ %dm", timer.RoundedMinutes(s.TotalMs()))
	if s.Paused {
		text += " ⏸"
	}
	return text
}

// statusText devolve a linha humana do timer atual.
func statusText(s *timer.State) string {
	mins := timer.RoundedMinutes(s.TotalMs())
	if s.Paused {
		return fmt.Sprintf("Timer pausado: %s — %dmin acumulados%s", s.Desc, mins, startedAt(s))
	}
	return fmt.Sprintf("Timer rodando: %s — %dmin%s", s.Desc, mins, startedAt(s))
}

// startedAt devolve " (iniciado às HH:MM)" ou "" quando Started não existe.
func startedAt(s *timer.State) string {
	if s.Started <= 0 {
		return ""
	}
	return fmt.Sprintf(" (iniciado às %s)", time.UnixMilli(s.Started).Format("15:04"))
}

func statusTimer(v *vault.Vault, prompt bool) error {
	s, err := timer.Load(v.StateFile())
	if errors.Is(err, timer.ErrNoTimer) {
		if prompt {
			return nil // prompt limpo, sem mensagem
		}
		fmt.Println("Nenhum timer rodando.")
		return nil
	}
	if err != nil {
		return err
	}
	if prompt {
		fmt.Print(promptText(s))
		return nil
	}
	fmt.Println(statusText(s))
	return nil
}

// timerDaylistPath resolve a daylist em que o timer começou, a partir do
// caminho relativo guardado no estado (não da data atual).
func timerDaylistPath(v *vault.Vault, s *timer.State) string {
	return filepath.Join(v.Path, filepath.FromSlash(s.File))
}

func stopTimer(v *vault.Vault) error {
	statePath := v.StateFile()

	s, err := timer.Load(statePath)
	if errors.Is(err, timer.ErrNoTimer) {
		fmt.Println("Nenhum timer rodando.")
		return nil
	}
	if err != nil {
		return err
	}

	file := timerDaylistPath(v, s)
	mins, err := s.Finish(file)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("daylist do timer não encontrada: %s (o estado foi mantido; use \"daylist timer stop\" de novo)", file)
	}
	switch {
	case errors.Is(err, timer.ErrLineNotFound):
		fmt.Println(`Não achei a linha na daylist. O estado foi mantido; corrija a linha e rode "daylist timer stop" de novo.`)
		return nil
	case errors.Is(err, timer.ErrAmbiguousLine):
		fmt.Println("Mais de uma linha corresponde à descrição; nada foi gravado. O estado foi mantido.")
		fmt.Println(err)
		return nil
	case errors.Is(err, timer.ErrAlreadyRecorded):
		if rmErr := os.Remove(statePath); rmErr != nil {
			fmt.Println("Aviso: não foi possível remover o estado do timer.")
		}
		fmt.Println("O tempo já estava registrado nesta linha; estado do timer removido.")
		return nil
	case err != nil:
		return err // o estado é mantido, nada se perde
	}

	if err := os.Remove(statePath); err != nil {
		fmt.Println("Aviso: não foi possível remover o estado do timer.")
	}
	fmt.Printf("Timer parado: %s — %dmin\n", s.Desc, mins)
	return nil
}

func init() {
	timerStatusCmd.Flags().BoolVar(&statusPrompt, "prompt", false, "saída curta para RPROMPT (ex.: ⏱ 25m)")

	timerCmd.AddCommand(timerStartCmd, timerPauseCmd, timerStopCmd, timerStatusCmd)
	rootCmd.AddCommand(timerCmd)
}
