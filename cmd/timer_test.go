package cmd

import (
	"errors"
	"io"
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
	if err := statusTimer(v, false); err != nil {
		t.Fatalf("statusTimer sem estado: %v", err)
	}
}

func TestStatusTimerCorruptState(t *testing.T) {
	v := newTestVault(t)
	if err := os.WriteFile(v.StateFile(), []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err := statusTimer(v, false)
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

func TestPromptTextRunning(t *testing.T) {
	s := &timer.State{
		Start: time.Now().Add(-25 * time.Minute).UnixMilli(),
	}

	if got, want := promptText(s), "⏱ 25m"; got != want {
		t.Errorf("promptText = %q, want %q", got, want)
	}
}

func TestPromptTextPaused(t *testing.T) {
	s := &timer.State{
		Paused:  true,
		Elapsed: (25 * time.Minute).Milliseconds(),
	}

	if got, want := promptText(s), "⏱ 25m ⏸"; got != want {
		t.Errorf("promptText = %q, want %q", got, want)
	}
}

func TestPromptTextZero(t *testing.T) {
	running := &timer.State{Start: time.Now().UnixMilli()}
	if got, want := promptText(running), "⏱ 0m"; got != want {
		t.Errorf("promptText rodando = %q, want %q", got, want)
	}

	paused := &timer.State{Paused: true}
	if got, want := promptText(paused), "⏱ 0m ⏸"; got != want {
		t.Errorf("promptText pausado = %q, want %q", got, want)
	}
}

func TestStatusTimerPromptNoState(t *testing.T) {
	v := newTestVault(t)

	out, err := captureStdout(t, func() error { return statusTimer(v, true) })
	if err != nil {
		t.Fatalf("statusTimer: %v", err)
	}
	if out != "" {
		t.Errorf("saída = %q, want vazia", out)
	}
}

func TestStatusTimerPromptRunning(t *testing.T) {
	v := newTestVault(t)
	s := &timer.State{
		Desc:  "escrever testes",
		Start: time.Now().Add(-25 * time.Minute).UnixMilli(),
	}
	if err := s.Save(v.StateFile()); err != nil {
		t.Fatalf("Save do estado: %v", err)
	}

	out, err := captureStdout(t, func() error { return statusTimer(v, true) })
	if err != nil {
		t.Fatalf("statusTimer: %v", err)
	}
	if out != "⏱ 25m" {
		t.Errorf("saída = %q, want %q", out, "⏱ 25m")
	}
}

func TestStatusPromptFlagRegistered(t *testing.T) {
	if timerStatusCmd.Flags().Lookup("prompt") == nil {
		t.Fatal("flag --prompt não registrada em timer status")
	}
}

// captureStdout executa fn trocando os.Stdout por um pipe e devolve a saída.
func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = old })

	callErr := fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close do pipe: %v", err)
	}
	data, readErr := io.ReadAll(r)
	if readErr != nil {
		t.Fatalf("leitura do pipe: %v", readErr)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("close da leitura: %v", err)
	}
	return string(data), callErr
}
