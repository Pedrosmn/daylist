package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"daylist/internal/planner"
	"daylist/internal/vault"
)

func addWeeklist(t *testing.T, v *vault.Vault, week string) string {
	t.Helper()
	path := v.WeeklistFile(week)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte("# "+week+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func TestWeeklistForTargetUsesTargetWeek(t *testing.T) {
	v := newTestVault(t)
	target := time.Date(2026, 11, 2, 0, 0, 0, 0, time.Local) // 2026-W45

	addWeeklist(t, v, "2026-W44")
	want := addWeeklist(t, v, "2026-W45")

	path, week, fallback, err := planner.WeeklistForTarget(v, target)
	if err != nil {
		t.Fatalf("weeklistForTarget: %v", err)
	}
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
	if week != "2026-W45" {
		t.Errorf("week = %q, want 2026-W45", week)
	}
	if fallback {
		t.Error("fallback = true, want false")
	}
}

func TestWeeklistForTargetFallsBackToPreviousWeek(t *testing.T) {
	v := newTestVault(t)
	target := time.Date(2026, 11, 2, 0, 0, 0, 0, time.Local) // 2026-W45

	want := addWeeklist(t, v, "2026-W44")

	path, week, fallback, err := planner.WeeklistForTarget(v, target)
	if err != nil {
		t.Fatalf("weeklistForTarget: %v", err)
	}
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
	if week != "2026-W44" {
		t.Errorf("week = %q, want 2026-W44", week)
	}
	if !fallback {
		t.Error("fallback = false, want true")
	}
}

func TestWeeklistForTargetNone(t *testing.T) {
	v := newTestVault(t)
	target := time.Date(2026, 11, 2, 0, 0, 0, 0, time.Local)

	targetPath := v.WeeklistFile("2026-W45")
	previousPath := v.WeeklistFile("2026-W44")

	path, week, fallback, err := planner.WeeklistForTarget(v, target)
	if err == nil {
		t.Fatal("weeklistForTarget deveria falhar sem nenhuma weeklist")
	}
	if path != "" || week != "" || fallback {
		t.Errorf("retorno inesperado: path=%q week=%q fallback=%v", path, week, fallback)
	}
	for _, want := range []string{"2026-W45", "2026-W44", targetPath, previousPath, "daylist week"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("erro não cita %q: %v", want, err)
		}
	}
}

func TestWeeklistForTargetSundayToMonday(t *testing.T) {
	v := newTestVault(t)
	sunday := time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local)
	monday := time.Date(2026, 11, 2, 0, 0, 0, 0, time.Local)

	if got := vault.ISOWeek(sunday); got != "2026-W44" {
		t.Fatalf("semana do domingo = %q, want 2026-W44", got)
	}
	if got := vault.ISOWeek(monday); got != "2026-W45" {
		t.Fatalf("semana da segunda = %q, want 2026-W45", got)
	}

	want := addWeeklist(t, v, "2026-W45")
	path, week, fallback, err := planner.WeeklistForTarget(v, monday)
	if err != nil {
		t.Fatalf("weeklistForTarget: %v", err)
	}
	if path != want || week != "2026-W45" || fallback {
		t.Errorf("got path=%q week=%q fallback=%v, want %q 2026-W45 false", path, week, fallback, want)
	}
}

func TestWeeklistForTargetYearBoundary(t *testing.T) {
	v := newTestVault(t)
	target := time.Date(2025, 12, 29, 0, 0, 0, 0, time.Local)

	if got := vault.ISOWeek(target); got != "2026-W01" {
		t.Fatalf("semana de 2025-12-29 = %q, want 2026-W01", got)
	}

	want := addWeeklist(t, v, "2026-W01")
	path, week, fallback, err := planner.WeeklistForTarget(v, target)
	if err != nil {
		t.Fatalf("weeklistForTarget: %v", err)
	}
	if path != want || week != "2026-W01" || fallback {
		t.Errorf("got path=%q week=%q fallback=%v, want %q 2026-W01 false", path, week, fallback, want)
	}
}

