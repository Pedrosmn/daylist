package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testNow() time.Time {
	// Quarta-feira da 2026-W44; a próxima semana é 2026-W45.
	return time.Date(2026, time.October, 28, 12, 0, 0, 0, time.Local)
}

func writeTestNote(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func readTestNote(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	return string(data)
}

func assertNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("arquivo %s deveria não existir (err = %v)", path, err)
	}
}

func TestWeekTarget(t *testing.T) {
	now := testNow()
	cases := []struct {
		name string
		opts weekOptions
		want string
	}{
		{"padrão", weekOptions{}, "2026-W45"},
		{"atual", weekOptions{Current: true}, "2026-W44"},
		{"próxima", weekOptions{Next: true}, "2026-W45"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := weekTarget(now, tc.opts); got != tc.want {
				t.Errorf("weekTarget = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRunWeekDefaultsToNext(t *testing.T) {
	v := newTestVault(t)
	source := v.WeeklistFile("2026-W44")
	writeTestNote(t, source, `# Tasks
- [ ] tarefa
---
# Hábitos
## Estudos
- [x] 5 min (U) - AWS
- [ ] 10 min - revisar
- [ ] ler
`)

	out, err := captureStdout(t, func() error {
		return runWeek(v, weekOptions{}, testNow())
	})
	if err != nil {
		t.Fatalf("runWeek: %v", err)
	}

	target := v.WeeklistFile("2026-W45")
	const want = `# Tasks

---
# Hábitos
## Estudos
- [ ] 5 min (U) - AWS
- [ ] 10 min - revisar
- [ ] ler
`
	if got := readTestNote(t, target); got != want {
		t.Errorf("weeklist diferente\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}
	if !strings.Contains(out, "Weeklist de 2026-W45 criada") {
		t.Errorf("saída não confirma criação: %q", out)
	}
}

func TestRunWeekCurrent(t *testing.T) {
	v := newTestVault(t)
	writeTestNote(t, v.WeeklistFile("2026-W43"), "# Hábitos\n## Estudos\n- [ ] 20 min (I) - estudar\n")

	if _, err := captureStdout(t, func() error {
		return runWeek(v, weekOptions{Current: true}, testNow())
	}); err != nil {
		t.Fatalf("runWeek: %v", err)
	}

	got := readTestNote(t, v.WeeklistFile("2026-W44"))
	if !strings.Contains(got, "- [ ] 20 min (I) - estudar") {
		t.Errorf("weeklist atual não copiou o hábito:\n%s", got)
	}
}

func TestRunWeekDryRun(t *testing.T) {
	v := newTestVault(t)
	writeTestNote(t, v.WeeklistFile("2026-W44"), "# Hábitos\n## Estudos\n- [ ] 5 min (U) - AWS\n")

	out, err := captureStdout(t, func() error {
		return runWeek(v, weekOptions{DryRun: true}, testNow())
	})
	if err != nil {
		t.Fatalf("runWeek: %v", err)
	}

	if !strings.Contains(out, "- [ ] 5 min (U) - AWS") {
		t.Errorf("dry-run não mostrou o hábito: %q", out)
	}
	assertNotExists(t, v.WeeklistFile("2026-W45"))
}

func TestRunWeekPrintPath(t *testing.T) {
	v := newTestVault(t)
	want := v.WeeklistFile("2026-W45")

	out, err := captureStdout(t, func() error {
		return runWeek(v, weekOptions{PrintPath: true}, testNow())
	})
	if err != nil {
		t.Fatalf("runWeek: %v", err)
	}
	if strings.TrimSpace(out) != want {
		t.Errorf("print-path = %q, want %q", strings.TrimSpace(out), want)
	}
	assertNotExists(t, want)
}

func TestRunWeekExisting(t *testing.T) {
	v := newTestVault(t)
	target := v.WeeklistFile("2026-W45")
	writeTestNote(t, target, "conteúdo existente\n")

	out, err := captureStdout(t, func() error {
		return runWeek(v, weekOptions{}, testNow())
	})
	if err != nil {
		t.Fatalf("runWeek: %v", err)
	}
	if !strings.Contains(out, "Weeklist já existe") {
		t.Errorf("saída não avisa que já existe: %q", out)
	}
	if got := readTestNote(t, target); got != "conteúdo existente\n" {
		t.Errorf("weeklist existente foi alterada: %q", got)
	}
}

func TestRunWeekMissingSource(t *testing.T) {
	v := newTestVault(t)

	if _, err := captureStdout(t, func() error {
		return runWeek(v, weekOptions{}, testNow())
	}); err != nil {
		t.Fatalf("runWeek: %v", err)
	}

	const want = "# Tasks\n\n---\n# Hábitos\n"
	if got := readTestNote(t, v.WeeklistFile("2026-W45")); got != want {
		t.Errorf("weeklist sem origem = %q, want %q", got, want)
	}
}

func TestWeekFlagsRegistered(t *testing.T) {
	for _, name := range []string{"current", "next", "dry-run", "print-path"} {
		if weekCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s não registrada em week", name)
		}
	}
}
