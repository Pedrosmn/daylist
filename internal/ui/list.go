package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"daylist/internal/theme"
)

// defaultListHeight é a janela usada antes do primeiro WindowSizeMsg. Ela é
// pequena o bastante para não dominar o scrollback e grande o bastante para
// mostrar contexto em listas longas.
const defaultListHeight = 15

// maxListItems limita a janela mesmo em terminais altos: o wizard é inline e
// não deve empurrar o histórico para fora da tela.
const maxListItems = 12

// listModel é a navegação compartilhada por Toggle e Pick: cursor, janela de
// itens visíveis, cabeçalho e rodapé de ajuda.
type listModel struct {
	items  []string
	cursor int
	offset int
	height int

	title string
	help  string
}

func newListModel(items []string, title, help string, cursor int) listModel {
	l := listModel{
		items:  items,
		title:  title,
		help:   help,
		height: defaultListHeight,
		cursor: cursor,
	}
	l.clamp()
	return l
}

// setHeight acompanha o redimensionamento do terminal.
func (l *listModel) setHeight(height int) {
	if height > 0 {
		l.height = height
	}
	l.clamp()
}

// visibleHeight devolve quantos itens cabem na janela, descontando o
// cabeçalho, a linha em branco e o rodapé de ajuda.
func (l listModel) visibleHeight() int {
	visible := l.height - 3
	if visible > maxListItems {
		visible = maxListItems
	}
	if visible < 3 {
		visible = 3
	}
	return visible
}

func (l *listModel) clamp() {
	if len(l.items) == 0 {
		l.cursor = 0
		l.offset = 0
		return
	}
	if l.cursor < 0 {
		l.cursor = 0
	}
	if l.cursor >= len(l.items) {
		l.cursor = len(l.items) - 1
	}
	l.ensureVisible()
}

func (l *listModel) ensureVisible() {
	visible := l.visibleHeight()

	if l.cursor < l.offset {
		l.offset = l.cursor
	}
	if l.cursor >= l.offset+visible {
		l.offset = l.cursor - visible + 1
	}

	maxOffset := len(l.items) - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	if l.offset > maxOffset {
		l.offset = maxOffset
	}
	if l.offset < 0 {
		l.offset = 0
	}
}

func (l *listModel) move(delta int) {
	if len(l.items) == 0 {
		return
	}
	l.cursor += delta
	if l.cursor < 0 {
		l.cursor = 0
	}
	if l.cursor >= len(l.items) {
		l.cursor = len(l.items) - 1
	}
	l.ensureVisible()
}

func (l *listModel) halfPage(direction int) {
	step := l.visibleHeight() / 2
	if step < 1 {
		step = 1
	}
	l.move(direction * step)
}

func (l *listModel) top() {
	l.cursor = 0
	l.ensureVisible()
}

func (l *listModel) bottom() {
	if len(l.items) > 0 {
		l.cursor = len(l.items) - 1
	}
	l.ensureVisible()
}

// window devolve o intervalo [start, end) de itens que deve ser desenhado.
func (l listModel) window() (int, int) {
	start := l.offset
	end := start + l.visibleHeight()
	if end > len(l.items) {
		end = len(l.items)
	}
	if start < 0 {
		start = 0
	}
	if start > end {
		start = end
	}
	return start, end
}

// updateNavigation trata as teclas comuns de navegação. O retorno indica se a
// tecla foi consumida; teclas de ação ficam para o modelo específico (Toggle
// ou Pick).
func (l *listModel) updateNavigation(key tea.KeyMsg) bool {
	switch key.String() {
	case "j", "down":
		l.move(1)
	case "k", "up":
		l.move(-1)
	case "g", "home":
		l.top()
	case "G", "end":
		l.bottom()
	case "ctrl+d":
		l.halfPage(1)
	case "ctrl+u":
		l.halfPage(-1)
	default:
		return false
	}
	return true
}

func (l listModel) render(renderItem func(index int, current bool) string) string {
	var b strings.Builder

	if l.title != "" {
		b.WriteString(theme.Heading.Render(l.title))
		b.WriteString("\n")
	}

	start, end := l.window()
	for i := start; i < end; i++ {
		b.WriteString(renderItem(i, i == l.cursor))
		b.WriteString("\n")
	}

	if l.help != "" {
		b.WriteString("\n")
		b.WriteString(theme.Help.Render(l.help))
	}
	return b.String()
}

// cursorLine destaca o item atual e deixa os demais alinhados.
func cursorLine(current bool, text string) string {
	if current {
		return theme.Selected.Render("▶ " + text)
	}
	return "  " + text
}
