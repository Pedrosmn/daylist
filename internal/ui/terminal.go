package ui

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
)

// hasTerminal informa se a UI nativa pode assumir stdin e stderr. O wizard
// desenha em stderr para manter stdout limpo para --dry-run e redirecionamentos.
func hasTerminal() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) && isatty.IsTerminal(os.Stderr.Fd())
}

// runProgram executa um modelo Bubble Tea inline (sem alt-screen), para que o
// histórico do terminal permaneça visível entre as etapas.
func runProgram(model tea.Model) (tea.Model, error) {
	return tea.NewProgram(
		model,
		tea.WithInput(os.Stdin),
		tea.WithOutput(os.Stderr),
	).Run()
}
