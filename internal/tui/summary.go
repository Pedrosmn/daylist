package tui

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"

	"daylist/internal/week"
)

var shortWeekdays = []string{"dom", "seg", "ter", "qua", "qui", "sex", "sáb"}

// summaryView desenha o resumo da semana da data visualizada. É somente
// leitura: nenhuma ação desta tela altera a daylist, a weeklist ou o feedback.
func (m Model) summaryView() string {
	var b strings.Builder
	b.WriteString(m.fitWidth(styleTitle.Render("resumo da semana") + "  " + styleSubtle.Render(weekLabel(m.day))))
	b.WriteString("\n\n")

	if m.weekReport == nil {
		b.WriteString(m.summaryMissingView())
		b.WriteString("\n")
		b.WriteString(m.fitWidth(styleHelp.Render("esc/q volta · resumo somente leitura")))
		return b.String()
	}

	report := *m.weekReport
	width := m.textWidth()

	bar := percentBar(report.Percent, report.Total, summaryBarWidth(width))
	b.WriteString(m.fitWidth(bar + "  " + fmt.Sprintf("%d%%", report.Percent)))
	b.WriteString("\n")
	b.WriteString(m.fitWidth(styleHelp.Render(fmt.Sprintf(
		"%d/%d hábitos · %d/%d dias", report.Checked, report.Total,
		report.DaysWithDaylist(), summaryDaysTotal(report)))))
	b.WriteString("\n")

	if m.summaryFeedback {
		b.WriteString(m.fitWidth(styleOK.Render("feedback.md: gerado")))
	} else {
		b.WriteString(m.fitWidth(styleHelp.Render("feedback.md: não gerado")))
	}
	b.WriteString("\n\n")

	b.WriteString(m.fitWidth(styleHeading.Render("Resumo por categoria")))
	b.WriteString("\n")
	if !report.HasHabits() {
		b.WriteString(m.fitWidth(styleHelp.Render("  Nenhum hábito na weeklist.")))
		b.WriteString("\n")
	} else {
		for _, category := range report.Categories {
			name := category.Name
			if name == "" {
				name = "Sem categoria"
			}
			header := fmt.Sprintf("## %s  %s", name, summaryScore(category.Checked, category.Total, category.Percent))
			b.WriteString(m.fitWidth(styleHeading.Render(header)))
			b.WriteString("\n")
			if len(category.Habits) == 0 {
				b.WriteString(m.fitWidth(styleHelp.Render("  (sem hábitos)")))
				b.WriteString("\n")
				continue
			}
			for _, habit := range category.Habits {
				line := fmt.Sprintf("  %s  %s", habit.Desc, summaryScore(habit.Checked, habit.Total, habit.Percent))
				b.WriteString(m.fitWidth(line))
				b.WriteString("\n")
			}
		}
	}

	b.WriteString("\n")
	b.WriteString(m.fitWidth(styleHeading.Render("Dias")))
	b.WriteString("\n")
	today := dateOnly(m.now)
	for _, day := range report.Days {
		b.WriteString(m.summaryDayLine(day, today))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.fitWidth(styleHelp.Render("esc/q volta · resumo somente leitura")))
	return b.String()
}

func (m Model) summaryMissingView() string {
	label := weekLabel(m.day)
	if m.weekReportErr != nil && !errors.Is(m.weekReportErr, os.ErrNotExist) {
		return m.fitWidth(styleError.Render("✗ não foi possível ler o resumo: "+m.weekReportErr.Error())) + "\n"
	}
	return m.fitWidth(fmt.Sprintf("Weeklist de %s não encontrada; não há resumo desta semana.", label)) + "\n"
}

func (m Model) summaryDayLine(day week.DayStat, today time.Time) string {
	label := fmt.Sprintf("%s %s", shortWeekdays[day.Date.Weekday()], day.Date.Format("02/01"))

	var line string
	switch {
	case day.Date.After(today):
		line = fmt.Sprintf("  %s  ·  futuro", label)
	case !day.HasDaylist:
		line = fmt.Sprintf("  %s  —  sem daylist", label)
	default:
		score := summaryScore(day.Checked, day.Total, day.Percent)
		if m.textWidth() >= 30 {
			line = fmt.Sprintf("  %s  %s  %s", label, percentBar(day.Percent, day.Total, 6), score)
		} else {
			line = fmt.Sprintf("  %s  %s", label, score)
		}
	}

	if sameDay(day.Date, today) {
		line += "  · hoje"
		line = styleSelected.Render(line)
	}
	return m.fitWidth(line)
}

// summaryScore usa — quando não há ocorrências, para não sugerir um desempenho
// de 0% em um hábito que sequer apareceu na semana.
func summaryScore(checked, total, percent int) string {
	if total == 0 {
		return "0/0 (—)"
	}
	return fmt.Sprintf("%d/%d (%d%%)", checked, total, percent)
}

func summaryDaysTotal(report week.Report) int {
	if len(report.Days) == 0 {
		return 7
	}
	return len(report.Days)
}

func summaryBarWidth(width int) int {
	if width <= 0 {
		return 20
	}
	barWidth := width - 6 // deixa espaço para "  100%"
	if barWidth < 4 {
		barWidth = width
	}
	if barWidth > 24 {
		barWidth = 24
	}
	if barWidth < 1 {
		barWidth = 1
	}
	return barWidth
}

// fitWidth evita que o resumo quebre em terminais estreitos.
func (m Model) fitWidth(line string) string {
	width := m.textWidth()
	if width <= 0 {
		return line
	}
	return ansi.Truncate(line, width, "…")
}
