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
	Time     string // "20 min", "5/15 min", "xST"...
	Flag     string // "", "I", "U", "IU"
	Desc     string
}

// Label é como o item aparece no seletor.
func (i Item) Label() string {
	if i.Category == "" {
		return i.Desc
	}
	return fmt.Sprintf("[%s] %s", i.Category, i.Desc)
}

// Resolved é um item já com tempo e flag definidos pelo usuário.
type Resolved struct {
	Item
	FinalTime string
	FinalFlag string
}

// - [ ] 30min (I) - descrição
var lineRe = regexp.MustCompile(`^- \[([ xX])\] (.+?) \((.*?)\) - (.*)$`)

// ParseWeeklist lê o arquivo da weeklist e devolve os itens pendentes.
func ParseWeeklist(path string) ([]Item, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(string(data)), nil
}

// Parse extrai os itens pendentes de # Tasks e # Hábitos, agrupando os
// hábitos pelas subcategorias (## ...). Mesmas regras do template do Obsidian:
// linhas fora do formato "- [ ] <tempo> (<flag>) - <desc>" são ignoradas.
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
		m := lineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if strings.TrimSpace(m[1]) != "" { // [x] ou [X] já concluído
			continue
		}
		items = append(items, Item{
			Section:  section,
			Category: category,
			Time:     strings.TrimSpace(m[2]),
			Flag:     strings.TrimSpace(m[3]),
			Desc:     strings.TrimSpace(m[4]),
		})
	}
	return items
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
