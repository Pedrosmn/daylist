package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"daylist/internal/daylist"
	"daylist/internal/planner"
	"daylist/internal/timer"
)

type fakeBackend struct {
	now         time.Time
	noteContent string
	noteExists  bool
	days        map[string]bool
	notes       map[string]string // conteúdo por data (YYYY-MM-DD), quando definido
	weekContent string
	weekExists  bool
	habits      []daylist.Item

	timer     *timer.State
	timerSave int
	finishMin int64
	finishErr error
	cleared   int

	plan           *planner.Plan
	planErr        error
	planned        int
	plannedTarget  time.Time
	written        int
	writtenTargets []time.Time
	writeErr       error
	createdWeek    int
	opened         int
	notified       []string

	savedNote  string
	noteSaves  int
	savedDays  []string
	savedByDay map[string]string
	savedWeek  string
	weekSaves  int
}

func newFake(note string) *fakeBackend {
	return &fakeBackend{
		now:         time.Date(2026, 11, 5, 10, 0, 0, 0, time.Local),
		noteContent: note,
		noteExists:  true,
	}
}

func sameDate(a, b time.Time) bool {
	return a.Format("2006-01-02") == b.Format("2006-01-02")
}

func (f *fakeBackend) Today() time.Time { return f.now }

func (f *fakeBackend) DaylistRelOn(day time.Time) string {
	return "days/" + day.Format("2006-01-02") + ".md"
}

func (f *fakeBackend) DaylistExistsOn(day time.Time) bool {
	key := day.Format("2006-01-02")
	if _, ok := f.notes[key]; ok {
		return true
	}
	if sameDate(day, f.now) {
		return f.noteExists
	}
	return f.days[key]
}

func (f *fakeBackend) LoadNoteOn(day time.Time) (*daylist.Note, error) {
	if content, ok := f.notes[day.Format("2006-01-02")]; ok {
		return daylist.ParseNote(content), nil
	}
	if !sameDate(day, f.now) || !f.noteExists {
		return nil, errors.New("daylist inexistente")
	}
	return daylist.ParseNote(f.noteContent), nil
}

