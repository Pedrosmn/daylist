package wizard

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"daylist/internal/daylist"
	"daylist/internal/ui"
)

type fakePrompter struct {
	toggles      []ui.ToggleResult
	toggleOpts   []ui.ToggleOptions
	picks        map[string]string
	inputs       []string
	confirms     []bool
	pickCalls    []string
	confirmCalls []string
}

func (f *fakePrompter) Toggle(opts ui.ToggleOptions) (ui.ToggleResult, error) {
	f.toggleOpts = append(f.toggleOpts, opts)
	if len(f.toggles) == 0 {
		return ui.ToggleResult{}, errors.New("nenhum toggle configurado")
	}
	res := f.toggles[0]
	f.toggles = f.toggles[1:]
	return res, nil
}

func (f *fakePrompter) Pick(prompt string, _ []string) (string, error) {
	f.pickCalls = append(f.pickCalls, prompt)
	if f.picks != nil {
		if value, ok := f.picks[prompt]; ok {
			return value, nil
		}
	}
	return "", nil
}

func (f *fakePrompter) Input(string) (string, error) {
	if len(f.inputs) == 0 {
		return "", errors.New("nenhum input configurado")
	}
	value := f.inputs[0]
	f.inputs = f.inputs[1:]
	return value, nil
}

func (f *fakePrompter) Confirm(prompt string) (bool, error) {
	f.confirmCalls = append(f.confirmCalls, prompt)
	if len(f.confirms) == 0 {
		return false, nil
	}
	value := f.confirms[0]
	f.confirms = f.confirms[1:]
	return value, nil
}

func stage(title string, items ...daylist.Item) daylist.Stage {
	return daylist.Stage{Title: title, Items: items}
}

func item(desc, time, flag string) daylist.Item {
	return daylist.Item{Section: "Hábitos", Category: "Geral", Desc: desc, Time: time, Flag: flag}
}

