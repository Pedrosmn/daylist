// Package tui implementa a interface de terminal do daylist com Bubble Tea.
package tui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"daylist/internal/daylist"
	"daylist/internal/timer"
)

type mode int

const (
	modeMain mode = iota
	modeHelp
	modeForm
	modeHabitPick
	modeWeeklist
	modeConfirm
)

type pendingAction int

const (
	pendingNone pendingAction = iota
	pendingDiscardForm
)

const maxUndo = 30

type tickMsg time.Time

type createdMsg struct{ err error }
type weekCreatedMsg struct{ err error }
type openedMsg struct{ err error }
type notifyFinishedMsg struct{}

// Model é o modelo Bubble Tea do daylist. Ele não toca no disco diretamente:
// todas as operações passam pelo Backend.
type Model struct {
	backend Backend

	now    time.Time
	note   *daylist.Note
	items  []daylist.NoteItem
	cursor int

	timer *timer.State
	mode  mode
	err   string
	st    string

	form    *formState
	pending pendingAction
	confirm string

	habitChoices []daylist.Item
	habitCursor  int

	week       *daylist.Note
	weekItems  []daylist.NoteItem
	weekCursor int

	undo []string

	width  int
	height int
}

// New cria o modelo e carrega o estado inicial. Erros de leitura ficam em
// Model.Err e podem ser exibidos pela view.
func New(backend Backend) *Model {
	m := &Model{backend: backend, now: backend.Today()}
	m.reload()
	return m
}

// Init inicia o relógio do cabeçalho e do timer.
func (m *Model) Init() tea.Cmd { return tickCmd() }

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// Update implementa tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tickMsg:
		m.now = m.backend.Today()
		return m, tea.Batch(tickCmd(), m.checkThresholds())

	case createdMsg:
		m.mode = modeMain
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.st = "Daylist de hoje criada."
		}
		m.reload()
		return m, nil

	case weekCreatedMsg:
		m.mode = modeMain
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.st = "Weeklist atual criada."
		}
		m.reload()
		return m, nil

	case openedMsg:
		if msg.err != nil {
			m.err = "não foi possível abrir no Obsidian: " + msg.err.Error()
		}
		return m, nil

	case notifyFinishedMsg:
		return m, nil

	case tea.KeyMsg:
		switch m.mode {
		case modeHelp:
			m.mode = modeMain
			return m, nil
		case modeMain:
			return m.updateMain(msg)
		case modeForm:
			return m.updateForm(msg)
		case modeHabitPick:
			return m.updateHabitPick(msg)
		case modeWeeklist:
			return m.updateWeeklist(msg)
		case modeConfirm:
			return m.updateConfirm(msg)
		}
	}

	return m, nil
}

func (m *Model) updateMain(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.err = ""

	switch key.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "?":
		m.mode = modeHelp
	case "j", "down":
		m.moveCursor(1)
	case "k", "up":
		m.moveCursor(-1)
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		if len(m.items) > 0 {
			m.cursor = len(m.items) - 1
		}
	case " ", "enter":
		m.toggleSelected()
	case "s":
		m.startTimer()
	case "p":
		m.toggleTimer()
	case "x":
		m.stopTimer()
	case "e":
		m.editMotivation()
	case "a":
		m.openExtraTask()
	case "A":
		m.openExtraHabit()
	case "u":
		m.undoLast()
	case "n":
		return m, m.createDaylist()
	case "c":
		return m, m.createWeeklist()
	case "W":
		m.showWeeklist()
	case "o":
		return m, m.openDaylist()
	case "r":
		m.reload()
		m.st = "Daylist recarregada."
	}
	return m, nil
}

func (m *Model) updateForm(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.String() == "esc" {
		if m.form != nil && m.form.dirty() {
			m.pending = pendingDiscardForm
			m.confirm = "Descartar a edição não salva?"
			m.mode = modeConfirm
			return m, nil
		}
		m.cancelForm()
		return m, nil
	}

	cmd, submit := m.form.Update(key)
	if submit {
		return m, m.submitForm()
	}
	return m, cmd
}

func (m *Model) updateConfirm(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "s", "y", "enter":
		switch m.pending {
		case pendingDiscardForm:
			m.cancelForm()
		}
		m.pending = pendingNone
		m.confirm = ""
		m.mode = modeMain
	case "n", "esc", "q":
		m.pending = pendingNone
		m.confirm = ""
		if m.form != nil {
			m.mode = modeForm
		} else {
			m.mode = modeMain
		}
	}
	return m, nil
}

