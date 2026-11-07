package timer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeWatchClock struct {
	now     time.Time
	sleeps  int
	onSleep func(time.Time)
}

func (c *fakeWatchClock) Now() time.Time { return c.now }

func (c *fakeWatchClock) Sleep(d time.Duration) {
	c.sleeps++
	if c.sleeps > 10_000 {
		panic("watch não terminou")
	}
	c.now = c.now.Add(d)
	if c.onSleep != nil {
		c.onSleep(c.now)
	}
}

func testWatchTimer(desc string, base time.Time, min, max int) *State {
	return &State{
		Desc:       desc,
		Start:      base.UnixMilli(),
		Started:    base.UnixMilli(),
		MinMinutes: min,
		MaxMinutes: max,
	}
}

func TestWatchNotifiesMinAndMaxOnce(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	pidPath := filepath.Join(dir, "watch.pid")
	base := time.Date(2026, 11, 7, 9, 0, 0, 0, time.Local)

	if err := testWatchTimer("tarefa", base, 5, 15).Save(statePath); err != nil {
		t.Fatalf("Save: %v", err)
	}

	clock := &fakeWatchClock{now: base}
	var calls []time.Time
	var bodies []string
	opts := WatchOptions{
		StatePath: statePath,
		PIDPath:   pidPath,
		Notify: func(title, body string) error {
			if title != "Daylist" {
				t.Errorf("título = %q, want Daylist", title)
			}
			calls = append(calls, clock.now)
			bodies = append(bodies, body)
			return nil
		},
		Now:   clock.Now,
		Sleep: clock.Sleep,
		Poll:  time.Minute,
	}

	if err := Watch(opts); err != nil {
		t.Fatalf("Watch: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("notificações = %v, want mínimo e máximo", bodies)
	}
	if !calls[0].Equal(base.Add(5*time.Minute)) || !strings.Contains(bodies[0], "mínimo atingido") {
		t.Errorf("primeira notificação = %v / %q", calls[0], bodies[0])
	}
	if !calls[1].Equal(base.Add(15*time.Minute)) || !strings.Contains(bodies[1], "máximo atingido") {
		t.Errorf("segunda notificação = %v / %q", calls[1], bodies[1])
	}

	loaded, err := Load(statePath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !loaded.MinNotified || !loaded.MaxNotified {
		t.Errorf("sinalizadores não persistidos: %+v", loaded)
	}
	if _, err := os.Stat(pidPath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("arquivo de PID não removido: err = %v", err)
	}

	// Reiniciar o watch com o mesmo estado não repete as notificações.
	if err := Watch(opts); err != nil {
		t.Fatalf("Watch repetido: %v", err)
	}
	if len(calls) != 2 {
		t.Errorf("watch repetido notificou de novo: %v", bodies)
	}

	// Um novo timer zera os sinalizadores e volta a notificar.
	base2 := base.Add(time.Hour)
	if err := testWatchTimer("outra tarefa", base2, 5, 15).Save(statePath); err != nil {
		t.Fatalf("Save novo timer: %v", err)
	}
	clock.now = base2
	if err := Watch(opts); err != nil {
		t.Fatalf("Watch novo timer: %v", err)
	}
	if len(calls) != 4 {
		t.Fatalf("notificações após reinício = %v, want 4", bodies)
	}
	if !calls[2].Equal(base2.Add(5*time.Minute)) || !calls[3].Equal(base2.Add(15*time.Minute)) {
		t.Errorf("notificações do novo timer = %v", calls[2:])
	}
}

func TestWatchPausedDoesNotNotifyUntilResume(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	pidPath := filepath.Join(dir, "watch.pid")
	base := time.Date(2026, 11, 7, 9, 0, 0, 0, time.Local)

	if err := testWatchTimer("tarefa", base, 5, 15).Save(statePath); err != nil {
		t.Fatalf("Save: %v", err)
	}

	paused := false
	pauseDone := false
	resumeDone := false
	clock := &fakeWatchClock{now: base}
	clock.onSleep = func(now time.Time) {
		switch {
		case !pauseDone && !now.Before(base.Add(2*time.Minute)):
			s, err := Load(statePath)
			if err != nil {
				t.Fatalf("Load para pausar: %v", err)
			}
			s.Paused = true
			s.Elapsed = now.Sub(base).Milliseconds()
			s.PausedAt = now.UnixMilli()
			if err := s.Save(statePath); err != nil {
				t.Fatalf("Save pausado: %v", err)
			}
			paused = true
			pauseDone = true
		case !resumeDone && paused && !now.Before(base.Add(10*time.Minute)):
			s, err := Load(statePath)
			if err != nil {
				t.Fatalf("Load para retomar: %v", err)
			}
			s.Start = now.UnixMilli()
			s.Paused = false
			s.PausedAt = 0
			if err := s.Save(statePath); err != nil {
				t.Fatalf("Save retomado: %v", err)
			}
			paused = false
			resumeDone = true
		}
	}

	var calls []time.Time
	opts := WatchOptions{
		StatePath: statePath,
		PIDPath:   pidPath,
		Notify: func(title, body string) error {
			calls = append(calls, clock.now)
			return nil
		},
		Now:   clock.Now,
		Sleep: clock.Sleep,
		Poll:  time.Minute,
	}

	if err := Watch(opts); err != nil {
		t.Fatalf("Watch: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("notificações = %v, want mínimo e máximo depois da retomada", calls)
	}
	if !calls[0].Equal(base.Add(13 * time.Minute)) {
		t.Errorf("mínimo notificado em %v, want %v", calls[0], base.Add(13*time.Minute))
	}
	if !calls[1].Equal(base.Add(23 * time.Minute)) {
		t.Errorf("máximo notificado em %v, want %v", calls[1], base.Add(23*time.Minute))
	}
}

func TestWatchNoTimer(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "missing.json")
	pidPath := filepath.Join(dir, "watch.pid")
	clock := &fakeWatchClock{now: time.Now()}

	called := false
	err := Watch(WatchOptions{
		StatePath: statePath,
		PIDPath:   pidPath,
		Notify: func(string, string) error {
			called = true
			return nil
		},
		Now:   clock.Now,
		Sleep: clock.Sleep,
	})
	if err != nil {
		t.Fatalf("Watch sem timer: %v", err)
	}
	if called {
		t.Error("não deveria notificar sem timer")
	}
	if _, err := os.Stat(pidPath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("arquivo de PID não removido: err = %v", err)
	}
}

func TestWatchWithoutRangeExits(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	pidPath := filepath.Join(dir, "watch.pid")
	base := time.Now()

	s := &State{Desc: "sem faixa", Start: base.UnixMilli()}
	if err := s.Save(statePath); err != nil {
		t.Fatalf("Save: %v", err)
	}
	clock := &fakeWatchClock{now: base}
	if err := Watch(WatchOptions{
		StatePath: statePath,
		PIDPath:   pidPath,
		Notify:    func(string, string) error { return nil },
		Now:       clock.Now,
		Sleep:     clock.Sleep,
		Poll:      time.Minute,
	}); err != nil {
		t.Fatalf("Watch sem faixa: %v", err)
	}
}

func TestWatchSingleInstanceAndStalePID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "watch.pid")

	release, already, err := acquireWatchLock(path)
	if err != nil || already {
		t.Fatalf("primeiro lock: already=%v err=%v", already, err)
	}
	if _, already, err := acquireWatchLock(path); err != nil || !already {
		t.Fatalf("segundo lock: already=%v err=%v, want already", already, err)
	}
	release()

	release2, already, err := acquireWatchLock(path)
	if err != nil || already {
		t.Fatalf("lock após release: already=%v err=%v", already, err)
	}
	release2()

	if err := os.WriteFile(path, []byte("99999999\n"), 0o644); err != nil {
		t.Fatalf("WriteFile PID velho: %v", err)
	}
	release3, already, err := acquireWatchLock(path)
	if err != nil || already {
		t.Fatalf("lock com PID velho: already=%v err=%v", already, err)
	}
	release3()
}

