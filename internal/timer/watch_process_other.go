//go:build !linux && !darwin && !freebsd && !netbsd && !openbsd && !dragonfly

package timer

import (
	"os/exec"
	"sync"
)

var fallbackLock sync.Mutex

func lockFile(string) (func(), error) {
	fallbackLock.Lock()
	return fallbackLock.Unlock, nil
}

func processAlive(int) bool { return false }

func signalWatchStop(int) {}

func detachProcess(*exec.Cmd) {}
