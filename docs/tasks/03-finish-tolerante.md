# Task 3 — `Finish` tolerante a linhas editadas

**Roadmap:** correção 2 · **Depende de:** tasks 1 e 2 · **Bloqueia:** task 4

## Problema

`internal/timer/timer.go:123` compara a linha exata (`if l == s.Line`). Qualquer edição — um espaço, ou marcar `[x]` à mão — gera `ErrLineNotFound`, e `cmd/timer.go:118` **apaga o estado**, perdendo o tempo registrado.

## Mudança

### `internal/timer/timer.go`

1. Normalização:

   ```go
   func normalize(s string) string {
       return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
   }
   ```

2. Novos sentinelas:

   ```go
   var (
       ErrLineNotFound  = errors.New("linha original não encontrada")
       ErrAmbiguousLine = errors.New("mais de uma linha corresponde à descrição")
       ErrAlreadyRecorded = errors.New("tempo já registrado nesta linha")
   )
   ```

3. `findLine(lines []string, desc string) (int, error)`, em camadas:

   - **a)** descrição normalizada em linha pendente (`- [ ] ...`);
   - **b)** descrição normalizada em linha concluída (`- [x] ...`);
   - **c)** correspondência parcial (`strings.Contains`, case-insensitive) **apenas se única**;
   - **d)** mais de um resultado parcial → `ErrAmbiguousLine`;
   - **e)** nada → `ErrLineNotFound`.

   Normalizar também o `\r` de CRLF antes de comparar.

4. `Finish`:

   - linha já com `⏱`: devolver `ErrAlreadyRecorded` (ou o tempo existente) sem duplicar o marcador;
   - linha `[x]` sem `⏱`: registrar o tempo normalmente;
   - manter a escrita atômica já existente (tmp + rename) e o formato de saída `- [x] ... ⏱ Nmin (HH:MM–HH:MM)`.

### `cmd/timer.go` (`stop`)

- `ErrLineNotFound`: **manter** o estado e imprimir instruções:
  `Não achei a linha na daylist. O estado foi mantido; corrija a linha e rode "daylist timer stop" de novo.`
- `ErrAmbiguousLine`: manter o estado e listar as candidatas.
- `ErrAlreadyRecorded`: remover o estado e avisar que o tempo já constava.
- Só remover `statePath` em sucesso ou em `ErrAlreadyRecorded`.

> A troca do arquivo usado no `stop` faz parte da task 4; aqui o foco é a busca tolerante e a preservação do estado.

## Testes (`internal/timer/timer_test.go`)

- `TestFinishExactLine`
- `TestFinishEditedLine` (espaços/maiúsculas diferentes)
- `TestFinishManuallyDoneLine` (não duplica `⏱`)
- `TestFinishPartialUnique`
- `TestFinishPartialAmbiguous` → `ErrAmbiguousLine`
- `TestFinishNotFoundKeepsDaylist` (arquivo da daylist intacto)
- `TestFinishCRLF`

## Critérios de aceite

- Editar espaços ou marcar `[x]` à mão não faz o tempo se perder.
- Correspondência ambígua nunca grava em item errado.
- Estado e daylist ficam intactos em qualquer erro.

## Verificação

```bash
go test ./internal/timer/...
go test ./cmd/...
```
