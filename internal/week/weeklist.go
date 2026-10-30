package week

import "strings"

// BuildWeeklist monta a nova weeklist a partir do conteúdo da weeklist de
// origem. Fonte vazia gera uma weeklist só com as seções base.
//
// Os hábitos são copiados preservando tempo, prioridade e descrição; apenas o
// checkbox é resetado para pendente.
func BuildWeeklist(source string) string {
	var habits []string
	inHabits := false

	for _, line := range strings.Split(source, "\n") {
		line = strings.TrimRight(line, "\r")

		if strings.HasPrefix(line, "# Hábitos") {
			inHabits = true
			continue
		}
		if inHabits && strings.HasPrefix(line, "# ") {
			inHabits = false
			continue
		}
		if !inHabits {
			continue
		}

		if strings.HasPrefix(line, "## ") {
			habits = append(habits, line)
			continue
		}

		habit, ok := ParseHabitLine(line)
		if !ok {
			continue
		}
		habits = append(habits, "- [ ] "+habit.Body)
	}

	var b strings.Builder
	b.WriteString("# Tasks\n\n---\n# Hábitos\n")
	for _, line := range habits {
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}