func resetNewFlags() {
	for _, name := range []string{"today", "tomorrow", "date", "dry-run"} {
		f := newCmd.Flags().Lookup(name)
		if f == nil {
			continue
		}
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	}
	newOpts = newOptions{}
}

func TestResolveTargetDate(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.Local)
	tomorrow := now.AddDate(0, 0, 1)
	pick := func(choice string) func(string, []string) (string, error) {
		return func(string, []string) (string, error) {
			return choice, nil
		}
	}
	pickErr := errors.New("picker falhou")

	cases := []struct {
		name        string
		opts        newOptions
		todayExists bool
		pick        func(string, []string) (string, error)
		want        time.Time
		wantErr     string
	}{
		{
			name: "today",
			opts: newOptions{Today: true},
			want: now,
		},
		{
			name: "tomorrow",
			opts: newOptions{Tomorrow: true},
			want: tomorrow,
		},
		{
			name: "date",
			opts: newOptions{Date: "2026-11-02"},
			want: time.Date(2026, 11, 2, 0, 0, 0, 0, time.Local),
		},
		{
			name:    "invalid date",
			opts:    newOptions{Date: "2026-11-2"},
			wantErr: "data inválida",
		},
		{
			name:        "default with today existing",
			todayExists: true,
			want:        tomorrow,
		},
		{
			name: "pick today",
			pick: pick("Hoje (2026-10-30)"),
			want: now,
		},
		{
			name: "pick tomorrow",
			pick: pick("Amanhã (2026-10-31)"),
			want: tomorrow,
		},
		{
			name: "cancel",
			pick: pick(""),
			want: time.Time{},
		},
		{
			name: "picker error",
			pick: func(string, []string) (string, error) {
				return "", pickErr
			},
			wantErr: "picker falhou",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pickFn := tc.pick
			if pickFn == nil {
				pickFn = func(string, []string) (string, error) {
					t.Fatal("picker não deveria ser chamado")
					return "", nil
				}
			}

			got, err := resolveTargetDate(now, tc.opts, tc.todayExists, pickFn)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("erro = %v, want contendo %q", err, tc.wantErr)
				}
				if !got.IsZero() {
					t.Errorf("data = %v, want zero em caso de erro", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveTargetDate: %v", err)
			}
			if tc.want.IsZero() {
				if !got.IsZero() {
					t.Errorf("data = %v, want zero", got)
				}
				return
			}
			if !got.Equal(tc.want) {
				t.Errorf("data = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResolveTargetDatePrompt(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.Local)
	var gotPrompt string
	var gotOptions []string

	target, err := resolveTargetDate(now, newOptions{}, false, func(prompt string, options []string) (string, error) {
		gotPrompt = prompt
		gotOptions = append([]string(nil), options...)
		return options[0], nil
	})
	if err != nil {
		t.Fatalf("resolveTargetDate: %v", err)
	}

	const wantPrompt = "A daylist de hoje não existe. Criar qual? "
	if gotPrompt != wantPrompt {
		t.Errorf("prompt = %q, want %q", gotPrompt, wantPrompt)
	}
	wantOptions := []string{"Hoje (2026-10-30)", "Amanhã (2026-10-31)"}
	if strings.Join(gotOptions, "|") != strings.Join(wantOptions, "|") {
		t.Errorf("opções = %q, want %q", gotOptions, wantOptions)
	}
	if !target.Equal(now) {
		t.Errorf("data = %v, want %v", target, now)
	}
}

func TestNewFlagsRegistered(t *testing.T) {
	for _, name := range []string{"today", "tomorrow", "date", "dry-run"} {
		if newCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s não registrada em new", name)
		}
	}

	foundAlias := false
	for _, alias := range newCmd.Aliases {
		if alias == "n" {
			foundAlias = true
			break
		}
	}
	if !foundAlias {
		t.Error("alias n não encontrado em new")
	}
}

func TestNewFlagsBindOptions(t *testing.T) {
	resetNewFlags()
	t.Cleanup(resetNewFlags)

	cases := []struct {
		name  string
		value string
		check func() bool
	}{
		{"today", "true", func() bool { return newOpts.Today }},
		{"tomorrow", "true", func() bool { return newOpts.Tomorrow }},
		{"date", "2026-11-02", func() bool { return newOpts.Date == "2026-11-02" }},
		{"dry-run", "true", func() bool { return newOpts.DryRun }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resetNewFlags()
			if err := newCmd.Flags().Set(tc.name, tc.value); err != nil {
				t.Fatalf("Set(%s): %v", tc.name, err)
			}
			if !tc.check() {
				t.Errorf("flag --%s não atualizou as opções", tc.name)
			}
		})
	}
}

func TestNewFlagsMutuallyExclusive(t *testing.T) {
	t.Cleanup(resetNewFlags)

	cases := [][]string{
		{"today", "tomorrow"},
		{"today", "date"},
		{"tomorrow", "date"},
	}
	for _, flags := range cases {
		t.Run(strings.Join(flags, "+"), func(t *testing.T) {
			resetNewFlags()
			for _, name := range flags {
				value := "true"
				if name == "date" {
					value = "2026-11-02"
				}
				if err := newCmd.Flags().Set(name, value); err != nil {
					t.Fatalf("Set(%s): %v", name, err)
				}
			}
			if err := newCmd.ValidateFlagGroups(); err == nil {
				t.Error("flags mutuamente exclusivas não geraram erro")
			}
		})
	}
}

func TestRunNewExistingDaylist(t *testing.T) {
	tv := newTestVault(t)
	oldV := v
	v = tv
	t.Cleanup(func() { v = oldV })

	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.Local)
	path := tv.DaylistFile(now)
	writeTestNote(t, path, "conteúdo existente\n")

	out, err := captureStdout(t, func() error {
		return runNew(tv, newOptions{Today: true}, now, nil)
	})
	if err != nil {
		t.Fatalf("runNew: %v", err)
	}
	if !strings.Contains(out, "já existe") {
		t.Errorf("saída não avisa que a daylist já existe: %q", out)
	}
	if got := readTestNote(t, path); got != "conteúdo existente\n" {
		t.Errorf("daylist existente foi alterada: %q", got)
	}
}

