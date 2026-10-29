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
	ErrNoTimer         = errors.New("nenhum timer rodando")
	ErrLineNotFound    = errors.New("linha original não encontrada")
	ErrAmbiguousLine   = errors.New("mais de uma linha corresponde à descrição")
	ErrAlreadyRecorded = errors.New("tempo já registrado nesta linha")

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

// RoundedMinutes converte milissegundos em minutos arredondados para o mais
// próximo. Sessões com tempo > 0 nunca resultam em 0.
func RoundedMinutes(ms int64) int64 {
	if ms <= 0 {
		return 0
	}
	mins := (ms + 30_000) / 60_000
	if mins == 0 {
		mins = 1
	}
	return mins
}

// Finish marca o item como [x] e acrescenta " ⏱ Nmin (HH:MM–HH:MM)".
// Grava num arquivo temporário e renomeia, para nunca corromper a daylist.
func (s *State) Finish(daylistFile string) (int64, error) {
	data, err := os.ReadFile(daylistFile)
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(data), "\n")
	idx, err := findLine(lines, s.Line)
	if err != nil {
		return 0, err
	}
	if strings.Contains(lines[idx], "⏱") {
		return 0, ErrAlreadyRecorded
	}

	ms := s.TotalMs()
	mins := RoundedMinutes(ms)
	startHM := time.UnixMilli(s.Started).Format("15:04")
	endHM := time.Now().Format("15:04")

	line := lines[idx]
	crlf := ""
	if strings.HasSuffix(line, "\r") {
		line = strings.TrimSuffix(line, "\r")
		crlf = "\r"
	}

	line = strings.Replace(line, "- [ ]", "- [x]", 1) +
		fmt.Sprintf(" ⏱ %dmin (%s–%s)", mins, startHM, endHM)
	lines[idx] = line + crlf

	return mins, writeAtomic(daylistFile, []byte(strings.Join(lines, "\n")))
}

// normalize reduz espaços em branco consecutivos e remove os das pontas.
func normalize(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

// normalizeLine deixa a linha comparável: sem espaços extras, sem diferença
// de caixa e sem distinguir "- [ ]" de "- [x]".
func normalizeLine(s string) string {
	n := strings.ToLower(normalize(s))
	return strings.Replace(n, "- [x]", "- [ ]", 1)
}

func isPending(n string) bool { return strings.HasPrefix(n, "- [ ] ") }
func isDone(n string) bool    { return strings.HasPrefix(n, "- [x] ") }

// lineDescription extrai a descrição de uma linha, ignorando o checkbox e o
// bloco "(...)" de duração/tipo. Se a entrada já for só a descrição, devolve
// ela normalizada.
func lineDescription(s string) string {
	n := normalizeLine(s)
	for _, prefix := range []string{"- [ ] ", "- [x] "} {
		if strings.HasPrefix(n, prefix) {
			n = strings.TrimPrefix(n, prefix)
			break
		}
	}
	if i := strings.Index(n, ") - "); i >= 0 {
		return strings.TrimSpace(n[i+4:])
	}
	return n
}

func ambiguousError(lines []string, idxs []int) error {
	candidates := make([]string, len(idxs))
	for i, idx := range idxs {
		candidates[i] = strings.TrimSpace(lines[idx])
	}
	return fmt.Errorf("%w:\n  - %s", ErrAmbiguousLine, strings.Join(candidates, "\n  - "))
}

// findLine localiza a linha da daylist correspondente, tolerando edições:
// primeiro compara linha pendente normalizada, depois concluída e, por fim,
// uma correspondência parcial que só é aceita se for única.
func findLine(lines []string, desc string) (int, error) {
	target := normalizeLine(desc)

	var pending, done []int
	for i, l := range lines {
		n := normalizeLine(l)
		if n != target {
			continue
		}
		switch {
		case isPending(n):
			pending = append(pending, i)
		case isDone(n):
			done = append(done, i)
		}
	}
	if len(pending) == 1 {
		return pending[0], nil
	}
	if len(pending) > 1 {
		return -1, ambiguousError(lines, pending)
	}
	if len(done) == 1 {
		return done[0], nil
	}
	if len(done) > 1 {
		return -1, ambiguousError(lines, done)
	}

	needle := strings.ToLower(lineDescription(desc))
	if needle == "" {
		return -1, ErrLineNotFound
	}
	var matches []int
	for i, l := range lines {
		if strings.Contains(strings.ToLower(normalize(l)), needle) {
			matches = append(matches, i)
		}
	}
	switch len(matches) {
	case 0:
		return -1, ErrLineNotFound
	case 1:
		return matches[0], nil
	default:
		return -1, ambiguousError(lines, matches)
	}
}
