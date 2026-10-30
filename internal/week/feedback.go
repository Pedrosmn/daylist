package week

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
)

// Day é o conteúdo de uma daylist que entra no cálculo do feedback.
type Day struct {
	Date    time.Time
	Content string
}

// HabitStat acumula as ocorrências de um hábito na semana.
type HabitStat struct {
	Category string
	Desc     string
	Checked  int
	Total    int
}

var timerSuffixRe = regexp.MustCompile(`\s*⏱.*$`)

// StripTimerSuffix remove " ⏱ ..." da descrição.
func StripTimerSuffix(desc string) string {
	return strings.TrimSpace(timerSuffixRe.ReplaceAllString(desc, ""))
}

// BuildFeedback monta o feedback a partir da weeklist e dos conteúdos dos
// dias da semana (na ordem segunda → domingo).
func BuildFeedback(week, weeklist string, days []Day, createdAt time.Time) string {
	type habitKey struct {
		category string
		desc     string
	}

	stats := map[habitKey]*HabitStat{}
	var statOrder []habitKey
	var categoryOrder []string
	seenCategory := map[string]bool{}

	addCategory := func(category string) {
		if category == "" || seenCategory[category] {
			return
		}
		seenCategory[category] = true
		categoryOrder = append(categoryOrder, category)
	}

	addHabit := func(category, desc string) *HabitStat {
		key := habitKey{category: category, desc: desc}
		if stat, ok := stats[key]; ok {
			return stat
		}
		stat := &HabitStat{Category: category, Desc: desc}
		stats[key] = stat
		statOrder = append(statOrder, key)
		return stat
	}

	// A weeklist define as categorias e os hábitos que devem aparecer mesmo
	// que nunca tenham sido usados na semana.
	inHabits, category := false, ""
	for _, line := range strings.Split(weeklist, "\n") {
		line = strings.TrimRight(line, "\r")

		switch {
		case strings.HasPrefix(line, "# Hábitos"):
			inHabits, category = true, ""
			continue
		case inHabits && strings.HasPrefix(line, "# "):
			inHabits = false
			continue
		case inHabits && strings.HasPrefix(line, "## "):
			category = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			addCategory(category)
			continue
		}
		if !inHabits {
			continue
		}

		habit, ok := ParseHabitLine(line)
		if !ok {
			continue
		}
		addHabit(category, StripTimerSuffix(habit.Desc))
	}

	// Cada ocorrência numa daylist incrementa o total; [x] incrementa também
	// o número de concluídos.
	for _, day := range days {
		inHabits, category = false, ""
		for _, line := range strings.Split(day.Content, "\n") {
			line = strings.TrimRight(line, "\r")

			switch {
			case strings.HasPrefix(line, "# Hábitos"):
				inHabits, category = true, ""
				continue
			case inHabits && strings.HasPrefix(line, "# "):
				inHabits = false
				continue
			case inHabits && strings.HasPrefix(line, "## "):
				category = strings.TrimSpace(strings.TrimPrefix(line, "## "))
				continue
			}
			if !inHabits {
				continue
			}

			habit, ok := ParseHabitLine(line)
			if !ok {
				continue
			}

			desc := StripTimerSuffix(habit.Desc)
			key := habitKey{category: category, desc: desc}
			stat, exists := stats[key]
			if !exists {
				addCategory(category)
				stat = addHabit(category, desc)
			}
			stat.Total++
			if habit.Checked {
				stat.Checked++
			}
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Feedback - Semana %s\n", week)
	fmt.Fprintf(&b, "Criado em: %s\n\n", createdAt.Format("2006-01-02 15:04"))
	for _, category := range categoryOrder {
		fmt.Fprintf(&b, "## %s\n\n", category)
	}
	b.WriteString("---\n# Resumo\n")

	if len(stats) == 0 {
		b.WriteString("Nenhum hábito encontrado na weeklist.\n")
		return b.String()
	}

	totalChecked, totalHabits := 0, 0
	for _, key := range statOrder {
		stat := stats[key]
		totalChecked += stat.Checked
		totalHabits += stat.Total
	}
	fmt.Fprintf(&b, "Hábitos concluídos na semana: %d/%d (%d%%)\n",
		totalChecked, totalHabits, percentage(totalChecked, totalHabits))

	for _, category := range categoryOrder {
		fmt.Fprintf(&b, "\n## %s\n", category)
		for _, key := range statOrder {
			stat := stats[key]
			if stat.Category != category {
				continue
			}
			fmt.Fprintf(&b, "%s %d/%d - (%d%%)\n",
				stat.Desc, stat.Checked, stat.Total, percentage(stat.Checked, stat.Total))
		}
	}

	return b.String()
}

func percentage(checked, total int) int {
	if total == 0 {
		return 0
	}
	return int(math.Round(float64(checked) / float64(total) * 100))
}
