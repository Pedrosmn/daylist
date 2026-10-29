// Package timer implementa o timer por item da daylist.
package timer

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

var (
	ErrNoTimer      = errors.New("nenhum timer rodando")
	ErrLineNotFound = errors.New("linha original não encontrada")

	// - [ ] 30min (I) - descrição
	pendingRe = regexp.MustCompile(`^- \[ \] .+\(.*\) - .+$`)
	descRe    = regexp.MustCompile(`^- \[ \] .+\(.*\) - `)
)

// State é salvo em JSON com os mesmos campos do script antigo.
// Todos os tempos em milissegundos (epoch).
type State struct {
	File     string `json:"file"`
	Line     string `json:"line"`
	Desc     string `json:"desc"`
	Start    int64  `json:"start"`
	Started  int64  `json:"started"`
	Elapsed  int64  `json:"elapsed"`
	Paused   bool   `json:"paused"`
	PausedAt int64  `json:"paused_at"`
}

func nowMs() int64 { return time.Now().UnixMilli() }

// New cria um timer para a linha escolhida. rel é o caminho da daylist relativo ao vault.
func New(line, rel string) *State {
	now := nowMs()
	return &State{
		File:    rel,
		Line:    line,
		Desc:    descRe.ReplaceAllString(line, ""),
		Start:   now,
		Started: now,
	}
}

func Load(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNoTimer
	}
	if err != nil {
		return nil, err
	}

	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("estado do timer inválido: %w", err)
	}
	return &s, nil
}

func (s *State) Save(path string) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return writeAtomic(path, data)
}

// writeAtomic grava data num arquivo temporário e renomeia para path, para
// nunca deixar o estado truncado se a escrita for interrompida.
func writeAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// PendingItems lista as linhas pendentes que ainda não têm ⏱.
func PendingItems(daylistFile string) ([]string, error) {
	data, err := os.ReadFile(daylistFile)
	if err != nil {
		return nil, err
	}

	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		if pendingRe.MatchString(line) && !strings.Contains(line, "⏱") {
			out = append(out, line)
		}
	}
	return out, nil
}

// Toggle pausa ou retoma. Retorna true se retomou.
func (s *State) Toggle() bool {
	now := nowMs()
	if !s.Paused {
		s.Elapsed += now - s.Start
		s.Paused = true
		s.PausedAt = now
		return false
	}
	s.Start = now
	s.Paused = false
	s.PausedAt = 0
	return true
}

func (s *State) TotalMs() int64 {
	if s.Paused {
		return s.Elapsed
	}
	return s.Elapsed + nowMs() - s.Start
}

// Finish marca o item como [x] e acrescenta " ⏱ Nmin (HH:MM–HH:MM)".
// Grava num arquivo temporário e renomeia, para nunca corromper a daylist.
func (s *State) Finish(daylistFile string) (int64, error) {
	data, err := os.ReadFile(daylistFile)
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(data), "\n")
	idx := -1
	for i, l := range lines {
		if l == s.Line {
			idx = i
			break
		}
	}
	if idx < 0 {
		return 0, ErrLineNotFound
	}

	ms := s.TotalMs()
	mins := (ms + 30_000) / 60_000
	if mins == 0 && ms > 0 {
		mins = 1
	}
	startHM := time.UnixMilli(s.Started).Format("15:04")
	endHM := time.Now().Format("15:04")

	lines[idx] = strings.Replace(lines[idx], "- [ ]", "- [x]", 1) +
		fmt.Sprintf(" ⏱ %dmin (%s–%s)", mins, startHM, endHM)

	return mins, writeAtomic(daylistFile, []byte(strings.Join(lines, "\n")))
}
