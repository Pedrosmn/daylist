package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"daylist/internal/vault"
	"daylist/internal/week"
)

func testRealBackend(t *testing.T, today time.Time) (*RealBackend, *vault.Vault) {
	t.Helper()
	v := &vault.Vault{Name: "Test", Path: t.TempDir()}
	return &RealBackend{Vault: v, Now: func() time.Time { return today }}, v
}

func localDate(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.Local)
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func TestWeekReportOnCurrentWeek(t *testing.T) {
	today := localDate(2026, time.November, 4) // quarta-feira, 2026-W45
	backend, v := testRealBackend(t, today)

	weeklist := `# Tasks
- [ ] tarefa
---
# Hábitos
## Saúde
- [ ] 5 min (U) - correr
- [ ] xST () - dormir
## Estudos
- [ ] 20 min - ler
`
	writeTestFile(t, v.WeeklistFile("2026-W45"), weeklist)
	writeTestFile(t, v.DaylistFile(localDate(2026, time.November, 1)), "# Hábitos\n## Saúde\n- [x] 5 min (U) - correr\n")
	writeTestFile(t, v.DaylistFile(localDate(2026, time.November, 2)), `# Hábitos
## Saúde
- [x] 5 min (U) - correr
- [ ] xST () - dormir
`)
	writeTestFile(t, v.DaylistFile(today), `# Hábitos
## Saúde
- [x] 5 min (U) - correr
## Estudos
- [x] 20 min - ler
`)
	// Daylist futura na mesma semana: deve ser ignorada.
	writeTestFile(t, v.DaylistFile(localDate(2026, time.November, 5)), "# Hábitos\n## Saúde\n- [x] 5 min (U) - correr\n")

	report, err := backend.WeekReportOn(today)
	if err != nil {
		t.Fatalf("WeekReportOn: %v", err)
	}

	if len(report.Days) != 7 {
		t.Fatalf("dias = %d, want 7", len(report.Days))
	}
	if got := report.DaysWithDaylist(); got != 2 {
		t.Errorf("dias com daylist = %d, want 2", got)
	}
	if report.Checked != 3 || report.Total != 4 || report.Percent != 75 {
		t.Errorf("totais = %d/%d (%d%%), want 3/4 (75%%)",
			report.Checked, report.Total, report.Percent)
	}

	wantDays := map[string]week.DayStat{
		"2026-11-02": {Date: localDate(2026, time.November, 2), HasDaylist: true, Checked: 1, Total: 2, Percent: 50},
		"2026-11-03": {Date: localDate(2026, time.November, 3)},
		"2026-11-04": {Date: localDate(2026, time.November, 4), HasDaylist: true, Checked: 2, Total: 2, Percent: 100},
		"2026-11-05": {Date: localDate(2026, time.November, 5)},
		"2026-11-06": {Date: localDate(2026, time.November, 6)},
		"2026-11-07": {Date: localDate(2026, time.November, 7)},
		"2026-11-08": {Date: localDate(2026, time.November, 8)},
	}
	for _, day := range report.Days {
		want := wantDays[day.Date.Format("2006-01-02")]
		if day != want {
			t.Errorf("dia %s = %+v, want %+v", day.Date.Format("2006-01-02"), day, want)
		}
	}
}

func TestWeekReportOnPastWeekReadsAllDays(t *testing.T) {
	today := localDate(2026, time.November, 9) // segunda-feira, 2026-W46
	backend, v := testRealBackend(t, today)

	writeTestFile(t, v.WeeklistFile("2026-W45"), "# Hábitos\n## Saúde\n- [ ] 5 min (U) - correr\n")
	writeTestFile(t, v.DaylistFile(localDate(2026, time.November, 2)), "# Hábitos\n## Saúde\n- [x] 5 min (U) - correr\n")
	writeTestFile(t, v.DaylistFile(localDate(2026, time.November, 8)), "# Hábitos\n## Saúde\n- [x] 5 min (U) - correr\n")

	report, err := backend.WeekReportOn(localDate(2026, time.November, 4))
	if err != nil {
		t.Fatalf("WeekReportOn: %v", err)
	}
	if got := report.DaysWithDaylist(); got != 2 {
		t.Errorf("dias com daylist = %d, want 2", got)
	}
	if report.Checked != 2 || report.Total != 2 {
		t.Errorf("totais = %d/%d, want 2/2", report.Checked, report.Total)
	}

	// A data visualizada define a semana: a semana atual ainda não tem weeklist.
	if _, err := backend.WeekReportOn(today); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("semana atual sem weeklist deveria devolver os.ErrNotExist, veio %v", err)
	}
}

