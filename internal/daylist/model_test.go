package daylist

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"daylist/internal/week"
)

const modelNote = `2026-11-05 08:00

# Tasks

- [ ] 40 min (I) - Escrever relatório ⏱ 12min (08:00–08:12)
> terminar antes do almoço
- [x] comprar café
texto desconhecido
---
# Hábitos

## Estudos
- [ ] xST (IU) - Estudar DP-900
- [x] 5 min (U) - AWS ⏱ 5min (09:00–09:05)
> revisar certificado
> segunda linha

## Rotina
- [ ] 5 min () - Fio dental
---
# Extra

- [ ] 15 min (I) - Task extra
`

func findNoteItem(t *testing.T, note *Note, desc string) NoteItem {
	t.Helper()
	for _, item := range note.Items() {
		if item.Desc == desc {
			return item
		}
	}
	t.Fatalf("item %q não encontrado em %+v", desc, note.Items())
	return NoteItem{}
}

func TestParseNote(t *testing.T) {
	note := ParseNote(modelNote)

	if got := note.String(); got != modelNote {
		t.Errorf("String não preservou o conteúdo\n--- got ---\n%q\n--- want ---\n%q", got, modelNote)
	}

	items := note.Items()
	if len(items) != 6 {
		t.Fatalf("esperava 6 itens, veio %d: %+v", len(items), items)
	}

	report := items[0]
	if report.Section != "Tasks" || report.Category != "" ||
		report.Checked || report.Time != "40 min" || report.Flag != "I" ||
		report.Desc != "Escrever relatório" ||
		report.Timer != "12min (08:00–08:12)" || report.TimerMinutes != 12 {
		t.Errorf("task inesperada: %+v", report)
	}
	if report.Raw != "- [ ] 40 min (I) - Escrever relatório ⏱ 12min (08:00–08:12)" {
		t.Errorf("Raw inesperado: %q", report.Raw)
	}
	if report.Motivation != "terminar antes do almoço" {
		t.Errorf("motivação inesperada: %q", report.Motivation)
	}
	if len(report.MotivationLines) != 1 || report.MotivationLines[0] != report.Line+1 {
		t.Errorf("linhas da motivação inesperadas: %+v", report.MotivationLines)
	}

	coffee := items[1]
	if !coffee.Checked || coffee.Desc != "comprar café" ||
		coffee.Time != "" || coffee.Flag != "" || coffee.Timer != "" {
		t.Errorf("task concluída inesperada: %+v", coffee)
	}

	dp900 := findNoteItem(t, note, "Estudar DP-900")
	if dp900.Section != "Hábitos" || dp900.Category != "Estudos" ||
		dp900.Time != "xST" || dp900.Flag != "IU" || dp900.Checked {
		t.Errorf("hábito inesperado: %+v", dp900)
	}

	aws := findNoteItem(t, note, "AWS")
	if !aws.Checked || aws.TimerMinutes != 5 ||
		aws.Motivation != "revisar certificado\nsegunda linha" ||
		len(aws.MotivationLines) != 2 {
		t.Errorf("hábito concluído inesperado: %+v", aws)
	}

	extra := findNoteItem(t, note, "Task extra")
	if extra.Section != "Extra" || extra.Category != "" ||
		extra.Time != "15 min" || extra.Flag != "I" {
		t.Errorf("extra inesperada: %+v", extra)
	}

	sections := note.Sections()
	if len(sections) != 3 {
		t.Fatalf("esperava 3 seções, veio %d: %+v", len(sections), sections)
	}
	if sections[0].Title != "Tasks" || len(sections[0].Items) != 2 {
		t.Errorf("seção Tasks inesperada: %+v", sections[0])
	}
	if sections[1].Title != "Hábitos" || len(sections[1].Categories) != 2 {
		t.Errorf("seção Hábitos inesperada: %+v", sections[1])
	}
	if sections[1].Categories[0].Title != "Estudos" ||
		len(sections[1].Categories[0].Items) != 2 {
		t.Errorf("categoria Estudos inesperada: %+v", sections[1].Categories[0])
	}
	if sections[2].Title != "Extra" || len(sections[2].Items) != 1 {
		t.Errorf("seção Extra inesperada: %+v", sections[2])
	}
}

