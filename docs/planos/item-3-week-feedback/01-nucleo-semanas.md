# Task 1 — Núcleo de semanas e parser de hábitos

**Depende de:** nada · **Bloqueia:** tasks 2–5
**Arquivos:** `internal/week/week.go`, `internal/week/habits.go`, `internal/week/week_test.go`, `internal/week/habits_test.go`

## Objetivo

Centralizar aritmética de semanas ISO e um parser tolerante de linhas de hábito, compartilhado por weeklist e feedback.

## Mudança

Criar o pacote `internal/week` com:

```go
// ParseWeek valida "YYYY-Www" e devolve a segunda-feira da semana, 00:00 local.
func ParseWeek(s string) (time.Time, error)

// WeekString devolve a semana ISO de t no formato YYYY-Www.
func WeekString(t time.Time) string

// PreviousWeek devolve a segunda-feira da semana anterior.
func PreviousWeek(weekStart time.Time) time.Time

// WeekDays devolve as sete datas (segunda a domingo) da semana.
func WeekDays(weekStart time.Time) []time.Time
```

Regras:

- `ParseWeek` aceita exatamente `YYYY-Www` (4 dígitos, hífen, `W`, 2 dígitos).
- Semana válida é validada pelo próprio `ISOWeek`: `2021-W53` deve falhar; `2020-W53` deve funcionar.
- A semana começa na segunda-feira, no fuso `time.Local`.
- `PreviousWeek` usa `AddDate(0, 0, -7)`, não decremento numérico da semana.
- `WeekDays` devolve exatamente 7 entradas, mesmo atravessando ano/bissexto.

Implementação sugerida para a segunda-feira da semana 1:

```go
jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.Local)
offset := (int(jan4.Weekday()) + 6) % 7 // segunda = 0
monday := jan4.AddDate(0, 0, -offset)
start := monday.AddDate(0, 0, (week-1)*7)
```

Depois confirmar com `start.ISOWeek()`; se não bater, retornar erro de semana inválida.

### Parser tolerante de hábitos (`internal/week/habits.go`)

```go
type HabitLine struct {
    Checked bool
    Body    string // texto depois do checkbox, preservado para a weeklist
    Desc    string // descrição sem tempo/flag, usada pelo feedback
}

// ParseHabitLine reconhece os mesmos formatos do parser da daylist, mas
// também aceita itens concluídos ([x]).
func ParseHabitLine(line string) (HabitLine, bool)
```

Formatos aceitos, nesta ordem:

| Formato | Exemplo | `Desc` |
|---|---|---|
| completo | `5 min (U) - AWS` | `AWS` |
| só tempo | `5 min - AWS` | `AWS` |
| só prioridade | `(U) - AWS` | `AWS` |
| livre | `AWS` | `AWS` |

Regras:

- O checkbox aceita espaço, `x` ou `X`; `Checked` reflete o estado.
- `Body` preserva tempo, prioridade e descrição exatamente como estão.
- Regexes equivalentes às de `internal/daylist`, aplicadas sobre o texto depois do checkbox:
  - completo: `^(.+?) \((.*?)\) - (.*)$`
  - só tempo: `(?i)^(x?st|\d+(?:/\d+)?\s*min) - (.*)$`
  - só prioridade: `^\(([IUiu]*)\) - (.*)$`
  - fallback: `Body` inteiro vira `Desc`.
- Linhas fora do formato de checklist (`- [ ] ...`) são rejeitadas.

## Testes (`internal/week/week_test.go`)

- `TestParseWeek`: `2026-W01` → `2025-12-29`.
- `TestParseWeekPrevious`: `2025-W52` → `2025-12-22`.
- `TestParseWeek53`: `2020-W53` válida; `2021-W53` inválida.
- `TestParseWeekInvalid`: `""`, `2026-01`, `2026-W1`, `2026-W00`, `2026-W54`.
- `TestWeekString`: `2025-12-29` → `2026-W01` e `2026-11-02` → `2026-W45`.
- `TestPreviousWeek`: `2026-01-05` → `2025-12-29`.
- `TestWeekDays`: `2025-12-29` até `2026-01-04`, na ordem e sem duplicatas.

## Testes (`internal/week/habits_test.go`)

Tabela cobrindo:

- `- [ ] 5 min (U) - AWS` → `Checked=false`, `Body="5 min (U) - AWS"`, `Desc="AWS"`.
- `- [x] 5 min (U) - AWS` → `Checked=true`, mesmo `Body`/`Desc`.
- `- [ ] 5 min - AWS` → `Desc="AWS"`.
- `- [ ] (U) - AWS` → `Desc="AWS"`.
- `- [ ] AWS` → `Desc="AWS"`.
- `- [ ] xST () - dormir` → `Desc="dormir"`.
- Linhas em branco, `## Categoria`, texto fora de checklist e CRLF.

## Critérios de aceite

- Toda semana retornada por `ParseWeek` tem `ISOWeek` igual à solicitada.
- Virada de ano e semana 53 são tratadas corretamente.
- `WeekDays` atravessa ano/mês sem erro.
- `ParseHabitLine` reconhece os quatro formatos e preserva `Body`.
- Nenhuma dependência de `vault` no pacote `week`.

## Verificação

```bash
gofmt -l internal/week
go vet ./internal/week/...
go test ./internal/week/... -race -cover
go build ./...
```
