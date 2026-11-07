package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"daylist/internal/timer"
)

// timerBarLayout descreve a barra em células, separada da renderização para
// facilitar os testes dos casos de limite e largura.
type timerBarLayout struct {
	Width   int // largura interna da barra, sem os colchetes
	Filled  int // células preenchidas pelo tempo decorrido
	MinMark int // índice da marcação do mínimo
	Min     int
	Max     int
	Current int
	Paused  bool
}

func newTimerBarLayout(s *timer.State, width int) (timerBarLayout, bool) {
	if s == nil || width < 3 {
		return timerBarLayout{}, false
	}
	min, max, ok := s.Limits()
	if !ok || max <= 0 {
		return timerBarLayout{}, false
	}

	barWidth := width - 2 // colchetes
	if barWidth < 1 {
		barWidth = 1
	}

	total := s.TotalMs()
	if total < 0 {
		total = 0
	}

	filled := int(math.Round(float64(total) / (float64(max) * 60_000) * float64(barWidth)))
	filled = clampInt(filled, 0, barWidth)

	minMark := int(math.Round(float64(min) / float64(max) * float64(barWidth)))
	minMark = clampInt(minMark, 0, barWidth-1)

	return timerBarLayout{
		Width:   barWidth,
		Filled:  filled,
		MinMark: minMark,
		Min:     min,
		Max:     max,
		Current: int(timer.RoundedMinutes(total)),
		Paused:  s.Paused,
	}, true
}

func (l timerBarLayout) label(width int) string {
	var text string
	if l.Min == l.Max {
		text = fmt.Sprintf("planejado %dm · agora %dm", l.Max, l.Current)
	} else {
		text = fmt.Sprintf("mín %dm · agora %dm · máx %dm", l.Min, l.Current, l.Max)
	}
	if l.Paused {
		text += " · ⏸ pausado"
	}

	if lipgloss.Width(text) > width {
		if l.Min == l.Max {
			text = fmt.Sprintf("%dm · %dm", l.Max, l.Current)
		} else {
			text = fmt.Sprintf("%dm·%dm·%dm", l.Min, l.Current, l.Max)
		}
		if l.Paused {
			text += " ⏸"
		}
	}
	if width > 0 && lipgloss.Width(text) > width {
		text = ansi.Truncate(text, width, "…")
	}
	return text
}

// timerBarView desenha a barra do timer e os rótulos abaixo dela. Devolve ""
// quando o item não tem faixa de tempo planejada.
func (m Model) timerBarView() string {
	layout, ok := newTimerBarLayout(m.timer, m.textWidth())
	if !ok {
		return ""
	}

	var b strings.Builder
	b.WriteString(styleSubtle.Render("["))
	for i := 0; i < layout.Width; i++ {
		switch {
		case i == layout.MinMark:
			b.WriteString(styleSelected.Render("┊"))
		case i < layout.Filled:
			b.WriteString(styleTimer.Render("█"))
		default:
			b.WriteString(styleSubtle.Render("░"))
		}
	}
	b.WriteString(styleSubtle.Render("]"))
	b.WriteString("\n")
	b.WriteString(styleHelp.Render(layout.label(m.textWidth())))
	b.WriteString("\n")
	return b.String()
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
