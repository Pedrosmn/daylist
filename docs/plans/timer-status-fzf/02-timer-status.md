# Task 2 — `timer status` humano e `daylist timer` sem subcomando

**Depende de:** task 1 · **Bloqueia:** task 3
**Arquivos:** `cmd/timer.go`, `cmd/timer_test.go`

## Problema

Hoje só é possível descobrir o timer atual indiretamente: `timer start` diz “Já tem um timer rodando”, `timer pause` mostra o acumulado e `timer stop` encerra. Não existe consulta somente-leitura, e `daylist timer` sem subcomando imprime a ajuda em vez do status.

## Mudança

### `cmd/timer.go`

1. Adicionar um formatador puro:

```go
// statusText devolve a linha humana do timer atual.
func statusText(s *timer.State) string {
    mins := timer.RoundedMinutes(s.TotalMs())
    if s.Paused {
        return fmt.Sprintf("Timer pausado: %s — %dmin acumulados%s",
            s.Desc, mins, startedAt(s))
    }
    return fmt.Sprintf("Timer rodando: %s — %dmin%s", s.Desc, mins, startedAt(s))
}

// startedAt devolve " (iniciado às HH:MM)" ou "" quando Started não existe.
func startedAt(s *timer.State) string {
    if s.Started <= 0 {
        return ""
    }
    return fmt.Sprintf(" (iniciado às %s)", time.UnixMilli(s.Started).Format("15:04"))
}
```

2. Adicionar o fluxo somente-leitura:

```go
func statusTimer(v *vault.Vault) error {
    s, err := timer.Load(v.StateFile())
    if errors.Is(err, timer.ErrNoTimer) {
        fmt.Println("Nenhum timer rodando.")
        return nil
    }
    if err != nil {
        return err
    }
    fmt.Println(statusText(s))
    return nil
}
```

3. Criar o subcomando:

```go
var timerStatusCmd = &cobra.Command{
    Use:   "status",
    Short: "Mostra o timer atual",
    Args:  cobra.NoArgs,
    RunE: func(cmd *cobra.Command, args []string) error {
        return statusTimer(v)
    },
}
```

4. Dar `RunE` ao comando pai para que `daylist timer` mostre o status:

```go
var timerCmd = &cobra.Command{
    Use:     "timer",
    Aliases: []string{"t"},
    Short:   "Controla o timer de um item da daylist",
    Args:    cobra.NoArgs,
    RunE: func(cmd *cobra.Command, args []string) error {
        return statusTimer(v)
    },
}
```

`--help` continua funcionando e mostrando os subcomandos. Registrar `timerStatusCmd` no `init`.

### Formatos

- Rodando: `Timer rodando: escrever testes — 25min (iniciado às 09:12)`
- Pausado: `Timer pausado: escrever testes — 25min acumulados (iniciado às 09:12)`
- Sem timer: `Nenhum timer rodando.`
- Estado sem `Started`: omitir o trecho `(iniciado às ...)`.

O status **não** altera o arquivo de estado e **não** abre o Obsidian.

## Testes (`cmd/timer_test.go`)

- `TestStatusTextRunning`: `Paused: false`, `Start` há 25min, `Started` fixo; conferir a linha esperada.
- `TestStatusTextPaused`: `Paused: true`, `Elapsed: 25min`; conferir “acumulados”.
- `TestStatusTextWithoutStarted`: `Started: 0`; conferir que não aparece “iniciado às”.
- `TestStatusTimerNoState`: vault temporário sem estado; `statusTimer` devolve `nil`.
- `TestStatusTimerCorruptState`: gravar JSON inválido em `v.StateFile()`; `statusTimer` devolve erro de estado inválido.
- `TestTimerParentCommandShowsStatus` (smoke): definir `v = newTestVault(t)` e chamar `timerCmd.RunE(timerCmd, nil)` com um estado válido; esperar `nil`.

## Critérios de aceite

- `daylist timer` e `daylist timer status` mostram o mesmo status.
- Funciona sem TTY e sem interação.
- Não modifica nem remove o estado do timer.
- Sem timer, imprime `Nenhum timer rodando.` e sai com código 0.
- Estado corrompido produz erro, não uma mensagem falsa de “nenhum timer”.

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
daylist timer start   # iniciar um item
daylist timer         # deve mostrar o status
daylist timer status  # idem
daylist timer pause
daylist timer status  # deve indicar pausado e minutos acumulados
```
