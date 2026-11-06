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

// ToggleAction é o que o usuário pediu ao sair da lista de uma etapa.
type ToggleAction int

const (
	// ToggleFinish conclui a etapa com os itens marcados.
	ToggleFinish ToggleAction = iota
	// ToggleBack volta para a etapa anterior.
	ToggleBack
	// ToggleCancel descarta o wizard.
	ToggleCancel
)

// ToggleOptions configura uma etapa do wizard.
type ToggleOptions struct {
	Prompt    string
	Items     []string
	Selected  []int
	Locked    []int // itens sempre marcados e que não podem ser desmarcados
	AllowBack bool
	Header    string
}

// ToggleResult devolve os itens marcados e a ação escolhida.
type ToggleResult struct {
	Selected []int
	Action   ToggleAction
}

// Toggle mostra uma lista onde Enter marca/desmarca o item destacado e l
// encerra a etapa. Devolve os índices marcados na ordem original.
// Esc devolve ErrCanceled.
func Toggle(prompt string, items []string) ([]int, error) {
	res, err := ToggleWithOptions(ToggleOptions{
		Prompt: prompt,
		Items:  items,
	})
	if err != nil {
		return nil, err
	}
	if res.Action == ToggleCancel {
		return nil, ErrCanceled
	}
	return res.Selected, nil
}

