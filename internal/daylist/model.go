package daylist

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"daylist/internal/marker"
)

var (
	// ErrNoteItemNotFound indica que a linha informada não é um item da daylist.
	ErrNoteItemNotFound = errors.New("item da daylist não encontrado")
	// ErrNoteItemExists indica que o hábito já está presente na daylist.
	ErrNoteItemExists = errors.New("item já existe na daylist")
	// ErrEmptyDescription indica que um Extra foi criado sem descrição.
	ErrEmptyDescription = errors.New("descrição vazia")
	// ErrNotHabit indica que o marcador #always foi usado fora de # Hábitos.
	ErrNotHabit = errors.New("#always só vale para hábitos")
)

// NoteItem é um item (- [ ] ...) de uma daylist. Line é o índice da linha no
// conteúdo; Raw preserva a linha original, sem o terminador de linha.
type NoteItem struct {
	Section         string
	Category        string
	Line            int
	Raw             string
	Checked         bool
	Time            string
	Flag            string
	Desc            string
	Always          bool // hábito com #always; sempre falso fora de # Hábitos
	Timer           string
	TimerMinutes    int
	Motivation      string
	MotivationLines []int
}

// NoteSection é uma seção de nível 1 da daylist (# Tasks, # Hábitos, # Extra
// ou qualquer outra). Items contém os itens que não estão em subcategorias.
type NoteSection struct {
	Title      string
	Line       int
	Categories []NoteCategory
	Items      []NoteItem
}

// NoteCategory é uma subcategoria de hábitos (## Estudos, ## Rotina...).
type NoteCategory struct {
	Title string
	Line  int
	Items []NoteItem
}

// Note é o modelo editável de uma daylist. As linhas não reconhecidas são
// preservadas na posição original; as operações só alteram as linhas afetadas.
type Note struct {
	lines    []string
	eol      string
	items    []NoteItem
	sections []NoteSection
}

// ExtraTask descreve uma task livre para a seção # Extra.
type ExtraTask struct {
	Time       string
	Flag       string
	Desc       string
	Motivation string
}

// ExtraHabit descreve um hábito escolhido da weeklist. Ele entra na categoria
// original dentro de # Hábitos, para contar no feedback naturalmente.
type ExtraHabit struct {
	Category   string
	Time       string
	Flag       string
	Desc       string
	Motivation string
}

var (
	noteItemRe       = regexp.MustCompile(`^- \[([ xX])\](?: ?(.*))?$`)
	noteTimerRe      = regexp.MustCompile(`\s*⏱️?\s*(.*?)\s*$`)
	noteTimeFlagRe   = regexp.MustCompile(`^(.+?) \((.*?)\) - (.*)$`)
	noteTimeOnlyRe   = regexp.MustCompile(`(?i)^(x?st|\d+(?:/\d+)?\s*min) - (.*)$`)
	noteFlagOnlyRe   = regexp.MustCompile(`^\(([IUiu]*)\) - (.*)$`)
	noteHeadingRe    = regexp.MustCompile(`^#\s+(.+?)\s*$`)
	noteSubheadingRe = regexp.MustCompile(`^##\s+(.+?)\s*$`)
	noteMinutesRe    = regexp.MustCompile(`(?i)^(\d+)\s*min`)
	noteCheckboxRe   = regexp.MustCompile(`^(- \[)([ xX])(\])`)
)

// ParseNote interpreta o conteúdo de uma daylist e devolve o modelo editável.
func ParseNote(content string) *Note {
	eol := "\n"
	if strings.Contains(content, "\r\n") {
		eol = "\r\n"
	}

	lines := strings.Split(content, "\n")
	for i := range lines {
		lines[i] = strings.TrimSuffix(lines[i], "\r")
	}

	n := &Note{lines: lines, eol: eol}
	n.refresh()
	return n
}

// LoadNote lê e interpreta uma daylist do disco.
func LoadNote(path string) (*Note, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseNote(string(data)), nil
}

// String devolve o conteúdo da nota, preservando o estilo de fim de linha
// (LF ou CRLF) detectado no parse.
func (n *Note) String() string {
	return strings.Join(n.lines, n.eol)
}

