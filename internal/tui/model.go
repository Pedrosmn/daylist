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
	"daylist/internal/planner"
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
	modePreview
	modeClose
)

type pendingAction int

const (
	pendingNone pendingAction = iota
	pendingDiscardForm
)

const maxUndo = 30

type tickMsg time.Time

type plannedMsg struct {
	plan   *planner.Plan
	target time.Time
	err    error
}
type weekCreatedMsg struct{ err error }
type openedMsg struct{ err error }
type notifyFinishedMsg struct{}

// Model é o modelo Bubble Tea do daylist. Ele não toca no disco diretamente:
// todas as operações passam pelo Backend.
type Model struct {
	backend Backend

	now    time.Time // horário atual, atualizado a cada tick
	day    time.Time // data da daylist carregada em note (00:00 local)
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

	preview *planner.Plan
	summary daylist.Summary

	habitChoices []daylist.Item
	habitCursor  int

	week       *daylist.Note
	weekPath   string
	weekItems  []daylist.NoteItem
	weekCursor int

	undo []string
	// Estado por data: cursor e undo pertencem à daylist visualizada. Guardar
	// por chave evita que voltar para hoje perca a seleção ou desfaça uma
	// edição no arquivo errado.
	undoByDay   map[string][]string
	cursorByDay map[string]int

	width  int
	height int
}

// New cria o modelo e carrega o estado inicial. Erros de leitura ficam em
// Model.Err e podem ser exibidos pela view.
func New(backend Backend) *Model {
	now := backend.Today()
	m := &Model{
		backend:     backend,
		now:         now,
		day:         dateOnly(now),
		undoByDay:   map[string][]string{},
		cursorByDay: map[string]int{},
	}
	m.reload()
	return m
}

// Init inicia o relógio do cabeçalho e do timer.
func (m *Model) Init() tea.Cmd {
	if m.timer != nil {
		_ = m.backend.StartTimerWatch()
	}
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// Update implementa tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if m.form != nil {
			m.form.layout(m.formContentWidth())
		}
		return m, nil

	case tickMsg:
		now := m.backend.Today()
		oldToday := dateOnly(m.now)
		newToday := dateOnly(now)
		if !sameDay(oldToday, newToday) {
			if sameDay(m.day, oldToday) {
				// Estava vendo o dia que era hoje: acompanha a virada.
				m.switchDay(newToday)
				m.st = "Novo dia: daylist de hoje recarregada."
				if m.mode == modeWeeklist || m.mode == modeForm || m.mode == modeHabitPick ||
					m.mode == modeConfirm || m.mode == modePreview || m.mode == modeClose {
					m.form = nil
					m.week = nil
					m.habitChoices = nil
					m.preview = nil
					m.mode = modeMain
				}
			} else {
				// Estava vendo ontem (ou outra data): mantém a data absoluta
				// carregada e apenas atualiza o relógio. Isso evita que uma
				// nota de ontem seja gravada no arquivo do novo dia.
				m.now = now
				m.reload()
				m.st = "O dia virou; você continua na " + m.daylistPhrase() + "."
			}
		} else {
			m.now = now
		}
		return m, tea.Batch(tickCmd(), m.checkThresholds())

	case plannedMsg:
		m.mode = modeMain
		switch {
		case errors.Is(msg.err, planner.ErrCanceled):
			m.st = "Criação cancelada."
		case errors.Is(msg.err, planner.ErrNoItems):
			m.st = "Nenhum item selecionado."
		case errors.Is(msg.err, planner.ErrDaylistExists):
			m.st = m.daylistExistsMessage(msg.target)
		case msg.err != nil:
			m.err = msg.err.Error()
		case msg.plan != nil:
			m.preview = msg.plan
			m.mode = modePreview
			m.st = ""
			m.err = ""
		}
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
		case modePreview:
			return m.updatePreview(msg)
		case modeClose:
			return m.updateClose(msg)
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
	case "y":
		m.toggleDay()
	case "t", "esc":
		m.goToday()
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
	case "m":
		m.editMotivation()
	case "a":
		m.openExtraTask()
	case "A":
		m.openExtraHabit()
	case "u":
		m.undoLast()
	case "n":
		return m, m.planDaylist(m.backend.Today())
	case "N":
		return m, m.planDaylist(m.backend.Today().AddDate(0, 0, 1))
	case "F":
		m.openClose()
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
		m.form.layout(m.formContentWidth())
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
	if m.day.IsZero() {
		m.day = dateOnly(m.now)
	}
	m.note = nil
	m.items = nil

	if m.backend.DaylistExistsOn(m.day) {
		note, err := m.backend.LoadNoteOn(m.day)
		if err != nil {
			m.err = "não foi possível ler a daylist: " + err.Error()
		} else {
			m.note = note
			m.items = note.Items()
		}
	}
	m.clampCursor()

	// O timer é único e pertence ao dia de hoje. Ao visualizar ontem, o
	// banner e os controles ficam fora da tela.
	m.timer = nil
	if !m.isToday() {
		return
	}
	state, err := m.backend.LoadTimer()
	if err != nil {
		m.err = "não foi possível ler o timer: " + err.Error()
	} else {
		m.timer = state
	}
}