func TestNoteCRLFRoundTripAndToggle(t *testing.T) {
	crlf := strings.ReplaceAll(modelNote, "\n", "\r\n")
	note := ParseNote(crlf)
	if got := note.String(); got != crlf {
		t.Fatalf("String não preservou CRLF\n--- got ---\n%q\n--- want ---\n%q", got, crlf)
	}

	report := findNoteItem(t, note, "Escrever relatório")
	if _, err := note.ToggleItem(report.Line); err != nil {
		t.Fatalf("ToggleItem: %v", err)
	}

	got := note.String()
	if !strings.Contains(got, "- [x] 40 min (I) - Escrever relatório ⏱ 12min (08:00–08:12)\r\n") {
		t.Errorf("checkbox não foi marcado:\n%s", got)
	}

	coffee := findNoteItem(t, note, "comprar café")
	if _, err := note.SetMotivation(coffee.Line, "café novo"); err != nil {
		t.Fatalf("SetMotivation: %v", err)
	}
	got = note.String()
	if !strings.Contains(got, "- [x] comprar café\r\n> café novo\r\n") {
		t.Errorf("motivação não foi inserida com CRLF:\n%q", got)
	}

	if !strings.Contains(got, "texto desconhecido\r\n") {
		t.Error("conteúdo não reconhecido foi perdido")
	}
	if !strings.Contains(got, "> terminar antes do almoço\r\n") {
		t.Error("motivação foi perdida")
	}
	if strings.Contains(strings.ReplaceAll(got, "\r\n", ""), "\n") {
		t.Error("toggle converteu CRLF em LF")
	}

	path := filepath.Join(t.TempDir(), "days", "2026-11-05.md")
	if err := note.Write(path); err != nil {
		t.Fatalf("Write: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != got {
		t.Errorf("Write gravou conteúdo diferente\n--- got ---\n%q", data)
	}
}

func TestSetAndRemoveMotivation(t *testing.T) {
	const content = "2026-11-05\n\n# Tasks\n\n- [ ] tarefa\n\n# Extra\n"
	note := ParseNote(content)
	item := findNoteItem(t, note, "tarefa")

	updated, err := note.SetMotivation(item.Line, "  primeira motivação  ")
	if err != nil {
		t.Fatalf("SetMotivation: %v", err)
	}
	if updated.Motivation != "primeira motivação" {
		t.Errorf("motivação = %q", updated.Motivation)
	}
	if !strings.Contains(note.String(), "- [ ] tarefa\n> primeira motivação\n") {
		t.Errorf("blockquote não foi inserido logo abaixo:\n%q", note.String())
	}

	updated, err = note.SetMotivation(item.Line, "segunda\nlinha")
	if err != nil {
		t.Fatalf("SetMotivation (edição): %v", err)
	}
	if updated.Motivation != "segunda\nlinha" || len(updated.MotivationLines) != 2 {
		t.Errorf("motivação editada inesperada: %+v", updated)
	}
	if strings.Count(note.String(), "primeira motivação") != 0 {
		t.Errorf("motivação antiga permaneceu:\n%q", note.String())
	}

	updated, err = note.RemoveMotivation(item.Line)
	if err != nil {
		t.Fatalf("RemoveMotivation: %v", err)
	}
	if updated.Motivation != "" || len(updated.MotivationLines) != 0 {
		t.Errorf("motivação não foi removida: %+v", updated)
	}
	if got := note.String(); got != content {
		t.Errorf("remoção não restaurou o conteúdo\n--- got ---\n%q\n--- want ---\n%q", got, content)
	}
}

func TestToggleItemNotFound(t *testing.T) {
	note := ParseNote("# Tasks\n\n- [ ] tarefa\n")
	_, err := note.ToggleItem(0)
	if !errors.Is(err, ErrNoteItemNotFound) {
		t.Fatalf("esperava ErrNoteItemNotFound, veio %v", err)
	}
}

func TestAddExtraTask(t *testing.T) {
	const content = "# Tasks\n\n---\n# Hábitos\n\n## Estudos\n- [ ] 10 min (I) - ler\n\n---\n# Extra\n\n"
	note := ParseNote(content)

	item, err := note.AddExtraTask(ExtraTask{
		Time:       "15 min",
		Flag:       "U",
		Desc:       "comprar leite",
		Motivation: "acabou em casa",
	})
	if err != nil {
		t.Fatalf("AddExtraTask: %v", err)
	}
	if item.Section != "Extra" || item.Category != "" || item.Checked ||
		item.Time != "15 min" || item.Flag != "U" || item.Desc != "comprar leite" {
		t.Errorf("item extra inesperado: %+v", item)
	}
	if item.Motivation != "acabou em casa" {
		t.Errorf("motivação extra inesperada: %+v", item)
	}

	const want = "# Tasks\n\n---\n# Hábitos\n\n## Estudos\n- [ ] 10 min (I) - ler\n\n---\n# Extra\n\n- [ ] 15 min (U) - comprar leite\n> acabou em casa\n"
	if got := note.String(); got != want {
		t.Errorf("extra task diferente\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}
}

func TestAddExtraTaskCreatesSectionAndTolerantFormat(t *testing.T) {
	note := ParseNote("# Tasks\n\n- [ ] tarefa\n")
	item, err := note.AddExtraTask(ExtraTask{Desc: "sem tempo"})
	if err != nil {
		t.Fatalf("AddExtraTask: %v", err)
	}
	if item.Section != "Extra" || item.Time != "" || item.Flag != "" {
		t.Errorf("item extra inesperado: %+v", item)
	}
	if !strings.Contains(note.String(), "---\n# Extra\n\n- [ ] sem tempo\n") {
		t.Errorf("seção Extra não foi criada corretamente:\n%q", note.String())
	}
}

func TestAddExtraHabit(t *testing.T) {
	const content = `# Hábitos

## Estudos
- [ ] 10 min (I) - ler

## Rotina
- [ ] 5 min () - fio

---
# Extra

`
	note := ParseNote(content)

	item, err := note.AddExtraHabit(ExtraHabit{
		Category: "Estudos",
		Time:     "20 min",
		Flag:     "U",
		Desc:     "AWS",
	})
	if err != nil {
		t.Fatalf("AddExtraHabit: %v", err)
	}
	if item.Section != "Hábitos" || item.Category != "Estudos" ||
		item.Time != "20 min" || item.Flag != "U" || item.Desc != "AWS" {
		t.Errorf("hábito extra inesperado: %+v", item)
	}
	if got := strings.Count(note.String(), "- [ ] 20 min (U) - AWS\n"); got != 1 {
		t.Errorf("hábito extra deveria aparecer uma vez, apareceu %d:\n%s", got, note.String())
	}

	// Categoria ausente: cria no fim de # Hábitos, antes de # Extra.
	item, err = note.AddExtraHabit(ExtraHabit{
		Category: "Social",
		Time:     "5/15 min",
		Desc:     "Almoço em família",
	})
	if err != nil {
		t.Fatalf("AddExtraHabit (nova categoria): %v", err)
	}
	if item.Category != "Social" {
		t.Errorf("categoria inesperada: %+v", item)
	}

	got := note.String()
	if !strings.Contains(got, "## Social\n\n- [ ] 5/15 min - Almoço em família\n") {
		t.Errorf("nova categoria não foi formatada corretamente:\n%s", got)
	}
	if strings.Index(got, "## Social") > strings.Index(got, "---\n# Extra") {
		t.Errorf("categoria deveria ficar dentro de # Hábitos:\n%s", got)
	}

	sections := note.Sections()
	if len(sections) != 2 || sections[0].Title != "Hábitos" ||
		len(sections[0].Categories) != 3 {
		t.Errorf("seções/categorias inesperadas: %+v", sections)
	}
}

func TestAddExtraHabitAvoidsDuplicates(t *testing.T) {
	note := ParseNote("# Hábitos\n\n## Estudos\n- [ ] 10 min (I) - Ler\n")
	_, err := note.AddExtraHabit(ExtraHabit{Category: "Estudos", Desc: "  ler  "})
	if !errors.Is(err, ErrNoteItemExists) {
		t.Fatalf("esperava ErrNoteItemExists, veio %v", err)
	}
	if got := strings.Count(strings.ToLower(note.String()), "- [ ] 10 min (i) - ler"); got != 1 {
		t.Errorf("duplicata foi inserida:\n%s", note.String())
	}

	_, err = note.AddExtraHabit(ExtraHabit{Category: "Estudos", Desc: ""})
	if !errors.Is(err, ErrEmptyDescription) {
		t.Fatalf("esperava ErrEmptyDescription, veio %v", err)
	}
}

// TestExtraHabitCountsInFeedback garante que o hábito Extra entra em # Hábitos
// na categoria original e é contado pelo feedback sem categoria artificial.
func TestExtraHabitCountsInFeedback(t *testing.T) {
	const weeklist = `# Hábitos
## Estudos
- [ ] 10 min (I) - ler
`
	note := ParseNote("# Hábitos\n\n## Estudos\n- [ ] 10 min (I) - ler\n")

	item, err := note.AddExtraHabit(ExtraHabit{
		Category: "Rotina",
		Time:     "5 min",
		Desc:     "Fio dental",
	})
	if err != nil {
		t.Fatalf("AddExtraHabit: %v", err)
	}
	if _, err := note.SetChecked(item.Line, true); err != nil {
		t.Fatalf("SetChecked: %v", err)
	}

	feedback := week.BuildFeedback(
		"2026-W45",
		weeklist,
		[]week.Day{{
			Date:    time.Date(2026, time.November, 5, 0, 0, 0, 0, time.Local),
			Content: note.String(),
		}},
		time.Date(2026, time.November, 8, 18, 30, 0, 0, time.Local),
	)

	if !strings.Contains(feedback, "## Rotina\n") {
		t.Errorf("categoria original não apareceu no feedback:\n%s", feedback)
	}
	if !strings.Contains(feedback, "Fio dental 1/1 - (100%)") {
		t.Errorf("hábito Extra não foi contado no feedback:\n%s", feedback)
	}
}

func TestParseNoteStripsAlwaysFromHabits(t *testing.T) {
	note := ParseNote("# Hábitos\n## Rotina\n- [ ] 5 min () - Fio dental #always\n# Tasks\n- [ ] comprar café #always\n")

	fio := findNoteItem(t, note, "Fio dental")
	if !fio.Always {
		t.Errorf("hábito deveria ter Always = true: %+v", fio)
	}
	if strings.Contains(fio.Desc, "#always") {
		t.Errorf("marcador não foi removido da descrição: %q", fio.Desc)
	}

	// Tasks não usam #always: o texto é preservado normalmente.
	var coffee NoteItem
	for _, item := range note.Items() {
		if item.Section == "Tasks" {
			coffee = item
		}
	}
	if coffee.Always || coffee.Desc != "comprar café #always" {
		t.Errorf("task tratada como hábito: %+v", coffee)
	}
}

func TestToggleAlwaysOnlyInHabits(t *testing.T) {
	content := "# Hábitos\n## Rotina\n- [ ] 5 min () - Fio dental\n# Tasks\n- [ ] tarefa\n"
	note := ParseNote(content)

	fio := findNoteItem(t, note, "Fio dental")
	item, err := note.ToggleAlways(fio.Line)
	if err != nil {
		t.Fatalf("ToggleAlways: %v", err)
	}
	if !item.Always || !strings.Contains(note.String(), "Fio dental #always") {
		t.Errorf("marcador não foi adicionado:\n%s", note.String())
	}

	item, err = note.ToggleAlways(item.Line)
	if err != nil {
		t.Fatalf("ToggleAlways (remoção): %v", err)
	}
	if item.Always || strings.Contains(note.String(), "#always") {
		t.Errorf("marcador não foi removido:\n%s", note.String())
	}

	task := NoteItem{}
	for _, it := range note.Items() {
		if it.Section == "Tasks" {
			task = it
		}
	}
	if _, err := note.ToggleAlways(task.Line); !errors.Is(err, ErrNotHabit) {
		t.Errorf("erro = %v, want ErrNotHabit", err)
	}
}

func TestToggleAlwaysPreservesTimeFlagAndCheckbox(t *testing.T) {
	content := "# Hábitos\n## Estudos\n- [x] 10/20 min (IU) - AWS\n"
	note := ParseNote(content)

	item := findNoteItem(t, note, "AWS")
	if _, err := note.ToggleAlways(item.Line); err != nil {
		t.Fatalf("ToggleAlways: %v", err)
	}

	got := note.String()
	const want = "# Hábitos\n## Estudos\n- [x] 10/20 min (IU) - AWS #always\n"
	if got != want {
		t.Errorf("conteúdo = %q, want %q", got, want)
	}
}

func TestToggleAlwaysCRLF(t *testing.T) {
	content := "# Hábitos\r\n## Rotina\r\n- [ ] 5 min () - Fio dental\r\n"
	note := ParseNote(content)

	item := findNoteItem(t, note, "Fio dental")
	if _, err := note.ToggleAlways(item.Line); err != nil {
		t.Fatalf("ToggleAlways: %v", err)
	}
	if !strings.Contains(note.String(), "- [ ] 5 min () - Fio dental #always\r\n") {
		t.Errorf("CRLF não preservado: %q", note.String())
	}
}

func TestSummarize(t *testing.T) {
	note := ParseNote(`# Tasks
- [x] 30 min (I) - feita ⏱ 25min (09:00–09:25)
- [ ] 20 min (U) - pendente
- [ ] 5/15 min () - faixa
# Hábitos
## Rotina
- [ ] 5 min () - água
- [x] xST () - sem tempo
`)

	summary := Summarize(note.Items())
	if !summary.HasPlanned {
		t.Fatal("deveria ter tempo planejado")
	}
	if summary.PlannedMin != 60 || summary.PlannedMax != 70 {
		t.Errorf("planejado = %d–%d, want 60–70", summary.PlannedMin, summary.PlannedMax)
	}
	if summary.ActualMin != 25 {
		t.Errorf("real = %d, want 25", summary.ActualMin)
	}
	if summary.Done != 2 || summary.Pending != 3 {
		t.Errorf("concluídos/pendentes = %d/%d, want 2/3", summary.Done, summary.Pending)
	}
}

func TestSummarizeWithoutPlannedTime(t *testing.T) {
	summary := Summarize(ParseNote("# Tasks\n- [ ] xST () - estudar\n").Items())
	if summary.HasPlanned {
		t.Error("xST não deveria contar como tempo planejado")
	}
	if summary.PlannedMin != 0 || summary.PlannedMax != 0 {
		t.Errorf("planejado = %d–%d, want 0–0", summary.PlannedMin, summary.PlannedMax)
	}
}