func TestRunNewDefaultUsesTomorrow(t *testing.T) {
	tv := newTestVault(t)
	oldV := v
	v = tv
	t.Cleanup(func() { v = oldV })

	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.Local)
	tomorrow := now.AddDate(0, 0, 1)
	writeTestNote(t, tv.DaylistFile(now), "hoje\n")
	writeTestNote(t, tv.DaylistFile(tomorrow), "amanhã\n")

	out, err := captureStdout(t, func() error {
		return runNew(tv, newOptions{}, now, nil)
	})
	if err != nil {
		t.Fatalf("runNew: %v", err)
	}
	if !strings.Contains(out, "2026-10-31") {
		t.Errorf("saída não se refere a amanhã: %q", out)
	}
}

func TestRunNewCanceled(t *testing.T) {
	tv := newTestVault(t)
	oldV := v
	v = tv
	t.Cleanup(func() { v = oldV })

	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.Local)
	out, err := captureStdout(t, func() error {
		return runNew(tv, newOptions{}, now, func(string, []string) (string, error) {
			return "", nil
		})
	})
	if err != nil {
		t.Fatalf("runNew: %v", err)
	}
	if !strings.Contains(out, "Cancelado.") {
		t.Errorf("saída não confirma cancelamento: %q", out)
	}
	assertNotExists(t, tv.DaylistFile(now))
}
