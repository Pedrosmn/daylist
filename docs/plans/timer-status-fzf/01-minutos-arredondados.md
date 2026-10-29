# Task 1 — Núcleo de minutos arredondados

**Depende de:** nada · **Bloqueia:** task 2
**Arquivos:** `internal/timer/timer.go`, `internal/timer/timer_test.go`

## Problema

A regra de arredondamento está embutida em `State.Finish`:

```go
ms := s.TotalMs()
mins := (ms + 30_000) / 60_000
if mins == 0 && ms > 0 {
    mins = 1
}
```

O futuro `timer status` precisa exibir exatamente os mesmos minutos. Se cada um tiver a sua própria conta, o status pode mostrar `1min` e o `Finish` gravar `2min` (ou vice-versa).

## Mudança

Em `internal/timer/timer.go`, extrair a regra para uma função exportada e usá-la no `Finish`:

```go
// RoundedMinutes converte milissegundos em minutos arredondados para o mais
// próximo. Sessões com tempo > 0 nunca resultam em 0.
func RoundedMinutes(ms int64) int64 {
    if ms <= 0 {
        return 0
    }
    mins := (ms + 30_000) / 60_000
    if mins == 0 {
        mins = 1
    }
    return mins
}
```

Em `Finish`:

```go
mins := RoundedMinutes(s.TotalMs())
```

Não alterar `TotalMs`, o formato da linha `⏱ Nmin (HH:MM–HH:MM)` nem a escrita atômica.

## Testes (`internal/timer/timer_test.go`)

Adicionar `TestRoundedMinutes` com tabela:

| Entrada | Esperado |
|---|---|
| `-1` | `0` |
| `0` | `0` |
| `1ms` | `1` |
| `29s` | `1` |
| `30s` | `1` |
| `59s` | `1` |
| `89s` | `1` |
| `90s` | `2` |
| `30min` | `30` |
| `1h30m40s` | `91` |

Manter `TestFinishRounding` como está; ele passa a validar indiretamente a integração com `Finish`.

## Critérios de aceite

- `Finish` continua com o mesmo comportamento e os testes existentes passam.
- A regra de arredondamento existe em um único lugar.
- `RoundedMinutes` devolve `0` para zero/negativo e nunca `0` para tempo positivo.

## Verificação

```bash
gofmt -l internal/timer
go vet ./internal/timer/...
go test ./internal/timer/... -race -cover
```
