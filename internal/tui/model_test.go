package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"daylist/internal/daylist"
	"daylist/internal/timer"
)

type fakeBackend struct {
	now         time.Time
	noteContent string
	noteExists  bool
	weekContent string
	weekExists  bool
	habits      []daylist.Item

	timer     *timer.State
	timerSave int
	finishMin int64
	finishErr error
	cleared   int

	created     int
	createdWeek int
	opened      int
	notified    []string

	savedNote string
	noteSaves int
	savedWeek string
	weekSaves int
}

func newFake(note string) *fakeBackend {
	return &fakeBackend{
		now:         time.Date(2026, 11, 5, 10, 0, 0, 0, time.Local),
		noteContent: note,
		noteExists:  true,
	}
}

func (f *fakeBackend) Today() time.Time    { return f.now }
func (f *fakeBackend) DaylistPath() string { return "/vault/days/today.md" }
func (f *fakeBackend) DaylistRel() string  { return "days/today.md" }
func (f *fakeBackend) DaylistExists() bool { return f.noteExists }

func (f *fakeBackend) LoadNote() (*daylist.Note, error) {
	if !f.noteExists {
		return nil, errors.New("daylist inexistente")
	}
	return daylist.ParseNote(f.noteContent), nil
}

func (f *fakeBackend) SaveNote(note *daylist.Note) error {
	f.savedNote = note.String()
	f.noteContent = f.savedNote
	f.noteSaves++
	return nil
}

func (f *fakeBackend) WeeklistPath() string { return "/vault/weeklist.md" }
func (f *fakeBackend) WeeklistExists() bool { return f.weekExists }

func (f *fakeBackend) LoadWeeklist() (*daylist.Note, error) {
	if !f.weekExists {
		return nil, errors.New("weeklist inexistente")
	}
	return daylist.ParseNote(f.weekContent), nil
}

func (f *fakeBackend) SaveWeeklist(note *daylist.Note) error {
	f.savedWeek = note.String()
	f.weekContent = f.savedWeek
	f.weekSaves++
	return nil
}

func (f *fakeBackend) WeeklistHabits() ([]daylist.Item, error) { return f.habits, nil }

func (f *fakeBackend) CreateDaylist() error {
	f.created++
	f.noteExists = true
	return nil
}

func (f *fakeBackend) CreateWeeklist() error {
	f.createdWeek++
	f.weekExists = true
	return nil
}

func (f *fakeBackend) OpenDaylist() error {
	f.opened++
	return nil
}

func (f *fakeBackend) LoadTimer() (*timer.State, error) { return f.timer, nil }

func (f *fakeBackend) SaveTimer(state *timer.State) error {
	f.timer = state
	f.timerSave++
	return nil
}

func (f *fakeBackend) FinishTimer(*timer.State) (int64, error) {
	if f.finishErr != nil {
		return 0, f.finishErr
	}
	return f.finishMin, nil
}

func (f *fakeBackend) ClearTimer() error {
	f.cleared++
	f.timer = nil
	return nil
}

func (f *fakeBackend) Notify(title, body string) error {
	f.notified = append(f.notified, title+": "+body)
	return nil
}

const testNote = `2026-11-05 08:00

# Tasks

- [ ] 40 min (I) - Escrever relatório
> terminar antes do almoço
- [x] comprar café
---
# Hábitos

## Estudos
- [ ] 5 min (U) - AWS
- [x] 5 min () - Fio dental
---
# Extra

- [ ] 15 min (I) - Task extra
`

