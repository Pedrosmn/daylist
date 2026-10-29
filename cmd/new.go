package cmd

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"daylist/internal/daylist"
	"daylist/internal/ui"
	"daylist/internal/vault"
)

var (
	newToday    bool
	newTomorrow bool
	newDate     string
	newDryRun   bool
)

var durationOptions = []string{
	"10 min", "15 min", "20 min", "30 min", "60 min", "120 min", "180 min", "Personalizado",
}

var flagOptions = []string{
	"Nenhuma", "Importante (I)", "Urgente (U)", "Importante e Urgente (IU)",
}

var newCmd = &cobra.Command{
	Use:     "new",
	Aliases: []string{"n"},
	Short:   "Cria a daylist no terminal (substitui o antigo -d)",
	Long: `Cria a daylist escolhendo as tasks pendentes da weeklist da semana da data alvo.

Se a weeklist da semana alvo não existir, usa a da semana anterior e avisa.
A seleção vai por etapas: primeiro # Tasks, depois cada categoria de hábito.
Em cada etapa, Enter marca/desmarca e "✅ Concluir" encerra; em seguida são
perguntados tempo e prioridade dos itens escolhidos naquela etapa.

Por padrão cria a daylist de amanhã. Se a daylist de hoje ainda não existir,
pergunta se você quer criar a de hoje ou a de amanhã. As flags pulam a pergunta.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		target, err := resolveTargetDate()
		if err != nil {
			return err
		}
		if target.IsZero() {
			fmt.Println("Cancelado.")
			return nil
		}
		return createDaylist(target, newDryRun)
	},
}

// resolveTargetDate decide qual daylist criar. Devolve a data zero quando o
// usuário cancela.
func resolveTargetDate() (time.Time, error) {
	now := time.Now()

	if newToday {
		return now, nil
	}
	if newTomorrow {
		return now.AddDate(0, 0, 1), nil
	}
	if newDate != "" {
		t, err := time.ParseInLocation("2006-01-02", newDate, time.Local)
		if err != nil {
			return time.Time{}, fmt.Errorf("data inválida %q (use YYYY-MM-DD)", newDate)
		}
		return t, nil
	}

	// Comportamento padrão: amanhã. Mas se a daylist de hoje não existe,
	// pergunta se o usuário quer a de hoje ou a de amanhã.
	if fileExists(v.DaylistFile(now)) {
		return now.AddDate(0, 0, 1), nil
	}

	options := []string{
		fmt.Sprintf("Hoje (%s)", now.Format("2006-01-02")),
		fmt.Sprintf("Amanhã (%s)", now.AddDate(0, 0, 1).Format("2006-01-02")),
	}
	choice, err := ui.Pick("A daylist de hoje não existe. Criar qual? ", options)
	if err != nil {
		return time.Time{}, err
	}
	if choice == "" {
		return time.Time{}, nil
	}
	if strings.HasPrefix(choice, "Hoje") {
		return now, nil
	}
	return now.AddDate(0, 0, 1), nil
}

// weeklistForTarget devolve a weeklist da semana da data alvo; se não
// existir, cai para a semana anterior.
func weeklistForTarget(v *vault.Vault, target time.Time) (path, week string, fallback bool, err error) {
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

// createDaylist executa o wizard por etapas e grava (ou só mostra, em dry-run).
func createDaylist(target time.Time, dryRun bool) error {
	path := v.DaylistFile(target)
	if fileExists(path) {
		fmt.Printf("Daylist de %s já existe: %s\n", target.Format("2006-01-02"), path)
		return nil
	}

	weeklistPath, week, fallback, err := weeklistForTarget(v, target)
	if err != nil {
		return err
	}
	if fallback {
		fmt.Printf("Weeklist de %s não encontrada; usando a de %s.\n", vault.ISOWeek(target), week)
	}

	items, err := daylist.ParseWeeklist(weeklistPath)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf(
			"weeklist %s não encontrada: %s\nrode `daylist week` para criá-la",
			week, weeklistPath,
		)
	}
	if err != nil {
		return err
	}
	if len(items) == 0 {
		fmt.Printf("Nenhum item pendente na weeklist de %s.\n", week)
		return nil
	}

	stages := daylist.Stages(items)
	var resolved []daylist.Resolved

	for i, stage := range stages {
		labels := make([]string, len(stage.Items))
		for j, it := range stage.Items {
			labels[j] = it.Desc
		}

		prompt := fmt.Sprintf("Etapa %d/%d · %s › ", i+1, len(stages), stage.Title)
		idxs, err := ui.Toggle(prompt, "✅ Concluir "+stage.Title, labels)
		if errors.Is(err, ui.ErrCanceled) {
			fmt.Println("Cancelado.")
			return nil
		}
		if err != nil {
			return err
		}

		selected := make([]daylist.Item, len(idxs))
		for k, idx := range idxs {
			selected[k] = stage.Items[idx]
		}

		stageResolved, err := resolveItems(selected)
		if err != nil {
			return err
		}
		if stageResolved == nil { // cancelou na duração/prioridade
			fmt.Println("Cancelado.")
			return nil
		}

		printStageSummary(stage.Title, stageResolved)
		resolved = append(resolved, stageResolved...)
	}

	if len(resolved) == 0 {
		fmt.Println("Nenhum item selecionado.")
		return nil
	}

	content := daylist.Render(resolved, time.Now())

	if dryRun {
		fmt.Printf("--- Daylist de %s (dry-run, nada foi gravado) ---\n", target.Format("2006-01-02"))
		fmt.Print(content)
		return nil
	}

	if err := daylist.Write(path, content); err != nil {
		return err
	}
	fmt.Printf("Daylist de %s criada: %s\n", target.Format("2006-01-02"), path)
	return nil
}

// printStageSummary mostra o que ficou definido na etapa antes de ir para a
// próxima.
func printStageSummary(title string, items []daylist.Resolved) {
	if len(items) == 0 {
		return
	}

	word := "itens"
	if len(items) == 1 {
		word = "item"
	}
	fmt.Printf("%s — %d %s:\n", title, len(items), word)
	for _, it := range items {
		fmt.Printf("  - %s (%s) - %s\n", it.FinalTime, it.FinalFlag, it.Desc)
	}
	fmt.Println()
}

// resolveItems pergunta tempo e prioridade do que ainda está indefinido.
// Devolve nil quando o usuário cancela.
func resolveItems(selected []daylist.Item) ([]daylist.Resolved, error) {
	resolved := make([]daylist.Resolved, 0, len(selected))

	for _, it := range selected {
		finalTime, ok, err := resolveTime(it)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, nil
		}

		finalFlag, ok, err := resolveFlag(it)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, nil
		}

		resolved = append(resolved, daylist.Resolved{
			Item:      it,
			FinalTime: finalTime,
			FinalFlag: finalFlag,
		})
	}
	return resolved, nil
}

func resolveTime(it daylist.Item) (string, bool, error) {
	current := strings.TrimSpace(it.Time)
	if current != "" && !isShortTime(current) {
		return current, true, nil
	}

	choice, err := ui.Pick(fmt.Sprintf("Duração de %q: ", it.Desc), durationOptions)
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
		value, err := ui.Input("Duração personalizada (min): ")
		if err != nil {
			return "", false, err
		}
		mins, convErr := strconv.Atoi(value)
		if convErr == nil && mins > 0 {
			return fmt.Sprintf("%d min", mins), true, nil
		}
		fmt.Fprintln(os.Stderr, "Valor inválido, tente de novo (Ctrl+C para sair).")
	}
}

func resolveFlag(it daylist.Item) (string, bool, error) {
	if it.Flag != "" {
		return it.Flag, true, nil
	}

	choice, err := ui.Pick(fmt.Sprintf("Prioridade de %q: ", it.Desc), flagOptions)
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func init() {
	newCmd.Flags().BoolVar(&newToday, "today", false, "cria a daylist de hoje")
	newCmd.Flags().BoolVar(&newTomorrow, "tomorrow", false, "cria a daylist de amanhã")
	newCmd.Flags().StringVar(&newDate, "date", "", "cria a daylist da data informada (YYYY-MM-DD)")
	newCmd.Flags().BoolVar(&newDryRun, "dry-run", false, "mostra o conteúdo sem gravar")
	newCmd.MarkFlagsMutuallyExclusive("today", "tomorrow", "date")

	rootCmd.AddCommand(newCmd)
}
