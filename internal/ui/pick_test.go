package ui

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// setStdin troca o reader compartilhado por um conteúdo fixo e devolve a
// função que restaura o original.
func setStdin(t *testing.T, content string) {
	t.Helper()
	old := stdin
	stdin = bufio.NewReader(strings.NewReader(content))
	t.Cleanup(func() { stdin = old })
}

func TestPickPlain(t *testing.T) {
	setStdin(t, "2\n")
	got, err := pickPlain("escolha: ", []string{"a", "b", "c"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "b" {
		t.Errorf("got = %q, want %q", got, "b")
	}
}

func TestPickPlainInvalidIsCancel(t *testing.T) {
	setStdin(t, "9\n")
	got, err := pickPlain("escolha: ", []string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("got = %q, want vazio", got)
	}
}

func TestTogglePlainTogglesAndFinishes(t *testing.T) {
	setStdin(t, "2\n3\n\n")
	got, err := togglePlain("escolha: ", "Concluir", []string{"a", "b", "c"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Errorf("got = %v, want [1 2]", got)
	}
}

func TestTogglePlainUntoggles(t *testing.T) {
	setStdin(t, "2\n2\n\n")
	got, err := togglePlain("escolha: ", "Concluir", []string{"a", "b", "c"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("got = %v, want vazio", got)
	}
}

func TestTogglePlainFinishRow(t *testing.T) {
	setStdin(t, "0\n")
	got, err := togglePlain("escolha: ", "Concluir", []string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("got = %v, want vazio", got)
	}
}

func TestTogglePlainEOFKeepsMarks(t *testing.T) {
	setStdin(t, "2\n")
	got, err := togglePlain("escolha: ", "Concluir", []string{"a", "b", "c"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != 1 {
		t.Errorf("got = %v, want [1]", got)
	}
}

func TestTogglePlainIgnoresInvalid(t *testing.T) {
	setStdin(t, "x\n9\n2\n\n")
	got, err := togglePlain("escolha: ", "Concluir", []string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != 1 {
		t.Errorf("got = %v, want [1]", got)
	}
}

func TestInputTrims(t *testing.T) {
	setStdin(t, "  45  \n")
	got, err := Input("min: ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "45" {
		t.Errorf("got = %q, want %q", got, "45")
	}
}

// fakeFzf instala um fzf falso no PATH. Com exitCode 0 ele devolve a primeira
// linha da entrada; com outro código apenas sai com esse código.
func fakeFzf(t *testing.T, exitCode int) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("teste usa um script /bin/sh")
	}

	dir := t.TempDir()
	argsFile := filepath.Join(dir, "fzf-args")
	script := filepath.Join(dir, "fzf")

	content := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$FZF_FAKE_ARGS\"\n"
	if exitCode == 0 {
		content += "IFS= read -r line\nprintf '%s\\n' \"$line\"\n"
	} else {
		content += fmt.Sprintf("exit %d\n", exitCode)
	}
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile do fzf falso: %v", err)
	}

	t.Setenv("FZF_FAKE_ARGS", argsFile)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return argsFile
}

func readFzfArgs(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile dos args: %v", err)
	}
	return strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
}

func hasArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

func TestPickFzfLayoutFlags(t *testing.T) {
	argsFile := fakeFzf(t, 0)

	got, err := Pick("escolha: ", []string{"a", "b"})
	if err != nil {
		t.Fatalf("Pick: %v", err)
	}
	if got != "a" {
		t.Fatalf("Pick = %q, want %q", got, "a")
	}

	args := readFzfArgs(t, argsFile)
	for _, flag := range []string{"--height=70%", "--border", "--info=inline", "--prompt=escolha: "} {
		if !hasArg(args, flag) {
			t.Errorf("flag %q ausente: %v", flag, args)
		}
	}
}

func TestToggleFzfLayoutFlags(t *testing.T) {
	argsFile := fakeFzf(t, 0)

	got, err := Toggle("escolha: ", "Concluir", []string{"a", "b"})
	if err != nil {
		t.Fatalf("Toggle: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Toggle = %v, want vazio", got)
	}

	args := readFzfArgs(t, argsFile)
	for _, flag := range []string{
		"--height=70%",
		"--border",
		"--info=inline",
		"--delimiter=\t",
		"--with-nth=2..",
	} {
		if !hasArg(args, flag) {
			t.Errorf("flag %q ausente: %v", flag, args)
		}
	}
	if !hasArg(args, "--bind=start:pos(2)") {
		t.Errorf("flag --bind ausente: %v", args)
	}
}

func TestPickFzfUsageError(t *testing.T) {
	fakeFzf(t, 2)

	if _, err := Pick("escolha: ", []string{"a", "b"}); err == nil {
		t.Fatal("Pick deveria propagar o erro de uso do fzf")
	}
}

func TestToggleFzfUsageError(t *testing.T) {
	fakeFzf(t, 2)

	_, err := Toggle("escolha: ", "Concluir", []string{"a", "b"})
	if err == nil {
		t.Fatal("Toggle deveria propagar o erro de uso do fzf")
	}
	if errors.Is(err, ErrCanceled) {
		t.Fatalf("erro de uso não pode virar ErrCanceled: %v", err)
	}
}
