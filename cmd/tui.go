package cmd

import (
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"daylist/internal/planner"
	"daylist/internal/tui"
	"daylist/internal/vault"
)

// runTUI abre o TUI da daylist de hoje. A criação da daylist e da weeklist
// reutiliza os comandos existentes, executados com o terminal liberado pelo
// Bubble Tea.
func runTUI(v *vault.Vault) error {
	backend := tui.NewRealBackend(v)
	backend.PlanDaylistFn = func() (*planner.Plan, error) {
		now := time.Now()
		return planner.Build(v, now, uiPrompter{}, os.Stdout, now)
	}
	backend.WriteDaylistFn = func(plan *planner.Plan) error {
		return planner.Write(v, plan)
	}
	backend.CreateWeeklistFn = func() error {
		return runWeek(v, weekOptions{Current: true}, time.Now())
	}

	program := tea.NewProgram(tui.New(backend), tea.WithAltScreen())
	_, err := program.Run()
	return err
}
