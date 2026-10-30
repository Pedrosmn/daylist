# Daylist — Revisão e roadmap

> Revisão feita em 2026-10-23, sobre o código em `~/projects/daylist` (Go 1.23.2, Cobra 1.10.2).
> Atualizado em 2026-10-29: correções 1–6 aplicadas e cobertas por testes.
> Atualizado em 2026-10-30: item 3 (week/feedback) portado para Go; o QuickAdd não é mais usado para gerar notas.
> Atualizado em 2026-10-30: item 4 (wizard) concluído com navegação Vim, voltar etapa, confirmação e tempo total.
> Atualizado em 2026-10-30: README e Makefile adicionados; workflow de CI criado e aguardando o primeiro run remoto.
> As referências `arquivo:linha` correspondem ao commit da revisão; podem mudar depois.

## Estado atual

O sistema está sólido: separação `cmd/` (comandos Cobra) e `internal/` (regras), escrita atômica das notas e do estado do timer, testes nas partes de parsing/seleção/timer/vault e um wizard por etapas com navegação Vim (sem busca). `week` e `feedback` agora são gerados em Go, sem QuickAdd; o Obsidian fica só para abrir/editar notas. O que falta está em melhorias de UX e engenharia.

| Comando | Estado |
|---|---|
| `daylist` | Abre a daylist de hoje no Obsidian; se não existir, cria pelo wizard no terminal |
| `daylist new` (`n`) | Wizard por etapas: Tasks → categorias de hábitos, com tempo/prioridade |
| `daylist week` (`w`) | Gera em Go, sem QuickAdd; copia hábitos preservando tempo/prioridade |
| `daylist feedback` (`f`) | Gera em Go, sem QuickAdd; resumo de hábitos por categoria |
| `daylist timer start/pause/stop/status` (`t`) | Funcional e robusto: estado atômico, busca tolerante, gravação na daylist de origem e status (incl. `--prompt`) |
| `daylist completion` | Gerado pelo Cobra |

Estrutura:

```
cmd/                  root, new, week, feedback, timer
internal/daylist/     parser tolerante + Stages + Render + Write
internal/week/        semanas ISO, parser de hábitos, weeklist e feedback
internal/timer/       estado do timer + marcação ⏱ na daylist
internal/wizard/      fluxo por etapas, voltar, resolução de tempo/prioridade e total
internal/ui/          Pick (fzf/menu) + Toggle (wizard) + Input + Confirm
internal/vault/       caminhos e URL do Obsidian
```

---

## ✅ Correções aplicadas (2026-10-29)

As correções 1–6 do roadmap foram implementadas e cobertas por testes:

1. **Timer grava na daylist de origem** — o `stop` resolve o arquivo a partir de `State.File`, inclusive quando o timer vira o dia.
2. **`Finish` tolerante a linhas editadas** — normalização de espaços/caixa, aceita `[x]` manual e correspondência parcial apenas quando única; erros específicos para ambiguidade e tempo já registrado.
3. **`Save` atômico** — estado gravado em `.tmp` + `rename`, sem deixar JSON truncado.
4. **Minutos arredondados** — arredonda para o minuto mais próximo e nunca grava `0min` em sessão com tempo > 0.
5. **Weeklist da semana alvo** — o wizard usa a semana da data alvo e cai para a anterior com aviso.
6. **Vault validado** — `Load` falha cedo com mensagem citando `DAYLIST_VAULT_PATH`.

## 🐞 Pendente

### Abertura de nota é Linux-only
`internal/vault/vault.go` (`openURL`) usa `xdg-open`. Se um dia usar macOS/WSL, vale escolher entre `open`, `xdg-open` e `wslview`.

---

## ✅ Maior ganho: `week` e `feedback` portados para Go (2026-10-30)

Os dois comandos agora geram os arquivos localmente, sem QuickAdd e sem arquivos de alvo:

- **`daylist week [--current|--next]`** copia os hábitos da semana anterior preservando tempo e prioridade; apenas o checkbox volta para `[ ]`. Aceita o formato completo (`5 min (U) - AWS`), só tempo (`5 min - AWS`), só prioridade (`(U) - AWS`) e texto livre (`AWS`).
- **`daylist feedback [--week YYYY-Www]`** conta os hábitos concluídos nas daylists de segunda a domingo, remove o sufixo `⏱` e monta o resumo por categoria.

Ambos têm `--dry-run` e `--print-path`, usam escrita atômica e ignoram daylists ausentes. Os scripts `scripts/weeklist_script.js` e `scripts/feedback_script.js` ficam no repositório como referência histórica.

---

## ✅ UX do wizard (2026-10-30)

- **Navegação Vim padrão** no fzf: `j`/`k`, `g`/`G`, `Ctrl-d`/`Ctrl-u`, `Enter`/`Espaço`, `h` para voltar, `l` para concluir e `q`/`Esc` para cancelar.
- **Sem busca digitável** no fzf: `--disabled` e teclas imprimíveis neutralizadas; a lista é navegada apenas por movimento e marcação.
- **“↩ Voltar etapa”**: `h` ou a linha de voltar reabre a etapa anterior preservando seleções, tempos e prioridades.
- **Header com contagem**: `Etapa 2/9 · Estudos · 3 marcados · 5 no total`.
- **Esc/q com itens marcados**: confirma “descartar tudo?” antes de perder as escolhas.
- **Tempo total planejado**: exibido ao final como valor único ou faixa (`1h45–1h55`); sem limite e sem alerta.
- **Duração personalizada**: continua aceitando número via `Input`.
- **Etapa “Extra”**: adiada; a seção `# Extra` continua vazia por enquanto.

