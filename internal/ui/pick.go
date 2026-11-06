// Package ui tem os helpers de interação com o terminal do wizard.
package ui

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"daylist/internal/theme"
)

// ErrCanceled indica que o usuário cancelou a interação (Esc).
var ErrCanceled = errors.New("cancelado")

// stdin é compartilhado entre as perguntas para não perder linhas já
// bufferizadas quando o input vem de um pipe.
var stdin = bufio.NewReader(os.Stdin)

// Pick deixa o usuário escolher um item. Com terminal interativo usa Bubble
// Tea; sem TTY cai num menu numerado. Retorna "" se cancelar.
func Pick(prompt string, items []string) (string, error) {
	if len(items) == 0 {
		return "", nil
	}
	if hasTerminal() {
		return pickNative(prompt, items)
	}
	return pickPlain(prompt, items)
}

type pickModel struct {
	list   listModel
	prompt string

	done   bool
	choice string
}

func newPickModel(prompt string, items []string) *pickModel {
	return &pickModel{
		prompt: strings.TrimSpace(prompt),
		list:   newListModel(items, strings.TrimSpace(prompt), pickHelp(), 0),
	}
}

func pickHelp() string {
	return "j/k mover · ctrl-d/u página · g/G início/fim · enter escolhe · q cancela"
}

func (m *pickModel) Init() tea.Cmd { return nil }

func (m *pickModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case "enter", " ":
			if len(m.list.items) > 0 {
				m.choice = m.list.items[m.list.cursor]
			}
			m.done = true
			return m, tea.Quit
		case "q", "esc", "ctrl+c":
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *pickModel) View() string {
	if m.done {
		if m.choice == "" {
			return theme.Subtle.Render(fmt.Sprintf("✗ %s — cancelado", m.prompt))
		}
		return theme.OK.Render(fmt.Sprintf("✓ %s — %s", m.prompt, m.choice))
	}

	return m.list.render(func(i int, current bool) string {
		return cursorLine(current, m.list.items[i])
	})
}

func pickNative(prompt string, items []string) (string, error) {
	model := newPickModel(prompt, items)
	if _, err := runProgram(model); err != nil {
		return "", err
	}
	return model.choice, nil
}

func pickPlain(prompt string, items []string) (string, error) {
	fmt.Fprintln(os.Stderr, prompt)
	for i, it := range items {
		fmt.Fprintf(os.Stderr, "%d) %s\n", i+1, it)
	}
	fmt.Fprint(os.Stderr, "#? ")

	line, err := stdin.ReadString('\n')
	if err != nil && line == "" {
		return "", nil
	}
	n, convErr := strconv.Atoi(strings.TrimSpace(line))
	if convErr != nil || n < 1 || n > len(items) {
		return "", nil
	}
	return items[n-1], nil
}