func (f *fakeBackend) SaveNoteOn(day time.Time, note *daylist.Note) error {
	key := day.Format("2006-01-02")
	content := note.String()

	if f.savedByDay == nil {
		f.savedByDay = map[string]string{}
	}
	f.savedByDay[key] = content
	f.savedDays = append(f.savedDays, key)
	f.noteSaves++

	if f.notes == nil {
		f.notes = map[string]string{}
	}
	f.notes[key] = content
	if sameDate(day, f.now) {
		f.savedNote = content
		f.noteContent = content
		f.noteExists = true
	}
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

func (f *fakeBackend) PlanDaylist(target time.Time) (*planner.Plan, error) {
	f.planned++
	f.plannedTarget = target
	if f.planErr != nil {
		return nil, f.planErr
	}
	return f.plan, nil
}

func (f *fakeBackend) WriteDaylist(plan *planner.Plan) error {
	if f.writeErr != nil {
		return f.writeErr
	}
	f.written++
	f.writtenTargets = append(f.writtenTargets, plan.Target)
	if sameDate(plan.Target, f.now) {
		f.savedNote = plan.Content
		f.noteContent = plan.Content
		f.noteExists = true
	}
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
	for _, want := range []string{"ainda não existe", "n  criar a daylist de hoje", "N  criar a daylist de amanhã"} {
		if !strings.Contains(view, want) {
			t.Errorf("view sem aviso de daylist ausente (%q):\n%s", want, view)
		}
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

func TestTickAtMidnightReloadsToday(t *testing.T) {
	fake := newFake("# Tasks\n- [ ] 10 min (I) - tarefa de ontem\n")
	fake.now = time.Date(2026, time.November, 5, 23, 59, 0, 0, time.Local)
	m := New(fake)

	fake.now = time.Date(2026, time.November, 6, 0, 1, 0, 0, time.Local)
	fake.notes = map[string]string{
		"2026-11-06": "# Tasks\n- [ ] 10 min (I) - tarefa de hoje\n",
	}

	m = updateModel(t, m, tickMsg(fake.now))

	if !sameDay(m.day, fake.now) {
		t.Fatalf("dia carregado = %v, want %v", m.day, fake.now)
	}
	view := m.View()
	if strings.Contains(view, "tarefa de ontem") {
		t.Errorf("view ainda mostra a daylist de ontem:\n%s", view)
	}
	if !strings.Contains(view, "tarefa de hoje") {
		t.Errorf("view não mostra a daylist de hoje:\n%s", view)
	}
	if !strings.Contains(m.st, "Novo dia") {
		t.Errorf("status = %q, want aviso de novo dia", m.st)
	}
}

func TestEditAfterMidnightReloadsBeforeWriting(t *testing.T) {
	fake := newFake("# Tasks\n- [ ] 10 min (I) - tarefa de ontem\n")
	fake.now = time.Date(2026, time.November, 5, 23, 59, 0, 0, time.Local)
	m := New(fake)

	// Sem esperar o tick: a primeira ação depois da meia-noite precisa
	// recarregar a daylist e não pode gravar a nota antiga no dia novo.
	fake.now = time.Date(2026, time.November, 6, 0, 1, 0, 0, time.Local)
	fake.notes = map[string]string{
		"2026-11-06": "# Tasks\n- [ ] 10 min (I) - tarefa de hoje\n",
	}

	m = updateModel(t, m, keyRunes(" "))
	if len(fake.savedDays) != 0 {
		t.Fatalf("gravou antes de recarregar: %v", fake.savedDays)
	}
	if !strings.Contains(m.st, "dia virou") {
		t.Errorf("status = %q, want aviso de virada de dia", m.st)
	}

	m = updateModel(t, m, keyRunes(" "))
	if len(fake.savedDays) != 1 || fake.savedDays[0] != "2026-11-06" {
		t.Fatalf("gravações = %v, want só 2026-11-06", fake.savedDays)
	}
	if got := fake.savedByDay["2026-11-06"]; !strings.Contains(got, "- [x] 10 min (I) - tarefa de hoje") {
		t.Errorf("item do dia novo não foi concluído:\n%s", got)
	}
	if _, ok := fake.savedByDay["2026-11-05"]; ok {
		t.Errorf("nota de ontem foi gravada depois da virada: %v", fake.savedByDay)
	}
}

func TestUndoAfterMidnightDoesNotRestoreOldDay(t *testing.T) {
	fake := newFake("# Tasks\n- [ ] 10 min (I) - tarefa de ontem\n")
	fake.now = time.Date(2026, time.November, 5, 23, 59, 0, 0, time.Local)
	m := New(fake)

	m = updateModel(t, m, keyRunes(" ")) // cria histórico de undo no dia 5
	if len(fake.savedDays) != 1 || fake.savedDays[0] != "2026-11-05" {
		t.Fatalf("setup: gravações = %v", fake.savedDays)
	}

	fake.now = time.Date(2026, time.November, 6, 0, 1, 0, 0, time.Local)
	fake.notes = map[string]string{
		"2026-11-06": "# Tasks\n- [ ] 10 min (I) - tarefa de hoje\n",
	}

	m = updateModel(t, m, keyRunes("u"))
	if len(fake.savedDays) != 1 {
		t.Fatalf("undo gravou depois da virada: %v", fake.savedDays)
	}
	if !strings.Contains(m.st, "dia virou") {
		t.Errorf("status = %q, want aviso de virada de dia", m.st)
	}

	m = updateModel(t, m, keyRunes("u"))
	if !strings.Contains(m.st, "Nada para desfazer") {
		t.Errorf("status = %q, want histórico vazio", m.st)
	}
	if len(fake.savedDays) != 1 {
		t.Errorf("undo deveria estar vazio, gravações = %v", fake.savedDays)
	}
}

func TestStartTimerAfterMidnightUsesCurrentDay(t *testing.T) {
	fake := newFake("# Tasks\n- [ ] 10 min (I) - tarefa de ontem\n")
	fake.now = time.Date(2026, time.November, 5, 23, 59, 0, 0, time.Local)
	m := New(fake)

	fake.now = time.Date(2026, time.November, 6, 0, 1, 0, 0, time.Local)
	fake.notes = map[string]string{
		"2026-11-06": "# Tasks\n- [ ] 10 min (I) - tarefa de hoje\n",
	}

	m = updateModel(t, m, keyRunes("s")) // recarrega e aborta
	m = updateModel(t, m, keyRunes("s")) // inicia no dia novo

	if fake.timer == nil {
		t.Fatal("timer não foi iniciado")
	}
	if fake.timer.File != "days/2026-11-06.md" {
		t.Errorf("arquivo do timer = %q, want days/2026-11-06.md", fake.timer.File)
	}
	if fake.timer.Line != "- [ ] 10 min (I) - tarefa de hoje" {
		t.Errorf("linha do timer = %q", fake.timer.Line)
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

func TestPreviewShowsFullDaylistTotalAndCreates(t *testing.T) {
	fake := newFake("")
	fake.noteExists = false
	content := `2026-11-05 08:00

# Tasks

- [ ] 20 min (I) - tarefa

---
# Hábitos

## Rotina
- [ ] 5 min () - Fio dental
`
	plan := &planner.Plan{
		Target:   fake.now,
		Week:     "2026-W45",
		TotalMin: 125,
		TotalMax: 125,
		HasTotal: true,
		LimitMin: 8 * 60,
		Content:  content,
	}
	fake.plan = plan
	m := New(fake)
	m.st = "Criando a daylist no wizard..."

	m = updateModel(t, m, plannedMsg{plan: plan})
	if m.mode != modePreview {
		t.Fatalf("mode = %v, want preview", m.mode)
	}
	view := m.View()
	for _, want := range []string{
		"Prévia da daylist",
		"- [ ] 20 min (I) - tarefa",
		"## Rotina",
		"- [ ] 5 min () - Fio dental",
		"Tempo planejado: 2h05",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("prévia não contém %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "acima do limite") {
		t.Errorf("prévia não deveria alertar abaixo do limite:\n%s", view)
	}
	if strings.Contains(view, "Criando a daylist no wizard") {
		t.Errorf("prévia manteve o status da tela anterior:\n%s", view)
	}

	m = updateModel(t, m, keyType(tea.KeyEnter))
	if fake.written != 1 || !fake.noteExists {
		t.Fatalf("daylist não foi gravada: written=%d exists=%v", fake.written, fake.noteExists)
	}
	if m.mode != modeMain || !strings.Contains(m.st, "criada") {
		t.Errorf("mode=%v status=%q", m.mode, m.st)
	}
}

func TestPreviewWarnsAboveDayLimit(t *testing.T) {
	fake := newFake("")
	fake.noteExists = false
	plan := &planner.Plan{
		Target:   fake.now,
		Week:     "2026-W45",
		TotalMin: 8*60 + 30,
		TotalMax: 9 * 60,
		HasTotal: true,
		LimitMin: 8 * 60,
		Content:  "conteúdo\n",
	}
	m := New(fake)

	m = updateModel(t, m, plannedMsg{plan: plan})
	view := m.View()
	if !strings.Contains(view, "acima do limite") || !strings.Contains(view, "limite 8h") {
		t.Errorf("alerta de limite ausente:\n%s", view)
	}
}

func TestPreviewCancelDoesNotWrite(t *testing.T) {
	fake := newFake("")
	fake.noteExists = false
	plan := &planner.Plan{Target: fake.now, Content: "conteúdo\n"}
	m := New(fake)

	m = updateModel(t, m, plannedMsg{plan: plan})
	m = updateModel(t, m, keyType(tea.KeyEsc))

	if m.mode != modeMain || fake.written != 0 || fake.noteExists {
		t.Errorf("cancelamento gravou algo: mode=%v written=%d exists=%v", m.mode, fake.written, fake.noteExists)
	}
}

func TestPlannedCanceledShowsStatus(t *testing.T) {
	fake := newFake("# Tasks\n")
	m := New(fake)

	m = updateModel(t, m, plannedMsg{err: planner.ErrCanceled})
	if m.mode != modeMain || !strings.Contains(m.st, "cancelada") {
		t.Errorf("mode=%v status=%q", m.mode, m.st)
	}
}

func TestCreateDaylistAlreadyExists(t *testing.T) {
	fake := newFake("# Tasks\n")
	m := New(fake)

	m = updateModel(t, m, keyRunes("n"))
	if !strings.Contains(m.st, "já existe") || !strings.Contains(m.st, "Pressione N") {
		t.Errorf("status = %q", m.st)
	}
}

func TestCreateTomorrowDaylist(t *testing.T) {
	fake := newFake(testNote)
	tomorrow := fake.now.AddDate(0, 0, 1)
	plan := &planner.Plan{
		Target:  tomorrow,
		Week:    "2026-W45",
		Content: "conteúdo de amanhã\n",
	}
	m := New(fake)

	m = updateModel(t, m, keyRunes("N"))
	if !strings.Contains(m.st, "amanhã") {
		t.Fatalf("status = %q, want menção a amanhã", m.st)
	}

	m = updateModel(t, m, plannedMsg{plan: plan, target: tomorrow})
	if m.mode != modePreview {
		t.Fatalf("mode = %v, want preview", m.mode)
	}
	if !strings.Contains(m.View(), tomorrow.Format("2006-01-02")) {
		t.Errorf("prévia não mostra a data de amanhã:\n%s", m.View())
	}

	m = updateModel(t, m, keyType(tea.KeyEnter))
	if fake.written != 1 || len(fake.writtenTargets) != 1 || !sameDate(fake.writtenTargets[0], tomorrow) {
		t.Fatalf("daylist de amanhã não foi gravada: written=%d targets=%v", fake.written, fake.writtenTargets)
	}
	if fake.noteContent != testNote {
		t.Error("a daylist de hoje foi alterada ao criar a de amanhã")
	}
	if !strings.Contains(m.st, tomorrow.Format("2006-01-02")) {
		t.Errorf("status = %q, want data de amanhã", m.st)
	}
}

func TestCreateTomorrowAlreadyExists(t *testing.T) {
	fake := newFake(testNote)
	tomorrow := fake.now.AddDate(0, 0, 1)
	fake.days = map[string]bool{tomorrow.Format("2006-01-02"): true}
	m := New(fake)

	m = updateModel(t, m, keyRunes("N"))
	if !strings.Contains(m.st, "amanhã") || !strings.Contains(m.st, "já existe") {
		t.Errorf("status = %q", m.st)
	}
	if strings.Contains(m.st, "Pressione N") {
		t.Errorf("status de amanhã não deveria sugerir N: %q", m.st)
	}
}

func TestCloseDayShowsSummary(t *testing.T) {
	fake := newFake(`# Tasks
- [x] 30 min (I) - feita ⏱ 25min (09:00–09:25)
- [ ] 20 min (U) - pendente
# Hábitos
## Rotina
- [ ] 5 min () - água
`)
	m := New(fake)

	m = updateModel(t, m, keyRunes("F"))
	if m.mode != modeClose {
		t.Fatalf("mode = %v, want close", m.mode)
	}
	view := m.View()
	for _, want := range []string{
		"Fechar o dia", "Planejado:  55 min", "Real:       25 min",
		"Concluídos: 1", "Pendentes:  2", "nada é alterado",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("resumo não contém %q:\n%s", want, view)
		}
	}

	m = updateModel(t, m, keyType(tea.KeyEsc))
	if m.mode != modeMain {
		t.Errorf("mode = %v, want main", m.mode)
	}
}

func TestCloseDayWithoutDaylistShowsError(t *testing.T) {
	fake := newFake("")
	fake.noteExists = false
	m := New(fake)

	m = updateModel(t, m, keyRunes("F"))
	if !strings.Contains(m.err, "antes de fechar") {
		t.Errorf("err = %q", m.err)
	}
	if m.mode != modeMain {
		t.Errorf("mode = %v, want main", m.mode)
	}
}

func TestHelpModeReturnsOnAnyKey(t *testing.T) {
	fake := newFake("# Tasks\n")
	m := New(fake)

	m = updateModel(t, m, keyRunes("?"))
	if m.mode != modeHelp {
		t.Fatalf("mode = %v, want help", m.mode)
	}
	view := m.View()
	if !strings.Contains(view, "Navegação") {
		t.Errorf("view de ajuda inesperada:\n%s", view)
	}
	if !strings.Contains(view, "N  criar a daylist de amanhã") {
		t.Errorf("ajuda não menciona a criação da daylist de amanhã:\n%s", view)
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
