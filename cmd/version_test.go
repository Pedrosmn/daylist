package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"daylist/internal/version"
)

func setTestVersion(t *testing.T) {
	t.Helper()

	oldVersion, oldCommit, oldDate := version.Version, version.Commit, version.Date
	oldRootVersion := rootCmd.Version
	t.Cleanup(func() {
		version.Version, version.Commit, version.Date = oldVersion, oldCommit, oldDate
		rootCmd.Version = oldRootVersion
	})

	version.Version = "1.2.3"
	version.Commit = "abc1234"
	version.Date = "2026-11-02"
	rootCmd.Version = version.String()
}

func TestVersionCommand(t *testing.T) {
	setTestVersion(t)

	out, err := captureStdout(t, func() error {
		versionCmd.Run(versionCmd, nil)
		return nil
	})
	if err != nil {
		t.Fatalf("version: %v", err)
	}

	const want = "daylist version 1.2.3 (commit abc1234, 2026-11-02)\n"
	if out != want {
		t.Errorf("saída = %q, want %q", out, want)
	}
}

func TestRootVersionFlag(t *testing.T) {
	setTestVersion(t)
	t.Setenv("DAYLIST_VAULT_PATH", filepath.Join(t.TempDir(), "vault-inexistente"))
	rootCmd.SetArgs([]string{"--version"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	out, err := captureStdout(t, func() error {
		return rootCmd.Execute()
	})
	if err != nil {
		t.Fatalf("--version: %v", err)
	}
	if !strings.Contains(out, "daylist version 1.2.3") {
		t.Errorf("saída não contém a versão: %q", out)
	}
}

func TestVersionDoesNotLoadVault(t *testing.T) {
	setTestVersion(t)
	t.Setenv("DAYLIST_VAULT_PATH", filepath.Join(t.TempDir(), "vault-inexistente"))
	rootCmd.SetArgs([]string{"version"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	out, err := captureStdout(t, func() error {
		return rootCmd.Execute()
	})
	if err != nil {
		t.Fatalf("version não deveria exigir vault: %v", err)
	}
	if !strings.Contains(out, "daylist version 1.2.3") {
		t.Errorf("saída não contém a versão: %q", out)
	}
}
