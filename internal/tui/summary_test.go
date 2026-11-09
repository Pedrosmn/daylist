package tui

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"daylist/internal/week"
)

func summaryReport() week.Report {
	report := weekLineReport()
	report.Categories = []week.CategoryStat{
		{
			Name: "Saúde", Checked: 3, Total: 4, Percent: 75,
			Habits: []week.HabitStat{
				{Category: "Saúde", Desc: "correr", Checked: 3, Total: 4, Percent: 75},
				{Category: "Saúde", Desc: "dormir", Checked: 0, Total: 0, Percent: 0},
			},
		},
		{
			Name: "Estudos", Checked: 2, Total: 3, Percent: 67,
			Habits: []week.HabitStat{
				{Category: "Estudos", Desc: "ler", Checked: 2, Total: 3, Percent: 67},
			},
		},
	}
	return report
}

func TestSummaryViewShowsWeekCategoriesDaysAndFeedback(t *testing.T) {
	fake := newFake("# Tasks\n- [ ] tarefa\n")
	report := summaryReport()
	report.Days[3].Checked = 0
	report.Days[3].Total = 0
	report.Days[3].Percent = 0
	fake.report = &report
	fake.feedback = false
	m := New(fake)
	m = updateModel(t, m, tea.WindowSizeMsg{Width: 80, Height: 40})

	m = updateModel(t, m, keyRunes("S"))
	if m.mode != modeSummary {
		t.Fatalf("mode = %v, want modeSummary", m.mode)
	}

	view := ansi.Strip(m.View())
	for _, want := range []string{
		"resumo da semana",
		"2026-W45",
		"62%",
		"13/21 hábitos",
		"4/7 dias",
		"feedback.md: não gerado",
		"Resumo por categoria",
		"## Saúde",
		"correr",
		"3/4 (75%)",
		"dormir",
		"0/0 (—)",
		"## Estudos",
		"ler",
		"Dias",
		"seg 02/11",
		"·  futuro",
		"· hoje",
		"esc/q volta",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("resumo não contém %q:\n%s", want, view)
		}
	}
}

func TestSummaryViewIsReadOnly(t *testing.T) {
	fake := newFake("# Tasks\n- [ ] 10 min (I) - tarefa\n")
	report := summaryReport()
	fake.report = &report
	m := New(fake)
	m = updateModel(t, m, keyRunes("S"))

	m = updateModel(t, m, keyRunes(" "))
	m = updateModel(t, m, keyRunes("j"))
	if m.mode != modeSummary {
		t.Errorf("mode = %v, want modeSummary", m.mode)
	}
	if fake.noteSaves != 0 {
		t.Errorf("resumo não deveria gravar a daylist (%d gravações)", fake.noteSaves)
	}

	m = updateModel(t, m, keyType(tea.KeyEsc))
	if m.mode != modeMain {
		t.Errorf("esc deveria voltar ao modeMain, veio %v", m.mode)
	}
}

func TestSummaryViewFeedbackGenerated(t *testing.T) {
	fake := newFake("# Tasks\n")
	report := summaryReport()
	fake.report = &report
	fake.feedback = true
	m := New(fake)

	m = updateModel(t, m, keyRunes("S"))
	if view := ansi.Strip(m.View()); !strings.Contains(view, "feedback.md: gerado") {
		t.Errorf("resumo deveria indicar o feedback gerado:\n%s", view)
	}
}

func TestSummaryViewMissingWeeklist(t *testing.T) {
	fake := newFake("# Tasks\n")
	fake.report = nil
	fake.reportErr = fmt.Errorf("weeklist de 2026-W45 não encontrada: %w", os.ErrNotExist)
	m := New(fake)

	m = updateModel(t, m, keyRunes("S"))
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "Weeklist de 2026-W45 não encontrada") {
		t.Errorf("resumo sem weeklist deveria explicar a ausência:\n%s", view)
	}
	if !strings.Contains(view, "esc/q volta") {
		t.Errorf("resumo sem weeklist deveria manter a ajuda:\n%s", view)
	}
}

