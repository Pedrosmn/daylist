package tui

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"daylist/internal/daylist"
	"daylist/internal/planner"
	"daylist/internal/timer"
	"daylist/internal/week"
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
	habitDays   []time.Time
	habitErr    error

	report      *week.Report
	reportErr   error
	reportDays  []time.Time
	feedback    bool
	feedbackDay []time.Time
	autoRun     int
	autoToday   []time.Time
	autoCreated bool
	autoErr     error

	timer       *timer.State
	timerSave   int
	finishMin   int64
	finishErr   error
	cleared     int
	watchStarts int
	watchStops  int

	plan           *planner.Plan
	planErr        error
	planned        int
	plannedTarget  time.Time
	written        int
	writtenTargets []time.Time
	writeErr       error
	createdWeek    int
	opened         int
	openedDay      time.Time
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

func (f *fakeBackend) WeeklistHabitsOn(day time.Time) ([]daylist.Item, error) {
	f.habitDays = append(f.habitDays, day)
	if f.habitErr != nil {
		return nil, f.habitErr
	}
	return f.habits, nil
}

func (f *fakeBackend) WeekReportOn(day time.Time) (*week.Report, error) {
	f.reportDays = append(f.reportDays, day)
	if f.reportErr != nil {
		return nil, f.reportErr
	}
	return f.report, nil
}

func (f *fakeBackend) FeedbackExistsOn(day time.Time) bool {
	f.feedbackDay = append(f.feedbackDay, day)
	return f.feedback
}

func (f *fakeBackend) AutoGeneratePreviousFeedback(today time.Time) (bool, error) {
	f.autoRun++
	f.autoToday = append(f.autoToday, today)
	if f.autoErr != nil {
		return false, f.autoErr
	}
	return f.autoCreated, nil
}

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

func (f *fakeBackend) OpenDaylistOn(day time.Time) error {
	f.opened++
	f.openedDay = day
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
	f.watchStops++
	f.timer = nil
	return nil
}

func (f *fakeBackend) StartTimerWatch() error {
	f.watchStarts++
	return nil
}

func (f *fakeBackend) ClaimThresholds(now time.Time) (*timer.State, string, bool, error) {
	if f.timer == nil {
		return nil, "", false, nil
	}
	if f.timer.Paused {
		return f.timer, "", false, nil
	}
	message, changed := f.timer.CheckThresholds(f.timer.TotalMsAt(now.UnixMilli()))
	return f.timer, message, changed, nil
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

	m = updateModel(t, m, keyRunes("m"))
	if m.mode != modeForm || m.form == nil {
		t.Fatalf("mode = %v, form = %+v", m.mode, m.form)
	}
	m.form.fields[0].SetValue("nova motivação")
	m = updateModel(t, m, keyType(tea.KeyEnter))

	if !strings.Contains(fake.savedNote, "> nova motivação") ||
		strings.Contains(fake.savedNote, "> antiga") {
		t.Errorf("motivação não foi atualizada:\n%s", fake.savedNote)
	}
}

