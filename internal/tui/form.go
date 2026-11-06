package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"daylist/internal/daylist"
)

type formKind int

const (
	formExtraTask formKind = iota
	formExtraHabit
	formMotivation
)

var (
	flagLabels = []string{
		"Nenhuma", "Importante (I)", "Urgente (U)", "Importante e Urgente (IU)",
	}
	flagValues = []string{"", "I", "U", "IU"}
)

// formState é um formulário simples, de uma ou quatro linhas. O foco 2 do
// Extra task é a prioridade, escolhida com ←/→.
type formState struct {
	kind    formKind
	inputs  []textinput.Model
	focus   int
	flagIdx int
	habit   daylist.Item
	line    int
	initial string
	err     string
}

func newInput(placeholder string) textinput.Model {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = placeholder
	input.CharLimit = 300
	input.Width = 60
	return input
}

func newExtraTaskForm() *formState {
	form := &formState{
		kind: formExtraTask,
		inputs: []textinput.Model{
			newInput("descrição da task"),
			newInput("ex.: 15 min (opcional)"),
			newInput(""),
			newInput("motivação (opcional)"),
		},
	}
	form.inputs[0].Focus()
	return form
}

func newExtraHabitForm(habit daylist.Item) *formState {
	form := &formState{
		kind:   formExtraHabit,
		habit:  habit,
		inputs: []textinput.Model{newInput("motivação (opcional)")},
	}
	form.inputs[0].Focus()
	return form
}

func newMotivationForm(line int, current string) *formState {
	form := &formState{
		kind:    formMotivation,
		line:    line,
		inputs:  []textinput.Model{newInput("motivação (vazio remove)")},
		initial: current,
	}
	form.inputs[0].SetValue(current)
	form.inputs[0].Focus()
	return form
}

func (f *formState) title() string {
	switch f.kind {
	case formExtraTask:
		return "Extra task"
	case formExtraHabit:
		return "Extra hábito · " + f.habit.Desc
	default:
		return "Motivação"
	}
}

func (f *formState) fieldLabel(index int) string {
	if f.kind == formExtraTask {
		switch index {
		case 0:
			return "Descrição"
		case 1:
			return "Tempo"
		case 2:
			return "Prioridade"
		case 3:
			return "Motivação"
		}
	}
	return "Motivação"
}

func (f *formState) flagLabel() string { return flagLabels[f.flagIdx] }

func (f *formState) flagValue() string { return flagValues[f.flagIdx] }

func (f *formState) lastField() int { return len(f.inputs) }

// dirty informa se o usuário já digitou ou mudou algo que seria perdido ao
// cancelar o formulário.
func (f *formState) dirty() bool {
	switch f.kind {
	case formExtraTask:
		return f.value(0) != "" || f.value(1) != "" || f.flagIdx != 0 || f.value(3) != ""
	default:
		return f.value(0) != strings.TrimSpace(f.initial)
	}
}

// Update trata uma tecla. O segundo retorno indica que o formulário deve ser
// enviado.
func (f *formState) Update(msg tea.Msg) (tea.Cmd, bool) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return f.updateInputs(msg), false
	}

	switch key.String() {
	case "esc":
		return nil, false
	case "ctrl+s":
		return nil, true
	case "tab", "down":
		f.moveFocus(1)
		return nil, false
	case "shift+tab", "up":
		f.moveFocus(-1)
		return nil, false
	case "left":
		if f.focus == 2 && f.kind == formExtraTask {
			f.flagIdx = (f.flagIdx + len(flagLabels) - 1) % len(flagLabels)
			return nil, false
		}
	case "right":
		if f.focus == 2 && f.kind == formExtraTask {
			f.flagIdx = (f.flagIdx + 1) % len(flagLabels)
			return nil, false
		}
	case "enter":
		if f.focus >= f.lastField()-1 {
			return nil, true
		}
		f.moveFocus(1)
		return nil, false
	}

	return f.updateInputs(msg), false
}

func (f *formState) updateInputs(msg tea.Msg) tea.Cmd {
	if f.focus < 0 || f.focus >= len(f.inputs) {
		return nil
	}
	var cmd tea.Cmd
	f.inputs[f.focus], cmd = f.inputs[f.focus].Update(msg)
	return cmd
}

func (f *formState) moveFocus(delta int) {
	total := f.lastField()
	f.focus = (f.focus + delta + total) % total
	for i := range f.inputs {
		if i == f.focus {
			f.inputs[i].Focus()
		} else {
			f.inputs[i].Blur()
		}
	}
}

func (f *formState) value(index int) string {
	if index < 0 || index >= len(f.inputs) {
		return ""
	}
	return strings.TrimSpace(f.inputs[index].Value())
}

func (f *formState) motivation() string {
	if len(f.inputs) == 0 {
		return ""
	}
	return strings.TrimSpace(f.inputs[len(f.inputs)-1].Value())
}