// Write grava a nota de forma atômica, preservando o estilo de fim de linha.
func (n *Note) Write(path string) error {
	return Write(path, n.String())
}

// Items devolve os itens na ordem em que aparecem na daylist.
func (n *Note) Items() []NoteItem {
	return cloneNoteItems(n.items)
}

// Sections devolve as seções e categorias na ordem em que aparecem.
func (n *Note) Sections() []NoteSection {
	return cloneSections(n.sections)
}

func cloneNoteItems(items []NoteItem) []NoteItem {
	out := make([]NoteItem, len(items))
	copy(out, items)
	for i := range out {
		out[i].MotivationLines = append([]int(nil), items[i].MotivationLines...)
	}
	return out
}

func cloneSections(sections []NoteSection) []NoteSection {
	out := make([]NoteSection, len(sections))
	for i, section := range sections {
		out[i] = NoteSection{
			Title: section.Title,
			Line:  section.Line,
			Items: cloneNoteItems(section.Items),
		}
		out[i].Categories = make([]NoteCategory, len(section.Categories))
		for j, category := range section.Categories {
			out[i].Categories[j] = NoteCategory{
				Title: category.Title,
				Line:  category.Line,
				Items: cloneNoteItems(category.Items),
			}
		}
	}
	return out
}

// ToggleItem alterna o checkbox do item na linha informada.
func (n *Note) ToggleItem(line int) (NoteItem, error) {
	item, err := n.itemAt(line)
	if err != nil {
		return NoteItem{}, err
	}
	return n.setChecked(line, !item.Checked)
}

// SetChecked define o checkbox do item na linha informada.
func (n *Note) SetChecked(line int, checked bool) (NoteItem, error) {
	if _, err := n.itemAt(line); err != nil {
		return NoteItem{}, err
	}
	return n.setChecked(line, checked)
}

func (n *Note) setChecked(line int, checked bool) (NoteItem, error) {
	mark := " "
	if checked {
		mark = "x"
	}
	n.lines[line] = noteCheckboxRe.ReplaceAllString(n.lines[line], "${1}"+mark+"${3}")
	n.refresh()
	return n.itemAt(line)
}

// ToggleAlways alterna o marcador #always na descrição de um hábito. Fora de
// # Hábitos devolve ErrNotHabit: tasks não usam esse marcador.
func (n *Note) ToggleAlways(line int) (NoteItem, error) {
	item, err := n.itemAt(line)
	if err != nil {
		return NoteItem{}, err
	}
	if item.Section != "Hábitos" {
		return NoteItem{}, fmt.Errorf("%w: linha %d", ErrNotHabit, line)
	}

	// Edita apenas o fim da linha para preservar o formato original (inclusive
	// parênteses vazios e espaçamento).
	if item.Always {
		n.lines[line], _ = marker.StripAlways(n.lines[line])
	} else {
		n.lines[line] = strings.TrimRight(n.lines[line], " \t") + " " + marker.Always
	}
	n.refresh()
	return n.itemAt(line)
}

// SetMotivation grava a motivação do item como blockquote logo abaixo dele.
// Texto vazio remove a motivação existente.
func (n *Note) SetMotivation(line int, text string) (NoteItem, error) {
	item, err := n.itemAt(line)
	if err != nil {
		return NoteItem{}, err
	}

	trimmed := strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
	if trimmed == "" {
		return n.removeMotivation(item)
	}

	replacement := make([]string, 0, strings.Count(trimmed, "\n")+1)
	for _, part := range strings.Split(trimmed, "\n") {
		part = strings.TrimSpace(part)
		if part == "" {
			replacement = append(replacement, ">")
			continue
		}
		replacement = append(replacement, "> "+part)
	}

	start, count := line+1, 0
	if len(item.MotivationLines) > 0 {
		start = item.MotivationLines[0]
		count = len(item.MotivationLines)
	}
	n.spliceLines(start, count, replacement)
	n.refresh()
	return n.itemAt(line)
}

