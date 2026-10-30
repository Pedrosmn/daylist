// Package wizard conduz o fluxo interativo do `daylist new`.
package wizard

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"daylist/internal/daylist"
	"daylist/internal/ui"
)

// Prompter é a interface mínima de terminal usada pelo wizard. Ela permite
// testar o fluxo sem TTY.
type Prompter interface {
	Pick(prompt string, items []string) (string, error)
	Toggle(opts ui.ToggleOptions) (ui.ToggleResult, error)
	Input(prompt string) (string, error)
	Confirm(prompt string) (bool, error)
}

// Options configura a execução do wizard.
type Options struct {
	Stages []daylist.Stage
	UI     Prompter
	Out    io.Writer
}

// Result é o resultado do wizard. Canceled indica que o usuário descartou o
// fluxo; HasTotal indica que alguma duração pôde ser somada.
type Result struct {
	Resolved []daylist.Resolved
	TotalMin int
	TotalMax int
	HasTotal bool
	Canceled bool
}

type stageState struct {
	marked   []int
	resolved map[int]daylist.Resolved
}

var (
	durationOptions = []string{
		"10 min", "15 min", "20 min", "30 min", "60 min", "120 min", "180 min", "Personalizado",
	}
	flagOptions = []string{
		"Nenhuma", "Importante (I)", "Urgente (U)", "Importante e Urgente (IU)",
	}

	minutesRangeRe = regexp.MustCompile(`(?i)^(\d+)(?:\s*/\s*(\d+))?\s*min$`)
	hoursRangeRe   = regexp.MustCompile(`(?i)^(\d+)\s*h(?:\s*(\d+)\s*(?:min|m)?)?$`)
)

// Run executa todas as etapas e devolve os itens resolvidos.
func Run(opts Options) (Result, error) {
	if opts.UI == nil {
		return Result{}, errors.New("wizard: UI não configurada")
	}
	out := opts.Out
	if out == nil {
		out = io.Discard
	}

	states := make([]stageState, len(opts.Stages))

	for i := 0; i < len(opts.Stages); {
		stage := opts.Stages[i]
		state := &states[i]

		labels := make([]string, len(stage.Items))
		for j, it := range stage.Items {
			labels[j] = it.Desc
		}

		res, err := opts.UI.Toggle(ui.ToggleOptions{
			Prompt:      fmt.Sprintf("Etapa %d/%d · %s › ", i+1, len(opts.Stages), stage.Title),
			FinishLabel: "✅ Concluir " + stage.Title,
			Items:       labels,
			Selected:    state.marked,
			AllowBack:   i > 0,
			Header:      fmt.Sprintf("Etapa %d/%d · %s", i+1, len(opts.Stages), stage.Title),
		})
		if err != nil {
			return Result{}, err
		}

		switch res.Action {
		case ui.ToggleCancel:
			state.marked = res.Selected
			discard, err := confirmDiscard(opts.UI, states)
			if err != nil {
				return Result{}, err
			}
			if !discard {
				continue
			}
			return Result{Canceled: true}, nil

		case ui.ToggleBack:
			state.marked = res.Selected
			if i > 0 {
				i--
			}

		case ui.ToggleFinish:
			state.marked = res.Selected
			resolved, canceled, err := resolveStage(opts.UI, out, stage.Items, state)
			if err != nil {
				return Result{}, err
			}
			if canceled {
				discard, err := confirmDiscard(opts.UI, states)
				if err != nil {
					return Result{}, err
				}
				if !discard {
					continue
				}
				return Result{Canceled: true}, nil
			}
			state.resolved = resolved
			printStageSummary(out, stage.Title, resolvedFor(state))
			i++

		default:
			return Result{}, fmt.Errorf("wizard: ação desconhecida %d", res.Action)
		}
	}

	var all []daylist.Resolved
	for _, state := range states {
		all = append(all, resolvedFor(&state)...)
	}

	result := Result{Resolved: all}
	result.TotalMin, result.TotalMax, result.HasTotal = totalTime(all)
	if result.HasTotal {
		fmt.Fprintf(out, "Tempo planejado: %s\n", formatRange(result.TotalMin, result.TotalMax))
	}
	return result, nil
}

// resolveStage pergunta tempo e prioridade apenas do que ainda não foi
// resolvido. Devolve canceled quando o usuário interrompe a resolução.
func resolveStage(p Prompter, out io.Writer, items []daylist.Item, state *stageState) (map[int]daylist.Resolved, bool, error) {
	resolved := make(map[int]daylist.Resolved, len(state.marked))

	for _, idx := range state.marked {
		if idx < 0 || idx >= len(items) {
			continue
		}
		if old, ok := state.resolved[idx]; ok {
			resolved[idx] = old
			continue
		}

		it, ok, err := resolveItem(p, out, items[idx])
		if err != nil {
			return nil, false, err
		}
		if !ok {
			return nil, true, nil
		}
		resolved[idx] = it
	}
	return resolved, false, nil
}

