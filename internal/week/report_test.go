package week

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestComputeReportCategoriesAndHabits(t *testing.T) {
	weeklist := `# Tasks
- [ ] 10 min (I) - tarefa
---
# Hábitos
## Saúde
- [ ] 5 min (U) - correr
- [ ] xST () - dormir
## Estudos
- [ ] 20 min - ler
- [ ] 5 min (U) - revisar
`
	days := []Day{
		{
			Date: localDate(2026, time.November, 2),
			Content: `# Hábitos
## Saúde
- [x] 5 min (U) - correr ⏱ 25min (09:00–09:25)
- [ ] xST () - dormir
## Estudos
- [x] (U) - planejar
## Social
- [x] livre
`,
		},
		{
			Date: localDate(2026, time.November, 3),
			Content: `# Hábitos
## Saúde
- [x] 5 min (U) - correr
- [x] xST () - dormir ⏱ 30min
## Estudos
- [x] 20 min - ler
`,
		},
	}

	report := ComputeReport("2026-W45", weeklist, days)

	wantCategories := []string{"Saúde", "Estudos", "Social"}
	if got := categoryNames(report); !reflect.DeepEqual(got, wantCategories) {
		t.Fatalf("categorias = %v, want %v", got, wantCategories)
	}

	wantHabits := map[string][]string{
		"Saúde":   {"correr", "dormir"},
		"Estudos": {"ler", "revisar", "planejar"},
		"Social":  {"livre"},
	}
	for _, category := range report.Categories {
		if got := habitNames(category); !reflect.DeepEqual(got, wantHabits[category.Name]) {
			t.Errorf("hábitos de %s = %v, want %v", category.Name, got, wantHabits[category.Name])
		}
	}

	wantHabitStats := map[string]HabitStat{
		"correr":   {Category: "Saúde", Desc: "correr", Checked: 2, Total: 2, Percent: 100},
		"dormir":   {Category: "Saúde", Desc: "dormir", Checked: 1, Total: 2, Percent: 50},
		"ler":      {Category: "Estudos", Desc: "ler", Checked: 1, Total: 1, Percent: 100},
		"revisar":  {Category: "Estudos", Desc: "revisar", Checked: 0, Total: 0, Percent: 0},
		"planejar": {Category: "Estudos", Desc: "planejar", Checked: 1, Total: 1, Percent: 100},
		"livre":    {Category: "Social", Desc: "livre", Checked: 1, Total: 1, Percent: 100},
	}
	for _, category := range report.Categories {
		for _, habit := range category.Habits {
			if want := wantHabitStats[habit.Desc]; habit != want {
				t.Errorf("hábito %s = %+v, want %+v", habit.Desc, habit, want)
			}
		}
	}

	wantCategoryStats := map[string]CategoryStat{
		"Saúde":   {Name: "Saúde", Checked: 3, Total: 4, Percent: 75},
		"Estudos": {Name: "Estudos", Checked: 2, Total: 2, Percent: 100},
		"Social":  {Name: "Social", Checked: 1, Total: 1, Percent: 100},
	}
	for _, category := range report.Categories {
		want := wantCategoryStats[category.Name]
		if category.Checked != want.Checked || category.Total != want.Total || category.Percent != want.Percent {
			t.Errorf("categoria %s = %d/%d (%d%%), want %d/%d (%d%%)",
				category.Name, category.Checked, category.Total, category.Percent,
				want.Checked, want.Total, want.Percent)
		}
	}

	if report.Checked != 6 || report.Total != 7 || report.Percent != 86 {
		t.Errorf("totais = %d/%d (%d%%), want 6/7 (86%%)",
			report.Checked, report.Total, report.Percent)
	}
	if !report.HasHabits() {
		t.Error("HasHabits deveria ser verdadeiro")
	}
}

