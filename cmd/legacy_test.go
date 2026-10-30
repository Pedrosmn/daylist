package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"daylist/internal/vault"
)

func legacyPath(v *vault.Vault, name string) string {
	return filepath.Join(v.Path, name)
}

func TestCleanupLegacyTargets(t *testing.T) {
	v := newTestVault(t)
	for _, name := range []string{legacyWeeklistTarget, legacyFeedbackTarget} {
		writeTestNote(t, filepath.Join(v.Path, name), "{}")
	}

	cleanupLegacyTargets(v)

	for _, name := range []string{legacyWeeklistTarget, legacyFeedbackTarget} {
		if _, err := os.Stat(filepath.Join(v.Path, name)); !os.IsNotExist(err) {
			t.Errorf("%s não foi removido (err = %v)", name, err)
		}
	}
}

func TestCleanupLegacyTargetsIgnoresMissing(t *testing.T) {
	cleanupLegacyTargets(newTestVault(t))
}
