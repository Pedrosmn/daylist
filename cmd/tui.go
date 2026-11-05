package cmd

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"daylist/internal/tui"
	"daylist/internal/vault"
)

// runTUI abre o TUI da daylist de hoje. A criação da daylist e da weeklist
// reutiliza os comandos existentes, executados com o terminal liberado pelo
// Bubble Tea.
func runTUI(v *vault.Vault) error {
	backend := tui.NewRealBackend(v)
	backend.CreateDaylistFn = func() error {
		return createDaylist(v, time.Now(), false)
	}
	backend.CreateWeeklistFn = func() error {
		return runWeek(v, weekOptions{Current: true}, time.Now())
	}

	program := tea.NewProgram(tui.New(backend), tea.WithAltScreen())
	_, err := program.Run()
	return err
}
