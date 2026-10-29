package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

	path, week, fallback, err := weeklistForTarget(v, target)
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

	path, week, fallback, err := weeklistForTarget(v, target)
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

	path, week, fallback, err := weeklistForTarget(v, target)
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
	path, week, fallback, err := weeklistForTarget(v, monday)
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
	path, week, fallback, err := weeklistForTarget(v, target)
	if err != nil {
		t.Fatalf("weeklistForTarget: %v", err)
	}
	if path != want || week != "2026-W01" || fallback {
		t.Errorf("got path=%q week=%q fallback=%v, want %q 2026-W01 false", path, week, fallback, want)
	}
}