func resolveItem(p Prompter, out io.Writer, it daylist.Item) (daylist.Resolved, bool, error) {
	finalTime, ok, err := resolveTime(p, out, it)
	if err != nil {
		return daylist.Resolved{}, false, err
	}
	if !ok {
		return daylist.Resolved{}, false, nil
	}

	finalFlag, ok, err := resolveFlag(p, it)
	if err != nil {
		return daylist.Resolved{}, false, err
	}
	if !ok {
		return daylist.Resolved{}, false, nil
	}

	return daylist.Resolved{Item: it, FinalTime: finalTime, FinalFlag: finalFlag}, true, nil
}

func resolveTime(p Prompter, out io.Writer, it daylist.Item) (string, bool, error) {
	current := strings.TrimSpace(it.Time)
	if current != "" && !isShortTime(current) {
		return current, true, nil
	}

	choice, err := p.Pick(fmt.Sprintf("Duração de %q: ", it.Desc), durationOptions)
	if err != nil {
		return "", false, err
	}
	if choice == "" { // Esc cancela o fluxo inteiro
		return "", false, nil
	}
	if choice != "Personalizado" {
		return choice, true, nil
	}

	for {
		value, err := p.Input("Duração personalizada (min): ")
		if err != nil {
			return "", false, err
		}
		mins, convErr := strconv.Atoi(value)
		if convErr == nil && mins > 0 {
			return fmt.Sprintf("%d min", mins), true, nil
		}
		fmt.Fprintln(out, "Valor inválido, tente de novo (Ctrl+C para sair).")
	}
}

func resolveFlag(p Prompter, it daylist.Item) (string, bool, error) {
	if it.Flag != "" {
		return it.Flag, true, nil
	}

	choice, err := p.Pick(fmt.Sprintf("Prioridade de %q: ", it.Desc), flagOptions)
	if err != nil {
		return "", false, err
	}
	if choice == "" { // Esc cancela
		return "", false, nil
	}
	return flagValue(choice), true, nil
}

func flagValue(label string) string {
	switch label {
	case "Importante (I)":
		return "I"
	case "Urgente (U)":
		return "U"
	case "Importante e Urgente (IU)":
		return "IU"
	default:
		return ""
	}
}

// isShortTime: tempos "ST"/"xST" na weeklist pedem duração na hora.
func isShortTime(t string) bool {
	switch strings.ToUpper(strings.TrimSpace(t)) {
	case "ST", "XST":
		return true
	}
	return false
}

func confirmDiscard(p Prompter, states []stageState) (bool, error) {
	if !hasSelection(states) {
		return true, nil
	}
	return p.Confirm("Descartar tudo? ")
}

func hasSelection(states []stageState) bool {
	for _, state := range states {
		if len(state.marked) > 0 {
			return true
		}
	}
	return false
}

func resolvedFor(state *stageState) []daylist.Resolved {
	out := make([]daylist.Resolved, 0, len(state.marked))
	for _, idx := range state.marked {
		if it, ok := state.resolved[idx]; ok {
			out = append(out, it)
		}
	}
	return out
}

func printStageSummary(out io.Writer, title string, items []daylist.Resolved) {
	if len(items) == 0 {
		return
	}

	word := "itens"
	if len(items) == 1 {
		word = "item"
	}
	fmt.Fprintf(out, "%s — %d %s:\n", title, len(items), word)
	for _, it := range items {
		fmt.Fprintf(out, "  - %s (%s) - %s\n", it.FinalTime, it.FinalFlag, it.Desc)
	}
	fmt.Fprintln(out)
}

func totalTime(items []daylist.Resolved) (min, max int, ok bool) {
	for _, it := range items {
		lo, hi, valid := parseTimeRange(it.FinalTime)
		if !valid {
			continue
		}
		min += lo
		max += hi
		ok = true
	}
	return min, max, ok
}

func parseTimeRange(value string) (min, max int, ok bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, 0, false
	}

	if m := minutesRangeRe.FindStringSubmatch(value); m != nil {
		lo, err := strconv.Atoi(m[1])
		if err != nil {
			return 0, 0, false
		}
		hi := lo
		if m[2] != "" {
			hi, err = strconv.Atoi(m[2])
			if err != nil {
				return 0, 0, false
			}
		}
		if lo > hi {
			lo, hi = hi, lo
		}
		return lo, hi, true
	}

	if m := hoursRangeRe.FindStringSubmatch(value); m != nil {
		hours, err := strconv.Atoi(m[1])
		if err != nil {
			return 0, 0, false
		}
		minutes := 0
		if m[2] != "" {
			minutes, err = strconv.Atoi(m[2])
			if err != nil {
				return 0, 0, false
			}
		}
		total := hours*60 + minutes
		return total, total, true
	}

	return 0, 0, false
}

func formatRange(min, max int) string {
	if min == max {
		return formatMinutes(min)
	}
	return formatMinutes(min) + "–" + formatMinutes(max)
}

func formatMinutes(total int) string {
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
