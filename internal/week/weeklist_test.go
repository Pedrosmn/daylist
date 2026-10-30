package week

import "testing"

const weeklistSource = `# Tasks
- [ ] tarefa qualquer

---
# Hábitos
## Saúde
- [x] 10 min (I) - correr
- [ ] xST () - dormir

## Estudos
- [ ] 20 min (U) - ler
- [ ] 5 min - revisar
- [ ] (I) - planejar
- [ ] livre
# Extra
- [ ] ignorar
`

func TestBuildWeeklistCopiesHabits(t *testing.T) {
	const want = `# Tasks

---
# Hábitos
## Saúde
- [ ] 10 min (I) - correr
- [ ] xST () - dormir
## Estudos
- [ ] 20 min (U) - ler
- [ ] 5 min - revisar
- [ ] (I) - planejar
- [ ] livre
`
	if got := BuildWeeklist(weeklistSource); got != want {
		t.Errorf("weeklist diferente\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}
}

func TestBuildWeeklistEmptySource(t *testing.T) {
	const want = "# Tasks\n\n---\n# Hábitos\n"
	if got := BuildWeeklist(""); got != want {
		t.Errorf("BuildWeeklist(\"\") = %q, want %q", got, want)
	}
}

func TestBuildWeeklistCRLF(t *testing.T) {
	lf := BuildWeeklist(weeklistSource)
	crlf := BuildWeeklist(toCRLF(weeklistSource))
	if lf != crlf {
		t.Errorf("CRLF gerou resultado diferente\n--- lf ---\n%q\n--- crlf ---\n%q", lf, crlf)
	}
}

func TestBuildWeeklistIgnoresNonChecklistLines(t *testing.T) {
	source := `# Hábitos
## Saúde
comentário
- [ ]
- [y] inválido
- [ ] 5 min (U) - válido
`
	const want = `# Tasks

---
# Hábitos
## Saúde
- [ ] 5 min (U) - válido
`
	if got := BuildWeeklist(source); got != want {
		t.Errorf("BuildWeeklist = %q, want %q", got, want)
	}
}

func toCRLF(s string) string {
	out := make([]byte, 0, len(s)+len(s)/20)
	for _, r := range s {
		if r == '\n' {
			out = append(out, '\r', '\n')
			continue
		}
		out = append(out, string(r)...)
	}
	return string(out)
}
