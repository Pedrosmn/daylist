package ui

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"daylist/internal/theme"
)

// ToggleAction é o que o usuário pediu ao sair da lista de uma etapa.
type ToggleAction int

const (
	// ToggleFinish conclui a etapa com os itens marcados.
	ToggleFinish ToggleAction = iota
	// ToggleBack volta para a etapa anterior.
	ToggleBack
	// ToggleCancel descarta o wizard.
	ToggleCancel
)

// ToggleOptions configura uma etapa do wizard.
type ToggleOptions struct {
	Prompt    string
	Items     []string
	Selected  []int
	Locked    []int // itens sempre marcados e que não podem ser desmarcados
	AllowBack bool
	Header    string
}

// ToggleResult devolve os itens marcados e a ação escolhida.
type ToggleResult struct {
	Selected []int
	Action   ToggleAction
}

// Toggle mostra uma lista onde Enter marca/desmarca o item destacado e l
// encerra a etapa. Devolve os índices marcados na ordem original.
// Esc devolve ErrCanceled.
func Toggle(prompt string, items []string) ([]int, error) {
	res, err := ToggleWithOptions(ToggleOptions{
		Prompt: prompt,
		Items:  items,
	})
	if err != nil {
		return nil, err
	}
	if res.Action == ToggleCancel {
		return nil, ErrCanceled
	}
	return res.Selected, nil
}

// ToggleWithOptions é a versão do Toggle usada pelo wizard. Além de marcar e
// desmarcar, permite voltar para a etapa anterior e preservar uma seleção.
// Com terminal interativo usa Bubble Tea; sem TTY cai no menu numerado.
func ToggleWithOptions(opts ToggleOptions) (ToggleResult, error) {
	if len(opts.Items) == 0 {
		return ToggleResult{Action: ToggleFinish}, nil
	}
	if hasTerminal() {
		return toggleNative(opts)
	}
	return togglePlainOptions(opts)
}

type toggleModel struct {
	opts   ToggleOptions
	list   listModel
	marked []bool
	locked []bool

	done   bool
	result ToggleResult
}

func newToggleModel(opts ToggleOptions) *toggleModel {
	marked := make([]bool, len(opts.Items))
	for _, idx := range opts.Selected {
		if idx >= 0 && idx < len(marked) {
			marked[idx] = true
		}
	}
	locked := make([]bool, len(opts.Items))
	for _, idx := range opts.Locked {
		if idx >= 0 && idx < len(locked) {
			marked[idx] = true
			locked[idx] = true
		}
	}

	cursor := 0
	for i, on := range marked {
		if on {
			cursor = i
			break
		}
	}

	return &toggleModel{
		opts:   opts,
		marked: marked,
		locked: locked,
		list:   newListModel(opts.Items, "", toggleHelp(opts.AllowBack), cursor),
	}
}

func toggleHelp(allowBack bool) string {
	parts := []string{"j/k mover", "ctrl-d/u página", "g/G início/fim", "espaço marca", "l conclui"}
	if allowBack {
		parts = append(parts, "h volta")
	}
	parts = append(parts, "q cancela")
	return strings.Join(parts, " · ")
}

func (m *toggleModel) title() string {
	base := strings.TrimSpace(m.opts.Header)
	if base == "" {
		base = strings.TrimSpace(m.opts.Prompt)
	}
	count := m.count()
	if base == "" {
		return fmt.Sprintf("%s · %d no total", markedCount(count), len(m.opts.Items))
	}
	return fmt.Sprintf("%s · %s · %d no total", base, markedCount(count), len(m.opts.Items))
}

func markedCount(count int) string {
	if count == 1 {
		return "1 marcado"
	}
	return fmt.Sprintf("%d marcados", count)
}

func (m *toggleModel) count() int {
	count := 0
	for _, on := range m.marked {
		if on {
			count++
		}
	}
	return count
}

func (m *toggleModel) Init() tea.Cmd { return nil }

func (m *toggleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.setHeight(msg.Height)
		return m, nil

	case tea.KeyMsg:
		if m.done {
			return m, nil
		}
		m.list.updateNavigation(msg)
		switch msg.String() {
		case " ", "enter":
			m.toggleCurrent()
		case "l":
			return m.finish()
		case "h":
			if m.opts.AllowBack {
				return m.back()
			}
		case "q", "esc", "ctrl+c":
			return m.cancel()
		}
	}
	return m, nil
}

func (m *toggleModel) toggleCurrent() {
	if len(m.marked) == 0 || m.locked[m.list.cursor] {
		return
	}
	m.marked[m.list.cursor] = !m.marked[m.list.cursor]
}

func (m *toggleModel) finish() (tea.Model, tea.Cmd) {
	m.result = ToggleResult{Selected: markedIndices(m.marked), Action: ToggleFinish}
	m.done = true
	return m, tea.Quit
}