// RemoveMotivation remove a motivação do item, se existir.
func (n *Note) RemoveMotivation(line int) (NoteItem, error) {
	item, err := n.itemAt(line)
	if err != nil {
		return NoteItem{}, err
	}
	return n.removeMotivation(item)
}

func (n *Note) removeMotivation(item NoteItem) (NoteItem, error) {
	if len(item.MotivationLines) == 0 {
		return item, nil
	}
	n.spliceLines(item.MotivationLines[0], len(item.MotivationLines), nil)
	n.refresh()
	return n.itemAt(item.Line)
}

// AddExtraTask insere uma task livre na seção # Extra.
func (n *Note) AddExtraTask(task ExtraTask) (NoteItem, error) {
	desc := strings.TrimSpace(task.Desc)
	if desc == "" {
		return NoteItem{}, ErrEmptyDescription
	}

	content := formatNoteItem(task.Time, task.Flag, desc)
	section := n.ensureSection("Extra")
	line := n.insertAtBlockEnd(section, false, []string{content})
	return n.finishInsert(line, task.Motivation)
}

// AddExtraHabit insere um hábito na categoria original dentro de # Hábitos.
// A descrição é comparada sem caixa e sem espaços extras; se o hábito já
// existir, devolve ErrNoteItemExists.
func (n *Note) AddExtraHabit(habit ExtraHabit) (NoteItem, error) {
	desc := strings.TrimSpace(habit.Desc)
	if desc == "" {
		return NoteItem{}, ErrEmptyDescription
	}
	if n.hasHabit(desc) {
		return NoteItem{}, fmt.Errorf("%w: %s", ErrNoteItemExists, desc)
	}

	content := formatNoteItem(habit.Time, habit.Flag, desc)
	section := n.ensureSection("Hábitos")
	category := strings.TrimSpace(habit.Category)

	var line int
	switch {
	case category == "":
		line = n.insertAtSectionRoot(section, content)
	case n.findCategory(section, category) >= 0:
		line = n.insertAtBlockEnd(n.findCategory(section, category), true, []string{content})
	default:
		line = n.insertAtBlockEnd(section, false, []string{"## " + category, "", content})
	}

	return n.finishInsert(line, habit.Motivation)
}

func (n *Note) finishInsert(line int, motivation string) (NoteItem, error) {
	n.refresh()
	item, err := n.itemAt(line)
	if err != nil {
		return NoteItem{}, err
	}
	if strings.TrimSpace(motivation) == "" {
		return item, nil
	}
	return n.SetMotivation(line, motivation)
}

func (n *Note) itemAt(line int) (NoteItem, error) {
	for _, item := range n.items {
		if item.Line == line {
			return item, nil
		}
	}
	return NoteItem{}, fmt.Errorf("%w: linha %d", ErrNoteItemNotFound, line)
}

func (n *Note) hasHabit(desc string) bool {
	for _, item := range n.items {
		if item.Section != "Hábitos" {
			continue
		}
		if equalDescription(item.Desc, desc) {
			return true
		}
	}
	return false
}

func equalDescription(a, b string) bool {
	return strings.EqualFold(normalizeSpaces(a), normalizeSpaces(b))
}

