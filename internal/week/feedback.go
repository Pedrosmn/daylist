package week

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Day é o conteúdo de uma daylist que entra no cálculo do feedback.
type Day struct {
	Date    time.Time
	Content string
}

var timerSuffixRe = regexp.MustCompile(`\s*⏱.*$`)

// StripTimerSuffix remove " ⏱ ..." da descrição.
func StripTimerSuffix(desc string) string {
	return strings.TrimSpace(timerSuffixRe.ReplaceAllString(desc, ""))
}

// BuildFeedback monta o feedback a partir da weeklist e dos conteúdos dos
// dias da semana (na ordem segunda → domingo). A saída é apenas a renderização
// do relatório compartilhado calculado por ComputeReport.
func BuildFeedback(week, weeklist string, days []Day, createdAt time.Time) string {
	report := ComputeReport(week, weeklist, days)

	var b strings.Builder
	fmt.Fprintf(&b, "# Feedback - Semana %s\n", week)
	fmt.Fprintf(&b, "Criado em: %s\n\n", createdAt.Format("2006-01-02 15:04"))
	for _, category := range report.Categories {
		if category.Name == "" {
			continue
		}
		fmt.Fprintf(&b, "## %s\n\n", category.Name)
	}
	b.WriteString("---\n# Resumo\n")

	if !report.HasHabits() {
		b.WriteString("Nenhum hábito encontrado na weeklist.\n")
		return b.String()
	}

	fmt.Fprintf(&b, "Hábitos concluídos na semana: %d/%d (%d%%)\n",
		report.Checked, report.Total, report.Percent)

	for _, category := range report.Categories {
		if category.Name == "" {
			continue
		}
		fmt.Fprintf(&b, "\n## %s\n", category.Name)
		for _, habit := range category.Habits {
			fmt.Fprintf(&b, "%s %d/%d - (%d%%)\n",
				habit.Desc, habit.Checked, habit.Total, habit.Percent)
		}
	}

	return b.String()
}