func TestMotivationShortcutTitleAndContextualHelp(t *testing.T) {
	fake := newFake("# Tasks\n- [ ] 10 min (I) - tarefa\n> antiga\n")
	m := New(fake)
	m = updateModel(t, m, tea.WindowSizeMsg{Width: 70, Height: 24})

	m = updateModel(t, m, keyRunes("e"))
	if m.mode == modeForm {
		t.Fatal("e não deveria mais abrir o formulário de motivação")
	}

	mainView := m.View()
	if !strings.Contains(mainView, "m motivação") {
		t.Errorf("rodapé da tela principal não anuncia m:\n%s", mainView)
	}

	m = updateModel(t, m, keyRunes("?"))
	helpView := m.View()
	if !strings.Contains(helpView, "m  editar ou remover a motivação") {
		t.Errorf("ajuda não anuncia m:\n%s", helpView)
	}
	if strings.Contains(helpView, "e  editar ou remover a motivação") {
		t.Errorf("ajuda ainda anuncia e:\n%s", helpView)
	}
	m = updateModel(t, m, keyRunes("x"))

	m = updateModel(t, m, keyRunes("m"))
	if m.mode != modeForm || m.form == nil {
		t.Fatalf("mode = %v, form = %+v", m.mode, m.form)
	}

	view := m.View()
	for _, want := range []string{
		"Motivação · 10 min (I) - tarefa",
		"enter salva · esc cancela",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("formulário não contém %q:\n%s", want, view)
		}
	}
	for _, unwanted := range []string{
		"Motivação: ",
		"tab/↑↓ campo",
		"←/→ prioridade",
	} {
		if strings.Contains(view, unwanted) {
			t.Errorf("formulário não deveria conter %q:\n%s", unwanted, view)
		}
	}
}

func TestExtraTaskKeepsFullFormHelp(t *testing.T) {
	fake := newFake("# Tasks\n")
	m := New(fake)
	m = updateModel(t, m, tea.WindowSizeMsg{Width: 70, Height: 24})
	m = updateModel(t, m, keyRunes("a"))

	view := m.View()
	for _, want := range []string{"tab/↑↓ campo", "←/→ prioridade", "enter/ctrl+s salva"} {
		if !strings.Contains(view, want) {
			t.Errorf("rodapé do Extra task não contém %q:\n%s", want, view)
		}
	}
}

func TestMotivationFormWrapsVisuallyAndSavesSingleLine(t *testing.T) {
	fake := newFake("# Tasks\n- [ ] 10 min (I) - tarefa\n")
	m := New(fake)
	m = updateModel(t, m, tea.WindowSizeMsg{Width: 40, Height: 24})
	m = updateModel(t, m, keyRunes("m"))

	text := "uma motivação bem longa para confirmar que o texto desce de linha sem rolar"
	m = updateModel(t, m, keyRunes(text))

	view := m.View()
	if strings.Contains(view, text) {
		t.Errorf("o texto não quebrou na largura disponível:\n%s", view)
	}
	for _, want := range []string{"uma motivação", "rolar"} {
		if !strings.Contains(view, want) {
			t.Errorf("texto quebrado perdeu %q:\n%s", want, view)
		}
	}

	m = updateModel(t, m, keyType(tea.KeyEnter))
	if m.mode != modeMain {
		t.Fatalf("mode = %v, want main; err=%q", m.mode, m.err)
	}
	if !strings.Contains(fake.savedNote, "> "+text) {
		t.Errorf("motivação não foi gravada em uma linha:\n%s", fake.savedNote)
	}
}

func TestExtraTaskMotivationWraps(t *testing.T) {
	fake := newFake("# Tasks\n")
	m := New(fake)
	m = updateModel(t, m, tea.WindowSizeMsg{Width: 44, Height: 24})
	m = updateModel(t, m, keyRunes("a"))

	text := "motivação do extra task longa o bastante para descer de linha"
	m.form.fields[3].SetValue(text)
	m.form.layout(m.formContentWidth())

	view := m.View()
	if strings.Contains(view, text) {
		t.Errorf("a motivação do Extra task não quebrou:\n%s", view)
	}
	for _, want := range []string{"motivação do extra", "linha"} {
		if !strings.Contains(view, want) {
			t.Errorf("texto quebrado perdeu %q:\n%s", want, view)
		}
	}
}

