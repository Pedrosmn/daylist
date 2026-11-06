package tui

import "daylist/internal/theme"

// Aliases mantêm a view do TUI inalterada enquanto os estilos passam a ser
// compartilhados com o wizard.
var (
	styleTitle    = theme.Title
	styleSubtle   = theme.Subtle
	styleHeading  = theme.Heading
	styleSelected = theme.Selected
	styleDone     = theme.Done
	styleMotivate = theme.Motivate
	styleError    = theme.Error
	styleOK       = theme.OK
	styleTimer    = theme.Timer
	styleHelp     = theme.Help
	styleForm     = theme.Form
)
