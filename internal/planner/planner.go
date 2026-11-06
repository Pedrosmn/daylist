// Package planner monta a daylist em memória, sem gravar, para permitir uma
// prévia antes da criação. O wizard continua sendo a fonte das escolhas.
package planner

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"daylist/internal/daylist"
	"daylist/internal/vault"
	"daylist/internal/wizard"
)

var (
	// ErrDaylistExists indica que a data alvo já tem uma daylist.
	ErrDaylistExists = errors.New("daylist já existe")
	// ErrCanceled indica que o usuário cancelou o wizard.
	ErrCanceled = errors.New("criação cancelada")
	// ErrNoItems indica que não há itens pendentes ou selecionados.
	ErrNoItems = errors.New("nenhum item selecionado")
)

// DefaultDayLimitMinutes é o limite diário de tempo planejado usado quando
// DAYLIST_DAY_LIMIT não está definido.
const DefaultDayLimitMinutes = 8 * 60

// Plan é uma daylist pronta para prévia, mas ainda não gravada.
type Plan struct {
	Target   time.Time
	Week     string
	Fallback bool

	Resolved []daylist.Resolved
	Always   []daylist.Resolved
	Content  string

	TotalMin int
	TotalMax int
	HasTotal bool
	LimitMin int
}

// OverLimit informa se o tempo planejado máximo passa do limite diário.
func (p *Plan) OverLimit() bool {
	return p.HasTotal && p.LimitMin > 0 && p.TotalMax > p.LimitMin
}

// TotalText devolve o tempo planejado em formato humano.
func (p *Plan) TotalText() string {
	if !p.HasTotal {
		return "sem tempo planejado"
	}
	return FormatRange(p.TotalMin, p.TotalMax)
}

// Build executa o wizard e devolve o plano da daylist, sem gravar nada.
func Build(
	v *vault.Vault,
	target time.Time,
	prompter wizard.Prompter,
	out io.Writer,
	now time.Time,
) (*Plan, error) {
	path := v.DaylistFile(target)
	if fileExists(path) {
		return nil, fmt.Errorf("%w: %s", ErrDaylistExists, path)
	}

	weeklistPath, week, fallback, err := WeeklistForTarget(v, target)
	if err != nil {
		return nil, err
	}
	if fallback && out != nil {
		fmt.Fprintf(out, "Weeklist de %s não encontrada; usando a de %s.\n", vault.ISOWeek(target), week)
	}

	items, err := daylist.ParseWeeklist(weeklistPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf(
			"weeklist %s não encontrada: %s\nrode `daylist week` para criá-la",
			week, weeklistPath,
		)
	}
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("%w: nenhum item pendente na weeklist de %s", ErrNoItems, week)
	}
	if prompter == nil {
		return nil, errors.New("planner: UI não configurada")
	}

	result, err := wizard.Run(wizard.Options{
		Stages: daylist.Stages(items),
		UI:     prompter,
		Out:    out,
	})
	if err != nil {
		return nil, err
	}
	if result.Canceled {
		return nil, ErrCanceled
	}
	if len(result.Resolved) == 0 {
		return nil, ErrNoItems
	}

	plan := &Plan{
		Target:   target,
		Week:     week,
		Fallback: fallback,
		Resolved: result.Resolved,
		Content:  daylist.Render(result.Resolved, now),
		TotalMin: result.TotalMin,
		TotalMax: result.TotalMax,
		HasTotal: result.HasTotal,
		LimitMin: DayLimitMinutes(),
	}
	for _, item := range result.Resolved {
		if item.Always {
			plan.Always = append(plan.Always, item)
		}
	}
	return plan, nil
}

// Write grava o plano na daylist da data alvo.
func Write(v *vault.Vault, plan *Plan) error {
	return daylist.Write(v.DaylistFile(plan.Target), plan.Content)
}

// DayLimitMinutes lê DAYLIST_DAY_LIMIT (ex.: "8h", "7h30", "480 min") e cai
// para 8h quando a variável está vazia ou inválida.
func DayLimitMinutes() int {
	value := strings.TrimSpace(os.Getenv("DAYLIST_DAY_LIMIT"))
	if value == "" {
		return DefaultDayLimitMinutes
	}
	_, max, ok := daylist.ParseTimeRange(value)
	if !ok || max <= 0 {
		return DefaultDayLimitMinutes
	}
	return max
}

// FormatRange formata uma faixa de minutos. Quando mínimo e máximo são iguais,
// mostra um valor único.
func FormatRange(min, max int) string {
	if min == max {
		return FormatMinutes(min)
	}
	return FormatMinutes(min) + "–" + FormatMinutes(max)
}

// FormatMinutes formata minutos como "45 min", "8h" ou "7h30".
func FormatMinutes(total int) string {
	if total < 60 {
		return fmt.Sprintf("%d min", total)
	}
	hours := total / 60
	minutes := total % 60
	if minutes == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh%02d", hours, minutes)
}

// WeeklistForTarget devolve a weeklist da semana da data alvo; se não existir,
// cai para a semana anterior.
func WeeklistForTarget(v *vault.Vault, target time.Time) (path, week string, fallback bool, err error) {
	targetWeek := vault.ISOWeek(target)
	targetPath := v.WeeklistFile(targetWeek)
	if fileExists(targetPath) {
		return targetPath, targetWeek, false, nil
	}

	previousWeek := vault.ISOWeek(target.AddDate(0, 0, -7))
	previousPath := v.WeeklistFile(previousWeek)
	if fileExists(previousPath) {
		return previousPath, previousWeek, true, nil
	}

	return "", "", false, fmt.Errorf(
		"weeklist não encontrada para %s (nem a da semana anterior, %s)\ncaminhos tentados:\n  - %s\n  - %s\nrode `daylist week` para criar a weeklist da semana",
		targetWeek, previousWeek, targetPath, previousPath,
	)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
