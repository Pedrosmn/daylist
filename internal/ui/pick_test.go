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

func TestPickPlainMulti(t *testing.T) {
	setStdin(t, "1 3\n")
	got, err := pickPlainMulti("escolha: ", []string{"a", "b", "c"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "a" || got[1] != "c" {
		t.Errorf("got = %v, want [a c]", got)
	}
}

func TestPickPlainMultiCommasAndDuplicates(t *testing.T) {
	setStdin(t, "2,4,2 9\n")
	got, err := pickPlainMulti("escolha: ", []string{"a", "b", "c", "d"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "b" || got[1] != "d" {
		t.Errorf("got = %v, want [b d]", got)
	}
}

func TestPickPlainMultiEmptyIsCancel(t *testing.T) {
	setStdin(t, "\n")
	got, err := pickPlainMulti("escolha: ", []string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("got = %v, want nil", got)
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
