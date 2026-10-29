package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadRejectsMissingVault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nao-existe")
	t.Setenv("DAYLIST_VAULT_PATH", path)

	v, err := Load()
	if err == nil {
		t.Fatal("Load deveria rejeitar um vault inexistente")
	}
	if v != nil {
		t.Errorf("Load devolveu vault com erro: %+v", v)
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("erro não cita o caminho %q: %v", path, err)
	}
	if !strings.Contains(err.Error(), "DAYLIST_VAULT_PATH") {
		t.Errorf("erro não cita DAYLIST_VAULT_PATH: %v", err)
	}
}

func TestLoadRejectsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "arquivo")
	if err := os.WriteFile(path, []byte("não sou um vault"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	t.Setenv("DAYLIST_VAULT_PATH", path)

	v, err := Load()
	if err == nil {
		t.Fatal("Load deveria rejeitar um caminho que é arquivo")
	}
	if v != nil {
		t.Errorf("Load devolveu vault com erro: %+v", v)
	}
}

func TestLoadAcceptsDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DAYLIST_VAULT_PATH", dir)

	v, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if v.Path != dir {
		t.Errorf("Path = %q, want %q", v.Path, dir)
	}
}

func TestISOWeek(t *testing.T) {
	// 2025-12-29 é segunda-feira da semana ISO 2026-W01.
	got := ISOWeek(time.Date(2025, 12, 29, 0, 0, 0, 0, time.Local))
	if got != "2026-W01" {
		t.Errorf("ISOWeek(2025-12-29) = %q, want 2026-W01", got)
	}
}

func TestWeeklistFile(t *testing.T) {
	v := &Vault{Path: filepath.Join(t.TempDir(), "vault")}

	want := filepath.Join(v.Path, filepath.FromSlash(WeeklistsDir), "2026-W01", "weeklist.md")
	if got := v.WeeklistFile("2026-W01"); got != want {
		t.Errorf("WeeklistFile = %q, want %q", got, want)
	}
}