func TestWeekReportOnMissingWeeklist(t *testing.T) {
	backend, _ := testRealBackend(t, localDate(2026, time.November, 4))

	if _, err := backend.WeekReportOn(localDate(2026, time.November, 4)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("erro = %v, want os.ErrNotExist", err)
	}
}

func TestAutoGeneratePreviousFeedback(t *testing.T) {
	today := localDate(2026, time.November, 9) // segunda-feira, 2026-W46
	backend, v := testRealBackend(t, today)

	writeTestFile(t, v.WeeklistFile("2026-W45"), "# Hábitos\n## Saúde\n- [ ] 5 min (U) - correr\n")
	writeTestFile(t, v.DaylistFile(localDate(2026, time.November, 2)), "# Hábitos\n## Saúde\n- [x] 5 min (U) - correr\n")
	writeTestFile(t, v.DaylistFile(localDate(2026, time.November, 3)), "# Hábitos\n## Saúde\n- [ ] 5 min (U) - correr\n")

	created, err := backend.AutoGeneratePreviousFeedback(today)
	if err != nil {
		t.Fatalf("AutoGeneratePreviousFeedback: %v", err)
	}
	if !created {
		t.Fatal("feedback da última semana encerrada deveria ser gerado")
	}

	path := v.FeedbackFile("2026-W45")
	content := readTestFile(t, path)
	if !strings.Contains(content, "# Feedback - Semana 2026-W45") {
		t.Errorf("feedback sem o cabeçalho da semana:\n%s", content)
	}
	if !strings.Contains(content, "Hábitos concluídos na semana: 1/2 (50%)") {
		t.Errorf("feedback não contou a semana:\n%s", content)
	}
	if !strings.Contains(content, "Criado em: 2026-11-09 00:00") {
		t.Errorf("feedback deveria usar a data de hoje:\n%s", content)
	}
	if !backend.FeedbackExistsOn(localDate(2026, time.November, 4)) {
		t.Error("FeedbackExistsOn deveria achar o feedback da semana de 2026-W45")
	}
	if backend.FeedbackExistsOn(today) {
		t.Error("FeedbackExistsOn não deveria achar feedback da semana atual")
	}

	// Idempotente: não sobrescreve nem gera de novo.
	writeTestFile(t, v.DaylistFile(localDate(2026, time.November, 2)), "# Hábitos\n## Saúde\n- [ ] 5 min (U) - correr\n")
	created, err = backend.AutoGeneratePreviousFeedback(today)
	if err != nil {
		t.Fatalf("segunda chamada: %v", err)
	}
	if created {
		t.Error("segunda chamada não deveria gerar o feedback novamente")
	}
	if got := readTestFile(t, path); got != content {
		t.Errorf("feedback existente foi sobrescrito\n--- antes ---\n%s\n--- depois ---\n%s", content, got)
	}
}

func TestAutoGeneratePreviousFeedbackSkipsMissingWeeklist(t *testing.T) {
	today := localDate(2026, time.November, 9)
	backend, v := testRealBackend(t, today)

	// Weeklist de uma semana antiga não autoriza backfill.
	writeTestFile(t, v.WeeklistFile("2026-W44"), "# Hábitos\n## Saúde\n- [ ] 5 min (U) - correr\n")

	created, err := backend.AutoGeneratePreviousFeedback(today)
	if err != nil {
		t.Fatalf("AutoGeneratePreviousFeedback: %v", err)
	}
	if created {
		t.Error("não deveria gerar sem a weeklist da última semana encerrada")
	}
	if _, err := os.Stat(v.FeedbackFile("2026-W44")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("não deveria fazer backfill de semana antiga: %v", err)
	}
}

func TestAutoGeneratePreviousFeedbackDoesNotOverwrite(t *testing.T) {
	today := localDate(2026, time.November, 9)
	backend, v := testRealBackend(t, today)

	writeTestFile(t, v.WeeklistFile("2026-W45"), "# Hábitos\n## Saúde\n- [ ] 5 min (U) - correr\n")
	const sentinel = "feedback escrito à mão\n"
	writeTestFile(t, v.FeedbackFile("2026-W45"), sentinel)

	created, err := backend.AutoGeneratePreviousFeedback(today)
	if err != nil {
		t.Fatalf("AutoGeneratePreviousFeedback: %v", err)
	}
	if created {
		t.Error("não deveria gerar por cima de um feedback existente")
	}
	if got := readTestFile(t, v.FeedbackFile("2026-W45")); got != sentinel {
		t.Errorf("feedback existente foi alterado: %q", got)
	}
}

