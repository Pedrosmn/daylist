// Package vault concentra tudo que sabe onde as coisas ficam no Obsidian.
package vault

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	DaysDir      = "Fleeting Notes/Daily/Days"
	WeeklistsDir = "Fleeting Notes/Daily/weeklists"
)

type Vault struct {
	Name string // nome do vault no Obsidian (usado nas URLs)
	Path string // caminho absoluto no disco
}

// Load usa os valores padrão, que podem ser sobrescritos por
// DAYLIST_VAULT_NAME e DAYLIST_VAULT_PATH.
func Load() (*Vault, error) {
	name := os.Getenv("DAYLIST_VAULT_NAME")
	if name == "" {
		name = "Vault 33"
	}

	path := os.Getenv("DAYLIST_VAULT_PATH")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, "documents", "Obsidian", "Vault 33")
	}

	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) || (err == nil && !info.IsDir()) {
		return nil, fmt.Errorf("vault não encontrado em %q (ajuste DAYLIST_VAULT_PATH)", path)
	}
	if err != nil {
		return nil, err
	}

	return &Vault{Name: name, Path: path}, nil
}

// ISOWeek devolve a semana no formato 2026-W43 (equivale a date +%G-W%V).
func ISOWeek(t time.Time) string {
	y, w := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", y, w)
}

// --- caminhos ---

// DaylistRel é o caminho relativo ao vault (usado no campo "file" do estado).
func (v *Vault) DaylistRel(t time.Time) string {
	return DaysDir + "/" + t.Format("2006-01-02") + ".md"
}

func (v *Vault) DaylistFile(t time.Time) string {
	return filepath.Join(v.Path, filepath.FromSlash(v.DaylistRel(t)))
}

func (v *Vault) WeeklistFile(week string) string {
	return filepath.Join(v.Path, filepath.FromSlash(WeeklistsDir), week, "weeklist.md")
}

func (v *Vault) FeedbackFile(week string) string {
	return filepath.Join(v.Path, filepath.FromSlash(WeeklistsDir), week, "feedback.md")
}

func (v *Vault) StateFile() string {
	return filepath.Join(v.Path, ".daylist-timer-state.json")
}

// --- Obsidian ---

// OpenNote abre a daylist do dia no Obsidian.
func (v *Vault) OpenNote(t time.Time) error {
	rel := strings.TrimSuffix(v.DaylistRel(t), ".md")
	parts := strings.Split(rel, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p) // "Fleeting Notes" -> "Fleeting%20Notes"
	}

	return openURL(fmt.Sprintf(
		"obsidian://open?vault=%s&file=%s",
		url.PathEscape(v.Name),
		strings.Join(parts, "/"),
	))
}

func openURL(u string) error {
	cmd := exec.Command("xdg-open", u)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release() // não espera o processo, igual ao "& disown"
}