// ToggleWithOptions é a versão do Toggle usada pelo wizard. Além de marcar e
// desmarcar, permite voltar para a etapa anterior e preservar uma seleção.
func ToggleWithOptions(opts ToggleOptions) (ToggleResult, error) {
	if len(opts.Items) == 0 {
		return ToggleResult{Action: ToggleFinish}, nil
	}
	if path, err := exec.LookPath("fzf"); err == nil {
		return toggleFzf(path, opts)
	}
	return togglePlainOptions(opts)
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

// Confirm pergunta uma confirmação. A opção segura ("Não, continuar") vem
// primeiro; Esc também conta como não.
func Confirm(prompt string) (bool, error) {
	choice, err := Pick(prompt, []string{"Não, continuar", "Sim, descartar"})
	if err != nil {
		return false, err
	}
	return choice == "Sim, descartar", nil
}

// fzfLayoutArgs mantém a lista compacta e preserva o contexto da etapa anterior.
func fzfLayoutArgs() []string {
	// --layout=reverse deixa a lista em ordem natural (primeiro item no topo)
	// e faz as ações down/up corresponderem a j/k.
	return []string{"--height=70%", "--border", "--info=inline", "--layout=reverse"}
}

// fzfNavArgs desliga a busca e adiciona os movimentos no estilo Vim.
func fzfNavArgs() []string {
	return []string{
		"--disabled",
		"--no-sort",
		"--bind=j:down,k:up,g:first,G:last,ctrl-d:half-page-down,ctrl-u:half-page-up",
	}
}

// fzfIgnoreArg neutraliza as teclas imprimíveis que não são atalhos. Sem isso,
// algumas versões do fzf ainda mostram os caracteres digitados na query mesmo
// com --disabled.
func fzfIgnoreArg(used ...rune) string {
	allowed := make(map[rune]bool, len(used))
	for _, r := range used {
		allowed[r] = true
	}

	var keys []string
	add := func(from, to rune) {
		for r := from; r <= to; r++ {
			if !allowed[r] {
				keys = append(keys, string(r)+":ignore")
			}
		}
	}
	add('a', 'z')
	add('A', 'Z')
	add('0', '9')
	for _, r := range "/.:_-+=;'[]?!@#$%^&*()|~\"" {
		if !allowed[r] {
			keys = append(keys, string(r)+":ignore")
		}
	}
	return "--bind=" + strings.Join(keys, ",")
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
	args := append(fzfLayoutArgs(), fzfNavArgs()...)
	args = append(args,
		"--prompt="+prompt,
		"--header=j/k mover · g/G início/fim · Enter escolhe · q cancela",
		"--bind=space:accept,l:accept,q:abort",
		fzfIgnoreArg('j', 'k', 'g', 'G', 'l', 'q'),
	)
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
// A lista contém apenas os itens: h e l são capturados por --expect.
func toggleFzf(path string, opts ToggleOptions) (ToggleResult, error) {
	marked := make([]bool, len(opts.Items))
	for _, idx := range opts.Selected {
		if idx >= 0 && idx < len(marked) {
			marked[idx] = true
		}
	}
	locked := make([]bool, len(opts.Items))
	for _, idx := range opts.Locked {
		if idx >= 0 && idx < len(locked) {
			marked[idx] = true
			locked[idx] = true
		}
	}

	itemRow := func(i int) int { return i + 1 }

	cursor := itemRow(0)
	for i, on := range marked {
		if on {
			cursor = itemRow(i)
			break
		}
	}

	expect := "l"
	used := []rune{'j', 'k', 'g', 'G', 'l', 'q'}
	if opts.AllowBack {
		expect = "l,h"
		used = append(used, 'h')
	}

	for {
		count := 0
		for _, m := range marked {
			if m {
				count++
			}
		}

		rows := make([]string, 0, len(opts.Items))
		for i, it := range opts.Items {
			mark := "[ ]"
			if marked[i] {
				mark = "[x]"
			}
			rows = append(rows, fmt.Sprintf("%d\t%s %s", i, mark, it))
		}

		pos := cursor
		if pos < 1 || pos > len(rows) {
			pos = itemRow(0)
		}

		args := append(fzfLayoutArgs(), fzfNavArgs()...)
		args = append(args,
			"--no-multi",
			"--delimiter=\t",
			"--with-nth=2..",
			"--prompt="+opts.Prompt,
			"--header="+toggleHeader(opts.Header, opts.AllowBack, count, len(opts.Items)),
			"--bind=space:accept,q:abort",
			"--expect="+expect,
			fzfIgnoreArg(used...),
			fmt.Sprintf("--bind=start:pos(%d)", pos),
		)
		cmd := exec.Command(path, args...)
		cmd.Stdin = strings.NewReader(strings.Join(rows, "\n"))
		cmd.Stderr = os.Stderr // a interface do fzf desenha aqui

		out, err := cmd.Output()
		if err != nil {
			if fzfCanceled(err) {
				return ToggleResult{Selected: markedIndices(marked), Action: ToggleCancel}, nil
			}
			return ToggleResult{}, err
		}

		// --expect imprime a tecla escolhida na primeira linha e o item
		// destacado na segunda (a tecla fica vazia quando é Enter/Espaço).
		lines := strings.Split(strings.TrimRight(string(out), "\r\n"), "\n")
		key := ""
		if len(lines) > 0 {
			key = lines[0]
		}
		line := ""
		if len(lines) > 1 {
			line = lines[len(lines)-1]
		}

		switch key {
		case "l":
			return ToggleResult{Selected: markedIndices(marked), Action: ToggleFinish}, nil
		case "h":
			if opts.AllowBack {
				return ToggleResult{Selected: markedIndices(marked), Action: ToggleBack}, nil
			}
			continue
		}

		if line == "" {
			return ToggleResult{Selected: markedIndices(marked), Action: ToggleCancel}, nil
		}
		id, _, _ := strings.Cut(line, "\t")
		n, convErr := strconv.Atoi(id)
		if convErr != nil || n < 0 || n >= len(opts.Items) {
			continue
		}
		if locked[n] {
			cursor = itemRow(n)
			continue
		}
		marked[n] = !marked[n]
		cursor = itemRow(n)
	}
}

func toggleHeader(custom string, allowBack bool, count, total int) string {
	help := "j/k mover · g/G início/fim · Enter marca · l conclui · q cancela"
	if allowBack {
		help = "j/k mover · g/G início/fim · Enter marca · h volta · l conclui · q cancela"
	}
	if custom == "" {
		return help
	}
	return fmt.Sprintf("%s · %d marcados · %d no total\n%s", custom, count, total, help)
}

// togglePlain é o fallback sem fzf. Ele continua numerado, mas aceita os
// comandos h, l e q para voltar, concluir e cancelar.
func togglePlain(prompt string, items []string) ([]int, error) {
	res, err := togglePlainOptions(ToggleOptions{
		Prompt: prompt,
		Items:  items,
	})
	if err != nil {
		return nil, err
	}
	if res.Action == ToggleCancel {
		return nil, ErrCanceled
	}
	return res.Selected, nil
}

func togglePlainOptions(opts ToggleOptions) (ToggleResult, error) {
	marked := make([]bool, len(opts.Items))
	for _, idx := range opts.Selected {
		if idx >= 0 && idx < len(marked) {
			marked[idx] = true
		}
	}
	locked := make([]bool, len(opts.Items))
	for _, idx := range opts.Locked {
		if idx >= 0 && idx < len(locked) {
			marked[idx] = true
			locked[idx] = true
		}
	}

	for {
		count := 0
		for _, m := range marked {
			if m {
				count++
			}
		}

		fmt.Fprintln(os.Stderr, opts.Prompt)
		if opts.Header != "" {
			fmt.Fprintf(os.Stderr, "%s · %d marcados · %d no total\n", opts.Header, count, len(opts.Items))
		}
		for i, it := range opts.Items {
			mark := "[ ]"
			if marked[i] {
				mark = "[x]"
			}
			fmt.Fprintf(os.Stderr, "%d) %s %s\n", i+1, mark, it)
		}
		hint := "Enter/l conclui, q cancela"
		if opts.AllowBack {
			hint = "Enter/l conclui, h volta, q cancela"
		}
		fmt.Fprintf(os.Stderr, "#? (%s) ", hint)

		line, err := stdin.ReadString('\n')
		if err != nil && line == "" {
			return ToggleResult{Selected: markedIndices(marked), Action: ToggleFinish}, nil // EOF conclui
		}

		line = strings.TrimSpace(line)
		switch line {
		case "", "l":
			return ToggleResult{Selected: markedIndices(marked), Action: ToggleFinish}, nil
		case "q":
			return ToggleResult{Selected: markedIndices(marked), Action: ToggleCancel}, nil
		case "h":
			if opts.AllowBack {
				return ToggleResult{Selected: markedIndices(marked), Action: ToggleBack}, nil
			}
			continue
		}

		n, convErr := strconv.Atoi(line)
		if convErr != nil || n < 1 || n > len(opts.Items) {
			continue
		}
		if locked[n-1] {
			continue
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