func TestRunBackPreservesSelection(t *testing.T) {
	p := &fakePrompter{
		toggles: []ui.ToggleResult{
			{Selected: []int{0}, Action: ui.ToggleFinish},
			{Action: ui.ToggleBack},
			{Selected: []int{0}, Action: ui.ToggleFinish},
			{Selected: []int{0}, Action: ui.ToggleFinish},
		},
	}

	result, err := Run(Options{
		Stages: []daylist.Stage{
			stage("Tasks", item("A", "10 min", "I"), item("B", "20 min", "U")),
			stage("Estudos", item("C", "30 min", "I")),
		},
		UI: p,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Canceled {
		t.Fatal("Canceled = true")
	}
	if len(result.Resolved) != 2 {
		t.Fatalf("resolved = %d, want 2", len(result.Resolved))
	}

	if len(p.toggleOpts) != 4 {
		t.Fatalf("toggles = %d, want 4", len(p.toggleOpts))
	}
	// A segunda entrada na primeira etapa deve reabrir com A marcado.
	if len(p.toggleOpts[2].Selected) != 1 || p.toggleOpts[2].Selected[0] != 0 {
		t.Errorf("seleção ao voltar = %v, want [0]", p.toggleOpts[2].Selected)
	}
	if p.toggleOpts[2].AllowBack {
		t.Error("primeira etapa não deveria permitir voltar")
	}
	if !p.toggleOpts[1].AllowBack {
		t.Error("segunda etapa deveria permitir voltar")
	}
}

func TestRunBackDoesNotReaskResolvedItems(t *testing.T) {
	p := &fakePrompter{
		toggles: []ui.ToggleResult{
			{Selected: []int{0}, Action: ui.ToggleFinish},
			{Action: ui.ToggleBack},
			{Selected: []int{0}, Action: ui.ToggleFinish},
			{Selected: []int{0}, Action: ui.ToggleFinish},
		},
		picks: map[string]string{
			`Duração de "A": `:    "Personalizado",
			`Prioridade de "A": `: "Importante (I)",
		},
		inputs: []string{"25"},
	}

	result, err := Run(Options{
		Stages: []daylist.Stage{
			stage("Tasks", item("A", "", "")),
			stage("Estudos", item("B", "10 min", "I")),
		},
		UI: p,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Resolved) != 2 {
		t.Fatalf("resolved = %d, want 2", len(result.Resolved))
	}
	if result.Resolved[0].FinalTime != "25 min" || result.Resolved[0].FinalFlag != "I" {
		t.Errorf("item A = %q (%q), want 25 min (I)", result.Resolved[0].FinalTime, result.Resolved[0].FinalFlag)
	}
	if len(p.pickCalls) != 2 {
		t.Errorf("picks = %d, want 2 (não deveria reperguntar ao voltar)", len(p.pickCalls))
	}
}

func TestRunCancelAsksConfirmation(t *testing.T) {
	p := &fakePrompter{
		toggles: []ui.ToggleResult{
			{Selected: []int{0}, Action: ui.ToggleCancel},
			{Selected: []int{0}, Action: ui.ToggleFinish},
		},
		confirms: []bool{false},
	}

	result, err := Run(Options{
		Stages: []daylist.Stage{stage("Tasks", item("A", "10 min", "I"))},
		UI:     p,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Canceled {
		t.Fatal("Canceled = true, want false")
	}
	if len(p.confirmCalls) != 1 {
		t.Fatalf("confirmações = %d, want 1", len(p.confirmCalls))
	}
	if len(result.Resolved) != 1 {
		t.Fatalf("resolved = %d, want 1", len(result.Resolved))
	}
}

func TestRunCancelDiscardsEverything(t *testing.T) {
	p := &fakePrompter{
		toggles: []ui.ToggleResult{
			{Selected: []int{0}, Action: ui.ToggleCancel},
		},
		confirms: []bool{true},
	}

	result, err := Run(Options{
		Stages: []daylist.Stage{stage("Tasks", item("A", "10 min", "I"))},
		UI:     p,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Canceled {
		t.Fatal("Canceled = false, want true")
	}
	if len(result.Resolved) != 0 {
		t.Errorf("resolved = %v, want vazio", result.Resolved)
	}
}

func TestRunTotalRange(t *testing.T) {
	p := &fakePrompter{
		toggles: []ui.ToggleResult{
			{Selected: []int{0, 1, 2}, Action: ui.ToggleFinish},
		},
	}

	var out bytes.Buffer
	result, err := Run(Options{
		Stages: []daylist.Stage{stage("Tasks",
			item("A", "40 min", "I"),
			item("B", "5/15 min", "I"),
			item("C", "60 min", "I"),
		)},
		UI:  p,
		Out: &out,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalMin != 105 || result.TotalMax != 115 {
		t.Errorf("total = %d–%d, want 105–115", result.TotalMin, result.TotalMax)
	}
	if !strings.Contains(out.String(), "Tempo planejado: 1h45–1h55") {
		t.Errorf("saída sem tempo total:\n%s", out.String())
	}
}

func TestRunCustomDurationUsesInput(t *testing.T) {
	p := &fakePrompter{
		toggles: []ui.ToggleResult{
			{Selected: []int{0}, Action: ui.ToggleFinish},
		},
		picks: map[string]string{
			`Duração de "A": `: "Personalizado",
		},
		inputs: []string{"45"},
	}

	result, err := Run(Options{
		Stages: []daylist.Stage{stage("Tasks", item("A", "", "I"))},
		UI:     p,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Resolved) != 1 {
		t.Fatalf("resolved = %d, want 1", len(result.Resolved))
	}
	if result.Resolved[0].FinalTime != "45 min" {
		t.Errorf("FinalTime = %q, want 45 min", result.Resolved[0].FinalTime)
	}
}

func TestParseTimeRange(t *testing.T) {
	tests := []struct {
		value string
		min   int
		max   int
		ok    bool
	}{
		{"40 min", 40, 40, true},
		{"5/15 min", 5, 15, true},
		{"15/5 min", 5, 15, true},
		{"1h", 60, 60, true},
		{"1h45", 105, 105, true},
		{"1h 45 min", 105, 105, true},
		{"", 0, 0, false},
		{"ST", 0, 0, false},
		{"indefinido", 0, 0, false},
	}
	for _, tt := range tests {
		min, max, ok := parseTimeRange(tt.value)
		if min != tt.min || max != tt.max || ok != tt.ok {
			t.Errorf("parseTimeRange(%q) = %d, %d, %v; want %d, %d, %v",
				tt.value, min, max, ok, tt.min, tt.max, tt.ok)
		}
	}
}

func TestFormatRange(t *testing.T) {
	tests := map[string]struct {
		min, max int
		want     string
	}{
		"minutos":   {45, 45, "45 min"},
		"hora":      {120, 120, "2h"},
		"faixa":     {105, 115, "1h45–1h55"},
		"atravessa": {55, 65, "55 min–1h05"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := formatRange(tt.min, tt.max); got != tt.want {
				t.Errorf("formatRange(%d, %d) = %q, want %q", tt.min, tt.max, got, tt.want)
			}
		})
	}
}
