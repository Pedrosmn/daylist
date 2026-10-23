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

func pickPlain(prompt string, items []string) (string, error) {
	fmt.Fprintln(os.Stderr, prompt)
	for i, it := range items {
		fmt.Fprintf(os.Stderr, "%d) %s\n", i+1, it)
	}
	fmt.Fprint(os.Stderr, "#? ")

	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return "", nil
	}
	n, convErr := strconv.Atoi(strings.TrimSpace(line))
	if convErr != nil || n < 1 || n > len(items) {
		return "", nil
	}
	return items[n-1], nil
}
