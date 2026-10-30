package cmd

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestFeedbackWeek(t *testing.T) {
	now := testNow()

	if got, err := feedbackWeek(now, feedbackOptions{}); err != nil || got != "2026-W44" {
		t.Errorf("feedbackWeek padrão = %q, %v; want 2026-W44, nil", got, err)
	}
	if got, err := feedbackWeek(now, feedbackOptions{Week: "2026-W45"}); err != nil || got != "2026-W45" {
		t.Errorf("feedbackWeek explícita = %q, %v; want 2026-W45, nil", got, err)
	}
	for _, invalid := range []string{"2026-W1", "2026-W00", "2026-W54", "abc"} {
		if _, err := feedbackWeek(now, feedbackOptions{Week: invalid}); err == nil {
			t.Errorf("feedbackWeek(%q) deveria falhar", invalid)
		}
	}
}

func TestRunFeedbackDefaultsToCurrent(t *testing.T) {
	v := newTestVault(t)
	writeTestNote(t, v.WeeklistFile("2026-W44"), `# Hábitos
## Estudos
- [ ] 5 min (U) - AWS
- [ ] 10 min - revisar
`)
	writeTestNote(t, v.DaylistFile(time.Date(2026, time.October, 26, 0, 0, 0, 0, time.Local)), `# Hábitos
## Estudos
- [x] 5 min (U) - AWS
- [ ] 10 min - revisar
`)
	writeTestNote(t, v.DaylistFile(time.Date(2026, time.October, 27, 0, 0, 0, 0, time.Local)), `# Hábitos
## Estudos
- [x] 5 min (U) - AWS
- [ ] 10 min - revisar
`)

	out, err := captureStdout(t, func() error {
		return runFeedback(v, feedbackOptions{}, testNow())
	})
	if err != nil {
		t.Fatalf("runFeedback: %v", err)
	}

	content := readTestNote(t, v.FeedbackFile("2026-W44"))
	for _, want := range []string{
		"Hábitos concluídos na semana: 2/4 (50%)",
		"AWS 2/2 - (100%)",
		"revisar 0/2 - (0%)",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("feedback não contém %q:\n%s", want, content)
		}
	}
	if !strings.Contains(out, "Feedback de 2026-W44 criado") {
		t.Errorf("saída não confirma criação: %q", out)
	}
}

func TestRunFeedbackExplicitWeek(t *testing.T) {
	v := newTestVault(t)
	writeTestNote(t, v.WeeklistFile("2026-W45"), "# Hábitos\n## Estudos\n- [ ] 5 min (U) - AWS\n")

	if _, err := captureStdout(t, func() error {
		return runFeedback(v, feedbackOptions{Week: "2026-W45"}, testNow())
	}); err != nil {
		t.Fatalf("runFeedback: %v", err)
	}
	if !strings.Contains(readTestNote(t, v.FeedbackFile("2026-W45")), "Semana 2026-W45") {
		t.Error("feedback não foi criado para a semana explícita")
	}
}

func TestRunFeedbackMissingWeeklist(t *testing.T) {
	v := newTestVault(t)
	err := runFeedback(v, feedbackOptions{Week: "2026-W45"}, testNow())
	if err == nil {
		t.Fatal("runFeedback deveria falhar sem weeklist")
	}
	if !strings.Contains(err.Error(), "2026-W45") || !strings.Contains(err.Error(), v.WeeklistFile("2026-W45")) {
		t.Errorf("erro não cita semana/caminho: %v", err)
	}
}

func TestRunFeedbackExisting(t *testing.T) {
	v := newTestVault(t)
	target := v.FeedbackFile("2026-W44")
	writeTestNote(t, target, "feedback existente\n")

	out, err := captureStdout(t, func() error {
		return runFeedback(v, feedbackOptions{}, testNow())
	})
	if err != nil {
		t.Fatalf("runFeedback: %v", err)
	}
	if !strings.Contains(out, "Feedback já existe") {
		t.Errorf("saída não avisa que já existe: %q", out)
	}
	if got := readTestNote(t, target); got != "feedback existente\n" {
		t.Errorf("feedback existente foi alterado: %q", got)
	}
}