---

## ⏱️ Timer: lacunas

- ✅ `timer status`, `timer status --prompt` e `timer` sem subcomando (entregues em 2026-10-29).
- Aviso/auto-stop quando o timer passar de N horas ou virar o dia.
- Estado do timer vive em `.daylist-timer-state.json` na raiz do vault; considerar `$XDG_STATE_HOME/daylist/` para não poluir/sincronizar o vault.

---

## 🧱 Engenharia

- ✅ `cmd/new.go` foi reduzido; o wizard agora vive em `internal/wizard`, com `Prompter` injetado, e é testável sem TTY.
- Cobertura adicionada em 2026-10-29: `timer.Finish` (linha editada, CRLF, arredondamento, já registrado), `vault` (validação, semana ISO, caminhos), fluxos de `timer stop` e weeklist alvo em `cmd`, e flags/erros do fzf em `ui`.
- Cobertura adicionada em 2026-10-30: `internal/week` (semanas ISO, parser tolerante, weeklist e feedback), comandos `week`/`feedback` (flags, dry-run, print-path, alvos legados), navegação/voltar/cancelar em `ui` e fluxo do wizard em `internal/wizard`.
- Testes que ainda faltam:
  - `cmd`: escolha da data alvo e mapeamento de flags.
- ✅ `Makefile`: `build`, `install`, `test`, `test-race`, `vet`, `fmt`, `fmt-check`, `lint`, `cover`, `ci` e `clean`; usa `go test -race -cover ./...`.
- ✅ `README.md`: instalação, configuração do vault, comandos, teclas do wizard, formato dos arquivos, desenvolvimento e limitações.
- ⏳ CI no GitHub Actions: workflow `.github/workflows/ci.yml` adicionado para rodar `make ci` em push/PR; primeiro run remoto pendente.
- `daylist version` + `--version` (ldflags) — útil com binário em `~/go/bin`.
- **Autodetectar o vault**: o Obsidian Flatpak registra os vaults em
  `~/.var/app/md.obsidian.Obsidian/config/obsidian/obsidian.json`
  (neste ambiente aponta para `/home/pedro/documents/Obsidian/Vault 33`). Ler esse arquivo (mais os caminhos de Linux/Snap/macOS) e usar `--vault` como override acaba com a dependência de `DAYLIST_VAULT_PATH`/`DAYLIST_VAULT_NAME`.

---

## 💡 Ideias de produto

- `daylist new --carry`: traz tasks não feitas de ontem para a daylist de hoje (fácil com o parser tolerante).
- Ao dar `timer stop`, marcar o item correspondente na **weeklist** como `[x]` — ou um comando `daylist week sync`.
- `daylist stats week`: tempo por categoria a partir dos `⏱` + taxa de conclusão (o feedback já tem a base).
- Streaks por hábito.
- `daylist today` / `daylist tomorrow` com `--dry-run` para pré-visualizar.

---

## Ordem sugerida

1. ✅ **Crítico e rápido**: correções 1–6 (timer + semana alvo + validação do vault) — concluído em 2026-10-29.
2. ✅ **Rápido**: `timer status` + `--prompt` e `--height` no fzf — concluído em 2026-10-29.
3. ✅ **Médio**: portar `week`/`feedback` para Go — concluído em 2026-10-30.
4. ✅ **Médio**: wizard com Vim, voltar etapa, confirmação no Esc e tempo total — concluído em 2026-10-30. A etapa Extra fica adiada.
5. **Contínuo**: ✅ README/Makefile; ⏳ CI (workflow pronto, aguardando run) e testes restantes de `cmd`.

---

## Apêndice — decisões já incorporadas no wizard por etapas

- `# Tasks` é a primeira etapa; depois cada categoria de hábito na ordem da weeklist.
- `j`/`k` navegam, `g`/`G` vão ao início/fim, `Ctrl-d`/`Ctrl-u` movem meia página e `Enter`/`Espaço` marcam/desmarcam.
- `h` volta para a etapa anterior; `l` conclui a etapa; `q`/`Esc` cancelam, com confirmação quando há seleção.
- Parser tolerante: `tempo (flag)`, `tempo`, `(flag)` e texto livre; `[x]` fica de fora.
- Duração é perguntada quando o tempo está vazio ou é `ST`/`xST`; prioridade, quando a flag está vazia.
- Tempo/prioridade são resolvidos por etapa, com resumo antes da próxima.
- O total planejado é exibido ao final; tempos compostos viram faixa (`5/15 min` → `5–15 min`).
- `new` grava a daylist de amanhã por padrão; se a de hoje não existir, pergunta qual criar; `--today`, `--tomorrow`, `--date` e `--dry-run` disponíveis.
- `new` nunca abre o Obsidian; o `daylist` sem argumento abre ao final.
