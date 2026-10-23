// Package daylist lê a weeklist e monta o conteúdo da daylist.
package daylist

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Item é uma linha pendente da weeklist candidata a entrar na daylist.
type Item struct {
	Section  string // "Tasks" ou "Hábitos"
	Category string // subcategoria de hábito (vazio em Tasks)
	Time     string // "20 min", "5/15 min", "xST"... (vazio = perguntar)
	Flag     string // "", "I", "U", "IU" (vazio = perguntar)
	Desc     string
}

// Resolved é um item já com tempo e flag definidos pelo usuário.
type Resolved struct {
	Item
	FinalTime string
	FinalFlag string
}

// Stage agrupa os itens que são escolhidos juntos, numa etapa do wizard.
type Stage struct {
	Title string
	Items []Item
}

// O parser é tolerante: aceita o formato completo, tempo sem flag, flag sem
// tempo e linhas livres. Só itens já concluídos ([x]) ficam de fora.
var (
	timeFlagRe = regexp.MustCompile(`^- \[([ xX])\] (.+?) \((.*?)\) - (.*)$`)
	timeOnlyRe = regexp.MustCompile(`(?i)^- \[([ xX])\] (x?st|\d+(?:/\d+)?\s*min) - (.*)$`)
	flagOnlyRe = regexp.MustCompile(`^- \[([ xX])\] \(([IUiu]*)\) - (.*)$`)
	plainRe    = regexp.MustCompile(`^- \[([ xX])\] (.+)$`)
)

// ParseWeeklist lê o arquivo da weeklist e devolve os itens pendentes.
func ParseWeeklist(path string) ([]Item, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(string(data)), nil
}

// Parse extrai os itens pendentes de # Tasks e # Hábitos, agrupando os
// hábitos pelas subcategorias (## ...).
func Parse(content string) []Item {
	var items []Item
	section, category := "", ""

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimRight(line, "\r")

		switch {
		case strings.HasPrefix(line, "# Tasks"):
			section, category = "Tasks", ""
			continue
		case strings.HasPrefix(line, "# Hábitos"):
			section, category = "Hábitos", ""
			continue
		case strings.HasPrefix(line, "# "):
			section, category = "", ""
			continue
		case strings.HasPrefix(line, "## "):
			category = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			continue
		}

		if section == "" {
			continue
		}
		it, ok := parseLine(line)
		if !ok {
			continue
		}
		it.Section = section
		it.Category = category
		items = append(items, it)
	}
	return items
}

// parseLine interpreta uma linha "- [ ] ..." pendente.
func parseLine(line string) (Item, bool) {
	if m := timeFlagRe.FindStringSubmatch(line); m != nil {
		if isDone(m[1]) {
			return Item{}, false
		}
		return Item{
			Time: strings.TrimSpace(m[2]),
			Flag: strings.TrimSpace(m[3]),
			Desc: strings.TrimSpace(m[4]),
		}, true
	}

	if m := timeOnlyRe.FindStringSubmatch(line); m != nil {
		if isDone(m[1]) {
			return Item{}, false
		}
		return Item{
			Time: strings.TrimSpace(m[2]),
			Desc: strings.TrimSpace(m[3]),
		}, true
	}

	if m := flagOnlyRe.FindStringSubmatch(line); m != nil {
		if isDone(m[1]) {
			return Item{}, false
		}
		return Item{
			Flag: strings.TrimSpace(m[2]),
			Desc: strings.TrimSpace(m[3]),
		}, true
	}

	if m := plainRe.FindStringSubmatch(line); m != nil {
		if isDone(m[1]) {
			return Item{}, false
		}
		return Item{Desc: strings.TrimSpace(m[2])}, true
	}

	return Item{}, false
}

func isDone(check string) bool {
	check = strings.TrimSpace(check)
	return check == "x" || check == "X"
}

// Stages separa os itens em etapas: Tasks primeiro, depois cada categoria de
// hábito na ordem em que aparece na weeklist. Etapas vazias não entram.
func Stages(items []Item) []Stage {
	var stages []Stage
	index := map[string]int{}

	add := func(key, title string) *Stage {
		if pos, ok := index[key]; ok {
			return &stages[pos]
		}
		index[key] = len(stages)
		stages = append(stages, Stage{Title: title})
		return &stages[len(stages)-1]
	}

	for _, it := range items {
		if it.Section != "Tasks" {
			continue
		}
		stage := add("tasks", "Tasks")
		stage.Items = append(stage.Items, it)
	}

	for _, it := range items {
		if it.Section != "Hábitos" {
			continue
		}
		title := it.Category
		if title == "" {
			title = "Hábitos"
		}
		stage := add("cat:"+title, title)
		stage.Items = append(stage.Items, it)
	}

	return stages
}

// Render devolve o conteúdo da daylist no mesmo formato do template do
// Obsidian (Tasks, depois Hábitos agrupados por categoria, depois Extra).
func Render(items []Resolved, createdAt time.Time) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%s\n\n", createdAt.Format("2006-01-02 15:04"))

	b.WriteString("# Tasks\n\n")
	for _, it := range items {
		if it.Section != "Tasks" {
			continue
		}
		fmt.Fprintf(&b, "- [ ] %s (%s) - %s\n", it.FinalTime, it.FinalFlag, it.Desc)
	}

	b.WriteString("\n---\n# Hábitos\n")
	for _, cat := range categories(items) {
		if cat != "" {
			fmt.Fprintf(&b, "\n## %s\n", cat)
		}
		for _, it := range items {
			if it.Section == "Hábitos" && it.Category == cat {
				fmt.Fprintf(&b, "- [ ] %s (%s) - %s\n", it.FinalTime, it.FinalFlag, it.Desc)
			}
		}
	}

	b.WriteString("\n---\n# Extra\n\n")
	return b.String()
}

// categories devolve as categorias de hábito na ordem em que aparecem.
func categories(items []Resolved) []string {
	var cats []string
	seen := map[string]bool{}
	for _, it := range items {
		if it.Section != "Hábitos" || seen[it.Category] {
			continue
		}
		seen[it.Category] = true
		cats = append(cats, it.Category)
	}
	return cats
}

// Write grava a daylist de forma atômica (tmp + rename), criando o diretório
// pai se necessário.
func Write(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
