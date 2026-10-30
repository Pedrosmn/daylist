package week

import (
	"regexp"
	"strings"
)

// HabitLine é uma linha de hábito já interpretada. Body preserva o conteúdo
// depois do checkbox (tempo, prioridade e descrição) para a weeklist; Desc
// contém só a descrição, usada pelo feedback.
type HabitLine struct {
	Checked bool
	Body    string
	Desc    string
}

var (
	checkboxRe = regexp.MustCompile(`^- \[([ xX])\] (.*)$`)

	// Mesmos formatos aceitos por internal/daylist, mas aplicados ao texto
	// depois do checkbox para podermos enxergar itens concluídos.
	habitTimeFlagRe = regexp.MustCompile(`^(.+?) \((.*?)\) - (.*)$`)
	habitTimeOnlyRe = regexp.MustCompile(`(?i)^(x?st|\d+(?:/\d+)?\s*min) - (.*)$`)
	habitFlagOnlyRe = regexp.MustCompile(`^\(([IUiu]*)\) - (.*)$`)
)

// ParseHabitLine reconhece os mesmos formatos do parser da daylist, mas
// também aceita itens concluídos ([x]).
func ParseHabitLine(line string) (HabitLine, bool) {
	line = strings.TrimRight(line, "\r")

	m := checkboxRe.FindStringSubmatch(line)
	if m == nil {
		return HabitLine{}, false
	}

	body := m[2]
	desc := strings.TrimSpace(body)

	if fm := habitTimeFlagRe.FindStringSubmatch(body); fm != nil {
		desc = strings.TrimSpace(fm[3])
	} else if tm := habitTimeOnlyRe.FindStringSubmatch(body); tm != nil {
		desc = strings.TrimSpace(tm[2])
	} else if fm := habitFlagOnlyRe.FindStringSubmatch(body); fm != nil {
		desc = strings.TrimSpace(fm[2])
	}

	return HabitLine{
		Checked: m[1] == "x" || m[1] == "X",
		Body:    body,
		Desc:    desc,
	}, true
}
