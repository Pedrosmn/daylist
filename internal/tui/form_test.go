package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestWrapFieldCursorFollowsWrappedText(t *testing.T) {
	form := newMotivationForm(0, "", "item")
	form.layout(20)
	field := &form.fields[0]

	text := "uma motivação bem longa que desce"
	for _, r := range text {
		field.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	if got := field.Value(); got != text {
		t.Fatalf("valor = %q, want %q", got, text)
	}
	height := visualHeight(field.Value(), field.width)
	info := field.area.LineInfo()
	if info.RowOffset != height-1 {
		t.Errorf("cursor na linha %d, want última de %d", info.RowOffset+1, height)
	}
}

func TestFormMotivationTitleWithoutSubject(t *testing.T) {
	form := newMotivationForm(0, "", "")
	if got := form.title(); got != "Motivação" {
		t.Errorf("título = %q, want Motivação", got)
	}
}
