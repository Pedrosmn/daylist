package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"daylist/internal/daylist"
	"daylist/internal/vault"
	"daylist/internal/week"
)

type feedbackOptions struct {
	Week      string
	DryRun    bool
	PrintPath bool
}

var feedbackOpts feedbackOptions

var feedbackCmd = &cobra.Command{
	Use:     "feedback",
	Aliases: []string{"f"},
	Short:   "Gera o feedback da semana",
	Long: `Gera o feedback da semana atual (ou da semana informada em --week),
contando os hábitos concluídos nas daylists de segunda a domingo.

Com --dry-run o conteúdo é mostrado sem gravar. Com --print-path apenas o
caminho do feedback é impresso.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFeedback(v, feedbackOpts, time.Now())
	},
}

func feedbackWeek(now time.Time, opts feedbackOptions) (string, error) {
	if opts.Week == "" {
		return vault.ISOWeek(now), nil
	}
	if _, err := week.ParseWeek(opts.Week); err != nil {
		return "", fmt.Errorf("semana inválida %q (use YYYY-Www)", opts.Week)
	}
	return opts.Week, nil
}

func runFeedback(v *vault.Vault, opts feedbackOptions, now time.Time) error {
	targetWeek, err := feedbackWeek(now, opts)
	if err != nil {
		return err
	}
	targetPath := v.FeedbackFile(targetWeek)

	if opts.PrintPath {
		fmt.Println(targetPath)
		return nil
	}

	if fileExists(targetPath) {
		fmt.Printf("Feedback já existe: %s\n", targetPath)
		return nil
	}

	weeklistPath := v.WeeklistFile(targetWeek)
	weeklist, err := os.ReadFile(weeklistPath)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("weeklist de %s não encontrada: %s", targetWeek, weeklistPath)
	}
	if err != nil {
		return err
	}

	start, err := week.ParseWeek(targetWeek)
	if err != nil {
		return err
	}

	var days []week.Day
	for _, day := range week.WeekDays(start) {
		path := v.DaylistFile(day)
		content, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		days = append(days, week.Day{Date: day, Content: string(content)})
	}

	content := week.BuildFeedback(targetWeek, string(weeklist), days, now)
	if opts.DryRun {
		fmt.Print(content)
		return nil
	}

	if err := daylist.Write(targetPath, content); err != nil {
		return err
	}
	cleanupLegacyTargets(v)
	fmt.Printf("Feedback de %s criado: %s\n", targetWeek, targetPath)
	return nil
}

func init() {
	feedbackCmd.Flags().StringVar(&feedbackOpts.Week, "week", "", "semana a gerar (YYYY-Www; padrão: atual)")
	feedbackCmd.Flags().BoolVar(&feedbackOpts.DryRun, "dry-run", false, "mostra o conteúdo sem gravar")
	feedbackCmd.Flags().BoolVar(&feedbackOpts.PrintPath, "print-path", false, "imprime o caminho sem gravar")
	feedbackCmd.MarkFlagsMutuallyExclusive("dry-run", "print-path")

	rootCmd.AddCommand(feedbackCmd)
}
