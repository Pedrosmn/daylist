package week

import (
	"math"
	"strings"
	"time"
)

// DayStat resume os hábitos de uma daylist da semana.
type DayStat struct {
	Date       time.Time
	HasDaylist bool
	Checked    int
	Total      int
	Percent    int
}

// HabitStat acumula as ocorrências de um hábito na semana.
type HabitStat struct {
	Category string
	Desc     string
	Checked  int
	Total    int
	Percent  int
}

// CategoryStat agrupa os hábitos de uma categoria na ordem da weeklist.
type CategoryStat struct {
	Name    string
	Checked int
	Total   int
	Percent int
	Habits  []HabitStat
}

// Report é o cálculo compartilhado entre o feedback e o TUI. Os hábitos são
// contados como no `daylist feedback`: [x] sobre as ocorrências da semana.
type Report struct {
	Categories []CategoryStat
	Days       []DayStat
	Checked    int
	Total      int
	Percent    int
}

// HasHabits informa se a semana tem ao menos um hábito conhecido, mesmo que
// sem ocorrências.
func (r Report) HasHabits() bool {
	for _, category := range r.Categories {
		if len(category.Habits) > 0 {
			return true
		}
	}
	return false
}

// DaysWithDaylist conta os dias da semana que têm daylist.
func (r Report) DaysWithDaylist() int {
	count := 0
	for _, day := range r.Days {
		if day.HasDaylist {
			count++
		}
	}
	return count
}

// ComputeReport calcula o resumo da semana a partir da weeklist e dos
// conteúdos das daylists. `week` define o esqueleto de sete dias; os dias sem
// daylist ficam marcados com HasDaylist falso. A semana atual deve receber
// apenas as daylists até hoje; semanas passadas recebem até domingo.
func ComputeReport(week, weeklist string, days []Day) Report {
	type habitKey struct {
		category string
		desc     string
	}

	stats := map[habitKey]*HabitStat{}
	var statOrder []habitKey
	var categoryOrder []string
	seenCategory := map[string]bool{}

	addCategory := func(category string) {
		if category == "" || seenCategory[category] {
			return
		}
		seenCategory[category] = true
		categoryOrder = append(categoryOrder, category)
	}

	addHabit := func(category, desc string) *HabitStat {
		key := habitKey{category: category, desc: desc}
		if stat, ok := stats[key]; ok {
			return stat
		}
		stat := &HabitStat{Category: category, Desc: desc}
		stats[key] = stat
		statOrder = append(statOrder, key)
		return stat
	}

	// A weeklist define as categorias e os hábitos que devem aparecer mesmo
	// que nunca tenham sido usados na semana.
	inHabits, category := false, ""
	for _, line := range strings.Split(weeklist, "\n") {
		line = strings.TrimRight(line, "\r")

		switch {
		case strings.HasPrefix(line, "# Hábitos"):
			inHabits, category = true, ""
			continue
		case inHabits && strings.HasPrefix(line, "# "):
			inHabits = false
			continue
		case inHabits && strings.HasPrefix(line, "## "):
			category = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			addCategory(category)
			continue
		}
		if !inHabits {
			continue
		}

		habit, ok := ParseHabitLine(line)
		if !ok {
			continue
		}
		addHabit(category, StripTimerSuffix(habit.Desc))
	}

	report := Report{}
	start, err := ParseWeek(week)
	dayIndex := map[string]int{}
	if err == nil {
		for _, day := range WeekDays(start) {
			dayIndex[day.Format("2006-01-02")] = len(report.Days)
			report.Days = append(report.Days, DayStat{Date: day})
		}
	}
	ensureDay := func(date time.Time) *DayStat {
		key := date.Format("2006-01-02")
		if idx, ok := dayIndex[key]; ok {
			return &report.Days[idx]
		}
		idx := len(report.Days)
		dayIndex[key] = idx
		report.Days = append(report.Days, DayStat{Date: date})
		return &report.Days[idx]
	}

	// Cada ocorrência numa daylist incrementa o total; [x] incrementa também
	// o número de concluídos. O mesmo recorte alimenta as estatísticas do dia.
	for _, day := range days {
		dayStat := ensureDay(day.Date)
		dayStat.HasDaylist = true

		inHabits, category = false, ""
		for _, line := range strings.Split(day.Content, "\n") {
			line = strings.TrimRight(line, "\r")

			switch {
			case strings.HasPrefix(line, "# Hábitos"):
				inHabits, category = true, ""
				continue
			case inHabits && strings.HasPrefix(line, "# "):
				inHabits = false
				continue
			case inHabits && strings.HasPrefix(line, "## "):
				category = strings.TrimSpace(strings.TrimPrefix(line, "## "))
				continue
			}
			if !inHabits {
				continue
			}

			habit, ok := ParseHabitLine(line)
			if !ok {
				continue
			}

			desc := StripTimerSuffix(habit.Desc)
			key := habitKey{category: category, desc: desc}
			stat, exists := stats[key]
			if !exists {
				addCategory(category)
				stat = addHabit(category, desc)
			}
			stat.Total++
			dayStat.Total++
			if habit.Checked {
				stat.Checked++
				dayStat.Checked++
			}
		}
		dayStat.Percent = percentage(dayStat.Checked, dayStat.Total)
	}

	for _, key := range statOrder {
		stat := stats[key]
		stat.Percent = percentage(stat.Checked, stat.Total)
		report.Checked += stat.Checked
		report.Total += stat.Total
	}
	report.Percent = percentage(report.Checked, report.Total)

	// Categorias da weeklist aparecem mesmo sem hábitos ou ocorrências.
	categoryIndex := map[string]int{}
	for _, name := range categoryOrder {
		categoryIndex[name] = len(report.Categories)
		report.Categories = append(report.Categories, CategoryStat{Name: name})
	}
	// Hábitos sem categoria (antes do primeiro "##") não têm cabeçalho, mas
	// continuam contando no total. Agrupa-os no fim para não perder o dado.
	for _, key := range statOrder {
		stat := stats[key]
		idx, ok := categoryIndex[stat.Category]
		if !ok {
			idx = len(report.Categories)
			categoryIndex[stat.Category] = idx
			report.Categories = append(report.Categories, CategoryStat{Name: stat.Category})
		}
		categoryStat := &report.Categories[idx]
		categoryStat.Habits = append(categoryStat.Habits, *stat)
		categoryStat.Checked += stat.Checked
		categoryStat.Total += stat.Total
	}
	for i := range report.Categories {
		categoryStat := &report.Categories[i]
		categoryStat.Percent = percentage(categoryStat.Checked, categoryStat.Total)
	}

	return report
}

func percentage(checked, total int) int {
	if total == 0 {
		return 0
	}
	return int(math.Round(float64(checked) / float64(total) * 100))
}
