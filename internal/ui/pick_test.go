package ui

import (
	"bufio"
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