func TestAutoGeneratePreviousFeedbackSkipsCurrentWeek(t *testing.T) {
	today := localDate(2026, time.November, 8) // domingo, 2026-W45
	backend, v := testRealBackend(t, today)

	writeTestFile(t, v.WeeklistFile("2026-W45"), "# Hábitos\n## Saúde\n- [ ] 5 min (U) - correr\n")

	created, err := backend.AutoGeneratePreviousFeedback(today)
	if err != nil {
		t.Fatalf("AutoGeneratePreviousFeedback: %v", err)
	}
	if created {
		t.Error("não deveria gerar o feedback da semana atual antes de ela encerrar")
	}
	if _, err := os.Stat(v.FeedbackFile("2026-W45")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("feedback da semana atual não deveria existir: %v", err)
	}
}

func TestWeekReportOnReadErrors(t *testing.T) {
	today := localDate(2026, time.November, 4) // quarta-feira, 2026-W45
	backend, v := testRealBackend(t, today)

	t.Run("weeklist não é arquivo", func(t *testing.T) {
		if err := os.MkdirAll(v.WeeklistFile("2026-W45"), 0o755); err != nil {
			t.Fatal(err)
		}
		_, err := backend.WeekReportOn(today)
		if err == nil || errors.Is(err, os.ErrNotExist) {
			t.Errorf("erro = %v, want erro de leitura diferente de os.ErrNotExist", err)
		}
	})

	t.Run("daylist não é arquivo", func(t *testing.T) {
		backend, v := testRealBackend(t, today)
		writeTestFile(t, v.WeeklistFile("2026-W45"), "# Hábitos\n## Saúde\n- [ ] 5 min (U) - correr\n")
		if err := os.MkdirAll(v.DaylistFile(localDate(2026, time.November, 2)), 0o755); err != nil {
			t.Fatal(err)
		}
		_, err := backend.WeekReportOn(today)
		if err == nil || errors.Is(err, os.ErrNotExist) {
			t.Errorf("erro = %v, want erro de leitura diferente de os.ErrNotExist", err)
		}
	})
}

func TestAutoGeneratePreviousFeedbackReadErrors(t *testing.T) {
	today := localDate(2026, time.November, 9) // segunda-feira, 2026-W46

	t.Run("weeklist não é arquivo", func(t *testing.T) {
		backend, v := testRealBackend(t, today)
		if err := os.MkdirAll(v.WeeklistFile("2026-W45"), 0o755); err != nil {
			t.Fatal(err)
		}
		created, err := backend.AutoGeneratePreviousFeedback(today)
		if err == nil || errors.Is(err, os.ErrNotExist) {
			t.Errorf("erro = %v, want erro de leitura diferente de os.ErrNotExist", err)
		}
		if created {
			t.Error("não deveria criar feedback quando a weeklist não pôde ser lida")
		}
	})

	t.Run("daylist não é arquivo", func(t *testing.T) {
		backend, v := testRealBackend(t, today)
		writeTestFile(t, v.WeeklistFile("2026-W45"), "# Hábitos\n## Saúde\n- [ ] 5 min (U) - correr\n")
		if err := os.MkdirAll(v.DaylistFile(localDate(2026, time.November, 2)), 0o755); err != nil {
			t.Fatal(err)
		}
		created, err := backend.AutoGeneratePreviousFeedback(today)
		if err == nil || errors.Is(err, os.ErrNotExist) {
			t.Errorf("erro = %v, want erro de leitura diferente de os.ErrNotExist", err)
		}
		if created {
			t.Error("não deveria criar feedback quando uma daylist não pôde ser lida")
		}
	})
}

func TestWriteNewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "feedback.md")

	created, err := writeNewFile(path, "primeiro\n")
	if err != nil {
		t.Fatalf("writeNewFile: %v", err)
	}
	if !created {
		t.Error("primeira gravação deveria criar o arquivo")
	}
	if got := readTestFile(t, path); got != "primeiro\n" {
		t.Errorf("conteúdo = %q, want primeiro", got)
	}

	created, err = writeNewFile(path, "segundo\n")
	if err != nil {
		t.Fatalf("segunda gravação: %v", err)
	}
	if created {
		t.Error("arquivo existente não deveria ser sobrescrito")
	}
	if got := readTestFile(t, path); got != "primeiro\n" {
		t.Errorf("conteúdo após segunda gravação = %q, want primeiro", got)
	}

	blocker := filepath.Join(dir, "bloqueio")
	writeTestFile(t, blocker, "não é diretório")
	created, err = writeNewFile(filepath.Join(blocker, "sub", "feedback.md"), "x")
	if err == nil {
		t.Error("MkdirAll deveria falhar quando o pai é um arquivo")
	}
	if created {
		t.Error("não deveria criar arquivo quando o diretório pai falha")
	}
}
