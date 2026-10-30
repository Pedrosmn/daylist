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

func TestTogglePlainBackAndCancel(t *testing.T) {
	setStdin(t, "h\n")
	res, err := togglePlainOptions(ToggleOptions{
		Prompt:    "escolha: ",
		Items:     []string{"a", "b"},
		AllowBack: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Action != ToggleBack {
		t.Errorf("action = %v, want ToggleBack", res.Action)
	}

	setStdin(t, "q\n")
	res, err = togglePlainOptions(ToggleOptions{
		Prompt: "escolha: ",
		Items:  []string{"a", "b"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Action != ToggleCancel {
		t.Errorf("action = %v, want ToggleCancel", res.Action)
	}
}

func TestTogglePlainStartsWithSelection(t *testing.T) {
	setStdin(t, "l\n")
	res, err := togglePlainOptions(ToggleOptions{
		Prompt:   "escolha: ",
		Items:    []string{"a", "b", "c"},
		Selected: []int{2},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Selected) != 1 || res.Selected[0] != 2 {
		t.Errorf("selected = %v, want [2]", res.Selected)
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

// fakeFzfReply instala um fzf falso que devolve uma resposta fixa.
func fakeFzfReply(t *testing.T, reply string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("teste usa um script /bin/sh")
	}

	dir := t.TempDir()
	argsFile := filepath.Join(dir, "fzf-args")
	script := filepath.Join(dir, "fzf")

	content := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$FZF_FAKE_ARGS\"\nprintf '%s\\n' \"$FZF_FAKE_REPLY\"\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile do fzf falso: %v", err)
	}

	t.Setenv("FZF_FAKE_ARGS", argsFile)
	t.Setenv("FZF_FAKE_REPLY", reply)
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
	for _, flag := range []string{
		"--height=70%",
		"--border",
		"--info=inline",
		"--layout=reverse",
		"--disabled",
		"--prompt=escolha: ",
	} {
		if !hasArg(args, flag) {
			t.Errorf("flag %q ausente: %v", flag, args)
		}
	}
	for _, arg := range args {
		if strings.Contains(arg, "j:down,k:up") {
			return
		}
	}
	t.Errorf("binds Vim ausentes: %v", args)
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
		"--layout=reverse",
		"--disabled",
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

func TestToggleWithOptionsBackAndInitialSelection(t *testing.T) {
	argsFile := fakeFzfReply(t, "-2\tvoltar")

	res, err := ToggleWithOptions(ToggleOptions{
		Prompt:      "escolha: ",
		FinishLabel: "Concluir",
		Items:       []string{"a", "b", "c"},
		Selected:    []int{1},
		AllowBack:   true,
		Header:      "Etapa 2/3 · Estudos",
	})
	if err != nil {
		t.Fatalf("ToggleWithOptions: %v", err)
	}
	if res.Action != ToggleBack {
		t.Fatalf("action = %v, want ToggleBack", res.Action)
	}
	if len(res.Selected) != 1 || res.Selected[0] != 1 {
		t.Fatalf("selected = %v, want [1]", res.Selected)
	}

	args := readFzfArgs(t, argsFile)
	if !hasArg(args, "--bind=start:pos(4)") {
		t.Errorf("posição inicial não considera as linhas de ação: %v", args)
	}
	for _, arg := range args {
		if strings.Contains(arg, "h:pos(2)+accept") {
			return
		}
	}
	t.Errorf("bind de voltar ausente: %v", args)
}

func TestConfirmUsesSafeDefault(t *testing.T) {
	fakeFzfReply(t, "Não, continuar")

	ok, err := Confirm("Descartar tudo? ")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("Confirm = true, want false")
	}
}

func TestConfirmCanDiscard(t *testing.T) {
	fakeFzfReply(t, "Sim, descartar")

	ok, err := Confirm("Descartar tudo? ")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("Confirm = false, want true")
	}
}

func TestFzfIgnoreArgKeepsVimKeys(t *testing.T) {
	arg := fzfIgnoreArg('j', 'k', 'g', 'G', 'h', 'l', 'q')
	if !strings.Contains(arg, "a:ignore") {
		t.Errorf("tecla imprimível não foi neutralizada: %s", arg)
	}
	for _, used := range []string{"j:ignore", "k:ignore", "g:ignore", "G:ignore", "h:ignore", "l:ignore", "q:ignore"} {
		if strings.Contains(arg, used) {
			t.Errorf("atalho %q não deveria ser ignorado: %s", used, arg)
		}
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
