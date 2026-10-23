package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"daylist/internal/timer"
	"daylist/internal/ui"
)

var timerCmd = &cobra.Command{
	Use:     "timer",
	Aliases: []string{"t"},
	Short:   "Controla o timer de um item da daylist",
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

var timerStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Para o timer e grava o tempo na daylist",
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

		mins, err := s.Finish(v.DaylistFile(time.Now()))
		if errors.Is(err, timer.ErrLineNotFound) {
			fmt.Println("Linha original não encontrada (foi editada?). Timer descartado sem gravar.")
			return os.Remove(statePath)
		}
		if err != nil {
			return err // o estado é mantido, nada se perde
		}

		if err := os.Remove(statePath); err != nil {
			fmt.Println("Aviso: não foi possível remover o estado do timer.")
		}
		fmt.Printf("Timer parado: %s — %dmin\n", s.Desc, mins)
		return nil
	},
}

func init() {
	timerCmd.AddCommand(timerStartCmd, timerPauseCmd, timerStopCmd)
	rootCmd.AddCommand(timerCmd)
}