func normalizeSpaces(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

func formatNoteItem(time, flag, desc string) string {
	return "- [ ] " + formatNoteBody(time, flag, desc)
}

func formatNoteBody(time, flag, desc string) string {
	time = strings.TrimSpace(time)
	flag = strings.TrimSpace(flag)
	desc = strings.TrimSpace(desc)

	switch {
	case time != "" && flag != "":
		return fmt.Sprintf("%s (%s) - %s", time, flag, desc)
	case time != "":
		return fmt.Sprintf("%s - %s", time, desc)
	case flag != "":
		return fmt.Sprintf("(%s) - %s", flag, desc)
	default:
		return desc
	}
}

// refresh reconstrói itens, seções e categorias a partir das linhas atuais.
func (n *Note) refresh() {
	n.items = nil
	n.sections = nil

	sectionIdx, categoryIdx := -1, -1
	var sectionTitle, category string

	for i := 0; i < len(n.lines); i++ {
		line := n.lines[i]

		if m := noteSubheadingRe.FindStringSubmatch(line); m != nil {
			category = strings.TrimSpace(m[1])
			categoryIdx = -1
			if sectionIdx >= 0 {
				n.sections[sectionIdx].Categories = append(n.sections[sectionIdx].Categories, NoteCategory{
					Title: category,
					Line:  i,
				})
				categoryIdx = len(n.sections[sectionIdx].Categories) - 1
			}
			continue
		}

		if m := noteHeadingRe.FindStringSubmatch(line); m != nil {
			sectionTitle = strings.TrimSpace(m[1])
			category = ""
			categoryIdx = -1
			n.sections = append(n.sections, NoteSection{Title: sectionTitle, Line: i})
			sectionIdx = len(n.sections) - 1
			continue
		}

		item, ok := parseNoteItem(line)
		if !ok || sectionIdx < 0 {
			continue
		}

		item.Section = sectionTitle
		item.Category = category
		item.Line = i
		item.Raw = line
		if sectionTitle == "Hábitos" {
			item.Desc, item.Always = marker.StripAlways(item.Desc)
		}

		for j := i + 1; j < len(n.lines) && isBlockquote(n.lines[j]); j++ {
			item.MotivationLines = append(item.MotivationLines, j)
		}
		if len(item.MotivationLines) > 0 {
			parts := make([]string, len(item.MotivationLines))
			for j, lineIdx := range item.MotivationLines {
				parts[j] = blockquoteText(n.lines[lineIdx])
			}
			item.Motivation = strings.Join(parts, "\n")
		}

		n.items = append(n.items, item)
		n.sections[sectionIdx].Items = append(n.sections[sectionIdx].Items, item)
		if categoryIdx >= 0 && category != "" {
			n.sections[sectionIdx].Categories[categoryIdx].Items = append(
				n.sections[sectionIdx].Categories[categoryIdx].Items,
				item,
			)
		}
	}
}

func parseNoteItem(line string) (NoteItem, bool) {
	m := noteItemRe.FindStringSubmatch(line)
	if m == nil {
		return NoteItem{}, false
	}

	item := NoteItem{Checked: m[1] == "x" || m[1] == "X"}
	body := strings.TrimSpace(m[2])

	if loc := noteTimerRe.FindStringSubmatchIndex(body); loc != nil {
		item.Timer = strings.TrimSpace(body[loc[2]:loc[3]])
		body = strings.TrimSpace(body[:loc[0]])
		if tm := noteMinutesRe.FindStringSubmatch(item.Timer); tm != nil {
			item.TimerMinutes, _ = strconv.Atoi(tm[1])
		}
	}

	switch {
	case noteTimeFlagRe.MatchString(body):
		parts := noteTimeFlagRe.FindStringSubmatch(body)
		item.Time = strings.TrimSpace(parts[1])
		item.Flag = strings.TrimSpace(parts[2])
		item.Desc = strings.TrimSpace(parts[3])
	case noteTimeOnlyRe.MatchString(body):
		parts := noteTimeOnlyRe.FindStringSubmatch(body)
		item.Time = strings.TrimSpace(parts[1])
		item.Desc = strings.TrimSpace(parts[2])
	case noteFlagOnlyRe.MatchString(body):
		parts := noteFlagOnlyRe.FindStringSubmatch(body)
		item.Flag = strings.TrimSpace(parts[1])
		item.Desc = strings.TrimSpace(parts[2])
	default:
		item.Desc = body
	}

	return item, true
}

func isBlockquote(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), ">")
}

func blockquoteText(line string) string {
	text := strings.TrimSpace(line)
	text = strings.TrimPrefix(text, ">")
	return strings.TrimSpace(text)
}

func (n *Note) findSection(title string) int {
	for i, line := range n.lines {
		if m := noteHeadingRe.FindStringSubmatch(line); m != nil &&
			strings.EqualFold(strings.TrimSpace(m[1]), title) {
			return i
		}
	}
	return -1
}

