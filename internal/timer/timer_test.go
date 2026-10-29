package timer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func testState() *State {
	return &State{
		File:     "2026/10/2026-10-29.md",
		Line:     "- [ ] 30min (I) - escrever testes",
		Desc:     "escrever testes",
		Start:    1_700_000_000_000,
		Started:  1_699_999_000_000,
		Elapsed:  600_000,
		Paused:   false,
		PausedAt: 0,
	}
}

func TestSaveAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	if err := testState().Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var got State
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("estado salvo não é JSON válido: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("arquivo temporário sobrou após Save: %s", e.Name())
		}
	}
}

func TestSaveKeepsPreviousOnFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	prev := testState()
	if err := prev.Save(path); err != nil {
		t.Fatalf("Save inicial: %v", err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile antes: %v", err)
	}

	// Um diretório no caminho do .tmp faz a escrita falhar de forma
	// determinística, mesmo quando os testes rodam como root.
	if err := os.Mkdir(path+".tmp", 0o755); err != nil {
		t.Fatalf("Mkdir %s.tmp: %v", path, err)
	}

	next := testState()
	next.Desc = "outro estado"
	if err := next.Save(path); err == nil {
		t.Fatal("Save deveria falhar com o caminho .tmp ocupado por um diretório")
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile depois: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("estado anterior foi alterado após falha\nantes: %s\ndepois: %s", before, after)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load depois da falha: %v", err)
	}
	if !reflect.DeepEqual(got, prev) {
		t.Fatalf("estado anterior não continua legível: got %+v, want %+v", got, prev)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	want := testState()
	want.Paused = true
	want.PausedAt = 1_700_000_500_000

	if err := want.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round-trip divergiu: got %+v, want %+v", got, want)
	}
}

// pausedState cria um estado pausado com o tempo informado, para não depender
// do relógio durante o teste.
func pausedState(line string, elapsed time.Duration) *State {
	return &State{
		Line:     line,
		Started:  1_699_999_000_000,
		Elapsed:  elapsed.Milliseconds(),
		Paused:   true,
		PausedAt: 1_700_000_000_000,
	}
}

// writeDaylist cria uma daylist temporária contendo a linha pendente.
func writeDaylist(t *testing.T, line string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "2026-10-29.md")
	if err := os.WriteFile(path, []byte("# Quinta\n\n"+line+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func TestFinishRounding(t *testing.T) {
	tests := []struct {
		name    string
		elapsed time.Duration
		want    int64
	}{
		{"20s arredonda para 1min", 20 * time.Second, 1},
		{"89s arredonda para 1min", 89 * time.Second, 1},
		{"90s arredonda para 2min", 90 * time.Second, 2},
		{"30min inteiros", 30 * time.Minute, 30},
		{"1h30m40s arredonda para 91min", time.Hour + 30*time.Minute + 40*time.Second, 91},
	}

	line := testState().Line
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := writeDaylist(t, line)
			s := pausedState(line, tc.elapsed)

			got, err := s.Finish(path)
			if err != nil {
				t.Fatalf("Finish: %v", err)
			}
			if got != tc.want {
				t.Fatalf("minutos = %d, want %d", got, tc.want)
			}

			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			content := string(data)
			if !strings.Contains(content, "- [x]") {
				t.Errorf("linha não foi marcada como concluída:\n%s", content)
			}
			if !strings.Contains(content, fmt.Sprintf(" ⏱ %dmin (", tc.want)) {
				t.Errorf("linha sem o tempo esperado (%dmin):\n%s", tc.want, content)
			}
		})
	}
}

func TestFinishZeroElapsed(t *testing.T) {
	line := testState().Line
	path := writeDaylist(t, line)
	s := pausedState(line, 0)

	got, err := s.Finish(path)
	if err != nil {
		t.Fatalf("Finish: %v", err)
	}
	if got != 0 {
		t.Fatalf("minutos = %d, want 0", got)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), " ⏱ 0min (") {
		t.Errorf("linha sem o tempo esperado (0min):\n%s", data)
	}
}
