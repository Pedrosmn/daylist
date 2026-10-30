# Plano — Item 3: portar `week` e `feedback` para Go

**Status:** implementado em 2026-10-30.
**Item do roadmap:** “3. Médio: portar `week`/`feedback` para Go”.
**Fonte de comportamento:** `scripts/weeklist_script.js` e `scripts/feedback_script.js`.
**Data do planejamento:** 2026-10-29.

## Objetivo

Gerar weeklist e feedback inteiramente em Go, sem QuickAdd e sem arquivos de alvo órfãos. O Obsidian continua sendo usado apenas para abrir/editar notas.

## Estado atual

- `cmd/week.go` e `cmd/feedback.go` só escolhem a semana, gravam `.daylist-weeklist-target.json`/`.daylist-feedback-target.json` e abrem uma URI `obsidian://quickadd`.
- Os scripts JS fazem o trabalho real: criar `weeklist.md` e `feedback.md`.
- O parser de `internal/daylist` **não serve** para esta etapa: ele ignora itens `[x]` e não preserva a ordem/estrutura necessária para copiar hábitos e calcular estatísticas.

## CLI proposta

| Comando | Flags | Padrão |
|---|---|---|
| `daylist week` | `--current`, `--next`, `--dry-run`, `--print-path` | próxima semana |
| `daylist feedback` | `--week YYYY-Www`, `--dry-run`, `--print-path` | semana atual |

Regras comuns:

- `--current`/`--next` e `--dry-run`/`--print-path` são mutuamente exclusivos.
- `--print-path` imprime apenas o caminho resolvido e não escreve.
- `--dry-run` imprime o conteúdo gerado e não escreve.
- Se o arquivo final já existir, imprime “já existe” e sai com 0 (exceto `--print-path`).
- Escrita sempre atômica, reutilizando `daylist.Write`.
- Arquivos de alvo legados são removidos best-effort apenas no caminho que grava (nunca em `--dry-run`/`--print-path`).
- O clock é injetado nas funções internas (`now time.Time`) para os testes não dependerem da data real.

## Tasks

| # | Task | Arquivos principais | Depende de |
|---|---|---|---|
| 1 | Núcleo de semanas e parser de hábitos | `internal/week/week.go`, `habits.go`, `_test.go` | — |
| 2 | Gerador puro de weeklist | `internal/week/weeklist.go`, `_test.go` | 1 |
| 3 | Comando `week` em Go | `cmd/week.go`, `cmd/week_test.go` | 2 |
| 4 | Gerador puro de feedback | `internal/week/feedback.go`, `_test.go` | 1 |
| 5 | Comando `feedback` em Go | `cmd/feedback.go`, `cmd/feedback_test.go` | 4 |
| 6 | Limpeza do QuickAdd e documentação | `internal/vault/vault.go`, `docs/roadmap.md` | 3, 5 |

**Ordem:** 1 → 2 → 3, 4 → 5, 6 por último. As duplas 2–3 e 4–5 podem ser feitas em paralelo depois da task 1.

## Pré-requisito

`scripts/` está untracked. Antes da implementação, commitar como referência:

```bash
git add scripts
git commit -m "docs: adiciona scripts de referência do QuickAdd"
```

## Decisões compartilhadas

- Criar o pacote `internal/week` para lógica pura de semana, hábitos, weeklist e feedback.
- **A nova weeklist copia os hábitos preservando tempo e prioridade** (ex.: `- [ ] 5 min (U) - AWS` continua `- [ ] 5 min (U) - AWS`). Apenas o checkbox é resetado de `[x]` para `[ ]`; nada de trocar tempo/flag por `xST ()`.
- **Copiar qualquer formato aceito pelo parser da daylist**, não só `tempo (prioridade) - desc`:
  - `5 min (U) - AWS`
  - `5 min - AWS`
  - `(U) - AWS`
  - `AWS`
- Usar um único parser tolerante de hábitos (`internal/week/habits.go`) na geração da weeklist e no feedback, para que linhas parciais também sejam contabilizadas.
- Não reutilizar `daylist.Parse`: ele ignora `[x]`, e precisamos enxergar o checkbox e preservar categorias.
- Reproduzir o formato dos scripts JS; divergências intencionais ficam em mensagens de erro/aviso, na preservação de tempo/prioridade e no suporte a formatos parciais.
- Manter `scripts/*.js` no repositório como referência até a paridade ser validada.
- Nenhuma dependência nova.
- Uma task por commit, com verificação antes de cada commit.

## Verificação global

```bash
gofmt -l .
go vet ./...
go test ./... -race
go build ./...
```

## Fora de escopo

- Melhorias de UX do wizard (item 4 do roadmap).
- Autodetecção do vault e abertura multi-plataforma.
- Alterar o formato da daylist/weeklist/feedback.
- Apagar os scripts JS nesta etapa.
- Sincronizar `timer stop` com a weeklist.

## Ao concluir

- Atualizar a tabela de comandos do roadmap.
- Marcar o item 3 como concluído.
- Documentar `week`/`feedback` no README, se ele existir.
