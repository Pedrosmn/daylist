//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package timer

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSpawnWatchDetached(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "args.txt")
	script := filepath.Join(dir, "watch.sh")
	content := fmt.Sprintf("#!/bin/sh\nprintf '%%s' \"$*\" > '%s'\n", marker)
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile script: %v", err)
	}

	oldExecutable := watchExecutable
	t.Cleanup(func() { watchExecutable = oldExecutable })
	watchExecutable = func() (string, error) { return script, nil }

	statePath := filepath.Join(dir, "state.json")
	pidPath := filepath.Join(dir, "watch.pid")
	if err := os.WriteFile(statePath, []byte("{}"), 0o644); err != nil {
		t.Fatalf("WriteFile estado: %v", err)
	}

	if err := SpawnWatch(statePath, pidPath); err != nil {
		t.Fatalf("SpawnWatch: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		data, err := os.ReadFile(marker)
		if err == nil {
			if got := string(data); got != "timer watch" {
				t.Errorf("argumentos do watch = %q, want %q", got, "timer watch")
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("watch não foi iniciado: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
