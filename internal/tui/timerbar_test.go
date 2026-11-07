package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"daylist/internal/timer"
)

func pausedTimer(min, max int, elapsed time.Duration) *timer.State {
	return &timer.State{
		Desc:       "tarefa",
		Paused:     true,
		Elapsed:    elapsed.Milliseconds(),
		MinMinutes: min,
		MaxMinutes: max,
	}
}

func TestTimerBarLayout(t *testing.T) {
	cases := []struct {
		name        string
		state       *timer.State
		width       int
		wantFilled  int
		wantMinMark int
		wantCurrent int
		wantOK      bool
	}{
		{"zero", pausedTimer(5, 15, 0), 82, 0, 27, 0, true},
		{"antes do mínimo", pausedTimer(5, 15, 2*time.Minute), 82, 11, 27, 2, true},
		{"mínimo atingido", pausedTimer(5, 15, 5*time.Minute), 82, 27, 27, 5, true},
		{"máximo atingido", pausedTimer(5, 15, 15*time.Minute), 82, 80, 27, 15, true},
		{"acima do máximo", pausedTimer(5, 15, 20*time.Minute), 82, 80, 27, 20, true},
		{"mínimo igual ao máximo", pausedTimer(10, 10, 3*time.Minute), 82, 24, 79, 3, true},
		{"sem faixa", &timer.State{Paused: true}, 82, 0, 0, 0, false},
		{"largura mínima", pausedTimer(5, 15, 0), 3, 0, 0, 0, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			layout, ok := newTimerBarLayout(c.state, c.width)
			if ok != c.wantOK {
				t.Fatalf("ok = %v, want %v", ok, c.wantOK)
			}
			if !ok {
				return
			}
			if layout.Filled != c.wantFilled || layout.MinMark != c.wantMinMark || layout.Current != c.wantCurrent {
				t.Errorf("layout = %+v, want filled=%d mark=%d current=%d",
					layout, c.wantFilled, c.wantMinMark, c.wantCurrent)
			}
		})
	}
}

func TestTimerBarViewShowsProgressAndLabels(t *testing.T) {
	fake := newFake("# Tasks\n- [ ] 10 min (I) - tarefa\n")
	fake.timer = pausedTimer(5, 15, 2*time.Minute)
	m := New(fake)
	m = updateModel(t, m, tea.WindowSizeMsg{Width: 40, Height: 24})

	view := m.timerBarView()
	lines := strings.Split(strings.TrimRight(ansi.Strip(view), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("linhas da barra = %d, want 2:\n%s", len(lines), view)
	}
	if width := lipgloss.Width(lines[0]); width != 40 {
		t.Errorf("largura da barra = %d, want 40", width)
	}
	for _, want := range []string{"[", "]", "█", "░", "┊"} {
		if !strings.Contains(lines[0], want) {
			t.Errorf("barra não contém %q: %q", want, lines[0])
		}
	}
	if !strings.Contains(lines[1], "mín 5m · agora 2m · máx 15m · ⏸ pausado") {
		t.Errorf("rótulos = %q", lines[1])
	}
}

func TestTimerBarNarrowTerminalFits(t *testing.T) {
	fake := newFake("# Tasks\n")
	fake.timer = pausedTimer(5, 15, 2*time.Minute)
	m := New(fake)
	m = updateModel(t, m, tea.WindowSizeMsg{Width: 10, Height: 24})

	lines := strings.Split(strings.TrimRight(ansi.Strip(m.timerBarView()), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("linhas da barra = %d, want 2", len(lines))
	}
	for _, line := range lines {
		if width := lipgloss.Width(line); width > 10 {
			t.Errorf("linha %q tem %d colunas, want <= 10", line, width)
		}
	}
	if !strings.Contains(lines[1], "5m·2m·15m") {
		t.Errorf("rótulo compacto = %q", lines[1])
	}
}

func TestTimerBarMinEqualsMaxUsesSingleEndMarker(t *testing.T) {
	fake := newFake("# Tasks\n")
	fake.timer = pausedTimer(10, 10, 3*time.Minute)
	m := New(fake)
	m = updateModel(t, m, tea.WindowSizeMsg{Width: 20, Height: 24})

	lines := strings.Split(strings.TrimRight(ansi.Strip(m.timerBarView()), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("linhas da barra = %d, want 2", len(lines))
	}
	if !strings.HasSuffix(lines[0], "┊]") {
		t.Errorf("marcador deveria estar no fim da barra: %q", lines[0])
	}
	if !strings.Contains(lines[1], "10m · 3m") {
		t.Errorf("rótulo = %q", lines[1])
	}
}

func TestTimerBarHiddenWithoutRange(t *testing.T) {
	fake := newFake("# Tasks\n")
	fake.timer = &timer.State{Desc: "tarefa", Start: time.Now().UnixMilli()}
	m := New(fake)
	m = updateModel(t, m, tea.WindowSizeMsg{Width: 40, Height: 24})

	if view := m.timerBarView(); view != "" {
		t.Errorf("barra deveria ficar oculta sem faixa de tempo:\n%s", view)
	}
}
