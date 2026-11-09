package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestInitSchedulesPreviousFeedback(t *testing.T) {
	fake := newFake("# Tasks\n")
	fake.now = localDate(2026, time.November, 9) // segunda-feira, 2026-W46
	fake.autoCreated = true
	m := New(fake)

	cmd := m.Init()
	batch, ok := cmd().(tea.BatchMsg)
	if !ok {
		t.Fatalf("Init deveria agendar relógio e feedback, veio %T", cmd())
	}
	if len(batch) != 2 {
		t.Fatalf("comandos do Init = %d, want 2", len(batch))
	}

	msg := batch[1]()
	generated, ok := msg.(feedbackGeneratedMsg)
	if !ok {
		t.Fatalf("segundo comando = %T, want feedbackGeneratedMsg", msg)
	}
	if generated.week != "2026-W45" || !generated.created || generated.err != nil {
		t.Errorf("mensagem = %+v, want semana 2026-W45 criada", generated)
	}
	if fake.autoRun != 1 {
		t.Errorf("AutoGeneratePreviousFeedback chamado %d vez(es), want 1", fake.autoRun)
	}

	m = updateModel(t, m, msg)
	if !strings.Contains(m.st, "Feedback de 2026-W45 gerado automaticamente.") {
		t.Errorf("status = %q", m.st)
	}
}

func TestAutoGenerateFeedbackAlreadyExistsDoesNotAnnounce(t *testing.T) {
	fake := newFake("# Tasks\n")
	fake.now = localDate(2026, time.November, 9)
	fake.autoCreated = false
	m := New(fake)

	m = updateModel(t, m, m.autoGenerateFeedback(fake.now)())
	if m.st != "" {
		t.Errorf("status = %q, want vazio quando o feedback já existe", m.st)
	}
	if m.err != "" {
		t.Errorf("erro = %q, want vazio", m.err)
	}
}

func TestAutoGenerateFeedbackErrorIsReported(t *testing.T) {
	fake := newFake("# Tasks\n")
	fake.now = localDate(2026, time.November, 9)
	fake.autoErr = errors.New("disco cheio")
	m := New(fake)

	m = updateModel(t, m, m.autoGenerateFeedback(fake.now)())
	if !strings.Contains(m.err, "não foi possível gerar o feedback da semana anterior") {
		t.Errorf("erro = %q", m.err)
	}
	if m.mode != modeMain {
		t.Errorf("mode = %v, want modeMain; o erro não pode bloquear o uso", m.mode)
	}
}

func TestAutoFeedbackOnRolloverSundayToMonday(t *testing.T) {
	sunday := localDate(2026, time.November, 8)
	monday := localDate(2026, time.November, 9)
	fake := newFake("# Tasks\n")
	fake.now = sunday
	fake.autoCreated = true
	m := New(fake)

	fake.now = monday
	next, cmd := m.Update(tickMsg(monday))
	m, ok := next.(*Model)
	if !ok {
		t.Fatalf("Update devolveu %T, want *Model", next)
	}

	// O tick devolve o relógio e o automático; executa só o automático.
	batch, ok := cmd().(tea.BatchMsg)
	if !ok {
		t.Fatalf("tick deveria agendar BatchMsg, veio %T", cmd())
	}
	if len(batch) != 2 {
		t.Fatalf("comandos do tick = %d, want relógio + feedback", len(batch))
	}
	msg := batch[1]()
	generated, ok := msg.(feedbackGeneratedMsg)
	if !ok {
		t.Fatalf("segundo comando = %T, want feedbackGeneratedMsg", msg)
	}
	if generated.week != "2026-W45" || !generated.created || generated.err != nil {
		t.Errorf("mensagem = %+v, want 2026-W45 criada", generated)
	}

	m = updateModel(t, m, msg)
	if !strings.Contains(m.st, "Feedback de 2026-W45 gerado automaticamente.") {
		t.Errorf("status = %q", m.st)
	}
	if fake.autoRun != 1 {
		t.Errorf("AutoGeneratePreviousFeedback chamado %d vez(es), want 1", fake.autoRun)
	}
}

func TestAutoFeedbackOnRolloverOtherDaysDoesNothing(t *testing.T) {
	fake := newFake("# Tasks\n")
	m := New(fake)

	cases := []struct {
		name       string
		old, new   time.Time
		wantCmdNil bool
	}{
		{"segunda para terça", localDate(2026, time.November, 9), localDate(2026, time.November, 10), true},
		{"sábado para domingo", localDate(2026, time.November, 7), localDate(2026, time.November, 8), true},
		{"domingo para segunda", localDate(2026, time.November, 8), localDate(2026, time.November, 9), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := m.autoFeedbackOnRollover(tc.old, tc.new)
			if gotNil := cmd == nil; gotNil != tc.wantCmdNil {
				t.Errorf("comando nulo = %v, want %v", gotNil, tc.wantCmdNil)
			}
		})
	}
}
