package cmd

import (
	"os"
	"path/filepath"

	"daylist/internal/vault"
)

const (
	legacyWeeklistTarget = ".daylist-weeklist-target.json"
	legacyFeedbackTarget = ".daylist-feedback-target.json"
)

// cleanupLegacyTargets remove os arquivos usados pelo fluxo antigo do
// QuickAdd. Erros são ignorados: são apenas resíduos de migração.
func cleanupLegacyTargets(v *vault.Vault) {
	for _, name := range []string{legacyWeeklistTarget, legacyFeedbackTarget} {
		_ = os.Remove(filepath.Join(v.Path, name))
	}
}
