# Task 4 — Gerador puro de feedback

**Depende de:** task 1 · **Bloqueia:** task 5
**Arquivos:** `internal/week/feedback.go`, `internal/week/feedback_test.go`

## Objetivo

Reproduzir `scripts/feedback_script.js` como lógica pura, sem disco e sem Obsidian, usando o parser tolerante compartilhado para aceitar também formatos parciais.

## Mudança

### Tipos e funções

```go
type Day struct {
    Date    time.Time
    Content string
}

type HabitStat struct {
    Category string
    Desc     string
    Checked  int
    Total    int
}

// StripTimerSuffix remove " ⏱ ..." da descrição.
func StripTimerSuffix(desc string) string

// BuildFeedback monta o feedback a partir da weeklist e dos conteúdos dos
// dias da semana (na ordem segunda → domingo).
func BuildFeedback(week, weeklist string, days []Day, createdAt time.Time) string
```

### Algoritmo

1. Percorrer a weeklist:
   - entrar em `# Hábitos` e sair no próximo `# `;
   - `## Categoria` entra em `categoryOrder` (sem duplicar);
   - usar `HabitLine, ok := ParseHabitLine(line)`; se `ok`, criar `HabitStat{Checked: 0, Total: 0}` com `Desc = StripTimerSuffix(HabitLine.Desc)`.
2. Percorrer os dias, na ordem recebida:
   - repetir o rastreamento de `# Hábitos`/`## Categoria`;
   - para cada linha de hábito:
     - `h, ok := ParseHabitLine(line)`; ignorar se `!ok`;
     - `desc := StripTimerSuffix(h.Desc)`;
     - obter/criar a estatística pela chave `categoria + "||" + desc`; se a categoria é nova, acrescentar ao fim de `categoryOrder`;
     - `Total++`;
     - se `h.Checked`, `Checked++`.
3. Ignorar `# Tasks`/`# Extra`.
4. Percentual: `0` quando `Total == 0`; senão `int(math.Round(float64(Checked)/float64(Total)*100))` (mesmo arredondamento do `Math.round` para positivos).
5. Montar o texto exatamente no formato do JS:

```text
# Feedback - Semana 2026-W45
Criado em: 2026-11-08 18:30

## Saúde

## Estudos

---
# Resumo
Hábitos concluídos na semana: 3/5 (60%)

## Saúde
correr 2/3 - (67%)
dormir 1/2 - (50%)
```

Sem nenhum hábito:

```text
Nenhum hábito encontrado na weeklist.
```

Detalhe de paridade: as categorias do topo vêm da weeklist e de novos hábitos encontrados nos dias; no resumo, só aparecem categorias que têm estatísticas.

## Testes (`internal/week/feedback_test.go`)

- `TestStripTimerSuffix`: `correr ⏱ 25min (09:00–09:25)` → `correr`; descrição sem sufixo fica igual.
- `TestBuildFeedbackNoHabits`: weeklist sem hábitos → mensagem “Nenhum hábito encontrado”; headings preservados.
- `TestBuildFeedbackWeeklistOnly`: hábito só na weeklist → `0/0 - (0%)`.
- `TestBuildFeedbackCounts`: dias com `[ ]` e `[x]`, incluindo tarefas que devem ser ignoradas.
- `TestBuildFeedbackPartialFormats`: `5 min - revisar`, `(I) - planejar` e `ler` também são contabilizados.
- `TestBuildFeedbackTimerSuffix`: linha da daylist com `⏱` conta na mesma estatística da versão sem sufixo.
- `TestBuildFeedbackNewCategory`: categoria que só existe na daylist aparece no fim.
- `TestBuildFeedbackPercentages`: 2/3 → 67%, 1/3 → 33%, total geral correto.
- `TestBuildFeedbackCRLF`: mesmo resultado de LF.
- `TestBuildFeedbackCreatedAt`: data/hora formatada como `YYYY-MM-DD HH:mm`.

## Critérios de aceite

- Estrutura e formato iguais aos do script; diferença intencional: aceitar também formatos parciais.
- `[x]` conta como concluído; `[ ]` soma apenas no total.
- O sufixo `⏱` não cria hábitos duplicados.
- Ordem de categorias estável: weeklist primeiro, depois descobertas nos dias.
- Percentuais usam arredondamento para o inteiro mais próximo.

## Verificação

```bash
gofmt -l internal/week
go vet ./internal/week/...
go test ./internal/week/... -race -cover
```
