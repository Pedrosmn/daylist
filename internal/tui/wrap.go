package tui

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// wrapText quebra o texto pela largura em colunas, preservando palavras e
// quebrando as que não cabem. Linhas já existentes no texto são mantidas e
// quebradas separadamente.
func wrapText(text string, width int) []string {
	if width < 1 {
		width = 1
	}

	lines := make([]string, 0, 1)
	for _, line := range strings.Split(text, "\n") {
		wrapped := ansi.Wrap(line, width, " ")
		lines = append(lines, strings.Split(wrapped, "\n")...)
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

// visualHeight devolve quantas linhas visuais o textarea usará para o texto na
// largura informada. É o mesmo algoritmo de wrap do textarea do Bubbles, mas
// reduzido à contagem de linhas.
func visualHeight(text string, width int) int {
	if width < 1 {
		width = 1
	}

	total := 0
	for _, line := range strings.Split(text, "\n") {
		total += wrappedLineCount([]rune(line), width)
	}
	if total < 1 {
		return 1
	}
	return total
}

func wrappedLineCount(runes []rune, width int) int {
	var (
		lines  = [][]rune{{}}
		word   = []rune{}
		row    int
		spaces int
	)

	for _, r := range runes {
		if unicode.IsSpace(r) {
			spaces++
		} else {
			word = append(word, r)
		}

		if spaces > 0 {
			if lipgloss.Width(string(lines[row]))+lipgloss.Width(string(word))+spaces > width {
				row++
				lines = append(lines, []rune{})
				lines[row] = append(lines[row], word...)
				lines[row] = append(lines[row], repeatSpaces(spaces)...)
				spaces = 0
				word = nil
			} else {
				lines[row] = append(lines[row], word...)
				lines[row] = append(lines[row], repeatSpaces(spaces)...)
				spaces = 0
				word = nil
			}
		} else if len(word) > 0 {
			// Se o último caractere for largo, ele pode não caber e precisa
			// descer para a próxima linha.
			lastCharLen := lipgloss.Width(string(word[len(word)-1]))
			if lipgloss.Width(string(word))+lastCharLen > width {
				if len(lines[row]) > 0 {
					row++
					lines = append(lines, []rune{})
				}
				lines[row] = append(lines[row], word...)
				word = nil
			}
		}
	}

	if lipgloss.Width(string(lines[row]))+lipgloss.Width(string(word))+spaces >= width {
		lines = append(lines, []rune{})
		lines[row+1] = append(lines[row+1], word...)
		spaces++
		lines[row+1] = append(lines[row+1], repeatSpaces(spaces)...)
	} else {
		lines[row] = append(lines[row], word...)
		spaces++
		lines[row] = append(lines[row], repeatSpaces(spaces)...)
	}

	return len(lines)
}

func repeatSpaces(n int) []rune {
	if n < 0 {
		n = 0
	}
	return []rune(strings.Repeat(" ", n))
}
