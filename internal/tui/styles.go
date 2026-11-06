package tui

import "github.com/charmbracelet/lipgloss"

var (
	styleTitle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	styleSubtle   = lipgloss.NewStyle().Faint(true)
	styleHeading  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	styleSelected = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	styleDone     = lipgloss.NewStyle().Faint(true).Strikethrough(true)
	styleMotivate = lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("13"))
	styleError    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9"))
	styleOK       = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	styleTimer    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	styleHelp     = lipgloss.NewStyle().Faint(true)
	styleForm     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
)
