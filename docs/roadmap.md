# Daylist — Revisão e roadmap

> Revisão feita em 2026-10-23, sobre o código em `~/projects/daylist` (Go 1.23.2, Cobra 1.10.2).
> Atualizado em 2026-10-29: correções 1–6 aplicadas e cobertas por testes.
> Atualizado em 2026-10-30: item 3 (week/feedback) portado para Go; o QuickAdd não é mais usado para gerar notas.
> Atualizado em 2026-10-30: item 4 (wizard) concluído com navegação Vim, voltar etapa, confirmação e tempo total.
> Atualizado em 2026-10-30: README e Makefile adicionados; CI validado no GitHub Actions; branch protection ativa na `main`.
> Atualizado em 2026-11-02: item 5.3 concluído; data alvo e flags de `new` cobertas por testes.
> Atualizado em 2026-11-02: `daylist version` e `--version` adicionados com `ldflags`.
> Atualizado em 2026-11-04: timer sinaliza mínimo/máximo do tempo planejado; itens adiados foram para `docs/backlog.md`.
> Atualizado em 2026-11-04: próximo ciclo definido (TUI, `#always`, motivações e Extra); tarefas detalhadas em `docs/backlog.md`.
> Atualizado em 2026-11-04: `#always` restrito a hábitos e incluído automaticamente nas daylists da semana; adicionadas prévia, undo, fechamento do dia e motivação no foco.
> Atualizado em 2026-11-04: `W1–W4` concluídos (marcador `#always` no parser, na próxima weeklist e no wizard).
> As referências `arquivo:linha` correspondem ao commit da revisão; podem mudar depois.
>
> Itens adiados e ideias de produto: [docs/backlog.md](backlog.md).

## Estado atual

O sistema está sólido: separação `cmd/` (comandos Cobra) e `internal/` (regras), escrita atômica das notas e do estado do timer, testes nas partes de parsing/seleção/timer/vault e um wizard por etapas com navegação Vim (sem busca). `week` e `feedback` agora são gerados em Go, sem QuickAdd; o Obsidian fica só para abrir/editar notas. O que falta está em melhorias de UX e engenharia; itens adiados estão em `docs/backlog.md`.

| Comando | Estado |
|---|---|
| `daylist` | Abre a daylist de hoje no Obsidian; se não existir, cria pelo wizard no terminal |
| `daylist new` (`n`) | Wizard por etapas: Tasks → categorias de hábitos, com tempo/prioridade |
| `daylist week` (`w`) | Gera em Go, sem QuickAdd; copia hábitos preservando tempo/prioridade |
| `daylist feedback` (`f`) | Gera em Go, sem QuickAdd; resumo de hábitos por categoria |
| `daylist timer start/pause/stop/status` (`t`) | Funcional e robusto: estado atômico, busca tolerante, gravação na daylist de origem, status (incl. `--prompt`) e aviso de mínimo/máximo |
| `daylist completion` | Gerado pelo Cobra |
| `daylist version` | Mostra versão, commit e data de build embutidos |

Estrutura:

```
cmd/                  root, new, week, feedback, timer, version
internal/daylist/     parser tolerante + Stages + Render + Write
internal/week/        semanas ISO, parser de hábitos, weeklist e feedback
internal/timer/       estado do timer + marcação ⏱ na daylist
internal/wizard/      fluxo por etapas, voltar, resolução de tempo/prioridade e total
internal/ui/          Pick (fzf/menu) + Toggle (wizard) + Input + Confirm
internal/tui/         (planejado) modelo, telas e ações do TUI
internal/vault/       caminhos e URL do Obsidian
```

## 🖥️ Próximo ciclo — TUI e metadados

As tarefas detalhadas estão em [docs/backlog.md](backlog.md).

- `daylist` sem argumentos abre o TUI; `daylist open` mantém a abertura no
  Obsidian.
- Ações do TUI: criar daylist, criar weeklist atual se faltar, ver e recarregar
  a daylist, marcar item como concluído, controlar o timer, adicionar Extra,
  editar motivação, desfazer a última edição, fechar o dia e abrir a nota no
  Obsidian.
- A criação da daylist reutiliza o wizard atual, suspendendo o TUI, e mostra uma
  prévia com os hábitos `#always`, o total planejado e alerta de sobrecarga.
- `#always` é exclusivo de hábitos. O hábito marcado é copiado para a próxima
  weeklist com o marcador e entra automaticamente, desmarcado, em toda daylist
  daquela semana; não depende de seleção manual e não é removido no wizard.
  Tasks não usam `#always` e continuam sem ser copiadas.
- Motivação fica somente na daylist, como blockquote abaixo do item, e aparece
  ao destacar o item ou iniciar o timer.
- Hábito Extra é escolhido entre os hábitos da weeklist e inserido na categoria
  original de `# Hábitos`, para entrar no feedback normalmente.
- Task Extra continua indo para `# Extra`.

## 🌿 Fluxo de trabalho

A partir de 2026-10-30, toda implementação nova começa em uma branch criada
antes do primeiro commit. Commits diretos na `main` são bloqueados pela
proteção da branch.

- `feat/...` para funcionalidades.
- `fix/...` para correções.
- `docs/...` para documentação.
- `ci/...` para CI e infraestrutura.
- `refactor/...` para refatorações sem mudança de comportamento.

O trabalho é commitado na branch e integrado por pull request. O merge deve
ocorrer somente com o check `Testes e build` verde.

