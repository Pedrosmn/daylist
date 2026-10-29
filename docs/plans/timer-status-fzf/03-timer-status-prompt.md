# Task 3 — `timer status --prompt`

**Depende de:** task 2 · **Bloqueia:** nada
**Arquivos:** `cmd/timer.go`, `cmd/timer_test.go`

## Objetivo

Ter uma saída curta e estável para colocar no `RPROMPT` do zsh, sem quebrar o prompt quando não há timer.

## Mudança

### `cmd/timer.go`

1. Criar o formatador puro:

```go
// promptText devolve a saída curta para RPROMPT.
func promptText(s *timer.State) string {
    text := fmt.Sprintf("⏱ %dm", timer.RoundedMinutes(s.TotalMs()))
    if s.Paused {
        text += " ⏸"
    }
    return text
}
```

2. Mudar `statusTimer` para receber o modo prompt:

```go
func statusTimer(v *vault.Vault, prompt bool) error {
    s, err := timer.Load(v.StateFile())
    if errors.Is(err, timer.ErrNoTimer) {
        if prompt {
            return nil // prompt limpo, sem mensagem
        }
        fmt.Println("Nenhum timer rodando.")
        return nil
    }
    if err != nil {
        return err
    }
    if prompt {
        fmt.Print(promptText(s)) // sem newline
        return nil
    }
    fmt.Println(statusText(s))
    return nil
}
```

3. Atualizar os chamadores:

- `timerCmd.RunE` → `statusTimer(v, false)`
- `timerStatusCmd.RunE` → `statusTimer(v, statusPrompt)`

4. Adicionar a flag no subcomando `status` (não no comando pai):

```go
var statusPrompt bool

// no init:
timerStatusCmd.Flags().BoolVar(&statusPrompt, "prompt", false,
    "saída curta para RPROMPT (ex.: ⏱ 25m)")
```

### Formatos

| Situação | Saída | Newline |
|---|---|---|
| Rodando | `⏱ 25m` | não |
| Pausado | `⏱ 25m ⏸` | não |
| Sem timer | *(nada)* | — |
| Estado corrompido | erro no stderr, código ≠ 0 | — |

Sem cores, sem ANSI e sem descrição do item (a descrição pode ser longa e quebrar o prompt).

## Testes (`cmd/timer_test.go`)

- `TestPromptTextRunning`: `Start` há 25min → `⏱ 25m`.
- `TestPromptTextPaused`: `Elapsed: 25min`, `Paused: true` → `⏱ 25m ⏸`.
- `TestPromptTextZero`: rodando com 0ms → `⏱ 0m`; pausado com 0ms → `⏱ 0m ⏸`.
- `TestStatusTimerPromptNoState`: capturar stdout e confirmar string vazia + `nil`.
- `TestStatusTimerPromptRunning`: capturar stdout e confirmar `⏱ 25m` (sem newline).
- `TestStatusPromptFlagRegistered`: `timerStatusCmd.Flags().Lookup("prompt") != nil`.

Para capturar stdout nos testes, usar `os.Pipe` e restaurar `os.Stdout` com `t.Cleanup`; os testes não devem rodar em paralelo.

## Critérios de aceite

- `daylist timer status --prompt` imprime exatamente uma linha curta quando há timer.
- Sem timer, a saída é vazia e o comando sai com 0 (o prompt não mostra lixo).
- Nenhum caractere de controle/ANSI.
- `daylist timer status` sem a flag continua com a saída humana da task 2.
- `daylist timer --prompt` não é aceito (flag existe só no `status`).

## Verificação

```bash
gofmt -l cmd internal/timer
go vet ./...
go test ./cmd/... -race
go test ./... -race
go build ./...
```

Manual:

```bash
daylist timer start
daylist timer status --prompt; echo
daylist timer pause
daylist timer status --prompt; echo
daylist timer stop
daylist timer status --prompt; echo   # deve imprimir linha vazia
```

Exemplo de uso no `~/.zshrc`:

```zsh
RPROMPT='$(daylist timer status --prompt)'
```
