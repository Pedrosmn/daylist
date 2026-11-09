package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"daylist/internal/week"
)

func weekLineReport() week.Report {
	days := []week.DayStat{
		{Date: localDate(2026, time.November, 2), HasDaylist: true, Checked: 3, Total: 5, Percent: 60},
		{Date: localDate(2026, time.November, 3), HasDaylist: true, Checked: 4, Total: 4, Percent: 100},
		{Date: localDate(2026, time.November, 4), HasDaylist: true, Checked: 3, Total: 5, Percent: 60},
		{Date: localDate(2026, time.November, 5), HasDaylist: true, Checked: 3, Total: 7, Percent: 43},
		{Date: localDate(2026, time.November, 6)},
		{Date: localDate(2026, time.November, 7)},
		{Date: localDate(2026, time.November, 8)},
	}
	return week.Report{
		Days:    days,
		Checked: 13,
		Total:   21,
		Percent: 62,
	}
}

func TestWeekLineLayoutFullLine(t *testing.T) {
	viewed := localDate(2026, time.November, 4)
	today := localDate(2026, time.November, 4)
	layout := newWeekLineLayout(weekLineReport(), viewed, today)

	line := ansi.Strip(layout.render(80))
	for _, want := range []string{
		"Semana 2026-W45",
		"62%",
		"13/21 hábitos",
		"4/7 dias",
		"hoje 3/5",
	} {
		if !strings.Contains(line, want) {
			t.Errorf("linha não contém %q: %q", want, line)
		}
	}
	if got := strings.Count(line, "█"); got != 6 {
		t.Errorf("células preenchidas = %d, want 6: %q", got, line)
	}
	if got := strings.Count(line, "░"); got != 4 {
		t.Errorf("células vazias = %d, want 4: %q", got, line)
	}
	if width := lipgloss.Width(layout.render(80)); width > 80 {
		t.Errorf("largura = %d, want <= 80", width)
	}
}

func TestWeekLineLayoutFitsNarrowTerminals(t *testing.T) {
	viewed := localDate(2026, time.November, 4)
	today := localDate(2026, time.November, 4)
	layout := newWeekLineLayout(weekLineReport(), viewed, today)

	for _, width := range []int{80, 60, 50, 40, 30, 20, 10, 5, 1} {
		line := layout.render(width)
		if got := lipgloss.Width(line); got > width {
			t.Errorf("largura %d: linha com %d colunas: %q", width, got, ansi.Strip(line))
		}
	}
}

func TestWeekLineLayoutCompactLabels(t *testing.T) {
	viewed := localDate(2026, time.November, 4)
	today := localDate(2026, time.November, 4)
	layout := newWeekLineLayout(weekLineReport(), viewed, today)

	line := ansi.Strip(layout.render(36))
	if !strings.Contains(line, "W45") {
		t.Errorf("largura 36 deveria usar a semana curta: %q", line)
	}
	if !strings.Contains(line, "hoje 3/5") {
		t.Errorf("largura 36 deveria manter o recorte de hoje: %q", line)
	}

	minimal := ansi.Strip(layout.render(20))
	if !strings.Contains(minimal, "W45") || !strings.Contains(minimal, "62%") {
		t.Errorf("largura 20 deveria manter semana e percentual: %q", minimal)
	}
}

func TestWeekLineLayoutTodayWithoutDaylist(t *testing.T) {
	report := weekLineReport()
	report.Days[2].HasDaylist = false

	line := ansi.Strip(newWeekLineLayout(report, localDate(2026, time.November, 4), localDate(2026, time.November, 4)).render(80))
	if !strings.Contains(line, "hoje —") {
		t.Errorf("recorte de hoje deveria mostrar —: %q", line)
	}
}

func TestWeekLineLayoutOmitsTodayOutsideWeek(t *testing.T) {
	line := ansi.Strip(newWeekLineLayout(
		weekLineReport(),
		localDate(2026, time.November, 4),
		localDate(2026, time.November, 9),
	).render(80))
	if strings.Contains(line, "hoje") {
		t.Errorf("semana passada não deveria ter recorte de hoje: %q", line)
	}
}

func TestWeekLineLayoutZeroTotals(t *testing.T) {
	report := week.Report{
		Days: []week.DayStat{
			{Date: localDate(2026, time.November, 2)},
			{Date: localDate(2026, time.November, 3)},
			{Date: localDate(2026, time.November, 4)},
			{Date: localDate(2026, time.November, 5)},
			{Date: localDate(2026, time.November, 6)},
			{Date: localDate(2026, time.November, 7)},
			{Date: localDate(2026, time.November, 8)},
		},
	}
	line := ansi.Strip(newWeekLineLayout(report, localDate(2026, time.November, 4), localDate(2026, time.November, 4)).render(80))
	if !strings.Contains(line, "0%") || !strings.Contains(line, "0/0") {
		t.Errorf("totais zerados inesperados: %q", line)
	}
	if strings.Contains(line, "█") {
		t.Errorf("barra zerada não deveria ter células preenchidas: %q", line)
	}
}

func TestWeekLineViewHiddenWithoutWeeklist(t *testing.T) {
	fake := newFake("# Tasks\n")
	fake.report = nil
	m := New(fake)

	if view := m.weekLineView(); view != "" {
		t.Errorf("linha deveria ficar oculta sem relatório:\n%s", view)
	}
}

func TestWeekLineViewAppearsAndUpdatesOnToggle(t *testing.T) {
	fake := newFake("# Hábitos\n## Saúde\n- [ ] 5 min (U) - correr\n")
	report := weekLineReport()
	fake.report = &report
	m := New(fake)
	m = updateModel(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})

	if line := ansi.Strip(m.weekLineView()); !strings.Contains(line, "13/21 hábitos") {
		t.Fatalf("linha inicial = %q", line)
	}

	updated := weekLineReport()
	updated.Checked++
	updated.Percent = 67
	fake.report = &updated

	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeySpace})
	if line := ansi.Strip(m.weekLineView()); !strings.Contains(line, "67%") {
		t.Errorf("linha não atualizou após marcar/desmarcar: %q", line)
	}
}

func TestWeekLineViewUsesDayRolloverReport(t *testing.T) {
	fake := newFake("# Tasks\n")
	report := weekLineReport()
	fake.report = &report
	m := New(fake)

	fake.now = fake.now.AddDate(0, 0, 1)
	fake.report = &week.Report{Percent: 100, Checked: 7, Total: 7, Days: report.Days}
	m = updateModel(t, m, tickMsg(fake.now))

	if line := ansi.Strip(m.weekLineView()); !strings.Contains(line, "100%") {
		t.Errorf("linha não acompanhou a virada do dia: %q", line)
	}
}
