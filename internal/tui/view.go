package tui

import (
	"fmt"
	"strings"
	"time"

	"daylist/internal/daylist"
	"daylist/internal/timer"
)

var weekdayNames = []string{
	"domingo", "segunda-feira", "terça-feira", "quarta-feira",
	"quinta-feira", "sexta-feira", "sábado",
}

// View implementa tea.Model.
func (m Model) View() string {
	switch m.mode {
	case modeHelp:
		return m.helpView()
	case modeForm:
		return m.formView()
	case modeHabitPick:
		return m.habitPickerView()
	case modeWeeklist:
		return m.weeklistView()
	case modeConfirm:
		return m.confirmView()
	default:
		return m.mainView()
	}
}

func (m Model) mainView() string {
	var b strings.Builder

	b.WriteString(styleTitle.Render("daylist"))
	b.WriteString("  ")
	b.WriteString(styleSubtle.Render(dateLabel(m.now) + " · " + weekLabel(m.now)))
	b.WriteString("\n")

	if m.timer != nil {
		b.WriteString(styleTimer.Render("◷ " + timer.StatusText(m.timer)))
		b.WriteString("  ")
		b.WriteString(styleTimer.Render(timer.PromptText(m.timer)))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	if m.note == nil {
		b.WriteString("A daylist de hoje ainda não existe.\n\n")
		b.WriteString("  n  criar pelo wizard (reutiliza a weeklist)\n")
		b.WriteString("  c  criar a weeklist atual, se estiver faltando\n")
		b.WriteString("  r  recarregar\n")
		b.WriteString("  q  sair\n")
	} else {
		b.WriteString(m.renderNote(m.note, m.selectedLine(), false))
	}

	b.WriteString("\n")
	b.WriteString(m.messageView())
	b.WriteString(styleHelp.Render("? ajuda · j/k mover · espaço marcar · s timer · p pausar · x parar"))
	b.WriteString("\n")
	b.WriteString(styleHelp.Render("e motivação · a Extra task · A Extra hábito · W weeklist · n nova · c weeklist · o abrir · q sair"))
	return b.String()
}

func (m Model) helpView() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("daylist · ajuda"))
	b.WriteString("\n\n")
	b.WriteString(styleHeading.Render("Navegação"))
	b.WriteString("\n")
	b.WriteString("  j/k, ↑/↓      mover o cursor\n")
	b.WriteString("  g/G, home/end início/fim da lista\n")
	b.WriteString("  espaço/enter  marcar ou desmarcar o item\n")
	b.WriteString("  r             recarregar a daylist do disco\n")
	b.WriteString("  q, ctrl+c     sair do TUI\n\n")

	b.WriteString(styleHeading.Render("Daylist e semana"))
	b.WriteString("\n")
	b.WriteString("  n  criar a daylist de hoje pelo wizard\n")
	b.WriteString("  c  criar a weeklist atual, se estiver faltando\n")
	b.WriteString("  W  ver a weeklist atual; espaço alterna #always em hábitos\n")
	b.WriteString("  o  abrir a daylist de hoje no Obsidian\n")
	b.WriteString("  u  desfazer a última edição da daylist\n\n")

	b.WriteString(styleHeading.Render("Timer"))
	b.WriteString("\n")
	b.WriteString("  s  iniciar o timer no item selecionado\n")
	b.WriteString("  p  pausar ou retomar o timer\n")
	b.WriteString("  x  parar o timer e gravar o tempo na daylist de origem\n\n")

	b.WriteString(styleHeading.Render("Extras e motivação"))
	b.WriteString("\n")
	b.WriteString("  a  adicionar Extra task (descrição, tempo, prioridade e motivação)\n")
	b.WriteString("  A  adicionar hábito Extra escolhido da weeklist\n")
	b.WriteString("  e  editar ou remover a motivação do item selecionado\n\n")

	b.WriteString(styleHelp.Render("qualquer tecla volta"))
	return b.String()
}

func (m Model) formView() string {
	form := m.form
	if form == nil {
		return m.mainView()
	}

	var b strings.Builder
	b.WriteString(styleTitle.Render(form.title()))
	b.WriteString("\n\n")

	if form.kind == formExtraHabit {
		b.WriteString(styleSubtle.Render(
			fmt.Sprintf("%s · %s · %s", form.habit.Category, form.habit.Time, form.habit.Flag)))
		b.WriteString("\n\n")
	}

	for i := 0; i < form.lastField(); i++ {
		label := form.fieldLabel(i)
		if form.kind == formExtraTask && i == 2 {
			line := fmt.Sprintf("%-11s ← %s →", label+":", form.flagLabel())
			if form.focus == i {
				line = styleSelected.Render(line)
			}
			b.WriteString(line)
			b.WriteString("\n")
			continue
		}
		b.WriteString(label)
		b.WriteString(": ")
		b.WriteString(form.inputs[i].View())
		b.WriteString("\n")
	}

	if form.err != "" {
		b.WriteString("\n")
		b.WriteString(styleError.Render("✗ " + form.err))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styleHelp.Render("tab/↑↓ campo · ←/→ prioridade · enter/ctrl+s salva · esc cancela"))
	return styleForm.Render(b.String())
}