func TestRunFeedbackDryRun(t *testing.T) {
	v := newTestVault(t)
	writeTestNote(t, v.WeeklistFile("2026-W44"), "# Hábitos\n## Estudos\n- [ ] 5 min (U) - AWS\n")

	out, err := captureStdout(t, func() error {
		return runFeedback(v, feedbackOptions{DryRun: true}, testNow())
	})
	if err != nil {
		t.Fatalf("runFeedback: %v", err)
	}
	if !strings.Contains(out, "# Feedback - Semana 2026-W44") {
		t.Errorf("dry-run não mostrou o feedback: %q", out)
	}
	assertNotExists(t, v.FeedbackFile("2026-W44"))
}

func TestRunFeedbackPrintPath(t *testing.T) {
	v := newTestVault(t)
	want := v.FeedbackFile("2026-W44")

	out, err := captureStdout(t, func() error {
		return runFeedback(v, feedbackOptions{PrintPath: true}, testNow())
	})
	if err != nil {
		t.Fatalf("runFeedback: %v", err)
	}
	if strings.TrimSpace(out) != want {
		t.Errorf("print-path = %q, want %q", strings.TrimSpace(out), want)
	}
	assertNotExists(t, want)
}

func TestFeedbackFlagsRegistered(t *testing.T) {
	for _, name := range []string{"week", "dry-run", "print-path"} {
		if feedbackCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s não registrada em feedback", name)
		}
	}
}

func TestRunWeekCleansLegacyTarget(t *testing.T) {
	v := newTestVault(t)
	writeTestNote(t, v.WeeklistFile("2026-W44"), "# Hábitos\n## Estudos\n- [ ] 5 min (U) - AWS\n")
	writeTestNote(t, legacyPath(v, legacyWeeklistTarget), "{}")
	writeTestNote(t, legacyPath(v, legacyFeedbackTarget), "{}")

	if _, err := captureStdout(t, func() error {
		return runWeek(v, weekOptions{}, testNow())
	}); err != nil {
		t.Fatalf("runWeek: %v", err)
	}
	assertNotExists(t, legacyPath(v, legacyWeeklistTarget))
	assertNotExists(t, legacyPath(v, legacyFeedbackTarget))
}

func TestRunWeekDryRunKeepsLegacyTargets(t *testing.T) {
	v := newTestVault(t)
	writeTestNote(t, v.WeeklistFile("2026-W44"), "# Hábitos\n## Estudos\n- [ ] 5 min (U) - AWS\n")
	writeTestNote(t, legacyPath(v, legacyWeeklistTarget), "{}")

	if _, err := captureStdout(t, func() error {
		return runWeek(v, weekOptions{DryRun: true}, testNow())
	}); err != nil {
		t.Fatalf("runWeek: %v", err)
	}
	if _, err := os.Stat(legacyPath(v, legacyWeeklistTarget)); err != nil {
		t.Errorf("dry-run removeu alvo legado: %v", err)
	}
}

func TestRunFeedbackCleansLegacyTarget(t *testing.T) {
	v := newTestVault(t)
	writeTestNote(t, v.WeeklistFile("2026-W44"), "# Hábitos\n## Estudos\n- [ ] 5 min (U) - AWS\n")
	writeTestNote(t, legacyPath(v, legacyWeeklistTarget), "{}")
	writeTestNote(t, legacyPath(v, legacyFeedbackTarget), "{}")

	if _, err := captureStdout(t, func() error {
		return runFeedback(v, feedbackOptions{}, testNow())
	}); err != nil {
		t.Fatalf("runFeedback: %v", err)
	}
	assertNotExists(t, legacyPath(v, legacyWeeklistTarget))
	assertNotExists(t, legacyPath(v, legacyFeedbackTarget))
}

func TestRunFeedbackDryRunKeepsLegacyTargets(t *testing.T) {
	v := newTestVault(t)
	writeTestNote(t, v.WeeklistFile("2026-W44"), "# Hábitos\n## Estudos\n- [ ] 5 min (U) - AWS\n")
	writeTestNote(t, legacyPath(v, legacyFeedbackTarget), "{}")

	if _, err := captureStdout(t, func() error {
		return runFeedback(v, feedbackOptions{DryRun: true}, testNow())
	}); err != nil {
		t.Fatalf("runFeedback: %v", err)
	}
	if _, err := os.Stat(legacyPath(v, legacyFeedbackTarget)); err != nil {
		t.Errorf("dry-run removeu alvo legado: %v", err)
	}
}
