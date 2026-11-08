package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
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

type fieldKind int

const (
	fieldText fieldKind = iota
	fieldPriority
	fieldWrap
)

const defaultFormWidth = 60

var (
	flagLabels = []string{
		"Nenhuma", "Importante (I)", "Urgente (U)", "Importante e Urgente (IU)",
	}
	flagValues = []string{"", "I", "U", "IU"}
)

// formField é um campo do formulário. Campos de texto usam textinput; o campo
// de motivação usa textarea para quebrar o texto em várias linhas visuais sem
// gravar quebras de linha. Prioridade é um campo especial, sem entrada livre.
type formField struct {
	kind  fieldKind
	label string
	input textinput.Model
	area  *textarea.Model
	width int
}

func newTextField(label, placeholder string) formField {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = placeholder
	input.CharLimit = 300
	input.Width = defaultFormWidth

	return formField{kind: fieldText, label: label, input: input}
}

func newWrapField(label, placeholder string) formField {
	area := textarea.New()
	area.Prompt = ""
	area.Placeholder = placeholder
	area.CharLimit = 300
	area.ShowLineNumbers = false
	area.MaxHeight = 0
	// O textarea do Bubbles pinta a linha do cursor com fundo preto por padrão.
	// No TUI queremos o campo transparente, alinhado ao restante do formulário.
	area.FocusedStyle.CursorLine = area.FocusedStyle.CursorLine.UnsetBackground()
	area.BlurredStyle.CursorLine = area.BlurredStyle.CursorLine.UnsetBackground()
	area.SetWidth(defaultFormWidth)
	area.SetHeight(1)

	return formField{kind: fieldWrap, label: label, area: &area, width: defaultFormWidth}
}

func newPriorityField() formField {
	return formField{kind: fieldPriority, label: "Prioridade"}
}

func (f *formField) prefix() string {
	if f.label == "" {
		return ""
	}
	return f.label + ": "
}

// layout ajusta a largura disponível e, no campo com wrap, a altura necessária
// para mostrar todo o texto, sem rolagem vertical.
func (f *formField) layout(width int) {
	if width < 1 {
		width = 1
	}
	previous := f.width
	f.width = width

	switch f.kind {
	case fieldText:
		f.input.Width = width
	case fieldWrap:
		f.area.SetWidth(width)
		text := f.area.Value()
		if text == "" {
			text = f.area.Placeholder
		}
		f.area.SetHeight(visualHeight(text, width))
		if previous != width {
			f.resetViewport()
		}
	}
}

func (f *formField) Focus() tea.Cmd {
	if f.kind == fieldPriority {
		return nil
	}
	if f.area != nil {
		return f.area.Focus()
	}
	return f.input.Focus()
}

func (f *formField) Blur() {
	if f.kind == fieldPriority {
		return
	}
	if f.area != nil {
		f.area.Blur()
		return
	}
	f.input.Blur()
}

func (f *formField) Value() string {
	if f.kind == fieldPriority {
		return ""
	}
	if f.area != nil {
		return f.area.Value()
	}
	return f.input.Value()
}

func (f *formField) SetValue(value string) {
	if f.kind == fieldPriority {
		return
	}
	if f.area != nil {
		f.area.SetValue(value)
		f.area.CursorEnd()
		return
	}
	f.input.SetValue(value)
}

func (f *formField) Update(msg tea.Msg) tea.Cmd {
	if f.kind == fieldPriority {
		return nil
	}

	var cmd tea.Cmd
	if f.area != nil {
		*f.area, cmd = f.area.Update(msg)
		f.resetViewport()
		f.layout(f.width)
	} else {
		f.input, cmd = f.input.Update(msg)
		f.layout(f.width)
	}
	return cmd
}

// resetViewport mantém a primeira linha visual sempre visível. O textarea
// ajusta a rolagem a cada tecla antes de sabermos a altura final do conteúdo;
// sem isso, ao crescer, o campo esconderia as primeiras linhas.
func (f *formField) resetViewport() {
	info := f.area.LineInfo()
	column := info.StartColumn + info.ColumnOffset
	value := flattenLines(f.area.Value())

	f.area.Reset()
	f.area.InsertString(value)
	f.area.SetCursor(column)
}

func (f *formField) View() string {
	if f.area != nil {
		return f.area.View()
	}
	return f.input.View()
}

