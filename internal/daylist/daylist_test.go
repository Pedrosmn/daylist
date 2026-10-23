package daylist

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Conteúdo real da weeklist 2026-W43 do vault.
const weeklistW43 = `# Tasks

- [ ] 40 min (I) - JOURNAL Pensar sobre um sistema de tracker social
- [ ] task tcc
- [ ] arrumar perfil no oxetech
---
# Hábitos

## Estudos
- [ ] xST (IU) - Estudar para DP-900
- [ ] xST () - Estudar Teo Me Why
- [ ] xST () - Estudar SQL
- [ ] xST (I) - Projeto Chess
- [ ] 5 min (U) - AWS

## Exercício Físico
- [ ] 20 min (IU) - Exercício Físico

## Social
- [ ] 20 min () - Almoço em família
- [ ] 5/15 min () - Tempo na área
- [ ] 1 min () - Sair de casa

## Mindfulness
- [ ] 20/40 min (IU) - Journal
- [ ] 10/20 min (I) - Leitura
- [ ] 15 min (IU) - Meditação

## Tédio
- [ ] 30 min () - fazer nada
- [ ] 20 min () - comer sem vídeo
- [ ] 15 min () - banho sem música

## Hobbies
- [ ] xST () - Hobby
- [ ] xST () - Jogar jogo
- [ ] xST () - ver filme
- [ ] xST () - ver série
- [ ] xST () - ouvir albúm

## Organização
- [ ] 5 min () - Daylist de amanhã
- [ ] 5 min (U) - Escrever motivações

## Rotina
- [ ] 25 min () - Entregar currículo
- [ ] 20 min () - Fio dental
`

func TestParseWeeklist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "weeklist.md")
	if err := os.WriteFile(path, []byte(weeklistW43), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := ParseWeeklist(path)
	if err != nil {
		t.Fatalf("ParseWeeklist: %v", err)
	}

	// 1 task válida + 24 hábitos (as linhas "task tcc" e "arrumar perfil no
	// oxetech" não têm tempo/flag e são ignoradas, como no template).
	if len(items) != 25 {
		t.Fatalf("esperava 25 itens, veio %d", len(items))
	}

	first := items[0]
	if first.Section != "Tasks" || first.Category != "" ||
		first.Time != "40 min" || first.Flag != "I" ||
		first.Desc != "JOURNAL Pensar sobre um sistema de tracker social" {
		t.Errorf("primeiro item inesperado: %+v", first)
	}

	byDesc := make(map[string]Item, len(items))
	for _, it := range items {
		byDesc[it.Desc] = it
	}

	if _, ok := byDesc["task tcc"]; ok {
		t.Error("linha sem tempo/flag não deveria ser parseada")
	}

	dp900 := byDesc["Estudar para DP-900"]
	if dp900.Section != "Hábitos" || dp900.Category != "Estudos" ||
		dp900.Time != "xST" || dp900.Flag != "IU" {
		t.Errorf("item xST inesperado: %+v", dp900)
	}

	composite := byDesc["Tempo na área"]
	if composite.Category != "Social" || composite.Time != "5/15 min" || composite.Flag != "" {
		t.Errorf("tempo composto inesperado: %+v", composite)
	}
}

func TestParseIgnoresDoneAndOtherSections(t *testing.T) {
	content := `# Tasks
- [x] 10 min (I) - já feita
- [ ] 20 min (U) - pendente
# Outra
- [ ] 30 min (I) - fora das seções
# Hábitos
- [X] 5 min () - também feita
## Estudos
- [ ] 60 min (IU) - estudar
`
	items := Parse(content)
	if len(items) != 2 {
		t.Fatalf("esperava 2 itens, veio %d: %+v", len(items), items)
	}
	if items[0].Desc != "pendente" || items[0].Section != "Tasks" {
		t.Errorf("item 0 inesperado: %+v", items[0])
	}
	if items[1].Desc != "estudar" || items[1].Category != "Estudos" {
		t.Errorf("item 1 inesperado: %+v", items[1])
	}
}

