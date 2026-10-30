package week

import (
	"testing"
	"time"
)

func feedbackTime() time.Time {
	return time.Date(2026, time.November, 8, 18, 30, 0, 0, time.Local)
}

func TestStripTimerSuffix(t *testing.T) {
	cases := map[string]string{
		"correr ⏱ 25min (09:00–09:25)": "correr",
		"dormir ⏱ 30min":               "dormir",
		"correr":                       "correr",
		"⏱ 25min":                      "",
	}
	for input, want := range cases {
		if got := StripTimerSuffix(input); got != want {
			t.Errorf("StripTimerSuffix(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestBuildFeedbackNoHabits(t *testing.T) {
	weeklist := `# Tasks
- [ ] tarefa
---
# Hábitos
## Saúde
`
	const want = `# Feedback - Semana 2026-W45
Criado em: 2026-11-08 18:30

## Saúde

---
# Resumo
Nenhum hábito encontrado na weeklist.
`
	if got := BuildFeedback("2026-W45", weeklist, nil, feedbackTime()); got != want {
		t.Errorf("feedback diferente\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}
}

func TestBuildFeedbackWeeklistOnly(t *testing.T) {
	weeklist := `# Hábitos
## Saúde
- [ ] 5 min (U) - correr
`
	const want = `# Feedback - Semana 2026-W45
Criado em: 2026-11-08 18:30

## Saúde

---
# Resumo
Hábitos concluídos na semana: 0/0 (0%)

## Saúde
correr 0/0 - (0%)
`
	if got := BuildFeedback("2026-W45", weeklist, nil, feedbackTime()); got != want {
		t.Errorf("feedback diferente\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}
}

func TestBuildFeedbackCounts(t *testing.T) {
	weeklist := `# Tasks
- [ ] 10 min (I) - tarefa
---
# Hábitos
## Saúde
- [ ] 5 min (U) - correr
- [ ] xST () - dormir
## Estudos
- [ ] 20 min - ler
`
	days := []Day{
		{
			Date: localDate(2026, time.November, 2),
			Content: `# Tasks
- [x] 10 min (I) - tarefa
---
# Hábitos
## Saúde
- [x] 5 min (U) - correr ⏱ 25min (09:00–09:25)
- [ ] xST () - dormir
## Estudos
- [ ] (U) - planejar
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
`,
		},
	}

	const want = `# Feedback - Semana 2026-W45
Criado em: 2026-11-08 18:30

## Saúde

## Estudos

## Social

---
# Resumo
Hábitos concluídos na semana: 4/6 (67%)

## Saúde
correr 2/2 - (100%)
dormir 1/2 - (50%)

## Estudos
ler 0/0 - (0%)
planejar 0/1 - (0%)

## Social
livre 1/1 - (100%)
`
	if got := BuildFeedback("2026-W45", weeklist, days, feedbackTime()); got != want {
		t.Errorf("feedback diferente\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}
}

func TestBuildFeedbackCRLF(t *testing.T) {
	weeklist := "# Hábitos\r\n## Saúde\r\n- [x] 5 min (U) - correr\r\n"
	day := "# Hábitos\r\n## Saúde\r\n- [x] 5 min (U) - correr\r\n"
	got := BuildFeedback("2026-W45", weeklist, []Day{{Content: day}}, feedbackTime())
	if got == "" {
		t.Fatal("feedback vazio")
	}
	if !contains(got, "correr 1/1 - (100%)") {
		t.Errorf("feedback CRLF não contou corretamente:\n%s", got)
	}
}

func TestPercentage(t *testing.T) {
	cases := []struct {
		checked, total, want int
	}{
		{0, 0, 0},
		{1, 3, 33},
		{2, 3, 67},
		{1, 2, 50},
		{3, 3, 100},
	}
	for _, tc := range cases {
		if got := percentage(tc.checked, tc.total); got != tc.want {
			t.Errorf("percentage(%d, %d) = %d, want %d", tc.checked, tc.total, got, tc.want)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