func (m *toggleModel) back() (tea.Model, tea.Cmd) {
	m.result = ToggleResult{Selected: markedIndices(m.marked), Action: ToggleBack}
	m.done = true
	return m, tea.Quit
}

func (m *toggleModel) cancel() (tea.Model, tea.Cmd) {
	m.result = ToggleResult{Selected: markedIndices(m.marked), Action: ToggleCancel}
	m.done = true
	return m, tea.Quit
}

func (m *toggleModel) View() string {
	if m.done {
		return m.summary()
	}

	list := m.list
	list.title = m.title()
	return list.render(func(i int, current bool) string {
		box := "[ ]"
		if m.marked[i] {
			box = "[x]"
		}
		return cursorLine(current, box+" "+m.opts.Items[i])
	})
}

func (m *toggleModel) summary() string {
	title := strings.TrimSpace(m.opts.Header)
	if title == "" {
		title = strings.TrimSpace(m.opts.Prompt)
	}
	if title == "" {
		title = "Seleção"
	}

	count := len(m.result.Selected)
	switch m.result.Action {
	case ToggleFinish:
		return theme.OK.Render(fmt.Sprintf("✓ %s — %s", title, selectedCount(count)))
	case ToggleBack:
		return theme.Subtle.Render(fmt.Sprintf("↩ %s — %s", title, selectedCount(count)))
	default:
		return theme.Subtle.Render(fmt.Sprintf("✗ %s — cancelado", title))
	}
}

func selectedCount(count int) string {
	switch count {
	case 0:
		return "nenhum selecionado"
	case 1:
		return "1 selecionado"
	default:
		return fmt.Sprintf("%d selecionados", count)
	}
}

func toggleNative(opts ToggleOptions) (ToggleResult, error) {
	model := newToggleModel(opts)
	if _, err := runProgram(model); err != nil {
		return ToggleResult{}, err
	}
	return model.result, nil
}

// togglePlain é o fallback sem TTY. Ele continua numerado, mas aceita os
// comandos h, l e q para voltar, concluir e cancelar.
func togglePlain(prompt string, items []string) ([]int, error) {
	res, err := togglePlainOptions(ToggleOptions{
		Prompt: prompt,
		Items:  items,
	})
	if err != nil {
		return nil, err
	}
	if res.Action == ToggleCancel {
		return nil, ErrCanceled
	}
	return res.Selected, nil
}

func togglePlainOptions(opts ToggleOptions) (ToggleResult, error) {
	marked := make([]bool, len(opts.Items))
	for _, idx := range opts.Selected {
		if idx >= 0 && idx < len(marked) {
			marked[idx] = true
		}
	}
	locked := make([]bool, len(opts.Items))
	for _, idx := range opts.Locked {
		if idx >= 0 && idx < len(locked) {
			marked[idx] = true
			locked[idx] = true
		}
	}

	for {
		count := 0
		for _, m := range marked {
			if m {
				count++
			}
		}

		fmt.Fprintln(os.Stderr, opts.Prompt)
		if opts.Header != "" {
			fmt.Fprintf(os.Stderr, "%s · %s · %d no total\n", opts.Header, markedCount(count), len(opts.Items))
		}
		for i, it := range opts.Items {
			mark := "[ ]"
			if marked[i] {
				mark = "[x]"
			}
			fmt.Fprintf(os.Stderr, "%d) %s %s\n", i+1, mark, it)
		}
		hint := "Enter/l conclui, q cancela"
		if opts.AllowBack {
			hint = "Enter/l conclui, h volta, q cancela"
		}
		fmt.Fprintf(os.Stderr, "#? (%s) ", hint)

		line, err := stdin.ReadString('\n')
		if err != nil && line == "" {
			return ToggleResult{Selected: markedIndices(marked), Action: ToggleFinish}, nil // EOF conclui
		}

		line = strings.TrimSpace(line)
		switch line {
		case "", "l":
			return ToggleResult{Selected: markedIndices(marked), Action: ToggleFinish}, nil
		case "q":
			return ToggleResult{Selected: markedIndices(marked), Action: ToggleCancel}, nil
		case "h":
			if opts.AllowBack {
				return ToggleResult{Selected: markedIndices(marked), Action: ToggleBack}, nil
			}
			continue
		}

		n, convErr := strconv.Atoi(line)
		if convErr != nil || n < 1 || n > len(opts.Items) {
			continue
		}
		if locked[n-1] {
			continue
		}
		marked[n-1] = !marked[n-1]
	}
}

func markedIndices(marked []bool) []int {
	var out []int
	for i, m := range marked {
		if m {
			out = append(out, i)
		}
	}
	return out
}
