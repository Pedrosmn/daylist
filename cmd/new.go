package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"daylist/internal/planner"
	"daylist/internal/ui"
	"daylist/internal/vault"
)

type newOptions struct {
	Today    bool
	Tomorrow bool
	Date     string
	DryRun   bool
}

var newOpts newOptions

var newCmd = &cobra.Command{
	Use:     "new",
	Aliases: []string{"n"},
	Short:   "Cria a daylist no terminal (substitui o antigo -d)",
	Long: `Cria a daylist escolhendo as tasks pendentes da weeklist da semana da data alvo.

Se a weeklist da semana alvo não existir, usa a da semana anterior e avisa.
A seleção vai por etapas: primeiro # Tasks, depois cada categoria de hábito.
Em cada etapa, j/k (ou Ctrl-d/Ctrl-u) navegam, g/G vão ao início/fim,
Enter/Espaço marcam/desmarcam, h volta, l conclui e q/Esc cancela.
Ao final, o tempo total planejado é exibido.

Por padrão cria a daylist de amanhã. Se a daylist de hoje ainda não existir,
pergunta se você quer criar a de hoje ou a de amanhã. As flags pulam a pergunta.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNew(v, newOpts, time.Now(), ui.Pick)
	},
}

// runNew resolve a data alvo e cria a daylist correspondente.
func runNew(v *vault.Vault, opts newOptions, now time.Time, pick func(string, []string) (string, error)) error {
	todayExists := fileExists(v.DaylistFile(now))

	target, err := resolveTargetDate(now, opts, todayExists, pick)
	if err != nil {
		return err
	}
	if target.IsZero() {
		fmt.Println("Cancelado.")
		return nil
	}
	return createDaylist(v, target, opts.DryRun)
}

// resolveTargetDate decide qual daylist criar. Devolve a data zero quando o
// usuário cancela.
func resolveTargetDate(
	now time.Time,
	opts newOptions,
	todayExists bool,
	pick func(string, []string) (string, error),
) (time.Time, error) {
	if opts.Today {
		return now, nil
	}
	if opts.Tomorrow {
		return now.AddDate(0, 0, 1), nil
	}
	if opts.Date != "" {
		t, err := time.ParseInLocation("2006-01-02", opts.Date, time.Local)
		if err != nil {
			return time.Time{}, fmt.Errorf("data inválida %q (use YYYY-MM-DD)", opts.Date)
		}
		return t, nil
	}

	// Comportamento padrão: amanhã. Mas se a daylist de hoje não existe,
	// pergunta se o usuário quer a de hoje ou a de amanhã.
	tomorrow := now.AddDate(0, 0, 1)
	if todayExists {
		return tomorrow, nil
	}

	options := []string{
		fmt.Sprintf("Hoje (%s)", now.Format("2006-01-02")),
		fmt.Sprintf("Amanhã (%s)", tomorrow.Format("2006-01-02")),
	}
	choice, err := pick("A daylist de hoje não existe. Criar qual? ", options)
	if err != nil {
		return time.Time{}, err
	}
	if choice == "" {
		return time.Time{}, nil
	}
	if strings.HasPrefix(choice, "Hoje") {
		return now, nil
	}
	return tomorrow, nil
}

// createDaylist monta o plano pelo wizard e grava (ou só mostra, em dry-run).
func createDaylist(v *vault.Vault, target time.Time, dryRun bool) error {
	path := v.DaylistFile(target)
	if fileExists(path) {
		fmt.Printf("Daylist de %s já existe: %s\n", target.Format("2006-01-02"), path)
		return nil
	}

	plan, err := planner.Build(v, target, uiPrompter{}, os.Stdout, time.Now())
	switch {
	case errors.Is(err, planner.ErrCanceled):
		fmt.Println("Cancelado.")
		return nil
	case errors.Is(err, planner.ErrNoItems):
		fmt.Println(err)
		return nil
	case errors.Is(err, planner.ErrDaylistExists):
		fmt.Printf("Daylist de %s já existe: %s\n", target.Format("2006-01-02"), path)
		return nil
	case err != nil:
		return err
	}

	if dryRun {
		fmt.Printf("--- Daylist de %s (dry-run, nada foi gravado) ---\n", target.Format("2006-01-02"))
		fmt.Print(plan.Content)
		return nil
	}

	if err := planner.Write(v, plan); err != nil {
		return err
	}
	fmt.Printf("Daylist de %s criada: %s\n", target.Format("2006-01-02"), path)
	return nil
}

type uiPrompter struct{}

func (uiPrompter) Pick(prompt string, items []string) (string, error) {
	return ui.Pick(prompt, items)
}

func (uiPrompter) Toggle(opts ui.ToggleOptions) (ui.ToggleResult, error) {
	return ui.ToggleWithOptions(opts)
}

func (uiPrompter) Input(prompt string) (string, error) {
	return ui.Input(prompt)
}

func (uiPrompter) Confirm(prompt string) (bool, error) {
	return ui.Confirm(prompt)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func init() {
	newCmd.Flags().BoolVar(&newOpts.Today, "today", false, "cria a daylist de hoje")
	newCmd.Flags().BoolVar(&newOpts.Tomorrow, "tomorrow", false, "cria a daylist de amanhã")
	newCmd.Flags().StringVar(&newOpts.Date, "date", "", "cria a daylist da data informada (YYYY-MM-DD)")
	newCmd.Flags().BoolVar(&newOpts.DryRun, "dry-run", false, "mostra o conteúdo sem gravar")
	newCmd.MarkFlagsMutuallyExclusive("today", "tomorrow", "date")

	rootCmd.AddCommand(newCmd)
}
