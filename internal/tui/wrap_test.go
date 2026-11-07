package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestWrapTextPreservesContentWithinWidth(t *testing.T) {
	text := "uma motivação bem longa que precisa aparecer completa em terminal estreito"
	lines := wrapText(text, 20)
	if len(lines) < 2 {
		t.Fatalf("linhas = %v, want quebra em mais de uma linha", lines)
	}
	for _, line := range lines {
		if width := lipgloss.Width(line); width > 20 {
			t.Errorf("linha %q tem %d colunas, want <= 20", line, width)
		}
	}

	got := strings.Join(strings.Fields(strings.Join(lines, " ")), " ")
	want := strings.Join(strings.Fields(text), " ")
	if got != want {
		t.Errorf("conteúdo quebrado = %q, want %q", got, want)
	}
}

func TestWrapTextKeepsLogicalLines(t *testing.T) {
	lines := wrapText("primeira\nsegunda", 40)
	if len(lines) != 2 || lines[0] != "primeira" || lines[1] != "segunda" {
		t.Errorf("linhas = %v, want duas linhas lógicas", lines)
	}
}

func TestVisualHeight(t *testing.T) {
	cases := []struct {
		text  string
		width int
		want  int
	}{
		{"", 20, 1},
		{"curto", 20, 1},
		{"1234567890123456789", 20, 1},
		{"12345678901234567890", 20, 2},
		{"uma\noutra", 20, 2},
	}
	for _, c := range cases {
		if got := visualHeight(c.text, c.width); got != c.want {
			t.Errorf("visualHeight(%q, %d) = %d, want %d", c.text, c.width, got, c.want)
		}
	}
}

func TestSingleLine(t *testing.T) {
	cases := map[string]string{
		"  uma linha  ":       "uma linha",
		"primeira\nsegunda":   "primeira segunda",
		"primeira\r\nsegunda": "primeira segunda",
		"primeira\rsegunda":   "primeira segunda",
		"\n\n":                "",
	}
	for input, want := range cases {
		if got := singleLine(input); got != want {
			t.Errorf("singleLine(%q) = %q, want %q", input, got, want)
		}
	}
}
