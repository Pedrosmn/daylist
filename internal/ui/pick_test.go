package ui

import (
	"bufio"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
	got, err := togglePlain("escolha: ", []string{"a", "b", "c"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Errorf("got = %v, want [1 2]", got)
	}
}

func TestTogglePlainUntoggles(t *testing.T) {
	setStdin(t, "2\n2\n\n")
	got, err := togglePlain("escolha: ", []string{"a", "b", "c"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("got = %v, want vazio", got)
	}
}

func TestTogglePlainIgnoresZeroAndFinishesWithL(t *testing.T) {
	setStdin(t, "0\n2\nl\n")
	got, err := togglePlain("escolha: ", []string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != 1 {
		t.Errorf("got = %v, want [1]", got)
	}
}

func TestTogglePlainEOFKeepsMarks(t *testing.T) {
	setStdin(t, "2\n")
	got, err := togglePlain("escolha: ", []string{"a", "b", "c"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != 1 {
		t.Errorf("got = %v, want [1]", got)
	}
}

func TestTogglePlainIgnoresInvalid(t *testing.T) {
	setStdin(t, "x\n9\n2\n\n")
	got, err := togglePlain("escolha: ", []string{"a", "b"})
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

func TestTogglePlainLockedStaysSelected(t *testing.T) {
	setStdin(t, "2\n2\nl\n")
	res, err := togglePlainOptions(ToggleOptions{
		Prompt:   "escolha: ",
		Items:    []string{"a", "b", "c"},
		Selected: []int{1},
		Locked:   []int{1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Selected) != 1 || res.Selected[0] != 1 {
		t.Errorf("selected = %v, want [1]", res.Selected)
	}
}

func TestTogglePlainLockedIsAutoSelected(t *testing.T) {
	setStdin(t, "l\n")
	res, err := togglePlainOptions(ToggleOptions{
		Prompt: "escolha: ",
		Items:  []string{"a", "b", "c"},
		Locked: []int{2},
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
	got, err := inputPlain("min: ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "45" {
		t.Errorf("got = %q, want %q", got, "45")
	}
}

// --- modelos Bubble Tea ---

func runeKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func typeKey(k tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: k}
}

func updateToggle(t *testing.T, m *toggleModel, msg tea.Msg) {
	t.Helper()
	got, _ := m.Update(msg)
	if got != m {
		t.Fatalf("Update devolveu outro modelo: %T", got)
	}
}

func TestListNavigationAndBounds(t *testing.T) {
	list := newListModel([]string{"a", "b", "c"}, "Lista", "ajuda", 0)

	list.updateNavigation(runeKey('j'))
	if list.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", list.cursor)
	}
	list.updateNavigation(typeKey(tea.KeyUp))
	if list.cursor != 0 {
		t.Fatalf("cursor = %d, want 0 depois de up", list.cursor)
	}
	list.updateNavigation(runeKey('k'))
	if list.cursor != 0 {
		t.Fatalf("cursor = %d, want 0 no início", list.cursor)
	}
	list.updateNavigation(runeKey('G'))
	if list.cursor != 2 {
		t.Fatalf("cursor = %d, want 2 no fim", list.cursor)
	}
	list.updateNavigation(runeKey('j'))
	if list.cursor != 2 {
		t.Fatalf("cursor = %d, want 2 (sem passar do fim)", list.cursor)
	}
	list.updateNavigation(runeKey('g'))
	if list.cursor != 0 {
		t.Fatalf("cursor = %d, want 0 no início", list.cursor)
	}
	if list.updateNavigation(runeKey('x')) {
		t.Error("tecla de ação não deveria ser consumida pela navegação")
	}
}

func TestListWindowKeepsCursorVisible(t *testing.T) {
	items := make([]string, 30)
	for i := range items {
		items[i] = string(rune('a' + i%26))
	}
	list := newListModel(items, "Lista", "", 0)
	list.setHeight(8) // 5 itens visíveis

	list.updateNavigation(runeKey('G'))
	start, end := list.window()
	if list.cursor < start || list.cursor >= end {
		t.Fatalf("cursor %d fora da janela %d-%d", list.cursor, start, end)
	}
	if end != len(items) {
		t.Errorf("fim da janela = %d, want %d", end, len(items))
	}

	list.updateNavigation(runeKey('g'))
	start, end = list.window()
	if start != 0 {
		t.Errorf("início da janela = %d, want 0", start)
	}
	if list.cursor < start || list.cursor >= end {
		t.Fatalf("cursor %d fora da janela %d-%d", list.cursor, start, end)
	}

	list.updateNavigation(typeKey(tea.KeyCtrlD))
	start, end = list.window()
	if list.cursor < start || list.cursor >= end {
		t.Fatalf("cursor %d fora da janela %d-%d depois de ctrl-d", list.cursor, start, end)
	}
}

func TestToggleModelInitialSelectionAndLocked(t *testing.T) {
	m := newToggleModel(ToggleOptions{
		Header:   "Etapa 1/2 · Tasks",
		Prompt:   "Etapa 1/2 · Tasks › ",
		Items:    []string{"a", "b", "c"},
		Selected: []int{1},
		Locked:   []int{2},
	})

	if !m.marked[1] || !m.marked[2] {
		t.Fatalf("seleção inicial = %v, want b e c marcados", m.marked)
	}
	if m.list.cursor != 1 {
		t.Errorf("cursor = %d, want 1 (primeiro marcado)", m.list.cursor)
	}
	if !strings.Contains(m.View(), "[x] b") || !strings.Contains(m.View(), "[x] c") {
		t.Errorf("view sem itens marcados:\n%s", m.View())
	}
	if !strings.Contains(m.View(), "2 marcados · 3 no total") {
		t.Errorf("view sem contador:\n%s", m.View())
	}
}

func TestToggleModelMarksAndIgnoresLocked(t *testing.T) {
	m := newToggleModel(ToggleOptions{
		Items:  []string{"a", "b", "c"},
		Locked: []int{2},
	})

	m.list.cursor = 0
	updateToggle(t, m, typeKey(tea.KeySpace))
	if !m.marked[0] {
		t.Fatal("espaço deveria marcar o item atual")
	}

	m.list.cursor = 2
	updateToggle(t, m, typeKey(tea.KeySpace))
	if !m.marked[2] {
		t.Error("item travado foi desmarcado")
	}

	updateToggle(t, m, runeKey('j'))
	if m.list.cursor != 2 {
		t.Errorf("cursor = %d, want 2 (limite)", m.list.cursor)
	}
}

func TestToggleModelFinishBackCancel(t *testing.T) {
	tests := []struct {
		name   string
		key    tea.KeyMsg
		back   bool
		action ToggleAction
	}{
		{name: "conclui", key: runeKey('l'), action: ToggleFinish},
		{name: "volta", key: runeKey('h'), back: true, action: ToggleBack},
		{name: "cancela", key: runeKey('q'), action: ToggleCancel},
		{name: "esc", key: typeKey(tea.KeyEsc), action: ToggleCancel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newToggleModel(ToggleOptions{
				Header:    "Etapa 1/2 · Tasks",
				Items:     []string{"a", "b"},
				Selected:  []int{0},
				AllowBack: tt.back,
			})
			updateToggle(t, m, tt.key)
			if m.result.Action != tt.action {
				t.Fatalf("action = %v, want %v", m.result.Action, tt.action)
			}
			if !m.done {
				t.Error("done = false")
			}
			if len(m.result.Selected) != 1 || m.result.Selected[0] != 0 {
				t.Errorf("selected = %v, want [0]", m.result.Selected)
			}
		})
	}
}

func TestToggleModelBackIgnoredOnFirstStage(t *testing.T) {
	m := newToggleModel(ToggleOptions{
		Items: []string{"a", "b"},
	})
	updateToggle(t, m, runeKey('h'))
	if m.done || m.result.Action != ToggleFinish {
		t.Fatal("h sem AllowBack deveria ser ignorado")
	}
}

func TestToggleModelSummaryIsCompact(t *testing.T) {
	m := newToggleModel(ToggleOptions{
		Header:   "Etapa 1/2 · Tasks",
		Items:    []string{"a", "b", "c"},
		Selected: []int{0, 2},
	})
	updateToggle(t, m, runeKey('l'))

	view := m.View()
	if !strings.Contains(view, "✓ Etapa 1/2 · Tasks — 2 selecionados") {
		t.Errorf("resumo = %q", view)
	}
	for _, marker := range []string{"▶", "[x]", "[ ]"} {
		if strings.Contains(view, marker) {
			t.Errorf("resumo ainda mostra a lista (%q): %q", marker, view)
		}
	}
}

func TestPickModelNavigationAndChoose(t *testing.T) {
	m := newPickModel("Duração: ", []string{"10 min", "20 min", "30 min"})

	m.Update(runeKey('j'))
	m.Update(runeKey('j'))
	if m.list.cursor != 2 {
		t.Fatalf("cursor = %d, want 2", m.list.cursor)
	}
	m.Update(runeKey('k'))
	_, cmd := m.Update(typeKey(tea.KeyEnter))
	if m.choice != "20 min" {
		t.Errorf("choice = %q, want 20 min", m.choice)
	}
	if cmd == nil {
		t.Error("enter deveria encerrar o programa")
	}
	if !strings.Contains(m.View(), "✓ Duração: — 20 min") {
		t.Errorf("resumo = %q", m.View())
	}
}

func TestPickModelCancel(t *testing.T) {
	m := newPickModel("Duração: ", []string{"10 min"})
	m.Update(typeKey(tea.KeyEsc))
	if m.choice != "" {
		t.Errorf("choice = %q, want vazio", m.choice)
	}
	if !strings.Contains(m.View(), "cancelado") {
		t.Errorf("resumo = %q", m.View())
	}
}

func TestInputModelAcceptsValue(t *testing.T) {
	m := newInputModel("Duração personalizada (min): ")
	m.Update(runeKey('4'))
	m.Update(runeKey('5'))
	_, cmd := m.Update(typeKey(tea.KeyEnter))

	if m.value != "45" {
		t.Errorf("value = %q, want 45", m.value)
	}
	if m.canceled {
		t.Error("canceled = true")
	}
	if cmd == nil {
		t.Error("enter deveria encerrar o programa")
	}
}

func TestInputModelCancel(t *testing.T) {
	m := newInputModel("Duração: ")
	m.Update(typeKey(tea.KeyEsc))
	if !m.canceled {
		t.Error("canceled = false")
	}
	if !strings.Contains(m.View(), "cancelado") {
		t.Errorf("resumo = %q", m.View())
	}
}

func TestConfirmModelSafeDefault(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyMsg
		yes  bool
	}{
		{name: "sim", key: runeKey('s'), yes: true},
		{name: "não", key: runeKey('n'), yes: false},
		{name: "esc", key: typeKey(tea.KeyEsc), yes: false},
		{name: "enter", key: typeKey(tea.KeyEnter), yes: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newConfirmModel("Descartar tudo? ")
			m.Update(tt.key)
			if m.yes != tt.yes {
				t.Errorf("yes = %v, want %v", m.yes, tt.yes)
			}
			if !m.done {
				t.Error("done = false")
			}
		})
	}
}
