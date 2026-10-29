# Plano — Timer: status, `--prompt` e altura do fzf

**Item do roadmap:** “2. Rápido: `timer status` + `--prompt` e `--height` no fzf”.
**Data do planejamento:** 2026-10-29.
**Escopo:** dar visibilidade ao timer em andamento e melhorar o conforto do wizard no fzf. Sem novas dependências.

## Estado atual

- `daylist timer start/pause/stop` já é robusto (estado atômico, `Finish` tolerante, gravação na daylist de origem).
- `internal/timer.State` guarda `Desc`, `Started`, `Start`, `Elapsed` e `Paused`; `TotalMs()` já devolve o tempo correto nos dois estados.
- Não existe `timer status`; `daylist timer` sem subcomando hoje imprime a ajuda.
- `internal/ui/pick.go` chama o fzf sem `--height`/`--border`, então cada etapa ocupa a tela inteira e apaga o contexto anterior.
- O arredondamento de minutos está embutido em `Finish`; o status precisa da mesma regra para não divergir.

## Tasks

| # | Task | Arquivo | Depende de |
|---|---|---|---|
| 1 | Núcleo de minutos arredondados | [01-minutos-arredondados.md](01-minutos-arredondados.md) | — |
| 2 | `timer status` humano e `daylist timer` | [02-timer-status.md](02-timer-status.md) | 1 |
| 3 | `timer status --prompt` | [03-timer-status-prompt.md](03-timer-status-prompt.md) | 2 |
| 4 | Altura e layout do fzf | [04-fzf-altura.md](04-fzf-altura.md) | — |

**Ordem de execução:** 1 → 2 → 3; a task 4 é independente e pode ser feita em paralelo ou por último.

## Decisões compartilhadas

- **Sem dependências novas** e sem mudar o formato do estado JSON.
- **Uma task por commit**, condicionado a `gofmt`, `go vet`, testes e build passando.
- **Status é somente leitura:** não cria, altera nem remove o estado do timer.
- **Sem timer não é erro:** `timer status` imprime a mensagem humana e sai com código 0; `--prompt` imprime nada e sai com 0.
- **Estado corrompido é erro:** o status propaga o erro de `timer.Load` em vez de escondê-lo.
- **Minutos do status seguem a mesma regra do `Finish`:** arredonda para o minuto mais próximo e usa no mínimo 1min para sessões > 0.
- **Saída humana:** `Timer rodando: <desc> — Nmin (iniciado às HH:MM)` e `Timer pausado: <desc> — Nmin acumulados (iniciado às HH:MM)`.
- **Saída para prompt:** `⏱ Nm` quando rodando e `⏱ Nm ⏸` quando pausado; linha única, sem ANSI.
- **Fallback sem fzf permanece intacto.**

## Verificação global

```bash
gofmt -l .
go vet ./...
go test ./... -race
go build ./...
```

Verificação manual:

```bash
daylist timer
daylist timer status
daylist timer status --prompt
# com zsh:
RPROMPT='$(daylist timer status --prompt)'
```

Para o fzf, abrir `daylist new` ou `daylist timer start` e conferir que a lista ocupa ~70% da tela, com borda.

## Fora de escopo

- Auto-stop/aviso ao virar o dia ou passar de N horas.
- Mover o estado para `$XDG_STATE_HOME`.
- `--json`, notificações, ícone animado ou integração com a barra do terminal.
- Alterar o comportamento de `start`/`pause`/`stop` além do necessário para reutilizar o núcleo de minutos.
- Mudanças na UX do wizard além do layout do fzf (voltar etapa, Esc com confirmação etc.).

## Ao concluir

- Atualizar o item 2 do roadmap para concluído, documentando os comandos e flags entregues.
- Se existir README, acrescentar `timer status` e o exemplo de `RPROMPT`.
- Fazer um commit por task concluída.