func keyRunes(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func keyType(t tea.KeyType) tea.KeyMsg { return tea.KeyMsg{Type: t} }

func updateModel(t *testing.T, m *Model, msg tea.Msg) *Model {
	t.Helper()
	next, _ := m.Update(msg)
	got, ok := next.(*Model)
	if !ok {
		t.Fatalf("Update devolveu %T, want *Model", next)
	}
	return got
}

func TestNewLoadsDaylistAndTimer(t *testing.T) {
	fake := newFake(testNote)
	fake.timer = &timer.State{
		Desc:    "Escrever relatório",
		Start:   time.Now().Add(-12 * time.Minute).UnixMilli(),
		Started: time.Now().Add(-12 * time.Minute).UnixMilli(),
	}

	m := New(fake)

	if len(m.items) != 5 {
		t.Fatalf("itens = %d, want 5", len(m.items))
	}
	view := m.View()
	for _, want := range []string{
		"Escrever relatório", "terminar antes do almoço", "AWS", "Task extra",
		"# Tasks", "# Hábitos", "## Estudos", "# Extra", "Timer rodando",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("view não contém %q:\n%s", want, view)
		}
	}
}

func TestMissingDaylistView(t *testing.T) {
	fake := newFake("")
	fake.noteExists = false

	m := New(fake)
	view := m.View()
	if !strings.Contains(view, "ainda não existe") || !strings.Contains(view, "n  criar") {
		t.Errorf("view sem aviso de daylist ausente:\n%s", view)
	}
}

func TestToggleAndUndo(t *testing.T) {
	fake := newFake("# Tasks\n- [ ] 10 min (I) - tarefa\n")
	m := New(fake)

	m = updateModel(t, m, keyRunes(" "))
	if !strings.Contains(fake.savedNote, "- [x] 10 min (I) - tarefa") {
		t.Fatalf("item não foi concluído: %q", fake.savedNote)
	}
	if !strings.Contains(m.st, "concluído") {
		t.Errorf("status = %q, want concluído", m.st)
	}

	m = updateModel(t, m, keyRunes("u"))
	if !strings.Contains(fake.savedNote, "- [ ] 10 min (I) - tarefa") {
		t.Errorf("undo não restaurou o item: %q", fake.savedNote)
	}
}

func TestAddExtraTaskForm(t *testing.T) {
	fake := newFake("# Tasks\n")
	m := New(fake)

	m = updateModel(t, m, keyRunes("a"))
	if m.mode != modeForm {
		t.Fatalf("mode = %v, want form", m.mode)
	}

	m = updateModel(t, m, keyRunes("Nova task"))
	m = updateModel(t, m, keyType(tea.KeyTab))
	m = updateModel(t, m, keyRunes("15 min"))
	m = updateModel(t, m, keyType(tea.KeyTab))
	m = updateModel(t, m, keyType(tea.KeyRight)) // Importante
	m = updateModel(t, m, keyType(tea.KeyRight)) // Urgente
	m = updateModel(t, m, keyType(tea.KeyTab))
	m = updateModel(t, m, keyRunes("porque sim"))
	m = updateModel(t, m, keyType(tea.KeyEnter))

	if m.mode != modeMain {
		t.Fatalf("mode = %v, want main; err=%q", m.mode, m.err)
	}
	if !strings.Contains(fake.savedNote, "# Extra") ||
		!strings.Contains(fake.savedNote, "- [ ] 15 min (U) - Nova task") ||
		!strings.Contains(fake.savedNote, "> porque sim") {
		t.Errorf("Extra task não foi gravada:\n%s", fake.savedNote)
	}
}

func TestDiscardFormAsksForConfirmation(t *testing.T) {
	fake := newFake("# Tasks\n")
	m := New(fake)

	m = updateModel(t, m, keyRunes("a"))
	m = updateModel(t, m, keyRunes("algo digitado"))
	m = updateModel(t, m, keyType(tea.KeyEsc))
	if m.mode != modeConfirm {
		t.Fatalf("mode = %v, want confirm", m.mode)
	}
	if !strings.Contains(m.View(), "Descartar") {
		t.Errorf("view de confirmação inesperada:\n%s", m.View())
	}

	m = updateModel(t, m, keyRunes("n"))
	if m.mode != modeForm || m.form == nil {
		t.Fatalf("cancelar a confirmação deveria voltar ao form: mode=%v", m.mode)
	}

	m = updateModel(t, m, keyType(tea.KeyEsc))
	m = updateModel(t, m, keyRunes("s"))
	if m.mode != modeMain || m.form != nil {
		t.Errorf("confirmação não descartou o form: mode=%v form=%+v", m.mode, m.form)
	}
}

