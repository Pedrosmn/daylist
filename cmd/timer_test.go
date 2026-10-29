package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"daylist/internal/timer"
	"daylist/internal/vault"
)

func newTestVault(t *testing.T) *vault.Vault {
	t.Helper()
	return &vault.Vault{Name: "teste", Path: t.TempDir()}
}

func TestTimerDaylistPath(t *testing.T) {
	v := &vault.Vault{Path: filepath.Join(string(filepath.Separator), "vault")}
	s := &timer.State{File: "Fleeting Notes/Daily/Days/2026-10-29.md"}

	want := filepath.Join(v.Path, "Fleeting Notes", "Daily", "Days", "2026-10-29.md")
	if got := timerDaylistPath(v, s); got != want {
		t.Fatalf("timerDaylistPath = %q, want %q", got, want)
	}
}

func TestTimerStopUsesStateDaylist(t *testing.T) {
	v := newTestVault(t)

	// Uma daylist que não é a de hoje: o stop deve gravar nela mesmo assim.
	rel := vault.DaysDir + "/2020-01-02.md"
	path := filepath.Join(v.Path, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	line := "- [ ] 30min (I) - tarefa de outro dia"
	if err := os.WriteFile(path, []byte("# 2020-01-02\n\n"+line+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	s := &timer.State{
		File:     rel,
		Line:     line,
		Desc:     "tarefa de outro dia",
		Started:  1_577_934_000_000,
		Elapsed:  (5 * time.Minute).Milliseconds(),
		Paused:   true,
		PausedAt: 1_577_934_300_000,
	}
	if err := s.Save(v.StateFile()); err != nil {
		t.Fatalf("Save do estado: %v", err)
	}

	if err := stopTimer(v); err != nil {
		t.Fatalf("stopTimer: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile da daylist: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "- [x]") || !strings.Contains(content, " ⏱ 5min (") {
		t.Fatalf("tempo não foi gravado na daylist do estado:\n%s", content)
	}

	if _, err := os.Stat(v.StateFile()); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("estado do timer não foi removido após sucesso: err = %v", err)
	}
	if _, err := os.Stat(v.DaylistFile(time.Now())); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("daylist de hoje foi criada/alterada indevidamente: err = %v", err)
	}
}

func TestTimerStopMissingDaylistKeepsState(t *testing.T) {
	v := newTestVault(t)

	rel := vault.DaysDir + "/2020-01-02.md"
	path := filepath.Join(v.Path, filepath.FromSlash(rel))
	s := &timer.State{
		File:     rel,
		Line:     "- [ ] 30min (I) - tarefa de outro dia",
		Desc:     "tarefa de outro dia",
		Started:  1_577_934_000_000,
		Elapsed:  (5 * time.Minute).Milliseconds(),
		Paused:   true,
		PausedAt: 1_577_934_300_000,
	}
	if err := s.Save(v.StateFile()); err != nil {
		t.Fatalf("Save do estado: %v", err)
	}

	err := stopTimer(v)
	if err == nil {
		t.Fatal("stopTimer deveria falhar quando a daylist do timer não existe")
	}
	if !strings.Contains(err.Error(), "daylist do timer não encontrada") || !strings.Contains(err.Error(), path) {
		t.Fatalf("erro sem a mensagem/caminho esperados: %v", err)
	}

	got, loadErr := timer.Load(v.StateFile())
	if loadErr != nil {
		t.Fatalf("estado foi removido ou corrompido após a falha: %v", loadErr)
	}
	if !reflect.DeepEqual(got, s) {
		t.Errorf("estado mudou após a falha: got %+v, want %+v", got, s)
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("daylist ausente foi criada indevidamente: err = %v", statErr)
	}
}

func TestStatusTextRunning(t *testing.T) {
	s := &timer.State{
		Desc:    "escrever testes",
		Start:   time.Now().Add(-25 * time.Minute).UnixMilli(),
		Started: time.Date(2026, 10, 29, 9, 12, 0, 0, time.Local).UnixMilli(),
	}

	want := "Timer rodando: escrever testes — 25min (iniciado às 09:12)"
	if got := statusText(s); got != want {
		t.Errorf("statusText = %q, want %q", got, want)
	}
}

func TestStatusTextPaused(t *testing.T) {
	s := &timer.State{
		Desc:    "escrever testes",
		Paused:  true,
		Elapsed: (25 * time.Minute).Milliseconds(),
		Started: time.Date(2026, 10, 29, 9, 12, 0, 0, time.Local).UnixMilli(),
	}

	want := "Timer pausado: escrever testes — 25min acumulados (iniciado às 09:12)"
	if got := statusText(s); got != want {
		t.Errorf("statusText = %q, want %q", got, want)
	}
}

func TestStatusTextWithoutStarted(t *testing.T) {
	s := &timer.State{
		Desc:    "escrever testes",
		Paused:  true,
		Elapsed: (5 * time.Minute).Milliseconds(),
	}

	want := "Timer pausado: escrever testes — 5min acumulados"
	if got := statusText(s); got != want {
		t.Errorf("statusText = %q, want %q", got, want)
	}
}

func TestStatusTimerNoState(t *testing.T) {
	v := newTestVault(t)
	if err := statusTimer(v); err != nil {
		t.Fatalf("statusTimer sem estado: %v", err)
	}
}

func TestStatusTimerCorruptState(t *testing.T) {
	v := newTestVault(t)
	if err := os.WriteFile(v.StateFile(), []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err := statusTimer(v)
	if err == nil || !strings.Contains(err.Error(), "estado do timer inválido") {
		t.Fatalf("erro = %v, want estado do timer inválido", err)
	}
}

func TestTimerParentCommandShowsStatus(t *testing.T) {
	old := v
	t.Cleanup(func() { v = old })
	v = newTestVault(t)

	s := &timer.State{
		Desc:     "escrever testes",
		Paused:   true,
		Elapsed:  (5 * time.Minute).Milliseconds(),
		Started:  1_699_999_000_000,
		PausedAt: 1_700_000_000_000,
	}
	if err := s.Save(v.StateFile()); err != nil {
		t.Fatalf("Save do estado: %v", err)
	}

	if err := timerCmd.RunE(timerCmd, nil); err != nil {
		t.Fatalf("timerCmd.RunE: %v", err)
	}
}
