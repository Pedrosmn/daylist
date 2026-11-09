package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"daylist/internal/daylist"
	"daylist/internal/planner"
	"daylist/internal/timer"
	"daylist/internal/vault"
	"daylist/internal/week"
)

// Backend concentra as operações de disco e de terminal usadas pelo TUI. O
// modelo só conhece esta interface, o que permite testá-lo com um backend
// falso, sem vault nem TTY.
type Backend interface {
	Today() time.Time
	DaylistRelOn(time.Time) string
	DaylistExistsOn(time.Time) bool
	LoadNoteOn(time.Time) (*daylist.Note, error)
	SaveNoteOn(time.Time, *daylist.Note) error

	WeeklistPath() string
	WeeklistExists() bool
	LoadWeeklist() (*daylist.Note, error)
	SaveWeeklist(*daylist.Note) error
	WeeklistHabitsOn(time.Time) ([]daylist.Item, error)
	WeekReportOn(time.Time) (*week.Report, error)
	FeedbackExistsOn(time.Time) bool
	AutoGeneratePreviousFeedback(time.Time) (bool, error)

	// PlanDaylist executa o wizard para a data alvo e devolve a prévia, sem gravar.
	PlanDaylist(time.Time) (*planner.Plan, error)
	WriteDaylist(*planner.Plan) error
	CreateWeeklist() error
	OpenDaylistOn(time.Time) error

	LoadTimer() (*timer.State, error)
	SaveTimer(*timer.State) error
	FinishTimer(*timer.State) (int64, error)
	ClearTimer() error
	StartTimerWatch() error
	ClaimThresholds(time.Time) (*timer.State, string, bool, error)
	Notify(title, body string) error
}

// RealBackend implementa Backend sobre um vault do Obsidian. Os callbacks de
// criação vêm do cmd para reutilizar o wizard e o gerador de weeklist sem criar
// dependência circular entre os pacotes.
type RealBackend struct {
	Vault *vault.Vault
	Now   func() time.Time

	PlanDaylistFn    func(time.Time) (*planner.Plan, error)
	WriteDaylistFn   func(*planner.Plan) error
	CreateWeeklistFn func() error
	NotifyFn         func(title, body string) error
}

// NewRealBackend cria o backend de produção para o vault informado.
func NewRealBackend(v *vault.Vault) *RealBackend {
	return &RealBackend{Vault: v, Now: time.Now}
}

func (b *RealBackend) Today() time.Time {
	if b.Now != nil {
		return b.Now()
	}
	return time.Now()
}

func (b *RealBackend) DaylistRelOn(day time.Time) string { return b.Vault.DaylistRel(day) }

func (b *RealBackend) DaylistExistsOn(day time.Time) bool {
	return fileExists(b.Vault.DaylistFile(day))
}

func (b *RealBackend) LoadNoteOn(day time.Time) (*daylist.Note, error) {
	return daylist.LoadNote(b.Vault.DaylistFile(day))
}

// SaveNoteOn grava a nota no arquivo da data informada, não na data atual.
// Isso evita que uma nota carregada antes da meia-noite seja gravada no dia
// seguinte caso o relógio vire entre a edição e a gravação.
func (b *RealBackend) SaveNoteOn(day time.Time, note *daylist.Note) error {
	return note.Write(b.Vault.DaylistFile(day))
}

func (b *RealBackend) WeeklistPath() string {
	return b.Vault.WeeklistFile(vault.ISOWeek(b.Today()))
}

func (b *RealBackend) WeeklistExists() bool { return fileExists(b.WeeklistPath()) }

func (b *RealBackend) LoadWeeklist() (*daylist.Note, error) {
	return daylist.LoadNote(b.WeeklistPath())
}

func (b *RealBackend) SaveWeeklist(note *daylist.Note) error {
	return note.Write(b.WeeklistPath())
}

// WeeklistHabitsOn devolve os hábitos pendentes da weeklist da semana da data
// informada. Eles são a origem dos hábitos Extra do TUI.
func (b *RealBackend) WeeklistHabitsOn(day time.Time) ([]daylist.Item, error) {
	items, err := daylist.ParseWeeklist(b.Vault.WeeklistFile(vault.ISOWeek(day)))
	if err != nil {
		return nil, err
	}
	var habits []daylist.Item
	for _, item := range items {
		if item.Section == "Hábitos" {
			habits = append(habits, item)
		}
	}
	return habits, nil
}

// WeekReportOn calcula o resumo da semana da data informada. Na semana atual
// lê as daylists até hoje; nas semanas passadas lê até domingo. Daylists
// ausentes e futuras ficam de fora. A weeklist ausente devolve os.ErrNotExist,
// para o TUI distinguir a semana sem dados.
func (b *RealBackend) WeekReportOn(day time.Time) (*week.Report, error) {
	weekString := vault.ISOWeek(day)
	weeklist, err := os.ReadFile(b.Vault.WeeklistFile(weekString))
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("weeklist de %s não encontrada: %w", weekString, os.ErrNotExist)
	}
	if err != nil {
		return nil, err
	}

	start, err := week.ParseWeek(weekString)
	if err != nil {
		return nil, err
	}
	today := dateOnly(b.Today())
	viewStart := weekStartOf(day)
	todayStart := weekStartOf(today)

	var days []week.Day
	if !viewStart.After(todayStart) {
		end := start.AddDate(0, 0, 6)
		if viewStart.Equal(todayStart) && today.Before(end) {
			end = today
		}
		for current := start; !current.After(end); current = current.AddDate(0, 0, 1) {
			content, err := os.ReadFile(b.Vault.DaylistFile(current))
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return nil, err
			}
			days = append(days, week.Day{Date: current, Content: string(content)})
		}
	}

	report := week.ComputeReport(weekString, string(weeklist), days)
	return &report, nil
}

