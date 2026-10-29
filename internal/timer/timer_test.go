package timer

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
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
