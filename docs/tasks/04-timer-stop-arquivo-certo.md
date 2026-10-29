# Task 4 — `timer stop` grava na daylist do estado

**Roadmap:** correção 1 · **Depende de:** task 3

## Problema

`cmd/timer.go:115` usa `v.DaylistFile(time.Now())` no `stop`. Se o timer começou às 23:50 e parou às 00:10, ele procura a daylist de amanhã (que pode nem existir) e o tempo se perde — ou pior, grava no arquivo errado.

O estado já guarda o caminho relativo correto em `timer.State.File` (`internal/timer/timer.go:27`).

## Mudança

Em `cmd/timer.go` (`timerStopCmd.RunE`):

```go
file := filepath.Join(v.Path, filepath.FromSlash(s.File))
mins, err := s.Finish(file)
```

- Importar `path/filepath`.
- Se `os.ReadFile` falhar com `os.ErrNotExist`, devolver erro claro **mantendo o estado**:
  `daylist do timer não encontrada: <path> (o estado foi mantido; use "daylist timer stop" de novo)`.
- Não remover o estado em nenhum caminho de erro.
- Manter os tratamentos de `ErrLineNotFound`/`ErrAmbiguousLine`/`ErrAlreadyRecorded` da task 3.

## Testes

O comando depende de TTY (`ui`) apenas no `start`; o `stop` é testável.

- `cmd/timer_test.go` (novo): escrever um estado com `File` apontando para uma daylist em `t.TempDir()` com item pendente, rodar o fluxo de `Finish` pelo caminho resolvido a partir do estado e conferir que o `⏱` foi para esse arquivo — inclusive quando o nome do arquivo não é a data de hoje.
- `TestTimerStopMissingDaylistKeepsState`.

> Se o teste via Cobra for custoso, extrair a resolução do caminho para um helper (`timerDaylistPath(v, s)`) e testá-lo diretamente.

## Critérios de aceite

- Tempo é gravado sempre na daylist em que o timer começou, mesmo virando o dia.
- Daylist ausente → erro claro, estado preservado.
- Nenhum arquivo de outra data é criado ou alterado.

## Verificação manual

```bash
export DAYLIST_VAULT_PATH=$(mktemp -d)
# criar daylist de ontem com item pendente e estado com file = essa daylist
daylist timer stop   # o ⏱ deve aparecer no arquivo de ontem
```