func TestComputeReportDayStats(t *testing.T) {
	weeklist := "# Hábitos\n## Saúde\n- [ ] 5 min (U) - correr\n"
	days := []Day{
		{
			Date: localDate(2026, time.November, 2),
			Content: `# Hábitos
## Saúde
- [x] 5 min (U) - correr
`,
		},
		{
			Date: localDate(2026, time.November, 4),
			Content: `# Hábitos
## Saúde
- [ ] 5 min (U) - correr
`,
		},
	}

	report := ComputeReport("2026-W45", weeklist, days)
	if len(report.Days) != 7 {
		t.Fatalf("Days = %d, want 7", len(report.Days))
	}

	want := map[string]DayStat{
		"2026-11-02": {Date: localDate(2026, time.November, 2), HasDaylist: true, Checked: 1, Total: 1, Percent: 100},
		"2026-11-03": {Date: localDate(2026, time.November, 3)},
		"2026-11-04": {Date: localDate(2026, time.November, 4), HasDaylist: true, Checked: 0, Total: 1, Percent: 0},
		"2026-11-05": {Date: localDate(2026, time.November, 5)},
		"2026-11-06": {Date: localDate(2026, time.November, 6)},
		"2026-11-07": {Date: localDate(2026, time.November, 7)},
		"2026-11-08": {Date: localDate(2026, time.November, 8)},
	}
	for _, day := range report.Days {
		wantDay, ok := want[day.Date.Format("2006-01-02")]
		if !ok {
			t.Errorf("dia inesperado: %s", day.Date)
			continue
		}
		if day != wantDay {
			t.Errorf("dia %s = %+v, want %+v", day.Date.Format("2006-01-02"), day, wantDay)
		}
	}

	if got := report.DaysWithDaylist(); got != 2 {
		t.Errorf("DaysWithDaylist = %d, want 2", got)
	}
}

func TestComputeReportUnusedHabit(t *testing.T) {
	weeklist := "# Hábitos\n## Saúde\n- [ ] 5 min (U) - correr\n"
	report := ComputeReport("2026-W45", weeklist, nil)

	if !report.HasHabits() {
		t.Fatal("hábito da weeklist deveria aparecer mesmo sem ocorrências")
	}
	if report.Total != 0 || report.Checked != 0 || report.Percent != 0 {
		t.Errorf("totais = %d/%d (%d%%), want 0/0 (0%%)",
			report.Checked, report.Total, report.Percent)
	}
	if len(report.Categories) != 1 {
		t.Fatalf("categorias = %d, want 1", len(report.Categories))
	}
	habit := report.Categories[0].Habits[0]
	if habit.Desc != "correr" || habit.Checked != 0 || habit.Total != 0 || habit.Percent != 0 {
		t.Errorf("hábito sem uso = %+v, want correr 0/0 (0%%)", habit)
	}
	if report.DaysWithDaylist() != 0 {
		t.Errorf("DaysWithDaylist = %d, want 0", report.DaysWithDaylist())
	}
	if len(report.Days) != 7 {
		t.Errorf("Days = %d, want 7", len(report.Days))
	}
}

func TestComputeReportNoHabits(t *testing.T) {
	report := ComputeReport("2026-W45", "", nil)

	if report.HasHabits() {
		t.Error("HasHabits deveria ser falso")
	}
	if len(report.Categories) != 0 {
		t.Errorf("categorias = %v, want vazio", report.Categories)
	}
	if report.Checked != 0 || report.Total != 0 || report.Percent != 0 {
		t.Errorf("totais = %d/%d (%d%%), want 0/0 (0%%)",
			report.Checked, report.Total, report.Percent)
	}
}

func TestComputeReportCRLF(t *testing.T) {
	weeklist := `# Hábitos
## Saúde
- [ ] 5 min (U) - correr
- [ ] xST () - dormir
## Estudos
- [ ] 20 min - ler
`
	day := `# Hábitos
## Saúde
- [x] 5 min (U) - correr ⏱ 25min
- [ ] xST () - dormir
## Estudos
- [x] 20 min - ler
`

	lf := ComputeReport("2026-W45", weeklist, []Day{{Date: localDate(2026, time.November, 2), Content: day}})
	crlf := ComputeReport("2026-W45", toCRLF(weeklist), []Day{{Date: localDate(2026, time.November, 2), Content: toCRLF(day)}})

	if !reflect.DeepEqual(lf, crlf) {
		t.Errorf("CRLF gerou relatório diferente\n--- lf ---\n%+v\n--- crlf ---\n%+v", lf, crlf)
	}
}

func TestComputeReportAlwaysAndTimerSuffix(t *testing.T) {
	weeklist := `# Hábitos
## Rotina
- [ ] 5 min (U) - corrida #always
- [ ] 5 min (U) - leitura
`
	day := `# Hábitos
## Rotina
- [x] 5 min (U) - corrida #always
- [x] 5 min (U) - leitura ⏱ 25min (09:00–09:25)
`

	report := ComputeReport("2026-W45", weeklist, []Day{{Date: localDate(2026, time.November, 2), Content: day}})

	if len(report.Categories) != 1 || len(report.Categories[0].Habits) != 2 {
		t.Fatalf("relatório inesperado: %+v", report)
	}
	if got := report.Categories[0].Habits[0]; got.Desc != "corrida" {
		t.Errorf("Desc = %q, want corrida sem #always", got.Desc)
	}
	if got := report.Categories[0].Habits[1]; got.Desc != "leitura" {
		t.Errorf("Desc = %q, want leitura sem sufixo ⏱", got.Desc)
	}
	for _, habit := range report.Categories[0].Habits {
		if habit.Checked != 1 || habit.Total != 1 || habit.Percent != 100 {
			t.Errorf("hábito = %+v, want 1/1 (100%%)", habit)
		}
	}
}