// formState é um formulário simples, de uma a quatro linhas. O campo de
// prioridade do Extra task é escolhido com ←/→.
type formState struct {
	kind    formKind
	fields  []formField
	focus   int
	flagIdx int
	habit   daylist.Item
	line    int
	initial string
	subject string
	err     string
}

func newExtraTaskForm() *formState {
	form := &formState{
		kind: formExtraTask,
		fields: []formField{
			newTextField("Descrição", "descrição da task"),
			newTextField("Tempo", "ex.: 15 min (opcional)"),
			newPriorityField(),
			newWrapField("Motivação", "motivação (opcional)"),
		},
	}
	form.layout(defaultFormWidth)
	form.moveFocus(0)
	return form
}

func newExtraHabitForm(habit daylist.Item) *formState {
	form := &formState{
		kind:   formExtraHabit,
		habit:  habit,
		fields: []formField{newWrapField("Motivação", "motivação (opcional)")},
	}
	form.layout(defaultFormWidth)
	form.moveFocus(0)
	return form
}

func newMotivationForm(line int, current, subject string) *formState {
	current = singleLine(current)
	form := &formState{
		kind:    formMotivation,
		line:    line,
		fields:  []formField{newWrapField("", "motivação (vazio remove)")},
		initial: current,
		subject: subject,
	}
	form.fields[0].SetValue(current)
	form.layout(defaultFormWidth)
	form.moveFocus(0)
	return form
}

func (f *formState) title() string {
	switch f.kind {
	case formExtraTask:
		return "Extra task"
	case formExtraHabit:
		return "Extra hábito · " + f.habit.Desc
	default:
		if f.subject != "" {
			return "Motivação · " + f.subject
		}
		return "Motivação"
	}
}

// help devolve o rodapé contextual: só anuncia teclas que funcionam no
// formulário atual.
func (f *formState) help() string {
	if f.kind == formExtraTask {
		return "tab/↑↓ campo · ←/→ prioridade · enter/ctrl+s salva · esc cancela"
	}
	return "enter salva · esc cancela"
}

// layout ajusta todos os campos à largura útil do formulário.
func (f *formState) layout(contentWidth int) {
	for i := range f.fields {
		field := &f.fields[i]
		if field.kind == fieldPriority {
			continue
		}
		field.layout(contentWidth - len([]rune(field.prefix())))
	}
}

func (f *formState) flagLabel() string { return flagLabels[f.flagIdx] }

func (f *formState) flagValue() string { return flagValues[f.flagIdx] }

func (f *formState) lastField() int { return len(f.fields) }

// dirty informa se o usuário já digitou ou mudou algo que seria perdido ao
// cancelar o formulário.
func (f *formState) dirty() bool {
	switch f.kind {
	case formExtraTask:
		return f.value(0) != "" || f.value(1) != "" || f.flagIdx != 0 || f.value(3) != ""
	default:
		return f.value(0) != f.initial
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
		if f.focusPriority() {
			f.flagIdx = (f.flagIdx + len(flagLabels) - 1) % len(flagLabels)
			return nil, false
		}
	case "right":
		if f.focusPriority() {
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

func (f *formState) focusPriority() bool {
	return f.focus >= 0 && f.focus < len(f.fields) && f.fields[f.focus].kind == fieldPriority
}

func (f *formState) updateInputs(msg tea.Msg) tea.Cmd {
	if f.focus < 0 || f.focus >= len(f.fields) {
		return nil
	}
	return f.fields[f.focus].Update(msg)
}

func (f *formState) moveFocus(delta int) {
	total := f.lastField()
	if total == 0 {
		return
	}
	f.focus = (f.focus + delta + total) % total
	for i := range f.fields {
		if i == f.focus {
			f.fields[i].Focus()
		} else {
			f.fields[i].Blur()
		}
	}
}

func (f *formState) value(index int) string {
	if index < 0 || index >= len(f.fields) {
		return ""
	}
	return singleLine(f.fields[index].Value())
}

func (f *formState) motivation() string {
	if len(f.fields) == 0 {
		return ""
	}
	return singleLine(f.fields[len(f.fields)-1].Value())
}

// singleLine mantém o modelo da motivação em uma única linha, mesmo quando o
// campo visual quebra o texto ou recebe uma colagem com quebras.
func singleLine(value string) string {
	return strings.TrimSpace(flattenLines(value))
}

// flattenLines troca quebras de linha por espaços, preservando os espaços das
// pontas para que o usuário consiga digitar e apagar normalmente.
func flattenLines(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	return strings.ReplaceAll(value, "\n", " ")
}
