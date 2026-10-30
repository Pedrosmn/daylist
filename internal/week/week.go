// Package week contém aritmética de semanas ISO e a geração de weeklists e
// feedbacks a partir do conteúdo das notas.
package week

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var weekRe = regexp.MustCompile(`^(\d{4})-W(\d{2})$`)

// ParseWeek valida "YYYY-Www" e devolve a segunda-feira da semana, 00:00 local.
func ParseWeek(s string) (time.Time, error) {
	m := weekRe.FindStringSubmatch(s)
	if m == nil {
		return time.Time{}, fmt.Errorf("semana inválida %q (use YYYY-Www)", s)
	}

	year, _ := strconv.Atoi(m[1])
	number, _ := strconv.Atoi(m[2])
	start, err := weekStart(year, number)
	if err != nil {
		return time.Time{}, fmt.Errorf("semana inválida %q: %w", s, err)
	}
	return start, nil
}

// weekStart devolve a segunda-feira da semana ISO número `number` do ano.
func weekStart(year, number int) (time.Time, error) {
	if number < 1 || number > 53 {
		return time.Time{}, fmt.Errorf("número da semana fora de 01–53")
	}

	// 4 de janeiro sempre pertence à semana ISO 1.
	jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.Local)
	offset := (int(jan4.Weekday()) + 6) % 7 // segunda = 0
	monday := jan4.AddDate(0, 0, -offset)
	start := monday.AddDate(0, 0, (number-1)*7)

	gotYear, gotWeek := start.ISOWeek()
	if gotYear != year || gotWeek != number {
		return time.Time{}, fmt.Errorf("o ano %d não tem a semana %02d", year, number)
	}
	return start, nil
}

// WeekString devolve a semana ISO de t no formato YYYY-Www.
func WeekString(t time.Time) string {
	year, number := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, number)
}

// PreviousWeek devolve a segunda-feira da semana anterior.
func PreviousWeek(weekStart time.Time) time.Time {
	return weekStart.AddDate(0, 0, -7)
}

// WeekDays devolve as sete datas (segunda a domingo) da semana.
func WeekDays(weekStart time.Time) []time.Time {
	days := make([]time.Time, 0, 7)
	for i := 0; i < 7; i++ {
		days = append(days, weekStart.AddDate(0, 0, i))
	}
	return days
}
