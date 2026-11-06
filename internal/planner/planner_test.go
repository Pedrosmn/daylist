package planner

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"daylist/internal/ui"
	"daylist/internal/vault"
)

type selectAllPrompter struct{}

func (selectAllPrompter) Pick(string, []string) (string, error) { return "Nenhuma", nil }
func (selectAllPrompter) Toggle(opts ui.ToggleOptions) (ui.ToggleResult, error) {
	selected := make([]int, len(opts.Items))
	for i := range selected {
		selected[i] = i
	}
	return ui.ToggleResult{Selected: selected, Action: ui.ToggleFinish}, nil
}
func (selectAllPrompter) Input(string) (string, error) { return "10", nil }
func (selectAllPrompter) Confirm(string) (bool, error) { return true, nil }

type cancelPrompter struct{ selectAllPrompter }

func (cancelPrompter) Toggle(ui.ToggleOptions) (ui.ToggleResult, error) {
	return ui.ToggleResult{Action: ui.ToggleCancel}, nil
}

func newTestVault(t *testing.T) *vault.Vault {
	t.Helper()
	return &vault.Vault{Name: "teste", Path: t.TempDir()}
}

func writeWeeklist(t *testing.T, v *vault.Vault, week, content string) {
	t.Helper()
	path := v.WeeklistFile(week)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestBuildReturnsPreviewWithoutWriting(t *testing.T) {
	t.Setenv("DAYLIST_DAY_LIMIT", "30 min")

	v := newTestVault(t)
	writeWeeklist(t, v, "2026-W45", `# Tasks
- [ ] 20 min (I) - tarefa
# Hábitos
## Rotina
- [ ] 5 min () - Fio dental #always
`)
	target := time.Date(2026, 11, 5, 0, 0, 0, 0, time.Local)

	plan, err := Build(v, target, selectAllPrompter{}, io.Discard, target)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if len(plan.Resolved) != 2 {
		t.Errorf("resolvidos = %d, want 2", len(plan.Resolved))
	}
	if len(plan.Always) != 1 || plan.Always[0].Desc != "Fio dental" {
		t.Errorf("always = %+v", plan.Always)
	}
	if plan.TotalMin != 25 || plan.TotalMax != 25 || !plan.HasTotal {
		t.Errorf("total = %d–%d (%v), want 25–25", plan.TotalMin, plan.TotalMax, plan.HasTotal)
	}
	if plan.LimitMin != 30 {
		t.Errorf("limite = %d, want 30", plan.LimitMin)
	}
	if plan.OverLimit() {
		t.Error("não deveria passar do limite")
	}
	if !strings.Contains(plan.Content, "- [ ] 20 min (I) - tarefa") ||
		!strings.Contains(plan.Content, "- [ ] 5 min () - Fio dental") {
		t.Errorf("conteúdo inesperado:\n%s", plan.Content)
	}
	if _, err := os.Stat(v.DaylistFile(target)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Build não deveria gravar a daylist: %v", err)
	}

	if err := Write(v, plan); err != nil {
		t.Fatalf("Write: %v", err)
	}
	data, err := os.ReadFile(v.DaylistFile(target))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "Fio dental") {
		t.Errorf("daylist gravada inesperada:\n%s", data)
	}
}

func TestBuildOverLimit(t *testing.T) {
	t.Setenv("DAYLIST_DAY_LIMIT", "20 min")

	v := newTestVault(t)
	writeWeeklist(t, v, "2026-W45", "# Tasks\n- [ ] 25 min (I) - tarefa\n")
	target := time.Date(2026, 11, 5, 0, 0, 0, 0, time.Local)

	plan, err := Build(v, target, selectAllPrompter{}, io.Discard, target)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !plan.OverLimit() {
		t.Errorf("deveria passar do limite: total=%d limite=%d", plan.TotalMax, plan.LimitMin)
	}
}

func TestBuildDaylistAlreadyExists(t *testing.T) {
	v := newTestVault(t)
	target := time.Date(2026, 11, 5, 0, 0, 0, 0, time.Local)
	path := v.DaylistFile(target)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte("existente\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := Build(v, target, selectAllPrompter{}, io.Discard, target)
	if !errors.Is(err, ErrDaylistExists) {
		t.Errorf("err = %v, want ErrDaylistExists", err)
	}
}

func TestBuildCanceled(t *testing.T) {
	v := newTestVault(t)
	writeWeeklist(t, v, "2026-W45", "# Tasks\n- [ ] 10 min (I) - tarefa\n")
	target := time.Date(2026, 11, 5, 0, 0, 0, 0, time.Local)

	_, err := Build(v, target, cancelPrompter{}, io.Discard, target)
	if !errors.Is(err, ErrCanceled) {
		t.Errorf("err = %v, want ErrCanceled", err)
	}
}

func TestBuildNoItems(t *testing.T) {
	v := newTestVault(t)
	writeWeeklist(t, v, "2026-W45", "# Tasks\n\n# Hábitos\n")
	target := time.Date(2026, 11, 5, 0, 0, 0, 0, time.Local)

	_, err := Build(v, target, selectAllPrompter{}, io.Discard, target)
	if !errors.Is(err, ErrNoItems) {
		t.Errorf("err = %v, want ErrNoItems", err)
	}
}

func TestDayLimitMinutes(t *testing.T) {
	cases := []struct {
		name string
		env  string
		want int
	}{
		{"padrão", "", 8 * 60},
		{"horas", "7h30", 7*60 + 30},
		{"minutos", "300 min", 300},
		{"inválido", "muito", 8 * 60},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DAYLIST_DAY_LIMIT", tc.env)
			if got := DayLimitMinutes(); got != tc.want {
				t.Errorf("DayLimitMinutes = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestFormatting(t *testing.T) {
	if got := FormatMinutes(45); got != "45 min" {
		t.Errorf("FormatMinutes(45) = %q", got)
	}
	if got := FormatMinutes(8 * 60); got != "8h" {
		t.Errorf("FormatMinutes(8h) = %q", got)
	}
	if got := FormatMinutes(7*60 + 30); got != "7h30" {
		t.Errorf("FormatMinutes(7h30) = %q", got)
	}
	if got := FormatRange(60, 90); got != "1h–1h30" {
		t.Errorf("FormatRange = %q", got)
	}
}