func TestSummaryViewPastWeekShowsAllDays(t *testing.T) {
	fake := newFake("# Tasks\n")
	fake.now = time.Date(2026, time.November, 9, 10, 0, 0, 0, time.Local)
	report := summaryReport()
	fake.report = &report
	fake.feedback = true
	m := New(fake)

	m = updateModel(t, m, keyRunes("y"))
	if m.mode != modeMain {
		t.Fatalf("y deveria manter o modeMain, veio %v", m.mode)
	}
	m = updateModel(t, m, keyRunes("S"))

	view := ansi.Strip(m.View())
	for _, want := range []string{"seg 02/11", "ter 03/11", "qua 04/11", "qui 05/11", "sex 06/11", "sáb 07/11", "dom 08/11"} {
		if !strings.Contains(view, want) {
			t.Errorf("semana passada deveria mostrar %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "·  futuro") {
		t.Errorf("semana passada não deveria ter dias futuros:\n%s", view)
	}
	if strings.Contains(view, "· hoje") {
		t.Errorf("semana passada não deveria destacar hoje:\n%s", view)
	}
}

func TestSummaryViewDrawsProgressBarAndZeroTotals(t *testing.T) {
	fake := newFake("# Tasks\n")
	report := summaryReport()
	fake.report = &report
	m := New(fake)
	m = updateModel(t, m, tea.WindowSizeMsg{Width: 80, Height: 40})
	m = updateModel(t, m, keyRunes("S"))

	view := ansi.Strip(m.View())
	if !strings.Contains(view, "█") || !strings.Contains(view, "░") {
		t.Errorf("resumo deveria desenhar a barra de progresso:\n%s", view)
	}
	if !strings.Contains(view, "0/0 (—)") {
		t.Errorf("hábito sem ocorrências deveria mostrar 0/0 (—):\n%s", view)
	}
}

func TestSummaryViewWithoutHabits(t *testing.T) {
	fake := newFake("# Tasks\n")
	report := week.Report{Days: summaryReport().Days}
	fake.report = &report
	m := New(fake)
	m = updateModel(t, m, tea.WindowSizeMsg{Width: 80, Height: 40})
	m = updateModel(t, m, keyRunes("S"))

	view := ansi.Strip(m.View())
	for _, want := range []string{"0%", "0/0 hábitos", "Nenhum hábito na weeklist."} {
		if !strings.Contains(view, want) {
			t.Errorf("resumo sem hábitos não contém %q:\n%s", want, view)
		}
	}
	var barLine string
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "0%") {
			barLine = line
			break
		}
	}
	if barLine == "" || strings.Contains(barLine, "█") {
		t.Errorf("barra total sem hábitos deveria ficar vazia: %q\n%s", barLine, view)
	}
}

func TestSummaryYesterdayUsesViewedDateAndWeekLineKeepsToday(t *testing.T) {
	fake := newFake("# Tasks\n")
	report := summaryReport()
	fake.report = &report
	m := New(fake)
	m = updateModel(t, m, tea.WindowSizeMsg{Width: 80, Height: 40})

	m = updateModel(t, m, keyRunes("y"))
	if sameDay(m.day, fake.now) {
		t.Fatal("y deveria mudar a data visualizada para ontem")
	}
	if line := ansi.Strip(m.weekLineView()); !strings.Contains(line, "hoje") || !strings.Contains(line, "W45") {
		t.Errorf("linha de ontem deveria continuar na semana atual com o recorte de hoje: %q", line)
	}

	m = updateModel(t, m, keyRunes("S"))
	view := ansi.Strip(m.View())
	for _, want := range []string{"2026-W45", "qua 04/11", "qui 05/11", "· hoje"} {
		if !strings.Contains(view, want) {
			t.Errorf("resumo de ontem não contém %q:\n%s", want, view)
		}
	}
}

func TestSummaryViewNarrowTerminalFits(t *testing.T) {
	fake := newFake("# Tasks\n")
	report := summaryReport()
	fake.report = &report

	for _, width := range []int{40, 20, 10, 5, 1} {
		m := New(fake)
		m = updateModel(t, m, tea.WindowSizeMsg{Width: width, Height: 40})
		m = updateModel(t, m, keyRunes("S"))

		lines := strings.Split(strings.TrimRight(ansi.Strip(m.View()), "\n"), "\n")
		for _, line := range lines {
			if got := lipgloss.Width(line); got > width {
				t.Errorf("largura %d: linha com %d colunas: %q", width, got, line)
			}
		}
	}
}