// isToday informa se a daylist carregada é a do relógio atual.
func (m *Model) isToday() bool {
	return sameDay(m.day, dateOnly(m.now))
}

func dayKey(t time.Time) string { return t.Format("2006-01-02") }

// saveDayMemory guarda cursor e undo da data que está saindo da tela.
func (m *Model) saveDayMemory() {
	if m.day.IsZero() {
		return
	}
	if m.undoByDay == nil {
		m.undoByDay = map[string][]string{}
	}
	if m.cursorByDay == nil {
		m.cursorByDay = map[string]int{}
	}
	key := dayKey(m.day)
	m.cursorByDay[key] = m.cursor
	m.undoByDay[key] = append([]string(nil), m.undo...)
}

// loadDayMemory restaura cursor e undo da data que está entrando na tela.
func (m *Model) loadDayMemory() {
	key := dayKey(m.day)
	m.cursor = m.cursorByDay[key]
	m.undo = append([]string(nil), m.undoByDay[key]...)
}

// switchDay troca a data visualizada preservando o estado de cada dia.
func (m *Model) switchDay(day time.Time) {
	m.saveDayMemory()
	m.day = dateOnly(day)
	m.loadDayMemory()
	m.reload()
}

// toggleDay alterna entre hoje e ontem. Se a tela estiver em uma data mais
// antiga (por exemplo, ontem antes da meia-noite), y volta para hoje.
func (m *Model) toggleDay() {
	today := dateOnly(m.backend.Today())
	if sameDay(m.day, today) {
		m.switchDay(today.AddDate(0, 0, -1))
		if m.note == nil {
			m.st = "A " + m.daylistPhrase() + " ainda não existe."
		} else {
			m.st = "Mostrando a " + m.daylistPhrase() + "."
		}
		return
	}
	m.goToday()
}

// goToday volta explicitamente para a daylist de hoje.
func (m *Model) goToday() {
	now := m.backend.Today()
	today := dateOnly(now)
	if sameDay(m.day, today) {
		m.now = now
		return
	}
	m.switchDay(today)
	if m.note == nil {
		m.st = "A daylist de hoje ainda não existe."
	} else {
		m.st = "De volta à daylist de hoje."
	}
}

// daylistPhrase descreve a data visualizada em mensagens ("daylist de hoje",
// "daylist de ontem" ou a data absoluta).
func (m *Model) daylistPhrase() string {
	today := dateOnly(m.now)
	switch {
	case sameDay(m.day, today):
		return "daylist de hoje"
	case sameDay(m.day, today.AddDate(0, 0, -1)):
		return "daylist de ontem"
	default:
		return "daylist de " + m.day.Format("02/01/2006")
	}
}

// viewDateWord descreve a data visualizada em rótulos curtos.
func (m *Model) viewDateWord() string {
	today := dateOnly(m.now)
	switch {
	case sameDay(m.day, today):
		return "hoje"
	case sameDay(m.day, today.AddDate(0, 0, -1)):
		return "ontem"
	default:
		return "de " + m.day.Format("02/01/2006")
	}
}

// weeklistPhrase descreve a weeklist correspondente à data visualizada.
func (m *Model) weeklistPhrase() string {
	if m.isToday() {
		return "weeklist atual"
	}
	if m.viewDateWord() == "ontem" {
		return "weeklist da semana de ontem"
	}
	return "weeklist da semana " + m.viewDateWord()
}

