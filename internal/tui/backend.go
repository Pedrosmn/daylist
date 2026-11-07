package tui

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"daylist/internal/daylist"
	"daylist/internal/planner"
	"daylist/internal/timer"
	"daylist/internal/vault"
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
	WeeklistHabits() ([]daylist.Item, error)

	// PlanDaylist executa o wizard para a data alvo e devolve a prévia, sem gravar.
	PlanDaylist(time.Time) (*planner.Plan, error)
	WriteDaylist(*planner.Plan) error
	CreateWeeklist() error
	OpenDaylist() error

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

// WeeklistHabits devolve os hábitos pendentes da weeklist atual. Eles são a
// origem dos hábitos Extra do TUI.
func (b *RealBackend) WeeklistHabits() ([]daylist.Item, error) {
	items, err := daylist.ParseWeeklist(b.WeeklistPath())
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

func (b *RealBackend) OpenDaylist() error { return b.Vault.OpenNote(b.Today()) }

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
