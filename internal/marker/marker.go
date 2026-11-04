// Package marker reconhece marcadores visíveis usados nas notas.
package marker

import (
	"regexp"
	"strings"
)

// Always é o marcador visível de hábitos que devem entrar em toda daylist da
// semana.
const Always = "#always"

var alwaysSuffixRe = regexp.MustCompile(`(?i)\s+#always\s*$`)

// StripAlways remove um #always no fim do texto e informa se ele existia.
// Marcadores no meio da descrição ou colados a outra palavra são preservados.
func StripAlways(s string) (string, bool) {
	loc := alwaysSuffixRe.FindStringIndex(s)
	if loc == nil {
		return s, false
	}
	return strings.TrimSpace(s[:loc[0]]), true
}