func TestMotivationFormEscCancelsWithoutSaving(t *testing.T) {
	fake := newFake("# Tasks\n- [ ] 10 min (I) - tarefa\n> antiga\n")
	m := New(fake)
	m = updateModel(t, m, tea.WindowSizeMsg{Width: 70, Height: 24})
	m = updateModel(t, m, keyRunes("m"))
	m = updateModel(t, m, keyRunes("alterada"))

	m = updateModel(t, m, keyType(tea.KeyEsc))
	if m.mode != modeConfirm {
		t.Fatalf("mode = %v, want confirmação", m.mode)
	}
	m = updateModel(t, m, keyRunes("s"))

	if m.mode != modeMain || fake.noteSaves != 0 {
		t.Fatalf("cancelamento gravou: mode=%v saves=%d", m.mode, fake.noteSaves)
	}
	if !strings.Contains(fake.noteContent, "> antiga") {
		t.Errorf("motivação original foi alterada:\n%s", fake.noteContent)
	}
}

func TestMainViewWrapsMotivation(t *testing.T) {
	motivation := "uma motivação bem longa que precisa aparecer completa mesmo em terminal estreito"
	fake := newFake("# Tasks\n- [ ] 10 min (I) - tarefa\n> " + motivation + "\n")
	m := New(fake)
	m = updateModel(t, m, tea.WindowSizeMsg{Width: 40, Height: 24})

	view := m.View()
	if strings.Contains(view, motivation) {
		t.Errorf("a motivação não quebrou na largura do terminal:\n%s", view)
	}
	for _, want := range []string{"uma motivação", "completa", "estreito"} {
		if !strings.Contains(view, want) {
			t.Errorf("motivação quebrada perdeu %q:\n%s", want, view)
		}
	}
}

