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

// PickMulti deixa o usuário escolher vários itens. Com fzf, Tab marca e
// Enter confirma; sem fzf, aceita índices separados por espaço/vírgula
// (ex.: "1 3 5"). Retorna nil se cancelar.
func PickMulti(prompt string, items []string) ([]string, error) {
	if len(items) == 0 {
		return nil, nil
	}
	if path, err := exec.LookPath("fzf"); err == nil {
		return pickFzfMulti(path, prompt, items)
	}
	return pickPlainMulti(prompt, items)
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

func pickFzf(path, prompt string, items []string) (string, error) {
	cmd := exec.Command(path, "--prompt="+prompt)
	cmd.Stdin = strings.NewReader(strings.Join(items, "\n"))
	cmd.Stderr = os.Stderr // a interface do fzf desenha aqui

	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", nil // Esc/Ctrl+C ou sem match
		}
		return "", err
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

func pickFzfMulti(path, prompt string, items []string) ([]string, error) {
	cmd := exec.Command(path,
		"--multi",
		"--no-sort",
		"--header=Tab marca vários, Enter confirma",
		"--prompt="+prompt,
	)
	cmd.Stdin = strings.NewReader(strings.Join(items, "\n"))
	cmd.Stderr = os.Stderr // a interface do fzf desenha aqui

	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, nil // Esc/Ctrl+C
		}
		return nil, err
	}

	var chosen []string
	for _, line := range strings.Split(strings.TrimRight(string(out), "\r\n"), "\n") {
		if line != "" {
			chosen = append(chosen, line)
		}
	}
	return chosen, nil
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

func pickPlainMulti(prompt string, items []string) ([]string, error) {
	fmt.Fprintln(os.Stderr, prompt)
	for i, it := range items {
		fmt.Fprintf(os.Stderr, "%d) %s\n", i+1, it)
	}
	fmt.Fprint(os.Stderr, "#? (ex.: 1 3 5) ")

	line, err := stdin.ReadString('\n')
	if err != nil && line == "" {
		return nil, nil
	}

	// TrimSpace antes de separar: o '\n' final não é separador para FieldsFunc.
	fields := strings.FieldsFunc(strings.TrimSpace(line), func(r rune) bool {
		return r == ' ' || r == ',' || r == '\t'
	})

	var chosen []string
	seen := map[int]bool{}
	for _, f := range fields {
		n, convErr := strconv.Atoi(f)
		if convErr != nil || n < 1 || n > len(items) || seen[n] {
			continue
		}
		seen[n] = true
		chosen = append(chosen, items[n-1])
	}
	return chosen, nil
}