func TestComputeReportInvalidWeekUsesProvidedDays(t *testing.T) {
	day := Day{
		Date:    localDate(2026, time.November, 2),
		Content: "# Hábitos\n## Saúde\n- [x] 5 min (U) - correr\n",
	}

	report := ComputeReport("semana-inválida", "# Hábitos\n## Saúde\n- [ ] 5 min (U) - correr\n", []Day{day})

	if len(report.Days) != 1 {
		t.Fatalf("Days = %d, want 1", len(report.Days))
	}
	if !report.Days[0].HasDaylist || report.Days[0].Checked != 1 || report.Days[0].Total != 1 {
		t.Errorf("dia = %+v, want daylist 1/1", report.Days[0])
	}
}

func TestBuildFeedbackUsesReportTotals(t *testing.T) {
	weeklist := "# Hábitos\n## Saúde\n- [ ] 5 min (U) - correr\n- [ ] xST () - dormir\n"
	days := []Day{{
		Date:    localDate(2026, time.November, 2),
		Content: "# Hábitos\n## Saúde\n- [x] 5 min (U) - correr\n- [ ] xST () - dormir\n",
	}}

	report := ComputeReport("2026-W45", weeklist, days)
	feedback := BuildFeedback("2026-W45", weeklist, days, feedbackTime())

	want := fmt.Sprintf("Hábitos concluídos na semana: %d/%d (%d%%)",
		report.Checked, report.Total, report.Percent)
	if !contains(feedback, want) {
		t.Errorf("feedback não usou os totais do relatório (%q):\n%s", want, feedback)
	}
}

func TestComputeReportUncategorizedDuplicateAndSectionEnd(t *testing.T) {
	weeklist := `# Hábitos
- [ ] 5 min (U) - solto
## Saúde
- [ ] 5 min (U) - correr
- [ ] 5 min (U) - correr
# Extra
- [ ] 5 min (U) - não é hábito
`
	days := []Day{{
		Date: localDate(2026, time.November, 2),
		Content: `# Hábitos
- [x] 5 min (U) - solto
## Saúde
- [x] 5 min (U) - correr
# Extra
- [x] 5 min (U) - não contar
`,
	}}

	report := ComputeReport("2026-W45", weeklist, days)

	if got := categoryNames(report); !reflect.DeepEqual(got, []string{"Saúde", ""}) {
		t.Fatalf("categorias = %v, want [Saúde \"\"]", got)
	}
	if len(report.Categories[0].Habits) != 1 {
		t.Errorf("hábito duplicado na weeklist criou ocorrências extras: %+v", report.Categories[0].Habits)
	}
	if report.Checked != 2 || report.Total != 2 || report.Percent != 100 {
		t.Errorf("totais = %d/%d (%d%%), want 2/2 (100%%)",
			report.Checked, report.Total, report.Percent)
	}
	if len(report.Days) != 7 || report.Days[0].Checked != 2 || report.Days[0].Total != 2 {
		t.Errorf("dia = %+v, want 2/2 na segunda", report.Days[0])
	}

	feedback := BuildFeedback("2026-W45", weeklist, days, feedbackTime())
	if !strings.Contains(feedback, "Hábitos concluídos na semana: 2/2 (100%)") {
		t.Errorf("feedback não contou o hábito sem categoria:\n%s", feedback)
	}
	if got := strings.Count(feedback, "## "); got != 2 {
		t.Errorf("cabeçalhos de categoria = %d, want 2 (sem categoria vazia):\n%s", got, feedback)
	}
	if strings.Contains(feedback, "\n## \n") {
		t.Errorf("feedback não deveria ter cabeçalho de categoria vazio:\n%s", feedback)
	}
}

func categoryNames(report Report) []string {
	names := make([]string, 0, len(report.Categories))
	for _, category := range report.Categories {
		names = append(names, category.Name)
	}
	return names
}

func habitNames(category CategoryStat) []string {
	names := make([]string, 0, len(category.Habits))
	for _, habit := range category.Habits {
		names = append(names, habit.Desc)
	}
	return names
}