func TestAddExtraTaskRequiresDescription(t *testing.T) {
	fake := newFake("# Tasks\n")
	m := New(fake)

	m = updateModel(t, m, keyRunes("a"))
	m = updateModel(t, m, keyType(tea.KeyTab))
	m = updateModel(t, m, keyType(tea.KeyTab))
	m = updateModel(t, m, keyType(tea.KeyTab))
	m = updateModel(t, m, keyType(tea.KeyEnter))

	if m.mode != modeForm {
		t.Fatalf("mode = %v, want form", m.mode)
	}
	if m.form == nil || !strings.Contains(m.form.err, "obrigatória") {
		t.Errorf("erro de descrição ausente: %+v", m.form)
	}
	if fake.noteSaves != 0 {
		t.Errorf("nada deveria ser gravado, saves=%d", fake.noteSaves)
	}
}

func TestEditMotivation(t *testing.T) {
	fake := newFake("# Tasks\n- [ ] 10 min (I) - tarefa\n> antiga\n")
	m := New(fake)

	m = updateModel(t, m, keyRunes("e"))
	if m.mode != modeForm || m.form == nil {
		t.Fatalf("mode = %v, form = %+v", m.mode, m.form)
	}
	m.form.inputs[0].SetValue("nova motivação")
	m = updateModel(t, m, keyType(tea.KeyEnter))

	if !strings.Contains(fake.savedNote, "> nova motivação") ||
		strings.Contains(fake.savedNote, "> antiga") {
		t.Errorf("motivação não foi atualizada:\n%s", fake.savedNote)
	}
}

func TestTimerStartPauseStop(t *testing.T) {
	fake := newFake("# Tasks\n- [ ] 10 min (I) - tarefa\n")
	fake.finishMin = 7
	m := New(fake)

	m = updateModel(t, m, keyRunes("s"))
	if fake.timer == nil || fake.timerSave != 1 {
		t.Fatalf("timer não iniciado: %+v", fake.timer)
	}
	if !strings.Contains(m.st, "Timer iniciado") {
		t.Errorf("status = %q", m.st)
	}

	m = updateModel(t, m, keyRunes("p"))
	if fake.timer == nil || !fake.timer.Paused {
		t.Fatalf("timer não pausado: %+v", fake.timer)
	}

	m = updateModel(t, m, keyRunes("x"))
	if fake.cleared != 1 || m.timer != nil {
		t.Errorf("timer não foi parado: cleared=%d timer=%+v", fake.cleared, m.timer)
	}
	if !strings.Contains(m.st, "Timer parado") || !strings.Contains(m.st, "7min") {
		t.Errorf("status = %q", m.st)
	}
}

func TestTimerThresholdsAreSharedAndNotifiedOnce(t *testing.T) {
	fake := newFake("# Tasks\n")
	fake.timer = &timer.State{
		Desc:       "tarefa",
		Paused:     true,
		Elapsed:    5 * time.Minute.Milliseconds(),
		MinMinutes: 5,
		MaxMinutes: 15,
	}
	m := New(fake)

	cmd := m.checkThresholds()
	if !strings.Contains(m.st, "mínimo atingido") {
		t.Fatalf("status = %q", m.st)
	}
	if cmd == nil {
		t.Fatal("notificação deveria ser agendada")
	}
	if msg := cmd(); msg == nil {
		t.Fatal("comando de notificação devolveu nil")
	}
	if len(fake.notified) != 1 || !strings.Contains(fake.notified[0], "mínimo atingido") {
		t.Errorf("notificações = %q", fake.notified)
	}

	m.checkThresholds()
	if len(fake.notified) != 1 {
		t.Errorf("notificação repetida: %q", fake.notified)
	}
}

