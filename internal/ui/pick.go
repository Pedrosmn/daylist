// Package ui tem os helpers de interação com o terminal.
package ui

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// ErrCanceled indica que o usuário cancelou a interação (Esc).
var ErrCanceled = errors.New("cancelado")

// stdin é compartilhado entre as perguntas para não perder linhas já
// bufferizadas quando o input vem de um pipe.
var stdin = bufio.NewReader(os.Stdin)

// Pick deixa o usuário escolher um item. Usa fzf se estiver instalado,
// senão cai num menu numerado. Retorna "" se cancelar.
func Pick(prompt string, items []string) (string, error) {
	if len(items) == 0 {
		return "", nil
	}
	if path, err := exec.LookPath("fzf"); err == nil {
		return pickFzf(path, prompt, items)
	}
	return pickPlain(prompt, items)
}

// Toggle mostra uma lista onde Enter marca/desmarca o item destacado e a
// linha "✅" encerra a etapa. Devolve os índices marcados na ordem original.
// Esc devolve ErrCanceled.
func Toggle(prompt, finishLabel string, items []string) ([]int, error) {
	if len(items) == 0 {
		return nil, nil
	}
	if path, err := exec.LookPath("fzf"); err == nil {
		return toggleFzf(path, prompt, finishLabel, items)
	}
	return togglePlain(prompt, finishLabel, items)
}

// Input lê uma linha digitada pelo usuário.
func Input(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)

	line, err := stdin.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// fzfLayoutArgs mantém a lista compacta e preserva o contexto da etapa anterior.
func fzfLayoutArgs() []string {
	return []string{"--height=70%", "--border", "--info=inline"}
}

// fzfCanceled reconhece os códigos de cancelamento/sem match do fzf:
// 1 = sem match, 130 = interrompido. Outros erros (ex.: 2, opção inválida)
// devem ser propagados.
func fzfCanceled(err error) bool {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return false
	}
	switch exitErr.ExitCode() {
	case 1, 130:
		return true
	default:
		return false
	}
}

func pickFzf(path, prompt string, items []string) (string, error) {
	args := append(fzfLayoutArgs(), "--prompt="+prompt)
	cmd := exec.Command(path, args...)
	cmd.Stdin = strings.NewReader(strings.Join(items, "\n"))
	cmd.Stderr = os.Stderr // a interface do fzf desenha aqui

	out, err := cmd.Output()
	if err != nil {
		if fzfCanceled(err) {
			return "", nil // Esc/Ctrl+C ou sem match
		}
		return "", err
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

// toggleFzf roda uma invocação de fzf por tecla. Cada linha começa com um ID
// oculto (--with-nth=2..), o que evita ambiguidade quando há labels iguais.
func toggleFzf(path, prompt, finishLabel string, items []string) ([]int, error) {
	marked := make([]bool, len(items))
	cursor := 2 // depois da linha "✅"; aprimorado a cada marcação

	for {
		count := 0
		for _, m := range marked {
			if m {
				count++
			}
		}

		rows := make([]string, 0, len(items)+1)
		rows = append(rows, fmt.Sprintf("-1\t%s (%d selecionado%s)", finishLabel, count, plural(count)))
		for i, it := range items {
			mark := "[ ]"
			if marked[i] {
				mark = "[x]"
			}
			rows = append(rows, fmt.Sprintf("%d\t%s %s", i, mark, it))
		}

		pos := cursor
		if pos < 1 || pos > len(rows) {
			pos = 1
		}

		cmd := exec.Command(path, append(fzfLayoutArgs(),
			"--no-multi",
			"--no-sort",
			"--delimiter=\t",
			"--with-nth=2..",
			"--prompt="+prompt,
			"--header=Enter marca/desmarca · Esc cancela",
			fmt.Sprintf("--bind=start:pos(%d)", pos),
		)...)
		cmd.Stdin = strings.NewReader(strings.Join(rows, "\n"))
		cmd.Stderr = os.Stderr // a interface do fzf desenha aqui

		out, err := cmd.Output()
		if err != nil {
			if fzfCanceled(err) {
				return nil, ErrCanceled
			}
			return nil, err
		}

		id, _, _ := strings.Cut(strings.TrimRight(string(out), "\r\n"), "\t")
		if id == "-1" {
			return markedIndices(marked), nil
		}

		n, convErr := strconv.Atoi(id)
		if convErr != nil || n < 0 || n >= len(items) {
			continue
		}
		marked[n] = !marked[n]
		cursor = n + 2
	}
}

func togglePlain(prompt, finishLabel string, items []string) ([]int, error) {
	marked := make([]bool, len(items))

	for {
		count := 0
		for _, m := range marked {
			if m {
				count++
			}
		}

		fmt.Fprintln(os.Stderr, prompt)
		fmt.Fprintf(os.Stderr, "0) %s (%d selecionado%s)\n", finishLabel, count, plural(count))
		for i, it := range items {
			mark := "[ ]"
			if marked[i] {
				mark = "[x]"
			}
			fmt.Fprintf(os.Stderr, "%d) %s %s\n", i+1, mark, it)
		}
		fmt.Fprint(os.Stderr, "#? (Enter conclui) ")

		line, err := stdin.ReadString('\n')
		if err != nil && line == "" {
			return markedIndices(marked), nil // EOF conclui
		}

		line = strings.TrimSpace(line)
		if line == "" {
			return markedIndices(marked), nil
		}

		n, convErr := strconv.Atoi(line)
		if convErr != nil || n < 0 || n > len(items) {
			continue
		}
		if n == 0 {
			return markedIndices(marked), nil
		}
		marked[n-1] = !marked[n-1]
	}
}

func markedIndices(marked []bool) []int {
	var out []int
	for i, m := range marked {
		if m {
			out = append(out, i)
		}
	}
	return out
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func pickPlain(prompt string, items []string) (string, error) {
	fmt.Fprintln(os.Stderr, prompt)
	for i, it := range items {
		fmt.Fprintf(os.Stderr, "%d) %s\n", i+1, it)
	}
	fmt.Fprint(os.Stderr, "#? ")

	line, err := stdin.ReadString('\n')
	if err != nil && line == "" {
		return "", nil
	}
	n, convErr := strconv.Atoi(strings.TrimSpace(line))
	if convErr != nil || n < 1 || n > len(items) {
		return "", nil
	}
	return items[n-1], nil
}
