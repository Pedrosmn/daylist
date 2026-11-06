// Package theme centraliza os estilos visuais compartilhados entre o TUI e o
// wizard. Manter as cores e os marcadores em um só lugar evita que as duas
// interfaces divirjam com o tempo.
package theme

import "github.com/charmbracelet/lipgloss"

var (
	Title    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	Subtle   = lipgloss.NewStyle().Faint(true)
	Heading  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	Selected = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	Done     = lipgloss.NewStyle().Faint(true).Strikethrough(true)
	Motivate = lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("13"))
	Error    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9"))
	OK       = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	Timer    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	Help     = lipgloss.NewStyle().Faint(true)
	Form     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
)
