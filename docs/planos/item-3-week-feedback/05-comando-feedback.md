# Task 5 — Comando `feedback` em Go

**Depende de:** task 4 · **Bloqueia:** task 6
**Arquivos:** `cmd/feedback.go`, `cmd/feedback_test.go`

## Objetivo

Substituir o QuickAdd de `feedback` pela geração local do markdown.

## CLI

```text
daylist feedback [--week YYYY-Www] [--dry-run|--print-path]
```

- Sem flags: semana ISO atual.
- `--week`: semana explícita, validada por `week.ParseWeek`.
- `--dry-run`: mostra o conteúdo, não grava.
- `--print-path`: mostra apenas o caminho resolvido, não grava.
- `--dry-run`/`--print-path` são mutuamente exclusivos.

Atualizar `Short`/`Long` para remover a menção a QuickAdd.

## Mudança

```go
type feedbackOptions struct {
    Week      string
    DryRun    bool
    PrintPath bool
}

var feedbackOpts feedbackOptions
```

### Resolução da semana

```go
func feedbackWeek(now time.Time, opts feedbackOptions) (string, error) {
    if opts.Week == "" {
        return vault.ISOWeek(now), nil
    }
    if _, err := week.ParseWeek(opts.Week); err != nil {
        return "", fmt.Errorf("semana inválida %q (use YYYY-Www)", opts.Week)
    }
    return opts.Week, nil
}
```

### Fluxo testável

```go
func runFeedback(v *vault.Vault, opts feedbackOptions, now time.Time) error
```

1. `targetWeek, err := feedbackWeek(now, opts)`.
2. `targetPath := v.FeedbackFile(targetWeek)`.
3. Se `PrintPath`: imprimir o caminho e retornar.
4. Se o feedback já existe: imprimir `Feedback já existe: <path>` e retornar `nil`.
5. Ler `v.WeeklistFile(targetWeek)`:
   - `os.ErrNotExist` → erro `weeklist de <semana> não encontrada: <path>`;
   - outros erros → retornar.
6. `start, _ := week.ParseWeek(targetWeek)`; para cada dia de `week.WeekDays(start)`:
   - tentar `os.ReadFile(v.DaylistFile(dia))`;
   - `os.ErrNotExist` → pular;
   - outros erros → retornar;
   - acrescentar `week.Day{Date: dia, Content: string(data)}`.
7. `content := week.BuildFeedback(targetWeek, string(weeklist), days, now)`.
8. `DryRun` → `fmt.Print(content)` e retornar.
9. `daylist.Write(targetPath, content)`.
10. Imprimir `Feedback de <targetWeek> criado: <targetPath>`.

O comando não chama `QuickAdd` nem grava arquivo de alvo. A limpeza dos alvos legados fica na task 6.

## Testes (`cmd/feedback_test.go`)

Usar `newTestVault(t)`, `now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.Local)` (`2026-W44`) e `captureStdout`.

- `TestFeedbackWeek`: sem flag → `2026-W44`; `Week: "2026-W45"` → aceita; `"2026-W1"`/`"abc"` → erro.
- `TestRunFeedbackDefaultsToCurrent`: criar weeklist `2026-W44` e duas daylists; conferir `feedback.md` com contagens.
- `TestRunFeedbackExplicitWeek`: criar `2026-W45` e rodar `--week 2026-W45`.
- `TestRunFeedbackMissingWeeklist`: erro cita a semana e o caminho.
- `TestRunFeedbackExisting`: criar feedback; rodar; conferir mensagem “já existe” e conteúdo inalterado.
- `TestRunFeedbackDryRun`: conferir conteúdo no stdout e ausência do arquivo.
- `TestRunFeedbackPrintPath`: conferir somente o caminho no stdout e nenhum arquivo criado.
- `TestRunFeedbackSkipsMissingDays`: só uma daylist existe; estatísticas refletem apenas ela.
- `TestFeedbackFlagsRegistered`: `--week`, `--dry-run`, `--print-path` existem.

## Critérios de aceite

- `daylist feedback` não abre o Obsidian nem o QuickAdd.
- Semana padrão e `--week` corretos; formato inválido falha com mensagem clara.
- Daylists ausentes são puladas sem erro.
- Weeklist ausente é erro, não sucesso silencioso.
- Feedback existente não é sobrescrito.
- `--dry-run`/`--print-path` não escrevem.
- Escrita atômica via `daylist.Write`.

## Verificação

```bash
gofmt -l cmd internal/week
go vet ./...
go test ./cmd/... -race
go test ./... -race
go build ./...
```

Manual:

```bash
DAYLIST_VAULT_PATH=$(mktemp -d) daylist feedback --dry-run
DAYLIST_VAULT_PATH=$(mktemp -d) daylist feedback --print-path
```
