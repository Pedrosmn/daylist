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

type weekOptions struct {
	Current   bool
	Next      bool
	DryRun    bool
	PrintPath bool
}

var weekOpts weekOptions

var weekCmd = &cobra.Command{
	Use:     "week",
	Aliases: []string{"w"},
	Short:   "Cria a weeklist da próxima semana",
	Long: `Cria a weeklist da próxima semana (ou da atual com --current), copiando os
hábitos da semana anterior. Tempo e prioridade são preservados; apenas o
checkbox volta para pendente.

Com --dry-run o conteúdo é mostrado sem gravar. Com --print-path apenas o
caminho da weeklist é impresso.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWeek(v, weekOpts, time.Now())
	},
}

func weekTarget(now time.Time, opts weekOptions) string {
	if opts.Current {
		return vault.ISOWeek(now)
	}
	return vault.ISOWeek(now.AddDate(0, 0, 7))
}

func runWeek(v *vault.Vault, opts weekOptions, now time.Time) error {
	targetWeek := weekTarget(now, opts)
	targetPath := v.WeeklistFile(targetWeek)

	if opts.PrintPath {
		fmt.Println(targetPath)
		return nil
	}

	if fileExists(targetPath) {
		fmt.Printf("Weeklist já existe: %s\n", targetPath)
		return nil
	}

	start, err := week.ParseWeek(targetWeek)
	if err != nil {
		return err
	}
	sourceWeek := week.WeekString(week.PreviousWeek(start))
	sourcePath := v.WeeklistFile(sourceWeek)

	source, err := os.ReadFile(sourcePath)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(os.Stderr,
			"Aviso: weeklist de %s não encontrada (%s); criando sem hábitos.\n",
			sourceWeek, sourcePath)
		source = nil
	} else if err != nil {
		return err
	}

	content := week.BuildWeeklist(string(source))
	if opts.DryRun {
		fmt.Print(content)
		return nil
	}

	if err := daylist.Write(targetPath, content); err != nil {
		return err
	}
	cleanupLegacyTargets(v)
	fmt.Printf("Weeklist de %s criada: %s\n", targetWeek, targetPath)
	return nil
}

func init() {
	weekCmd.Flags().BoolVar(&weekOpts.Current, "current", false, "cria a weeklist da semana atual")
	weekCmd.Flags().BoolVar(&weekOpts.Next, "next", false, "cria a weeklist da próxima semana (padrão)")
	weekCmd.Flags().BoolVar(&weekOpts.DryRun, "dry-run", false, "mostra o conteúdo sem gravar")
	weekCmd.Flags().BoolVar(&weekOpts.PrintPath, "print-path", false, "imprime o caminho sem gravar")
	weekCmd.MarkFlagsMutuallyExclusive("current", "next")
	weekCmd.MarkFlagsMutuallyExclusive("dry-run", "print-path")

	rootCmd.AddCommand(weekCmd)
}