func TestClaimThresholdsDeduplicates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	now := time.Date(2026, 11, 7, 9, 0, 0, 0, time.Local)
	s := testWatchTimer("tarefa", now.Add(-5*time.Minute), 5, 15)
	if err := s.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	first, err := ClaimThresholds(path, now)
	if err != nil {
		t.Fatalf("ClaimThresholds: %v", err)
	}
	if !first.Changed || !strings.Contains(first.Message, "mínimo atingido") {
		t.Fatalf("primeiro claim = %+v", first)
	}

	second, err := ClaimThresholds(path, now)
	if err != nil {
		t.Fatalf("ClaimThresholds repetido: %v", err)
	}
	if second.Changed {
		t.Errorf("segundo claim duplicou a notificação: %+v", second)
	}
}

func TestClaimThresholdsPausedDoesNotChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	now := time.Now()
	s := &State{
		Desc:       "tarefa",
		Paused:     true,
		Elapsed:    (5 * time.Minute).Milliseconds(),
		MinMinutes: 5,
		MaxMinutes: 15,
	}
	if err := s.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	update, err := ClaimThresholds(path, now)
	if err != nil {
		t.Fatalf("ClaimThresholds: %v", err)
	}
	if update.Changed {
		t.Errorf("timer pausado não deveria notificar: %+v", update)
	}
}

func TestStopWatchRemovesStalePID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "watch.pid")
	if err := os.WriteFile(path, []byte("99999999\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	StopWatch(path)
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("arquivo de PID não removido: err = %v", err)
	}
}

func TestClearStateRemovesStateAndPID(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	pidPath := filepath.Join(dir, "watch.pid")
	if err := testWatchTimer("tarefa", time.Now(), 5, 15).Save(statePath); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := os.WriteFile(pidPath, []byte("99999999\n"), 0o644); err != nil {
		t.Fatalf("WriteFile PID: %v", err)
	}

	if err := ClearState(statePath, pidPath); err != nil {
		t.Fatalf("ClearState: %v", err)
	}
	for _, path := range []string{statePath, pidPath} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s não foi removido: err = %v", path, err)
		}
	}
}

func TestSpawnWatchNoState(t *testing.T) {
	dir := t.TempDir()
	err := SpawnWatch(filepath.Join(dir, "missing.json"), filepath.Join(dir, "watch.pid"))
	if !errors.Is(err, ErrNoTimer) {
		t.Errorf("SpawnWatch = %v, want ErrNoTimer", err)
	}
}

func TestSpawnWatchAlreadyRunning(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	pidPath := filepath.Join(dir, "watch.pid")
	if err := os.WriteFile(statePath, []byte("{}"), 0o644); err != nil {
		t.Fatalf("WriteFile estado: %v", err)
	}
	if err := os.WriteFile(pidPath, []byte(fmt.Sprintf("%d\n", os.Getpid())), 0o644); err != nil {
		t.Fatalf("WriteFile PID: %v", err)
	}
	if err := SpawnWatch(statePath, pidPath); err != nil {
		t.Errorf("SpawnWatch com watch vivo: %v", err)
	}
}