// ensureViewedDay garante que a data visualizada continua sendo a mesma nota
// carregada. Se o dia virou enquanto a tela mostrava "hoje", recarrega o novo
// dia e devolve false para o chamador abortar a ação. Se a tela mostrava ontem,
// mantém a data absoluta e permite a edição no arquivo correto.
func (m *Model) ensureViewedDay() bool {
	now := m.backend.Today()
	oldToday := dateOnly(m.now)
	newToday := dateOnly(now)
	if !sameDay(oldToday, newToday) {
		if sameDay(m.day, oldToday) {
			m.switchDay(newToday)
			m.st = "O dia virou; a daylist de hoje foi recarregada. Repita a ação."
			return false
		}
		m.now = now
		return true
	}
	m.now = now
	return true
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
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
	if !m.ensureViewedDay() {
		return
	}
	if len(m.undo) == 0 {
		m.st = "Nada para desfazer."
		return
	}
	snapshot := m.undo[len(m.undo)-1]
	m.undo = m.undo[:len(m.undo)-1]

	note := daylist.ParseNote(snapshot)
	if err := m.backend.SaveNoteOn(m.day, note); err != nil {
		m.err = "não foi possível desfazer: " + err.Error()
		return
	}
	m.reload()
	m.st = "Última edição desfeita."
}

func (m *Model) toggleSelected() {
	if !m.ensureViewedDay() {
		return
	}
	item, ok := m.selected()
	if !ok {
		return
	}

	m.pushUndo()
	if _, err := m.note.ToggleItem(item.Line); err != nil {
		m.err = err.Error()
		return
	}
	if err := m.backend.SaveNoteOn(m.day, m.note); err != nil {
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
	if !m.ensureViewedDay() {
		return
	}
	if !m.isToday() {
		m.err = "edição de motivação disponível apenas na daylist de hoje."
		return
	}
	item, ok := m.selected()
	if !ok {
		return
	}
	current := strings.ReplaceAll(item.Motivation, "\n", " ")
	m.form = newMotivationForm(item.Line, current, itemText(item.Time, item.Flag, item.Desc))
	m.form.layout(m.formContentWidth())
	m.mode = modeForm
}

func (m *Model) openExtraTask() {
	if !m.ensureViewedDay() {
		return
	}
	if m.note == nil {
		m.err = "crie a " + m.daylistPhrase() + " antes de adicionar um Extra."
		return
	}
	m.form = newExtraTaskForm()
	m.form.layout(m.formContentWidth())
	m.mode = modeForm
}

func (m *Model) openExtraHabit() {
	if !m.ensureViewedDay() {
		return
	}
	if m.note == nil {
		m.err = "crie a " + m.daylistPhrase() + " antes de adicionar um Extra."
		return
	}

	habits, err := m.backend.WeeklistHabitsOn(m.day)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			m.err = m.weeklistPhrase() + " não encontrada"
			if m.isToday() {
				m.err += "; pressione c para criá-la."
			} else {
				m.err += "."
			}
			return
		}
		m.err = "não foi possível ler a weeklist: " + err.Error()
		return
	}
	if len(habits) == 0 {
		m.st = "Nenhum hábito pendente na " + m.weeklistPhrase() + "."
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
	if !m.ensureViewedDay() {
		m.form = nil
		m.mode = modeMain
		return nil
	}
	if form.kind == formMotivation && !m.isToday() {
		form.err = "edição de motivação disponível apenas na daylist de hoje."
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
		item, err = m.note.SetMotivation(form.line, form.motivation())
	}

	if err != nil {
		form.err = err.Error()
		return nil
	}

	if err := m.backend.SaveNoteOn(m.day, m.note); err != nil {
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
		if form.motivation() == "" {
			m.st = "Motivação removida."
		} else {
			m.st = "Motivação atualizada."
		}
	}
	return nil
}

// --- timer ---

// ensureTimerDay recusa ações de timer em qualquer data que não seja hoje. O
// timer é único no vault e continua associado à daylist em que foi iniciado.
func (m *Model) ensureTimerDay() bool {
	if !m.ensureViewedDay() {
		return false
	}
	if !m.isToday() {
		m.err = "timer disponível apenas na daylist de hoje."
		return false
	}
	return true
}

