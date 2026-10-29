# Task 2 — Minutos arredondados em sessões curtas

**Roadmap:** correção 4 · **Depende de:** nada · **Bloqueia:** task 3

## Problema

`internal/timer/timer.go:132` (`State.Finish`) trunca o tempo: `mins := s.TotalMs() / 60000`. Sessões entre 30s e 59s, e qualquer fração perdida, aparecem como `0min` (ou um minuto a menos) na daylist.

## Mudança

Em `internal/timer/timer.go` (`Finish`):

```go
ms := s.TotalMs()
mins := (ms + 30_000) / 60_000
if mins == 0 && ms > 0 {
    mins = 1
}
```

- Arredonda para o minuto mais próximo.
- Garante mínimo de `1min` para qualquer sessão com tempo > 0.
- Sessões pausadas continuam usando `Elapsed` (comportamento atual de `TotalMs`).

## Testes (`internal/timer/timer_test.go`)

Montar o estado com `Elapsed` preenchido e `Paused: true` para não depender do relógio:

- `TestFinishRounding`:
  - 20s → `1min`
  - 89s → `1min`
  - 90s → `2min`
  - 30min → `30min`
  - 1h30m40s → `91min`
- `TestFinishZeroElapsed`: 0s continua `0min` (não inventar tempo).

## Critérios de aceite

- Nenhuma sessão > 0s resulta em `0min`.
- Arredondamento segue o minuto mais próximo.
- Formato da linha permanece `⏱ Nmin (HH:MM–HH:MM)`.

## Verificação

```bash
go test ./internal/timer/...
```
