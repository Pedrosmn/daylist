package timer

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// PromptText devolve a saída curta do timer, adequada para RPROMPT ou para o
// cabeçalho do TUI.
func PromptText(s *State) string {
	total := s.TotalMs()
	mins := RoundedMinutes(total)
	text := fmt.Sprintf("⏱ %dm", mins)
	if min, max, ok := s.Limits(); ok {
		switch {
		case total >= int64(max)*60_000:
			text = fmt.Sprintf("⏱ %dm⚠", mins)
		case total >= int64(min)*60_000:
			text = fmt.Sprintf("⏱ %dm✓", mins)
		default:
			text = fmt.Sprintf("⏱ %d/%dm", mins, max)
		}
	}
	if s.Paused {
		text += " ⏸"
	}
	return text
}

// StatusText devolve a linha humana do timer atual.
func StatusText(s *State) string {
	total := s.TotalMs()
	mins := RoundedMinutes(total)
	details := LimitsText(s, total)
	if s.Paused {
		return fmt.Sprintf("Timer pausado: %s — %dmin acumulados%s%s", s.Desc, mins, details, StartedAt(s))
	}
	return fmt.Sprintf("Timer rodando: %s — %dmin%s%s", s.Desc, mins, details, StartedAt(s))
}

// LimitsText devolve o planejado e o limite alcançado, se houver.
func LimitsText(s *State, total int64) string {
	min, max, ok := s.Limits()
	if !ok {
		return ""
	}

	parts := []string{"planejado " + FormatLimits(min, max)}
	switch {
	case total >= int64(max)*60_000:
		parts = append(parts, "máximo atingido")
	case total >= int64(min)*60_000:
		parts = append(parts, "mínimo atingido")
	}
	return " (" + strings.Join(parts, "; ") + ")"
}

// FormatLimits mostra um tempo planejado único ou em faixa.
func FormatLimits(min, max int) string {
	if min == max {
		return fmt.Sprintf("%dmin", max)
	}
	return fmt.Sprintf("%d–%dmin", min, max)
}

// StartedAt devolve " (iniciado às HH:MM)" ou "" quando Started não existe.
func StartedAt(s *State) string {
	if s.Started <= 0 {
		return ""
	}
	return fmt.Sprintf(" (iniciado às %s)", time.UnixMilli(s.Started).Format("15:04"))
}

// CheckThresholds sinaliza, no máximo uma vez por limite, que o tempo mínimo
// ou máximo foi atingido. Devolve a mensagem e se o estado mudou (para o
// chamador persistir os sinalizadores).
func (s *State) CheckThresholds(total int64) (string, bool) {
	minReached, maxReached := s.Thresholds(total)
	switch {
	case maxReached:
		if s.MaxNotified {
			return "", false
		}
		s.MaxNotified = true
		s.MinNotified = true
		return fmt.Sprintf("Tempo máximo atingido: %s (%dmin)", s.Desc, RoundedMinutes(total)), true
	case minReached:
		if s.MinNotified {
			return "", false
		}
		s.MinNotified = true
		return fmt.Sprintf("Tempo mínimo atingido: %s (%dmin)", s.Desc, RoundedMinutes(total)), true
	default:
		return "", false
	}
}

// NotifyDesktop envia uma notificação pelo notify-send, quando disponível.
func NotifyDesktop(title, body string) error {
	if _, err := exec.LookPath("notify-send"); err != nil {
		return err
	}
	return exec.Command("notify-send", "-a", "daylist", title, body).Run()
}