func (m *Model) cancelForm() {
	m.form = nil
	m.mode = modeMain
	m.st = "Edição cancelada."
}

func (m *Model) updateHabitPick(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc", "q":
		m.mode = modeMain
	case "j", "down":
		m.moveHabitCursor(1)
	case "k", "up":
		m.moveHabitCursor(-1)
	case " ", "enter":
		if len(m.habitChoices) == 0 {
			return m, nil
		}
		habit := m.habitChoices[m.habitCursor]
		m.form = newExtraHabitForm(habit)
		m.mode = modeForm
	}
	return m, nil
}

func (m *Model) updateWeeklist(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.err = ""

	switch key.String() {
	case "esc", "q", "W":
		m.mode = modeMain
	case "j", "down":
		m.moveWeekCursor(1)
	case "k", "up":
		m.moveWeekCursor(-1)
	case "g", "home":
		m.weekCursor = 0
	case "G", "end":
		if len(m.weekItems) > 0 {
			m.weekCursor = len(m.weekItems) - 1
		}
	case "c":
		if !m.backend.WeeklistExists() {
			return m, m.createWeeklist()
		}
	case " ", "enter":
		m.toggleAlways()
	}
	return m, nil
}

// --- carregamento ---

func (m *Model) reload() {
	m.now = m.backend.Today()
	m.note = nil
	m.items = nil

	if m.backend.DaylistExists() {
		note, err := m.backend.LoadNote()
		if err != nil {
			m.err = "não foi possível ler a daylist: " + err.Error()
		} else {
			m.note = note
			m.items = note.Items()
		}
	}
	m.clampCursor()

	state, err := m.backend.LoadTimer()
	if err != nil {
		m.err = "não foi possível ler o timer: " + err.Error()
	} else {
		m.timer = state
	}
}

func (m *Model) clampCursor() {
	if len(m.items) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.items) {
		m.cursor = len(m.items) - 1
	}
}

func (m *Model) moveCursor(delta int) {
	if len(m.items) == 0 {
		return
	}
	m.cursor += delta
	m.clampCursor()
}

func (m *Model) selected() (daylist.NoteItem, bool) {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return daylist.NoteItem{}, false
	}
	return m.items[m.cursor], true
}

func (m *Model) selectLine(line int) {
	for i, item := range m.items {
		if item.Line == line {
			m.cursor = i
			return
		}
	}
	m.clampCursor()
}

func (m *Model) moveHabitCursor(delta int) {
	if len(m.habitChoices) == 0 {
		return
	}
	m.habitCursor = (m.habitCursor + delta + len(m.habitChoices)) % len(m.habitChoices)
}

func (m *Model) moveWeekCursor(delta int) {
	if len(m.weekItems) == 0 {
		return
	}
	m.weekCursor += delta
	if m.weekCursor < 0 {
		m.weekCursor = 0
	}
	if m.weekCursor >= len(m.weekItems) {
		m.weekCursor = len(m.weekItems) - 1
	}
}

func (m *Model) selectWeekLine(line int) {
	for i, item := range m.weekItems {
		if item.Line == line {
			m.weekCursor = i
			return
		}
	}
}

// --- ações de edição ---

func (m *Model) pushUndo() {
	if m.note == nil {
		return
	}
	m.undo = append(m.undo, m.note.String())
	if len(m.undo) > maxUndo {
		m.undo = m.undo[1:]
	}
}

func (m *Model) undoLast() {
	if len(m.undo) == 0 {
		m.st = "Nada para desfazer."
		return
	}
	snapshot := m.undo[len(m.undo)-1]
	m.undo = m.undo[:len(m.undo)-1]

	note := daylist.ParseNote(snapshot)
	if err := m.backend.SaveNote(note); err != nil {
		m.err = "não foi possível desfazer: " + err.Error()
		return
	}
	m.reload()
	m.st = "Última edição desfeita."
}

func (m *Model) toggleSelected() {
	item, ok := m.selected()
	if !ok {
		return
	}

	m.pushUndo()
	if _, err := m.note.ToggleItem(item.Line); err != nil {
		m.err = err.Error()
		return
	}
	if err := m.backend.SaveNote(m.note); err != nil {
		m.err = "não foi possível gravar a daylist: " + err.Error()
		m.reload()
		return
	}

	m.reload()
	m.selectLine(item.Line)
	if item.Checked {
		m.st = "Item reaberto."
	} else {
		m.st = "Item concluído."
	}
}

