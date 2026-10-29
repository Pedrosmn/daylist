# Correções críticas 1–5 — visão geral

> Base: [`docs/roadmap.md`](../roadmap.md), seção **“🐞 Correções prioritárias”** e **“Ordem sugerida”** (item 1).
> Repo: `~/projects/daylist` (Go 1.23.2, Cobra 1.10.2).

## Objetivo

Eliminar as cinco falhas de maior risco apontadas no roadmap, com foco em **não perder tempo registrado pelo timer** e **não montar a daylist a partir da weeklist errada**. Todas as mudanças devem ser cobertas por testes automatizados.

## Tasks

| # | Task | Arquivo | Depende de |
|---|---|---|---|
| 1 | `Save` do estado atômico | [01-save-atomico.md](01-save-atomico.md) | — |
| 2 | Minutos arredondados em sessões curtas | [02-arredondamento-minutos.md](02-arredondamento-minutos.md) | — |
| 3 | `Finish` tolerante a linhas editadas | [03-finish-tolerante.md](03-finish-tolerante.md) | 1, 2 |
| 4 | `timer stop` grava na daylist do estado | [04-timer-stop-arquivo-certo.md](04-timer-stop-arquivo-certo.md) | 3 |
| 5 | Wizard usa a weeklist da semana alvo | [05-weeklist-semana-alvo.md](05-weeklist-semana-alvo.md) | — |
| 6 | Validar o caminho do vault (opcional) | [06-validar-vault.md](06-validar-vault.md) | — |

## Fora de escopo

- Portar `week`/`feedback` para Go (item 3 do roadmap).
- Fallback de hábitos entre semanas no script do QuickAdd (pedido separado).
- UX do wizard, `timer status`, `Makefile`/README (itens 2, 4 e 5 do roadmap).
- Abertura multi-plataforma (correção 7).

## Ordem de execução

1. Tasks 1 e 2 (independentes, base para o resto).
2. Task 3 (`Finish` tolerante).
3. Task 4 (`cmd/timer.go`).
4. Task 5 (semana alvo).
5. Task 6 (opcional).
6. `gofmt -w`, `go vet ./...`, `go test ./...`.

## Commits

Ao concluir cada task, faça um commit antes de seguir para a próxima:

- Rode primeiro a verificação da task e a verificação global.
- Inclua no commit apenas as mudanças da task (código, testes e a documentação ajustada por ela).
- Use uma mensagem descritiva no padrão do repositório, por exemplo:
  `fix(timer): grava estado de forma atômica`.
- Não misture duas tasks no mesmo commit.

## Verificação global

```bash
gofmt -l .
go vet ./...
go test ./... -race -cover
go build ./...
```

## Critérios de aceite globais

- Nenhum caminho de erro do timer apaga o estado silenciosamente.
- Tempo nunca é gravado em uma daylist diferente da que o timer começou.
- Nenhuma sessão > 0s resulta em `0min`.
- A daylist é montada a partir da weeklist da semana da data alvo (ou da anterior, com aviso).
- `go vet` e `go test ./...` passam.