func TestParseWeeklistNotFound(t *testing.T) {
	_, err := ParseWeeklist(filepath.Join(t.TempDir(), "nao-existe.md"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("esperava os.ErrNotExist, veio %v", err)
	}
}

// TestRenderGolden compara com o arquivo real gerado pelo Obsidian
// (2026-10-10.md), inclusive a linha em branco final.
func TestRenderGolden(t *testing.T) {
	items := []Resolved{
		{Item: Item{Section: "Hábitos", Category: "Estudos", Time: "60 min", Flag: "I", Desc: "Projeto Chess"}, FinalTime: "60 min", FinalFlag: "I"},
		{Item: Item{Section: "Hábitos", Category: "Estudos", Time: "5 min", Flag: "U", Desc: "AWS"}, FinalTime: "5 min", FinalFlag: "U"},
		{Item: Item{Section: "Hábitos", Category: "Exercício Físico", Time: "20 min", Flag: "IU", Desc: "Exercício Físico"}, FinalTime: "20 min", FinalFlag: "IU"},
		{Item: Item{Section: "Hábitos", Category: "Hobbies", Time: "60 min", Flag: "", Desc: "Hobby"}, FinalTime: "60 min", FinalFlag: ""},
		{Item: Item{Section: "Hábitos", Category: "Organização", Time: "5 min", Flag: "", Desc: "Daylist de amanhã"}, FinalTime: "5 min", FinalFlag: ""},
	}

	createdAt := time.Date(2026, 10, 9, 20, 28, 0, 0, time.Local)

	const want = `2026-10-09 20:28

# Tasks


---
# Hábitos

## Estudos
- [ ] 60 min (I) - Projeto Chess
- [ ] 5 min (U) - AWS

## Exercício Físico
- [ ] 20 min (IU) - Exercício Físico

## Hobbies
- [ ] 60 min () - Hobby

## Organização
- [ ] 5 min () - Daylist de amanhã

---
# Extra

`

	if got := Render(items, createdAt); got != want {
		t.Errorf("render diferente do esperado\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}
}

func TestRenderGroupsTasksFirst(t *testing.T) {
	items := []Resolved{
		{Item: Item{Section: "Hábitos", Category: "Estudos", Desc: "estudar"}, FinalTime: "20 min", FinalFlag: "I"},
		{Item: Item{Section: "Tasks", Desc: "escrever"}, FinalTime: "10 min", FinalFlag: "U"},
	}

	got := Render(items, time.Date(2026, 10, 23, 8, 0, 0, 0, time.Local))

	wantTasks := "2026-10-23 08:00\n\n# Tasks\n\n- [ ] 10 min (U) - escrever\n\n---\n# Hábitos\n\n## Estudos\n- [ ] 20 min (I) - estudar\n\n---\n# Extra\n\n"
	if got != wantTasks {
		t.Errorf("ordem inesperada\n--- got ---\n%q\n", got)
	}
}

func TestLabel(t *testing.T) {
	if got := (Item{Desc: "tarefa"}).Label(); got != "tarefa" {
		t.Errorf("label sem categoria = %q", got)
	}
	if got := (Item{Category: "Estudos", Desc: "SQL"}).Label(); got != "[Estudos] SQL" {
		t.Errorf("label com categoria = %q", got)
	}
}

func TestWriteCreatesParentsAndReplacesAtomically(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Fleeting Notes", "Daily", "Days", "2026-10-24.md")
	if err := Write(path, "primeiro\n"); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := Write(path, "segundo\n"); err != nil {
		t.Fatalf("Write (2): %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "segundo\n" {
		t.Errorf("conteúdo = %q", data)
	}
	if _, err := os.Stat(path + ".tmp"); !errors.Is(err, os.ErrNotExist) {
		t.Error("arquivo temporário não deveria sobrar")
	}
}