func (m *Model) startTimer() {
	if !m.ensureTimerDay() {
		return
	}
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

	state := timer.New(item.Raw, m.backend.DaylistRelOn(m.day))
	if err := m.backend.SaveTimer(state); err != nil {
		m.err = "não foi possível iniciar o timer: " + err.Error()
		return
	}
	if err := m.backend.StartTimerWatch(); err != nil && !errors.Is(err, timer.ErrNoTimer) {
		m.err = "timer iniciado, mas não foi possível iniciar as notificações: " + err.Error()
	}

	m.timer = state
	m.st = "Timer iniciado: " + state.Desc
	if item.Motivation != "" {
		m.st += " — " + strings.ReplaceAll(item.Motivation, "\n", " ")
	}
}

func (m *Model) toggleTimer() {
	if !m.ensureTimerDay() {
		return
	}
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
	if !m.ensureTimerDay() {
		return
	}
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
	state, message, changed, err := m.backend.ClaimThresholds(m.now)
	if err != nil {
		m.err = "não foi possível verificar os limites do timer: " + err.Error()
		return nil
	}
	if state != nil {
		m.timer = state
	}
	if !changed {
		return nil
	}
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

func (m *Model) planDaylist(target time.Time) tea.Cmd {
	if m.backend.DaylistExistsOn(target) {
		m.st = m.daylistExistsMessage(target)
		return nil
	}

	var plan *planner.Plan
	m.st = fmt.Sprintf("Criando a daylist de %s no wizard...", m.dateWord(target))
	return tea.Exec(execFunc(func() error {
		var err error
		plan, err = m.backend.PlanDaylist(target)
		return err
	}), func(err error) tea.Msg {
		return plannedMsg{plan: plan, target: target, err: err}
	})
}

// dateWord descreve a data alvo em relação ao dia atual do TUI.
func (m *Model) dateWord(t time.Time) string {
	today := m.backend.Today()
	switch t.Format("2006-01-02") {
	case today.Format("2006-01-02"):
		return "hoje"
	case today.AddDate(0, 0, 1).Format("2006-01-02"):
		return "amanhã"
	default:
		return t.Format("2006-01-02")
	}
}

func (m *Model) daylistExistsMessage(target time.Time) string {
	msg := fmt.Sprintf("A daylist de %s já existe.", m.dateWord(target))
	if m.dateWord(target) == "hoje" {
		msg += " Pressione N para criar a de amanhã."
	}
	return msg
}

func (m *Model) openClose() {
	if !m.ensureViewedDay() {
		return
	}
	if m.note == nil {
		m.err = "crie a " + m.daylistPhrase() + " antes de fechar o dia."
		return
	}
	m.summary = daylist.Summarize(m.items)
	m.mode = modeClose
}

func (m *Model) updateClose(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc", "q", "F", "enter":
		m.mode = modeMain
	}
	return m, nil
}

func (m *Model) updatePreview(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc", "q", "n":
		m.preview = nil
		m.mode = modeMain
		m.st = "Criação cancelada."
	case "enter", "s":
		if m.preview == nil {
			m.mode = modeMain
			return m, nil
		}
		if err := m.backend.WriteDaylist(m.preview); err != nil {
			m.err = "não foi possível criar a daylist: " + err.Error()
			return m, nil
		}
		target := m.preview.Target
		m.preview = nil
		m.mode = modeMain
		m.reload()
		m.st = fmt.Sprintf("Daylist de %s criada.", target.Format("2006-01-02"))
	}
	return m, nil
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
	if !m.ensureViewedDay() {
		return nil
	}
	if !m.backend.DaylistExistsOn(m.day) {
		m.err = "a " + m.daylistPhrase() + " ainda não existe."
		return nil
	}
	return func() tea.Msg { return openedMsg{err: m.backend.OpenDaylistOn(m.day)} }
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
	m.weekPath = m.backend.WeeklistPath()
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
	if m.weekPath != "" && m.weekPath != m.backend.WeeklistPath() {
		m.week = nil
		m.weekItems = nil
		m.mode = modeMain
		m.st = "A semana virou; abra a weeklist atual novamente."
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
