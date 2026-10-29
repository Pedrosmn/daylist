package timer

import (
	"bytes"
	"encoding/json"
	"errors"
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
	return writeDaylistContent(t, "# Quinta\n\n"+line+"\n")
}

// writeDaylistContent cria uma daylist temporária com o conteúdo exato.
func writeDaylistContent(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "2026-10-29.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
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

func TestFinishExactLine(t *testing.T) {
	line := testState().Line
	path := writeDaylist(t, line)
	s := pausedState(line, 5*time.Minute)

	got, err := s.Finish(path)
	if err != nil {
		t.Fatalf("Finish: %v", err)
	}
	if got != 5 {
		t.Fatalf("minutos = %d, want 5", got)
	}

	content := readFile(t, path)
	if !strings.Contains(content, "escrever testes ⏱ 5min (") {
		t.Errorf("linha sem o tempo esperado:\n%s", content)
	}
}

func TestFinishEditedLine(t *testing.T) {
	// Espaços extras e caixa diferente não podem impedir o registro.
	edited := "  - [ ]   30min (I) -   Escrever TESTES  "
	path := writeDaylist(t, edited)
	s := pausedState(testState().Line, 5*time.Minute)

	got, err := s.Finish(path)
	if err != nil {
		t.Fatalf("Finish: %v", err)
	}
	if got != 5 {
		t.Fatalf("minutos = %d, want 5", got)
	}

	content := readFile(t, path)
	if !strings.Contains(content, "- [x]") || !strings.Contains(content, " ⏱ 5min (") {
		t.Errorf("linha editada não recebeu o tempo:\n%s", content)
	}
}

func TestFinishManuallyDoneLine(t *testing.T) {
	done := "- [x] 30min (I) - escrever testes"
	path := writeDaylist(t, done)
	s := pausedState(testState().Line, 5*time.Minute)

	got, err := s.Finish(path)
	if err != nil {
		t.Fatalf("Finish: %v", err)
	}
	if got != 5 {
		t.Fatalf("minutos = %d, want 5", got)
	}

	content := readFile(t, path)
	if n := strings.Count(content, "⏱"); n != 1 {
		t.Fatalf("marcador ⏱ aparece %d vez(es), want 1:\n%s", n, content)
	}
	if !strings.Contains(content, "escrever testes ⏱ 5min (") {
		t.Errorf("linha sem o tempo esperado:\n%s", content)
	}
}

func TestFinishPartialUnique(t *testing.T) {
	path := writeDaylist(t, "- [ ] 45min (I) - escrever testes de integração")
	s := pausedState(testState().Line, 5*time.Minute)

	got, err := s.Finish(path)
	if err != nil {
		t.Fatalf("Finish: %v", err)
	}
	if got != 5 {
		t.Fatalf("minutos = %d, want 5", got)
	}

	content := readFile(t, path)
	if !strings.Contains(content, "- [x] 45min (I) - escrever testes de integração ⏱ 5min (") {
		t.Errorf("correspondência parcial não recebeu o tempo:\n%s", content)
	}
}

func TestFinishPartialAmbiguous(t *testing.T) {
	path := writeDaylistContent(t, "# Quinta\n\n"+
		"- [ ] 45min (I) - escrever testes de integração\n"+
		"- [ ] 45min (II) - escrever testes unitários\n")
	before := readFile(t, path)
	s := pausedState(testState().Line, 5*time.Minute)

	_, err := s.Finish(path)
	if !errors.Is(err, ErrAmbiguousLine) {
		t.Fatalf("erro = %v, want ErrAmbiguousLine", err)
	}
	if !strings.Contains(err.Error(), "integração") || !strings.Contains(err.Error(), "unitários") {
		t.Errorf("erro não lista as candidatas: %v", err)
	}
	if after := readFile(t, path); after != before {
		t.Errorf("daylist foi alterada em caso ambíguo\nantes:\n%s\ndepois:\n%s", before, after)
	}
}

func TestFinishNotFoundKeepsDaylist(t *testing.T) {
	path := writeDaylist(t, "- [ ] 30min (I) - outra coisa")
	before := readFile(t, path)
	s := pausedState(testState().Line, 5*time.Minute)

	_, err := s.Finish(path)
	if !errors.Is(err, ErrLineNotFound) {
		t.Fatalf("erro = %v, want ErrLineNotFound", err)
	}
	if after := readFile(t, path); after != before {
		t.Errorf("daylist foi alterada quando a linha não foi encontrada\nantes:\n%s\ndepois:\n%s", before, after)
	}
}

func TestFinishCRLF(t *testing.T) {
	line := testState().Line
	path := writeDaylistContent(t, "# Quinta\r\n\r\n"+line+"\r\n")
	s := pausedState(line, 5*time.Minute)

	got, err := s.Finish(path)
	if err != nil {
		t.Fatalf("Finish: %v", err)
	}
	if got != 5 {
		t.Fatalf("minutos = %d, want 5", got)
	}

	content := readFile(t, path)
	if strings.Contains(content, "\r ⏱") {
		t.Errorf("quebra de linha CRLF foi corrompida:\n%s", content)
	}
	if !strings.Contains(content, "escrever testes ⏱ 5min (") {
		t.Errorf("linha sem o tempo esperado:\n%s", content)
	}
	if !strings.HasSuffix(content, ")\r\n") {
		t.Errorf("arquivo CRLF perdeu o formato:\n%q", content)
	}
}

func TestFinishAlreadyRecorded(t *testing.T) {
	line := testState().Line + " ⏱ 10min (09:00–09:10)"
	path := writeDaylist(t, line)
	before := readFile(t, path)
	s := pausedState(testState().Line, 5*time.Minute)

	_, err := s.Finish(path)
	if !errors.Is(err, ErrAlreadyRecorded) {
		t.Fatalf("erro = %v, want ErrAlreadyRecorded", err)
	}
	if n := strings.Count(readFile(t, path), "⏱"); n != 1 {
		t.Errorf("marcador ⏱ duplicado (%d ocorrências)", n)
	}
	if after := readFile(t, path); after != before {
		t.Errorf("daylist foi alterada quando o tempo já estava registrado\nantes:\n%s\ndepois:\n%s", before, after)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	return string(data)
}