func (m *Model) editMotivation() {
	item, ok := m.selected()
	if !ok {
		return
	}
	current := strings.ReplaceAll(item.Motivation, "\n", " ")
	m.form = newMotivationForm(item.Line, current)
	m.mode = modeForm
}

func (m *Model) openExtraTask() {
	if m.note == nil {
		m.err = "crie a daylist de hoje antes de adicionar um Extra."
		return
	}
	m.form = newExtraTaskForm()
	m.mode = modeForm
}

func (m *Model) openExtraHabit() {
	if m.note == nil {
		m.err = "crie a daylist de hoje antes de adicionar um Extra."
		return
	}

	habits, err := m.backend.WeeklistHabits()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			m.err = "weeklist atual não encontrada; pressione c para criá-la."
			return
		}
		m.err = "não foi possível ler a weeklist: " + err.Error()
		return
	}
	if len(habits) == 0 {
		m.st = "Nenhum hábito pendente na weeklist atual."
		return
	}

	m.habitChoices = habits
	m.habitCursor = 0
	m.mode = modeHabitPick
}

func (m *Model) submitForm() tea.Cmd {
	form := m.form
	if form == nil || m.note == nil {
		return nil
	}

	var (
		item daylist.NoteItem
		err  error
	)

	switch form.kind {
	case formExtraTask:
		desc := form.value(0)
		if desc == "" {
			form.err = "a descrição é obrigatória."
			return nil
		}
		m.pushUndo()
		item, err = m.note.AddExtraTask(daylist.ExtraTask{
			Time:       form.value(1),
			Flag:       form.flagValue(),
			Desc:       desc,
			Motivation: form.motivation(),
		})

	case formExtraHabit:
		m.pushUndo()
		item, err = m.note.AddExtraHabit(daylist.ExtraHabit{
			Category:   form.habit.Category,
			Time:       form.habit.Time,
			Flag:       form.habit.Flag,
			Desc:       form.habit.Desc,
			Motivation: form.motivation(),
		})

	case formMotivation:
		m.pushUndo()
		item, err = m.note.SetMotivation(form.line, form.value(0))
	}

	if err != nil {
		form.err = err.Error()
		return nil
	}

	if err := m.backend.SaveNote(m.note); err != nil {
		m.err = "não foi possível gravar a daylist: " + err.Error()
		m.form = nil
		m.mode = modeMain
		m.reload()
		return nil
	}

	m.form = nil
	m.mode = modeMain
	m.reload()
	m.selectLine(item.Line)

	switch form.kind {
	case formExtraTask:
		m.st = "Extra task adicionada."
	case formExtraHabit:
		m.st = "Hábito Extra adicionado em " + form.habit.Category + "."
	default:
		if strings.TrimSpace(form.value(0)) == "" {
			m.st = "Motivação removida."
		} else {
			m.st = "Motivação atualizada."
		}
	}
	return nil
}

// --- timer ---

func (m *Model) startTimer() {
	if m.note == nil {
		m.err = "crie a daylist de hoje antes de iniciar o timer."
		return
	}
	if m.timer != nil {
		m.st = "Já tem um timer rodando: " + m.timer.Desc
		return
	}

	item, ok := m.selected()
	if !ok {
		m.err = "selecione um item para iniciar o timer."
		return
	}
	if item.Checked {
		m.err = "item já concluído; reabra antes de iniciar o timer."
		return
	}
	if item.Timer != "" {
		m.err = "este item já tem tempo registrado."
		return
	}

	state := timer.New(item.Raw, m.backend.DaylistRel())
	if err := m.backend.SaveTimer(state); err != nil {
		m.err = "não foi possível iniciar o timer: " + err.Error()
		return
	}

	m.timer = state
	m.st = "Timer iniciado: " + state.Desc
	if item.Motivation != "" {
		m.st += " — " + strings.ReplaceAll(item.Motivation, "\n", " ")
	}
}

