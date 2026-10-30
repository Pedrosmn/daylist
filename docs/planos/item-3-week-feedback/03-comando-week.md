# Task 3 — Comando `week` em Go

**Depende de:** task 2 · **Bloqueia:** task 6
**Arquivos:** `cmd/week.go`, `cmd/week_test.go`

## Objetivo

Substituir o QuickAdd de `week` pela geração local da weeklist.

## CLI

```text
daylist week [--current|--next] [--dry-run|--print-path]
```

- Sem flags: próxima semana.
- `--current`: semana atual (recuperação).
- `--next`: próxima semana (explícito).
- `--dry-run`: mostra o conteúdo, não grava.
- `--print-path`: mostra apenas o caminho resolvido, não grava.
- `--current`/`--next` e `--dry-run`/`--print-path` são mutuamente exclusivos.

Atualizar `Short`/`Long` para remover a menção a QuickAdd.

## Mudança

### Estruturas e flags

```go
type weekOptions struct {
    Current   bool
    Next      bool
    DryRun    bool
    PrintPath bool
}

var weekOpts weekOptions
```

Registrar as flags no `init` e usar `MarkFlagsMutuallyExclusive`.

### Resolução da semana

```go
func weekTarget(now time.Time, opts weekOptions) string {
    if opts.Current {
        return vault.ISOWeek(now)
    }
    return vault.ISOWeek(now.AddDate(0, 0, 7))
}
```

### Fluxo testável

```go
func runWeek(v *vault.Vault, opts weekOptions, now time.Time) error
```

1. `targetWeek := weekTarget(now, opts)`.
2. `targetPath := v.WeeklistFile(targetWeek)`.
3. Se `PrintPath`: imprimir `targetPath` e retornar.
4. Se o arquivo já existe: imprimir `Weeklist já existe: <path>` e retornar `nil`.
5. Calcular a semana de origem:

   ```go
   start, err := week.ParseWeek(targetWeek)
   sourceWeek := week.WeekString(week.PreviousWeek(start))
   sourcePath := v.WeeklistFile(sourceWeek)
   ```

6. Ler a origem. Se `os.ErrNotExist`, avisar em stderr (`Weeklist de <semana> não encontrada; criando sem hábitos.`) e usar `source := ""`. Outros erros: retornar.
7. `content := week.BuildWeeklist(source)`.
8. Se `DryRun`: `fmt.Print(content)` e retornar.
9. `daylist.Write(targetPath, content)`.
10. Imprimir `Weeklist de <targetWeek> criada: <targetPath>`.

O comando não chama `QuickAdd` nem grava arquivo de alvo. A limpeza dos alvos legados fica na task 6.

## Testes (`cmd/week_test.go`)

Usar `v := newTestVault(t)`, `now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.Local)` (quarta da `2026-W44`) e reutilizar `captureStdout` de `cmd/timer_test.go`.

- `TestWeekTarget`: opts vazio → `2026-W45`; `Current` → `2026-W44`; `Next` → `2026-W45`.
- `TestRunWeekDefaultsToNext`: criar weeklist `2026-W44` com hábitos; rodar; conferir `2026-W45/weeklist.md` com hábitos pendentes e tempo/prioridade preservados.
- `TestRunWeekPartialFormats`: origem com `5 min - revisar` e `ler`; conferir cópia dos dois.
- `TestRunWeekCurrent`: criar `2026-W43`; rodar com `Current`; conferir `2026-W44`.
- `TestRunWeekDryRun`: criar origem; rodar com `DryRun`; conferir conteúdo no stdout e ausência do arquivo alvo.
- `TestRunWeekPrintPath`: conferir somente o caminho no stdout e nenhum arquivo criado.
- `TestRunWeekExisting`: criar o alvo; rodar; conferir mensagem “já existe” e conteúdo inalterado.
- `TestRunWeekMissingSource`: sem origem; rodar; conferir weeklist criada só com `# Tasks`/`# Hábitos`.
- `TestWeekFlagsRegistered`: `--current`, `--next`, `--dry-run`, `--print-path` existem.

## Critérios de aceite

- `daylist week` não abre o Obsidian nem o QuickAdd.
- Default e `--current`/`--next` escolhem a semana correta.
- `--dry-run` e `--print-path` não escrevem nada.
- Weeklist existente não é sobrescrita.
- Origem ausente gera weeklist vazia com aviso, como no script.
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
DAYLIST_VAULT_PATH=$(mktemp -d) daylist week --dry-run
DAYLIST_VAULT_PATH=$(mktemp -d) daylist week --print-path
```