func (m Model) confirmView() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("Confirmação"))
	b.WriteString("\n\n")
	b.WriteString(m.confirm)
	b.WriteString("\n\n")
	b.WriteString(styleSelected.Render("s/enter  sim"))
	b.WriteString("\n")
	b.WriteString("n/esc    não\n")
	return styleForm.Render(b.String())
}

func (m Model) habitPickerView() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("Extra hábito"))
	b.WriteString("\n\n")
	b.WriteString("Escolha um hábito pendente da weeklist:\n\n")

	for i, habit := range m.habitChoices {
		cursor := "  "
		if i == m.habitCursor {
			cursor = "▶ "
		}
		meta := ""
		if habit.Time != "" {
			meta += habit.Time + " "
		}
		if habit.Flag != "" {
			meta += "(" + habit.Flag + ") "
		}
		if meta != "" {
			meta += "- "
		}
		line := fmt.Sprintf("%s%s%s · %s", cursor, meta, habit.Desc, habit.Category)
		if i == m.habitCursor {
			line = styleSelected.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styleHelp.Render("j/k mover · enter escolhe · esc cancela"))
	return b.String()
}

func (m Model) weeklistView() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("weeklist"))
	b.WriteString("  ")
	b.WriteString(styleSubtle.Render(weekLabel(m.now)))
	b.WriteString("\n\n")

	if m.week == nil {
		b.WriteString("Weeklist indisponível.\n")
		return b.String()
	}

	b.WriteString(m.renderNote(m.week, m.weekSelectedLine(), true))
	b.WriteString("\n")
	b.WriteString(m.messageView())
	b.WriteString(styleHelp.Render("espaço alterna #always em hábitos · c cria se faltar · esc volta"))
	return b.String()
}

func (m Model) renderNote(note *daylist.Note, selectedLine int, week bool) string {
	var b strings.Builder

	for _, section := range note.Sections() {
		b.WriteString(styleHeading.Render("# " + section.Title))
		b.WriteString("\n")

		for _, item := range section.Items {
			if item.Category != "" {
				continue
			}
			b.WriteString(m.renderItem(item, item.Line == selectedLine, week))
		}

		for _, category := range section.Categories {
			b.WriteString("  ")
			b.WriteString(styleHeading.Render("## " + category.Title))
			b.WriteString("\n")
			for _, item := range category.Items {
				b.WriteString(m.renderItem(item, item.Line == selectedLine, week))
			}
		}
	}

	return b.String()
}

func (m Model) renderItem(item daylist.NoteItem, selected, week bool) string {
	cursor := "  "
	if selected {
		cursor = "▶ "
	}
	box := "[ ]"
	if item.Checked {
		box = "[x]"
	}

	meta := ""
	if item.Time != "" {
		meta += item.Time + " "
	}
	if item.Flag != "" {
		meta += "(" + item.Flag + ") "
	}
	if meta != "" {
		meta += "- "
	}

	line := fmt.Sprintf("%s%s %s%s", cursor, box, meta, item.Desc)
	if week && item.Always {
		line += " #always"
	}
	if item.Timer != "" {
		line += " ⏱ " + item.Timer
	}

	switch {
	case selected:
		line = styleSelected.Render(line)
	case item.Checked:
		line = styleDone.Render(line)
	}
	b := line + "\n"

	if item.Motivation != "" {
		for _, part := range strings.Split(item.Motivation, "\n") {
			b += "      " + styleMotivate.Render("> "+part) + "\n"
		}
	}
	return b
}

func (m Model) selectedLine() int {
	item, ok := m.selected()
	if !ok {
		return -1
	}
	return item.Line
}

func (m Model) weekSelectedLine() int {
	if m.weekCursor < 0 || m.weekCursor >= len(m.weekItems) {
		return -1
	}
	return m.weekItems[m.weekCursor].Line
}

func (m Model) messageView() string {
	switch {
	case m.err != "":
		return styleError.Render("✗ "+m.err) + "\n"
	case m.st != "":
		return styleOK.Render(m.st) + "\n"
	default:
		return ""
	}
}

func dateLabel(t time.Time) string {
	return fmt.Sprintf("%s, %02d/%02d/%04d", weekdayNames[t.Weekday()], t.Day(), int(t.Month()), t.Year())
}

func weekLabel(t time.Time) string {
	year, week := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}