func (m *Model) toggleTimer() {
	if m.timer == nil {
		m.st = "Nenhum timer rodando."
		return
	}

	resumed := m.timer.Toggle()
	if err := m.backend.SaveTimer(m.timer); err != nil {
		m.err = "não foi possível gravar o timer: " + err.Error()
		return
	}

	if resumed {
		m.st = "Timer retomado: " + m.timer.Desc
		return
	}
	m.st = fmt.Sprintf("Timer pausado: %s — %dmin acumulados", m.timer.Desc, m.timer.TotalMs()/60_000)
}

func (m *Model) stopTimer() {
	if m.timer == nil {
		m.st = "Nenhum timer rodando."
		return
	}

	desc := m.timer.Desc
	mins, err := m.backend.FinishTimer(m.timer)
	switch {
	case errors.Is(err, timer.ErrAlreadyRecorded):
		_ = m.backend.ClearTimer()
		m.timer = nil
		m.reload()
		m.st = "O tempo já estava registrado; estado do timer removido."
		return
	case errors.Is(err, timer.ErrLineNotFound), errors.Is(err, timer.ErrAmbiguousLine):
		m.err = err.Error() + " O estado foi mantido; ajuste a daylist e tente de novo."
		return
	case err != nil:
		m.err = "não foi possível parar o timer: " + err.Error()
		return
	}

	if err := m.backend.ClearTimer(); err != nil {
		m.err = "tempo gravado, mas não foi possível remover o estado do timer: " + err.Error()
	}
	m.timer = nil
	m.reload()
	m.st = fmt.Sprintf("Timer parado: %s — %dmin", desc, mins)
}

func (m *Model) checkThresholds() tea.Cmd {
	if m.timer == nil {
		return nil
	}
	message, changed := m.timer.CheckThresholds(m.timer.TotalMs())
	if !changed {
		return nil
	}
	_ = m.backend.SaveTimer(m.timer)
	m.st = message
	return func() tea.Msg {
		_ = m.backend.Notify("Daylist", message)
		return notifyFinishedMsg{}
	}
}

// --- daylist e weeklist ---

type execFunc func() error

func (f execFunc) Run() error          { return f() }
func (f execFunc) SetStdin(io.Reader)  {}
func (f execFunc) SetStdout(io.Writer) {}
func (f execFunc) SetStderr(io.Writer) {}

func (m *Model) createDaylist() tea.Cmd {
	if m.backend.DaylistExists() {
		m.st = "A daylist de hoje já existe."
		return nil
	}
	m.st = "Criando a daylist no wizard..."
	return tea.Exec(execFunc(m.backend.CreateDaylist), func(err error) tea.Msg {
		return createdMsg{err: err}
	})
}

func (m *Model) createWeeklist() tea.Cmd {
	if m.backend.WeeklistExists() {
		m.st = "A weeklist desta semana já existe."
		return nil
	}
	return tea.Exec(execFunc(m.backend.CreateWeeklist), func(err error) tea.Msg {
		return weekCreatedMsg{err: err}
	})
}

func (m *Model) openDaylist() tea.Cmd {
	if !m.backend.DaylistExists() {
		m.err = "a daylist de hoje ainda não existe."
		return nil
	}
	return func() tea.Msg { return openedMsg{err: m.backend.OpenDaylist()} }
}

func (m *Model) showWeeklist() {
	if !m.backend.WeeklistExists() {
		m.st = "Weeklist desta semana não existe; pressione c para criá-la."
		return
	}

	note, err := m.backend.LoadWeeklist()
	if err != nil {
		m.err = "não foi possível ler a weeklist: " + err.Error()
		return
	}

	m.week = note
	m.weekItems = note.Items()
	m.weekCursor = 0
	for i, item := range m.weekItems {
		if item.Section == "Hábitos" {
			m.weekCursor = i
			break
		}
	}
	m.mode = modeWeeklist
}

func (m *Model) toggleAlways() {
	if m.week == nil || len(m.weekItems) == 0 {
		return
	}
	item := m.weekItems[m.weekCursor]
	if item.Section != "Hábitos" {
		m.st = "Apenas hábitos usam #always."
		return
	}

	updated, err := m.week.ToggleAlways(item.Line)
	if err != nil {
		m.err = err.Error()
		return
	}
	if err := m.backend.SaveWeeklist(m.week); err != nil {
		m.err = "não foi possível gravar a weeklist: " + err.Error()
		return
	}

	m.weekItems = m.week.Items()
	m.selectWeekLine(item.Line)
	if updated.Always {
		m.st = "Hábito marcado como #always."
	} else {
		m.st = "Marcador #always removido."
	}
}