func TestMainViewWrapsMultilineMotivation(t *testing.T) {
	fake := newFake("# Tasks\n- [ ] 10 min (I) - tarefa\n> primeira linha longa\n> segunda linha longa\n")
	m := New(fake)
	m = updateModel(t, m, tea.WindowSizeMsg{Width: 44, Height: 24})

	view := m.View()
	for _, want := range []string{"primeira linha longa", "segunda linha longa"} {
		if !strings.Contains(view, want) {
			t.Errorf("motivação com múltiplas linhas perdeu %q:\n%s", want, view)
		}
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
	if fake.watchStarts != 1 {
		t.Errorf("watch não iniciado junto com o timer: starts=%d", fake.watchStarts)
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
	if fake.watchStops != 1 {
		t.Errorf("watch não foi encerrado junto com o timer: stops=%d", fake.watchStops)
	}
	if !strings.Contains(m.st, "Timer parado") || !strings.Contains(m.st, "7min") {
		t.Errorf("status = %q", m.st)
	}
}

func TestTimerThresholdsAreSharedAndNotifiedOnce(t *testing.T) {
	fake := newFake("# Tasks\n")
	fake.timer = &timer.State{
		Desc:       "tarefa",
		Start:      fake.now.Add(-5 * time.Minute).UnixMilli(),
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

func TestTimerThresholdsPausedDoNotNotify(t *testing.T) {
	fake := newFake("# Tasks\n")
	fake.timer = &timer.State{
		Desc:       "tarefa",
		Paused:     true,
		Elapsed:    (5 * time.Minute).Milliseconds(),
		MinMinutes: 5,
		MaxMinutes: 15,
	}
	m := New(fake)

	if cmd := m.checkThresholds(); cmd != nil {
		t.Fatal("timer pausado não deveria agendar notificação")
	}
	if len(fake.notified) != 0 {
		t.Errorf("notificações com timer pausado = %q", fake.notified)
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
	for _, want := range []string{
		"S  ver o resumo da semana",
		"feedback da última semana: automático a partir de segunda",
		"não há atalho g para feedback",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("ajuda não contém %q:\n%s", want, view)
		}
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

func TestToggleBetweenTodayAndYesterday(t *testing.T) {
	fake := newFake("# Tasks\n- [ ] 10 min (I) - tarefa de hoje\n")
	fake.notes = map[string]string{
		"2026-11-04": "# Tasks\n- [ ] 10 min (I) - tarefa de ontem\n",
	}
	fake.timer = &timer.State{Desc: "timer de hoje"}

	m := New(fake)
	if !m.isToday() {
		t.Fatalf("modelo iniciou fora de hoje: %v", m.day)
	}
	if m.timer == nil {
		t.Fatal("timer de hoje não foi carregado")
	}

	m = updateModel(t, m, keyRunes("y"))
	if got := m.day.Format("2006-01-02"); got != "2026-11-04" {
		t.Fatalf("data visualizada = %s, want 2026-11-04", got)
	}
	if len(m.items) != 1 || !strings.Contains(m.items[0].Desc, "ontem") {
		t.Fatalf("itens de ontem não carregados: %+v", m.items)
	}
	if m.timer != nil {
		t.Errorf("banner do timer deveria sumir em ontem: %+v", m.timer)
	}
	view := m.View()
	for _, want := range []string{"ontem", "04/11/2026", "tarefa de ontem", "timer e motivação desabilitados"} {
		if !strings.Contains(view, want) {
			t.Errorf("view de ontem não contém %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "Timer rodando") {
		t.Errorf("banner do timer apareceu em ontem:\n%s", view)
	}

	m = updateModel(t, m, keyRunes("t"))
	if !sameDay(m.day, fake.now) {
		t.Fatalf("t não voltou para hoje: %v", m.day)
	}
	if m.timer == nil {
		t.Error("timer de hoje não foi recarregado ao voltar")
	}
	view = m.View()
	if strings.Contains(view, "tarefa de ontem") || !strings.Contains(view, "tarefa de hoje") {
		t.Errorf("view de hoje inesperada:\n%s", view)
	}
}

func TestYesterdayMissingShowsMessageWithoutLeavingTUI(t *testing.T) {
	fake := newFake("# Tasks\n- [ ] tarefa de hoje\n")
	m := New(fake)

	m = updateModel(t, m, keyRunes("y"))
	if got := m.day.Format("2006-01-02"); got != "2026-11-04" {
		t.Fatalf("data visualizada = %s, want 2026-11-04", got)
	}
	if m.note != nil {
		t.Fatalf("não deveria haver nota de ontem: %+v", m.note)
	}
	if m.mode != modeMain {
		t.Errorf("mode = %v, want main", m.mode)
	}
	view := m.View()
	for _, want := range []string{"daylist de ontem ainda não existe", "t/esc", "y"} {
		if !strings.Contains(view, want) {
			t.Errorf("view sem aviso de ontem ausente (%q):\n%s", want, view)
		}
	}

	m = updateModel(t, m, keyType(tea.KeyEsc))
	if !sameDay(m.day, fake.now) || m.note == nil {
		t.Errorf("esc não voltou para hoje: day=%v note=%+v", m.day, m.note)
	}
}

func TestYesterdayEditsWriteToYesterdayFile(t *testing.T) {
	fake := newFake("# Tasks\n- [ ] 10 min (I) - tarefa de hoje\n")
	fake.notes = map[string]string{
		"2026-11-04": "# Tasks\n- [ ] 10 min (I) - tarefa de ontem\n",
	}
	m := New(fake)

	m = updateModel(t, m, keyRunes("y"))
	m = updateModel(t, m, keyRunes(" "))
	if len(fake.savedDays) != 1 || fake.savedDays[0] != "2026-11-04" {
		t.Fatalf("toggle gravou em %v, want só 2026-11-04", fake.savedDays)
	}
	if got := fake.savedByDay["2026-11-04"]; !strings.Contains(got, "- [x] 10 min (I) - tarefa de ontem") {
		t.Errorf("item de ontem não foi concluído:\n%s", got)
	}
	if _, ok := fake.savedByDay["2026-11-05"]; ok {
		t.Errorf("arquivo de hoje foi alterado ao editar ontem: %v", fake.savedByDay)
	}

	// Extra task na tela de ontem também precisa gravar no arquivo de ontem.
	m = updateModel(t, m, keyRunes("a"))
	m = updateModel(t, m, keyRunes("Extra de ontem"))
	m = updateModel(t, m, keyType(tea.KeyCtrlS))
	if len(fake.savedDays) != 2 || fake.savedDays[1] != "2026-11-04" {
		t.Fatalf("Extra gravou em %v, want ontem", fake.savedDays)
	}
	if got := fake.savedByDay["2026-11-04"]; !strings.Contains(got, "# Extra") ||
		!strings.Contains(got, "Extra de ontem") {
		t.Errorf("Extra não foi para a daylist de ontem:\n%s", got)
	}
	if _, ok := fake.savedByDay["2026-11-05"]; ok {
		t.Errorf("arquivo de hoje foi alterado pelo Extra de ontem: %v", fake.savedByDay)
	}
}

func TestUndoIsolatedPerDay(t *testing.T) {
	fake := newFake("# Tasks\n- [ ] 10 min (I) - tarefa de hoje\n")
	fake.notes = map[string]string{
		"2026-11-04": "# Tasks\n- [ ] 10 min (I) - tarefa de ontem\n",
	}
	m := New(fake)

	m = updateModel(t, m, keyRunes(" ")) // undo de hoje
	m = updateModel(t, m, keyRunes("y"))
	m = updateModel(t, m, keyRunes("u")) // ontem ainda não tem undo
	if !strings.Contains(m.st, "Nada para desfazer") {
		t.Errorf("status = %q, want histórico vazio em ontem", m.st)
	}
	if len(fake.savedDays) != 1 {
		t.Fatalf("undo em ontem gravou: %v", fake.savedDays)
	}

	m = updateModel(t, m, keyRunes(" ")) // cria undo de ontem
	m = updateModel(t, m, keyRunes("u")) // desfaz só ontem
	if len(fake.savedDays) != 3 || fake.savedDays[1] != "2026-11-04" || fake.savedDays[2] != "2026-11-04" {
		t.Fatalf("sequência de gravações em ontem = %v", fake.savedDays)
	}

	m = updateModel(t, m, keyRunes("t"))
	m = updateModel(t, m, keyRunes("u")) // desfaz o undo que ficou guardado de hoje
	if len(fake.savedDays) != 4 || fake.savedDays[3] != "2026-11-05" {
		t.Fatalf("undo de hoje não foi isolado: %v", fake.savedDays)
	}
	for _, day := range []string{"2026-11-04", "2026-11-05"} {
		if got := fake.savedByDay[day]; !strings.Contains(got, "- [ ] 10 min (I)") {
			t.Errorf("daylist %s não voltou ao estado inicial:\n%s", day, got)
		}
	}
}

func TestYesterdayDisablesTimerAndMotivation(t *testing.T) {
	fake := newFake("# Tasks\n- [ ] 10 min (I) - tarefa de hoje\n")
	fake.notes = map[string]string{
		"2026-11-04": "# Tasks\n- [ ] 10 min (I) - tarefa de ontem\n",
	}
	fake.timer = &timer.State{Desc: "timer de hoje"}
	m := New(fake)

	m = updateModel(t, m, keyRunes("y"))
	if m.timer != nil {
		t.Fatalf("timer deveria estar oculto em ontem: %+v", m.timer)
	}

	m = updateModel(t, m, keyRunes("s"))
	if !strings.Contains(m.err, "timer") || !strings.Contains(m.err, "hoje") {
		t.Errorf("err ao iniciar timer em ontem = %q", m.err)
	}
	if fake.timerSave != 0 {
		t.Errorf("timer foi salvo em ontem: saves=%d", fake.timerSave)
	}

	m = updateModel(t, m, keyRunes("p"))
	if fake.timerSave != 0 {
		t.Errorf("timer foi pausado em ontem: saves=%d", fake.timerSave)
	}
	m = updateModel(t, m, keyRunes("x"))
	if fake.cleared != 0 {
		t.Errorf("timer foi parado em ontem: clears=%d", fake.cleared)
	}

	m = updateModel(t, m, keyRunes("m"))
	if m.mode == modeForm {
		t.Error("formulário de motivação abriu em ontem")
	}
	if !strings.Contains(m.err, "motivação") || !strings.Contains(m.err, "hoje") {
		t.Errorf("err ao editar motivação em ontem = %q", m.err)
	}

	m = updateModel(t, m, keyRunes("t"))
	if m.timer == nil {
		t.Error("timer de hoje deveria voltar ao retornar para hoje")
	}
}

func TestYesterdayExtraHabitUsesViewedWeeklist(t *testing.T) {
	fake := newFake("# Tasks\n")
	fake.notes = map[string]string{
		"2026-11-04": "# Tasks\n",
	}
	fake.habits = []daylist.Item{{
		Section: "Hábitos", Category: "Rotina", Time: "5 min", Desc: "Fio dental",
	}}
	m := New(fake)

	m = updateModel(t, m, keyRunes("y"))
	m = updateModel(t, m, keyRunes("A"))
	if len(fake.habitDays) != 1 || !sameDate(fake.habitDays[0], m.day) {
		t.Fatalf("weeklist consultada em %v, want %v", fake.habitDays, m.day)
	}
	m = updateModel(t, m, keyType(tea.KeyEnter))
	m = updateModel(t, m, keyType(tea.KeyEnter))

	if len(fake.savedDays) != 1 || fake.savedDays[0] != "2026-11-04" {
		t.Fatalf("hábito gravou em %v, want só ontem", fake.savedDays)
	}
	if got := fake.savedByDay["2026-11-04"]; !strings.Contains(got, "## Rotina") ||
		!strings.Contains(got, "5 min - Fio dental") {
		t.Errorf("hábito Extra não foi para ontem:\n%s", got)
	}
}

func TestYesterdayExtraHabitWithoutWeeklistWarns(t *testing.T) {
	fake := newFake("# Tasks\n")
	fake.notes = map[string]string{
		"2026-11-04": "# Tasks\n",
	}
	fake.habitErr = os.ErrNotExist
	m := New(fake)

	m = updateModel(t, m, keyRunes("y"))
	m = updateModel(t, m, keyRunes("A"))
	if !strings.Contains(m.err, "weeklist da semana de ontem não encontrada") {
		t.Errorf("err = %q", m.err)
	}
	if m.mode != modeMain {
		t.Errorf("mode = %v, want main", m.mode)
	}
}

func TestMidnightWhileViewingYesterdayKeepsAbsoluteDate(t *testing.T) {
	fake := newFake("# Tasks\n- [ ] tarefa de hoje\n")
	fake.now = time.Date(2026, time.November, 5, 23, 59, 0, 0, time.Local)
	fake.notes = map[string]string{
		"2026-11-04": "# Tasks\n- [ ] tarefa de ontem\n",
	}
	m := New(fake)

	m = updateModel(t, m, keyRunes("y"))
	if got := m.day.Format("2006-01-02"); got != "2026-11-04" {
		t.Fatalf("data visualizada = %s", got)
	}

	fake.now = time.Date(2026, time.November, 6, 0, 1, 0, 0, time.Local)
	fake.notes["2026-11-06"] = "# Tasks\n- [ ] tarefa do novo dia\n"

	m = updateModel(t, m, tickMsg(fake.now))
	if got := m.day.Format("2006-01-02"); got != "2026-11-04" {
		t.Fatalf("tick mudou a data visualizada para %s", got)
	}
	if !sameDate(m.now, fake.now) {
		t.Errorf("relógio = %v, want %v", m.now, fake.now)
	}
	if !strings.Contains(m.st, "continua na daylist de 04/11/2026") {
		t.Errorf("status = %q", m.st)
	}
	if m.note == nil || !strings.Contains(m.note.String(), "tarefa de ontem") {
		t.Errorf("nota de ontem foi perdida: %+v", m.note)
	}

	m = updateModel(t, m, keyRunes(" "))
	if len(fake.savedDays) != 1 || fake.savedDays[0] != "2026-11-04" {
		t.Fatalf("edição depois da virada gravou em %v", fake.savedDays)
	}
	if _, ok := fake.savedByDay["2026-11-06"]; ok {
		t.Errorf("edição de ontem vazou para o novo dia: %v", fake.savedByDay)
	}

	m = updateModel(t, m, keyRunes("r"))
	if got := m.day.Format("2006-01-02"); got != "2026-11-04" {
		t.Errorf("reload depois da virada mudou a data para %s", got)
	}

	m = updateModel(t, m, keyRunes("t"))
	if got := m.day.Format("2006-01-02"); got != "2026-11-06" {
		t.Fatalf("t não voltou para o novo hoje: %s", got)
	}
	if m.note == nil || !strings.Contains(m.note.String(), "tarefa do novo dia") {
		t.Errorf("daylist do novo dia não foi carregada: %+v", m.note)
	}
	if _, ok := fake.savedByDay["2026-11-06"]; ok {
		t.Errorf("voltar para hoje não deveria gravar: %v", fake.savedByDay)
	}
}

func TestMidnightKeepsOpenExtraFormOnYesterday(t *testing.T) {
	fake := newFake("# Tasks\n- [ ] tarefa de hoje\n")
	fake.now = time.Date(2026, time.November, 5, 23, 59, 0, 0, time.Local)
	fake.notes = map[string]string{
		"2026-11-04": "# Tasks\n- [ ] tarefa de ontem\n",
	}
	m := New(fake)

	m = updateModel(t, m, keyRunes("y"))
	m = updateModel(t, m, keyRunes("a"))
	m = updateModel(t, m, keyRunes("Extra noturno"))

	fake.now = time.Date(2026, time.November, 6, 0, 1, 0, 0, time.Local)
	fake.notes["2026-11-06"] = "# Tasks\n- [ ] tarefa do novo dia\n"
	m = updateModel(t, m, tickMsg(fake.now))

	if m.mode != modeForm {
		t.Fatalf("mode = %v, want form preservado", m.mode)
	}
	if m.form == nil || m.form.value(0) != "Extra noturno" {
		t.Fatalf("estado de edição perdido: %+v", m.form)
	}

	m = updateModel(t, m, keyType(tea.KeyCtrlS))
	if len(fake.savedDays) != 1 || fake.savedDays[0] != "2026-11-04" {
		t.Fatalf("form depois da virada gravou em %v", fake.savedDays)
	}
	if got := fake.savedByDay["2026-11-04"]; !strings.Contains(got, "Extra noturno") {
		t.Errorf("Extra não foi para a daylist de ontem:\n%s", got)
	}
}

func TestToggleYesterdayPreservesCursorPerDay(t *testing.T) {
	fake := newFake("# Tasks\n- [ ] item a\n- [ ] item b\n- [ ] item c\n")
	fake.notes = map[string]string{
		"2026-11-04": "# Tasks\n- [ ] item x\n",
	}
	m := New(fake)

	m = updateModel(t, m, keyRunes("j"))
	if m.cursor != 1 {
		t.Fatalf("cursor antes = %d, want 1", m.cursor)
	}
	m = updateModel(t, m, keyRunes("y"))
	if m.cursor != 0 {
		t.Errorf("cursor em ontem = %d, want 0", m.cursor)
	}
	m = updateModel(t, m, keyRunes("t"))
	if m.cursor != 1 {
		t.Errorf("cursor ao voltar para hoje = %d, want 1", m.cursor)
	}
	item, ok := m.selected()
	if !ok || item.Desc != "item b" {
		t.Errorf("seleção ao voltar = %+v, want item b", item)
	}
}
