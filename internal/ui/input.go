package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"daylist/internal/theme"
)

// Input lê uma linha digitada pelo usuário. Com terminal interativo usa
// textinput em Bubble Tea; sem TTY lê uma linha de stdin.
func Input(prompt string) (string, error) {
	if hasTerminal() {
		return inputNative(prompt)
	}
	return inputPlain(prompt)
}

func inputPlain(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)

	line, err := stdin.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

type inputModel struct {
	prompt string
	input  textinput.Model

	done     bool
	canceled bool
	value    string
}

func newInputModel(prompt string) *inputModel {
	input := textinput.New()
	input.Prompt = ""
	input.CharLimit = 300
	input.Width = 60
	input.Focus()

	return &inputModel{
		prompt: strings.TrimSpace(prompt),
		input:  input,
	}
}

func (m *inputModel) Init() tea.Cmd { return textinput.Blink }

func (m *inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		width := msg.Width - 4
		if width > 0 {
			m.input.Width = width
		}
		return m, nil

	case tea.KeyMsg:
		if m.done {
			return m, nil
		}
		switch msg.String() {
		case "enter":
			m.value = strings.TrimSpace(m.input.Value())
			m.done = true
			return m, tea.Quit
		case "esc", "ctrl+c":
			m.canceled = true
			m.done = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *inputModel) View() string {
	if m.done {
		if m.canceled {
			return theme.Subtle.Render(fmt.Sprintf("✗ %s — cancelado", m.prompt))
		}
		return theme.OK.Render(fmt.Sprintf("✓ %s — %s", m.prompt, m.value))
	}

	var b strings.Builder
	b.WriteString(theme.Heading.Render(m.prompt))
	b.WriteString("\n")
	b.WriteString(theme.Selected.Render("> "))
	b.WriteString(m.input.View())
	b.WriteString("\n\n")
	b.WriteString(theme.Help.Render("enter confirma · esc cancela"))
	return b.String()
}

func inputNative(prompt string) (string, error) {
	model := newInputModel(prompt)
	if _, err := runProgram(model); err != nil {
		return "", err
	}
	if model.canceled {
		return "", ErrCanceled
	}
	return model.value, nil
}

type confirmModel struct {
	prompt string

	done bool
	yes  bool
}

func newConfirmModel(prompt string) *confirmModel {
	return &confirmModel{prompt: strings.TrimSpace(prompt)}
}

func (m *confirmModel) Init() tea.Cmd { return nil }

func (m *confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok || m.done {
		return m, nil
	}

	switch key.String() {
	case "s", "y":
		m.yes = true
		m.done = true
		return m, tea.Quit
	case "n", "q", "esc", "enter", "ctrl+c":
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

func (m *confirmModel) View() string {
	if m.done {
		answer := "Não, continuar"
		if m.yes {
			answer = "Sim, descartar"
		}
		return theme.OK.Render(fmt.Sprintf("✓ %s — %s", m.prompt, answer))
	}

	var b strings.Builder
	b.WriteString(theme.Heading.Render(m.prompt))
	b.WriteString("\n")
	b.WriteString("  s        Sim, descartar\n")
	b.WriteString(theme.Selected.Render("▶ n/esc    Não, continuar"))
	b.WriteString("\n\n")
	b.WriteString(theme.Help.Render("s sim · n/esc não · enter mantém o padrão seguro"))
	return b.String()
}

// Confirm pergunta uma confirmação. A opção segura ("Não, continuar") vem
// primeiro; Esc também conta como não.
func Confirm(prompt string) (bool, error) {
	if hasTerminal() {
		model := newConfirmModel(prompt)
		if _, err := runProgram(model); err != nil {
			return false, err
		}
		return model.yes, nil
	}

	choice, err := pickPlain(prompt, []string{"Não, continuar", "Sim, descartar"})
	if err != nil {
		return false, err
	}
	return choice == "Sim, descartar", nil
}