func TestWeeklistToggleAlwaysOnlyOnHabits(t *testing.T) {
	fake := newFake("# Tasks\n")
	fake.weekExists = true
	fake.weekContent = `# Tasks
- [ ] tarefa
---
# Hábitos
## Rotina
- [ ] 5 min () - Fio dental
`
	m := New(fake)

	m = updateModel(t, m, keyRunes("W"))
	if m.mode != modeWeeklist {
		t.Fatalf("mode = %v, want weeklist", m.mode)
	}
	if got := m.weekItems[m.weekCursor].Desc; got != "Fio dental" {
		t.Fatalf("cursor inicial = %q, want Fio dental", got)
	}

	m = updateModel(t, m, keyRunes(" "))
	if !strings.Contains(fake.savedWeek, "Fio dental #always") {
		t.Errorf("#always não foi adicionado:\n%s", fake.savedWeek)
	}
	if !strings.Contains(m.st, "#always") {
		t.Errorf("status = %q", m.st)
	}

	m = updateModel(t, m, keyRunes(" "))
	if strings.Contains(fake.savedWeek, "#always") {
		t.Errorf("#always não foi removido:\n%s", fake.savedWeek)
	}

	m.weekCursor = 0
	m = updateModel(t, m, keyRunes(" "))
	if !strings.Contains(m.st, "Apenas hábitos") {
		t.Errorf("status = %q", m.st)
	}
}

func TestAddExtraHabitFromWeeklist(t *testing.T) {
	fake := newFake("# Tasks\n")
	fake.weekExists = true
	fake.habits = []daylist.Item{{
		Section: "Hábitos", Category: "Rotina", Time: "5 min", Desc: "Fio dental",
	}}
	m := New(fake)

	m = updateModel(t, m, keyRunes("A"))
	if m.mode != modeHabitPick {
		t.Fatalf("mode = %v, want habit pick", m.mode)
	}
	m = updateModel(t, m, keyType(tea.KeyEnter))
	if m.mode != modeForm {
		t.Fatalf("mode = %v, want form", m.mode)
	}
	m = updateModel(t, m, keyRunes("todos os dias"))
	m = updateModel(t, m, keyType(tea.KeyEnter))

	if !strings.Contains(fake.savedNote, "## Rotina") ||
		!strings.Contains(fake.savedNote, "5 min - Fio dental") ||
		!strings.Contains(fake.savedNote, "> todos os dias") {
		t.Errorf("hábito Extra não foi gravado:\n%s", fake.savedNote)
	}
}

func TestCreateDaylistWhenMissing(t *testing.T) {
	fake := newFake("")
	fake.noteExists = false
	m := New(fake)

	m = updateModel(t, m, keyRunes("n"))
	if m.st == "" {
		t.Error("status de criação não foi definido")
	}

	fake.noteExists = true
	m = updateModel(t, m, keyRunes("n"))
	if !strings.Contains(m.st, "já existe") {
		t.Errorf("status = %q", m.st)
	}
}

func TestHelpModeReturnsOnAnyKey(t *testing.T) {
	fake := newFake("# Tasks\n")
	m := New(fake)

	m = updateModel(t, m, keyRunes("?"))
	if m.mode != modeHelp {
		t.Fatalf("mode = %v, want help", m.mode)
	}
	if !strings.Contains(m.View(), "Navegação") {
		t.Errorf("view de ajuda inesperada:\n%s", m.View())
	}
	m = updateModel(t, m, keyRunes("x"))
	if m.mode != modeMain {
		t.Errorf("mode = %v, want main", m.mode)
	}
}

func TestOpenWithoutDaylistShowsError(t *testing.T) {
	fake := newFake("")
	fake.noteExists = false
	m := New(fake)

	m = updateModel(t, m, keyRunes("o"))
	if !strings.Contains(m.err, "ainda não existe") {
		t.Errorf("err = %q", m.err)
	}
	if fake.opened != 0 {
		t.Errorf("OpenDaylist chamado %d vezes", fake.opened)
	}
}