**Status da proteção:** ativa desde 2026-10-30. A `main` exige pull request e o
check `Testes e build` verde, precisa estar atualizada antes do merge e bloqueia
force push e deleção. As regras valem também para administradores.

---

## ✅ Correções aplicadas (2026-10-29)

As correções 1–6 do roadmap foram implementadas e cobertas por testes:

1. **Timer grava na daylist de origem** — o `stop` resolve o arquivo a partir de `State.File`, inclusive quando o timer vira o dia.
2. **`Finish` tolerante a linhas editadas** — normalização de espaços/caixa, aceita `[x]` manual e correspondência parcial apenas quando única; erros específicos para ambiguidade e tempo já registrado.
3. **`Save` atômico** — estado gravado em `.tmp` + `rename`, sem deixar JSON truncado.
4. **Minutos arredondados** — arredonda para o minuto mais próximo e nunca grava `0min` em sessão com tempo > 0.
5. **Weeklist da semana alvo** — o wizard usa a semana da data alvo e cai para a anterior com aviso.
6. **Vault validado** — `Load` falha cedo com mensagem citando `DAYLIST_VAULT_PATH`.

## 🧭 Escopo atual

- Linux-only por decisão; suporte a macOS/WSL fica fora do escopo.
- Autodetecção do vault continua adiada em `docs/backlog.md`.
- O próximo ciclo é o TUI com `#always` para hábitos, motivações e Extra; o
  wizard não terá uma etapa Extra separada.
- A TUI também terá prévia da daylist, undo, fechamento do dia e motivação no
  momento do foco.
- O timer sinaliza mínimo e máximo do tempo planejado, sem parar automaticamente.

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
- **Hábitos “sempre”**: hábitos com `#always` entram automaticamente em toda
  daylist da semana, desmarcados, sem seleção manual e sem opção de remover no
  wizard. Tasks não usam `#always`.
- **Etapa “Extra”**: não será uma etapa do wizard; a inclusão de Extra será uma
  ação do TUI.

---

## ⏱️ Timer: lacunas

- ✅ `timer status`, `timer status --prompt` e `timer` sem subcomando (entregues em 2026-10-29).
- ✅ Aviso visual no status/prompt ao atingir o mínimo e o máximo do tempo planejado; notificação desktop via `notify-send`. O máximo apenas sinaliza, sem auto-stop.
- Estado do timer vive em `.daylist-timer-state.json` na raiz do vault; mover para `$XDG_STATE_HOME/daylist/` está no backlog.

---

## 🧱 Engenharia

- ✅ `cmd/new.go` foi reduzido; o wizard agora vive em `internal/wizard`, com `Prompter` injetado, e é testável sem TTY.
- Cobertura adicionada em 2026-10-29: `timer.Finish` (linha editada, CRLF, arredondamento, já registrado), `vault` (validação, semana ISO, caminhos), fluxos de `timer stop` e weeklist alvo em `cmd`, e flags/erros do fzf em `ui`.
- Cobertura adicionada em 2026-10-30: `internal/week` (semanas ISO, parser tolerante, weeklist e feedback), comandos `week`/`feedback` (flags, dry-run, print-path, alvos legados), navegação/voltar/cancelar em `ui` e fluxo do wizard em `internal/wizard`.
- Cobertura adicionada em 2026-11-02: `cmd/new` (data alvo, padrão Hoje/Amanhã, cancelamento, erro do picker e flags).
- Cobertura adicionada em 2026-11-02: `internal/version` (formato) e `cmd` (comando `version`, `--version` e ausência de vault).
- Cobertura adicionada em 2026-11-04: limites mínimo/máximo do timer (parser, estado antigo, status/prompt e notificação única).
- ✅ `Makefile`: `build`, `install`, `test`, `test-race`, `vet`, `fmt`, `fmt-check`, `lint`, `cover`, `ci` e `clean`; usa `go test -race -cover ./...` e embute versão/commit/data no build.
- ✅ `README.md`: instalação, configuração do vault, comandos, teclas do wizard, formato dos arquivos, desenvolvimento e limitações.
- ✅ CI no GitHub Actions: workflow `.github/workflows/ci.yml` roda `make ci` em push/PR; primeiro run verde em 2026-10-30.
- ✅ `daylist version` + `--version` (ldflags) — útil com binário em `~/go/bin`.
- Itens adiados, incluindo autodetecção do vault, estão em [docs/backlog.md](backlog.md).

---

## 💡 Ideias de produto

As ideias de produto foram movidas para [docs/backlog.md](backlog.md) para manter
o roadmap focado no escopo atual.

---

## Ordem sugerida

1. ✅ **Crítico e rápido**: correções 1–6 (timer + semana alvo + validação do vault) — concluído em 2026-10-29.
2. ✅ **Rápido**: `timer status` + `--prompt` e `--height` no fzf — concluído em 2026-10-29.
3. ✅ **Médio**: portar `week`/`feedback` para Go — concluído em 2026-10-30.
4. ✅ **Médio**: wizard com Vim, voltar etapa, confirmação no Esc e tempo total — concluído em 2026-10-30. A inclusão de Extra será feita pelo TUI, não pelo wizard.
5. **Contínuo**: ✅ README/Makefile, CI, testes de `cmd` e versionamento do binário.
6. **Próximo ciclo**: TUI, `#always` para hábitos, motivações, Extra, prévia da daylist, undo e fechamento do dia — tarefas em [docs/backlog.md](backlog.md).

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
