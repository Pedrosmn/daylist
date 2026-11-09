package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"daylist/internal/week"
)

// weekLineLayout separa o conteúdo da linha compacta da renderização, para
// testar barras, percentuais e larguras sem depender do modelo inteiro.
type weekLineLayout struct {
	Week      string
	ShortWeek string
	Percent   int
	Checked   int
	Total     int
	DaysWith  int
	DaysTotal int
	Today     *week.DayStat
}

func newWeekLineLayout(report week.Report, viewed, today time.Time) weekLineLayout {
	layout := weekLineLayout{
		Week:      weekLabel(viewed),
		ShortWeek: shortWeekLabel(weekLabel(viewed)),
		Percent:   report.Percent,
		Checked:   report.Checked,
		Total:     report.Total,
		DaysWith:  report.DaysWithDaylist(),
		DaysTotal: len(report.Days),
	}
	for i := range report.Days {
		if sameDay(report.Days[i].Date, today) {
			day := report.Days[i]
			layout.Today = &day
			break
		}
	}
	if layout.DaysTotal == 0 {
		layout.DaysTotal = 7
	}
	return layout
}

func shortWeekLabel(week string) string {
	if i := strings.LastIndex(week, "-"); i >= 0 {
		return week[i+1:]
	}
	return week
}

type weekLineFormat struct {
	BarWidth int
	FullWeek bool
	Habits   bool
	Days     bool
	Today    bool
}

func (l weekLineLayout) line(format weekLineFormat) string {
	var b strings.Builder

	if format.FullWeek {
		b.WriteString(styleSubtle.Render("Semana " + l.Week))
	} else {
		b.WriteString(styleSubtle.Render(l.ShortWeek))
	}
	b.WriteString(" ")

	if format.BarWidth > 0 {
		b.WriteString(l.bar(format.BarWidth))
		b.WriteString(" ")
	}

	b.WriteString(styleHelp.Render(fmt.Sprintf("%d%%", l.Percent)))
	if format.Habits {
		b.WriteString(styleHelp.Render(fmt.Sprintf(" · %d/%d hábitos", l.Checked, l.Total)))
	} else {
		b.WriteString(styleHelp.Render(fmt.Sprintf(" · %d/%d", l.Checked, l.Total)))
	}
	if format.Days {
		b.WriteString(styleHelp.Render(fmt.Sprintf(" · %d/%d dias", l.DaysWith, l.DaysTotal)))
	}
	if format.Today && l.Today != nil {
		if l.Today.HasDaylist {
			b.WriteString(styleHelp.Render(fmt.Sprintf(" · hoje %d/%d", l.Today.Checked, l.Today.Total)))
		} else {
			b.WriteString(styleHelp.Render(" · hoje —"))
		}
	}

	return b.String()
}

func (l weekLineLayout) bar(width int) string {
	return percentBar(l.Percent, l.Total, width)
}

// percentBar desenha a barra preenchida conforme o percentual. Sem ocorrências
// a barra fica vazia, evitando uma leitura enganosa de 100%.
func percentBar(percent, total, width int) string {
	filled := 0
	if total > 0 && width > 0 {
		filled = int(math.Round(float64(percent) / 100 * float64(width)))
		filled = clampInt(filled, 0, width)
	}

	var b strings.Builder
	for i := 0; i < width; i++ {
		if i < filled {
			b.WriteString(styleTimer.Render("█"))
		} else {
			b.WriteString(styleSubtle.Render("░"))
		}
	}
	return b.String()
}

// render escolhe o formato mais completo que cabe na largura e, no limite,
// trunca a versão mínima.
func (l weekLineLayout) render(width int) string {
	formats := []weekLineFormat{
		{BarWidth: 10, FullWeek: true, Habits: true, Days: true, Today: true},
		{BarWidth: 8, FullWeek: true, Habits: false, Days: true, Today: true},
		{BarWidth: 6, FullWeek: false, Habits: false, Days: true, Today: true},
		{BarWidth: 4, FullWeek: false, Habits: false, Days: false, Today: true},
		{BarWidth: 0, FullWeek: false, Habits: false, Days: false, Today: true},
		{BarWidth: 0, FullWeek: false, Habits: false, Days: false, Today: false},
	}
	last := ""
	for _, format := range formats {
		line := l.line(format)
		last = line
		if width <= 0 || lipgloss.Width(line) <= width {
			return line
		}
	}
	if width <= 0 {
		return last
	}
	return ansi.Truncate(last, width, "…")
}

// weekLineView devolve a linha compacta da semana da data visualizada. Fica
// oculta quando a weeklist não existe.
func (m Model) weekLineView() string {
	if m.weekReport == nil {
		return ""
	}
	layout := newWeekLineLayout(*m.weekReport, m.day, dateOnly(m.now))
	return layout.render(m.textWidth())
}