func (n *Note) findCategory(section int, title string) int {
	end := n.blockEnd(section, false)
	for i := section + 1; i < end; i++ {
		if m := noteSubheadingRe.FindStringSubmatch(n.lines[i]); m != nil &&
			strings.EqualFold(strings.TrimSpace(m[1]), title) {
			return i
		}
	}
	return -1
}

// ensureSection devolve a linha do cabeçalho da seção, criando-a no fim da
// nota quando necessário.
func (n *Note) ensureSection(title string) int {
	if idx := n.findSection(title); idx >= 0 {
		return idx
	}

	if len(n.lines) == 1 && n.lines[0] == "" {
		n.lines = nil
	}
	if len(n.lines) > 0 && strings.TrimSpace(n.lines[len(n.lines)-1]) != "" {
		n.lines = append(n.lines, "")
	}
	if len(n.lines) > 0 {
		n.lines = append(n.lines, "---")
	}
	n.lines = append(n.lines, "# "+title, "", "")
	return n.findSection(title)
}

// blockEnd devolve o índice da próxima linha que encerra o bloco iniciado em
// start. Em categorias, o próximo ## também encerra o bloco.
func (n *Note) blockEnd(start int, category bool) int {
	for i := start + 1; i < len(n.lines); i++ {
		if strings.TrimSpace(n.lines[i]) == "---" {
			return i
		}
		if noteHeadingRe.MatchString(n.lines[i]) {
			return i
		}
		if category && noteSubheadingRe.MatchString(n.lines[i]) {
			return i
		}
	}
	return len(n.lines)
}

// contentInsertIndex escolhe onde inserir conteúdo dentro de um bloco: depois
// do último conteúdo, ou depois do cabeçalho e de suas linhas em branco quando
// o bloco está vazio.
func (n *Note) contentInsertIndex(start, end int) int {
	last := -1
	for i := start + 1; i < end; i++ {
		if strings.TrimSpace(n.lines[i]) != "" {
			last = i
		}
	}
	if last >= 0 {
		return last + 1
	}

	limit := end
	if limit == len(n.lines) && limit > 0 && strings.TrimSpace(n.lines[limit-1]) == "" {
		limit--
	}
	insert := start + 1
	for insert < limit && strings.TrimSpace(n.lines[insert]) == "" {
		insert++
	}
	return insert
}

// insertAtBlockEnd insere repl ao fim de um bloco e devolve a linha do item
// inserido (a última linha não vazia que não seja cabeçalho).
func (n *Note) insertAtBlockEnd(start int, category bool, repl []string) int {
	end := n.blockEnd(start, category)
	insert := n.contentInsertIndex(start, end)

	if len(repl) > 0 && noteSubheadingRe.MatchString(repl[0]) &&
		insert > 0 && strings.TrimSpace(n.lines[insert-1]) != "" {
		repl = append([]string{""}, repl...)
	}

	n.spliceLines(insert, 0, repl)
	for i := len(repl) - 1; i >= 0; i-- {
		line := repl[i]
		if strings.TrimSpace(line) != "" &&
			!noteHeadingRe.MatchString(line) && !noteSubheadingRe.MatchString(line) {
			return insert + i
		}
	}
	return insert
}

// insertAtSectionRoot insere um item sem categoria logo abaixo do cabeçalho,
// antes da primeira subcategoria.
func (n *Note) insertAtSectionRoot(section int, content string) int {
	insert := section + 1
	if insert < len(n.lines) && strings.TrimSpace(n.lines[insert]) == "" {
		insert++
	}

	repl := []string{content}
	if insert < len(n.lines) && noteSubheadingRe.MatchString(n.lines[insert]) {
		repl = append(repl, "")
	}
	n.spliceLines(insert, 0, repl)
	return insert
}

func (n *Note) spliceLines(start, count int, repl []string) {
	out := make([]string, 0, len(n.lines)-count+len(repl))
	out = append(out, n.lines[:start]...)
	out = append(out, repl...)
	out = append(out, n.lines[start+count:]...)
	n.lines = out
}