// FeedbackExistsOn informa se o feedback da semana da data já foi gerado.
func (b *RealBackend) FeedbackExistsOn(day time.Time) bool {
	return fileExists(b.Vault.FeedbackFile(vault.ISOWeek(day)))
}

// AutoGeneratePreviousFeedback gera somente o feedback da última semana
// encerrada, se a weeklist existir e o arquivo ainda não existir. A criação é
// exclusiva para nunca sobrescrever um feedback existente; semanas anteriores
// não recebem backfill.
func (b *RealBackend) AutoGeneratePreviousFeedback(today time.Time) (bool, error) {
	createdAt := today
	today = dateOnly(today)
	previous := week.PreviousWeek(weekStartOf(today))
	weekString := vault.ISOWeek(previous)

	feedbackPath := b.Vault.FeedbackFile(weekString)
	if fileExists(feedbackPath) {
		return false, nil
	}

	weeklist, err := os.ReadFile(b.Vault.WeeklistFile(weekString))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	var days []week.Day
	for _, day := range week.WeekDays(previous) {
		content, err := os.ReadFile(b.Vault.DaylistFile(day))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return false, err
		}
		days = append(days, week.Day{Date: day, Content: string(content)})
	}

	return writeNewFile(feedbackPath, week.BuildFeedback(weekString, string(weeklist), days, createdAt))
}

// weekStartOf devolve a segunda-feira da semana da data.
func weekStartOf(day time.Time) time.Time {
	start, err := week.ParseWeek(vault.ISOWeek(day))
	if err != nil {
		return dateOnly(day)
	}
	return start
}

// writeNewFile grava o conteúdo apenas se o arquivo ainda não existir. Devolve
// true quando a gravação aconteceu. O O_EXCL evita sobrescrever um arquivo
// criado entre a verificação e a escrita.
func writeNewFile(path, content string) (bool, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, os.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if _, err := file.WriteString(content); err != nil {
		file.Close()
		os.Remove(path)
		return false, err
	}
	if err := file.Close(); err != nil {
		os.Remove(path)
		return false, err
	}
	return true, nil
}

func (b *RealBackend) PlanDaylist(target time.Time) (*planner.Plan, error) {
	if b.PlanDaylistFn == nil {
		return nil, errors.New("planejamento da daylist não configurado")
	}
	return b.PlanDaylistFn(target)
}

func (b *RealBackend) WriteDaylist(plan *planner.Plan) error {
	if b.WriteDaylistFn == nil {
		return errors.New("gravação da daylist não configurada")
	}
	return b.WriteDaylistFn(plan)
}

func (b *RealBackend) CreateWeeklist() error {
	if b.CreateWeeklistFn == nil {
		return errors.New("criação da weeklist não configurada")
	}
	return b.CreateWeeklistFn()
}

func (b *RealBackend) OpenDaylistOn(day time.Time) error { return b.Vault.OpenNote(day) }

func (b *RealBackend) LoadTimer() (*timer.State, error) {
	state, err := timer.Load(b.Vault.StateFile())
	if errors.Is(err, timer.ErrNoTimer) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return state, nil
}

func (b *RealBackend) SaveTimer(state *timer.State) error {
	return state.Save(b.Vault.StateFile())
}

func (b *RealBackend) FinishTimer(state *timer.State) (int64, error) {
	path := filepath.Join(b.Vault.Path, filepath.FromSlash(state.File))
	return state.Finish(path)
}

func (b *RealBackend) ClearTimer() error {
	return timer.ClearState(b.Vault.StateFile(), b.Vault.WatchPIDFile())
}

// StartTimerWatch garante o daemon de notificação para o timer em andamento.
func (b *RealBackend) StartTimerWatch() error {
	return timer.SpawnWatch(b.Vault.StateFile(), b.Vault.WatchPIDFile())
}

// ClaimThresholds verifica e persiste os limites de forma atômica, para o TUI
// não duplicar a notificação do watch.
func (b *RealBackend) ClaimThresholds(now time.Time) (*timer.State, string, bool, error) {
	update, err := timer.ClaimThresholds(b.Vault.StateFile(), now)
	if errors.Is(err, timer.ErrNoTimer) {
		return nil, "", false, nil
	}
	if err != nil {
		return nil, "", false, err
	}
	return update.State, update.Message, update.Changed, nil
}

func (b *RealBackend) Notify(title, body string) error {
	if b.NotifyFn != nil {
		return b.NotifyFn(title, body)
	}
	return timer.NotifyDesktop(title, body)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
