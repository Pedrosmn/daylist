# Daylist — Revisão e roadmap

> Revisão feita em 2026-10-23, sobre o código em `~/projects/daylist` (Go 1.23.2, Cobra 1.10.2).
> Atualizado em 2026-10-29: correções 1–6 aplicadas e cobertas por testes.
> As referências `arquivo:linha` correspondem ao commit da revisão; podem mudar depois.

## Estado atual

O sistema está sólido: separação `cmd/` (comandos Cobra) e `internal/` (regras), escrita atômica das notas e do estado do timer, testes nas partes de parsing/seleção/timer/vault e um wizard por etapas com busca fuzzy. O que falta está concentrado na **dependência residual do Obsidian** para `week`/`feedback` e em melhorias de UX e engenharia.

| Comando | Estado |
|---|---|
| `daylist` | Abre a daylist de hoje no Obsidian; se não existir, cria pelo wizard no terminal |
| `daylist new` (`n`) | Wizard por etapas: Tasks → categorias de hábitos, com tempo/prioridade |
| `daylist week` (`w`) | Ainda via QuickAdd (Obsidian) |
| `daylist feedback` (`f`) | Ainda via QuickAdd (Obsidian) |
| `daylist timer start/pause/stop` (`t`) | Funcional e robusto: estado atômico, busca tolerante e gravação na daylist de origem |
| `daylist completion` | Gerado pelo Cobra |

Estrutura:

```
cmd/                  root, new, week, feedback, timer
internal/daylist/     parser tolerante + Stages + Render + Write
internal/timer/       estado do timer + marcação ⏱ na daylist
internal/ui/          Pick (fzf/menu) + Toggle (wizard) + Input
internal/vault/       caminhos, URL do Obsidian, arquivos de alvo
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

## 🚀 Maior ganho: portar `week` e `feedback` para Go

Hoje os dois dependem do Obsidian e **falham em silêncio**: o CLI abre a URI, imprime sucesso e deixa `.daylist-weeklist-target.json` / `.daylist-feedback-target.json` na raiz do vault.

Os scripts são pequenos e o parser novo cobre quase tudo:

| Script | Tamanho | O que faz | Esforço |
|---|---|---|---|
| `weeklist_script.js` | ~70 linhas | copia hábitos da semana anterior, resetando tempo/flag para `xST ()` | baixo |
| `feedback_script.js` | ~135 linhas | conta hábitos concluídos por categoria na semana e monta o resumo | baixo/médio |

**Benefícios:** nenhuma dependência do QuickAdd para criar nada; sem arquivos de alvo órfãos; saída testável; o Obsidian fica só para abrir/editar notas.

**Comandos resultantes:** `daylist week [--current|--next]`, `daylist feedback [--week YYYY-Www]`, ambos com `--dry-run` e `--print-path`.

---

## 🎨 UX do wizard

- **fzf com altura** (`--height=70% --border --info=inline`): hoje cada etapa limpa a tela; com altura, os resumos das etapas anteriores continuam visíveis.
- **“↩ Voltar etapa”**: revisar uma categoria sem refazer as anteriores.
- **Header com contagem**: `Etapa 2/9 · Estudos · 3 marcados · 5 no total`.
- **Esc com itens marcados**: confirmar “descartar tudo?” (hoje perde todas as escolhas sem aviso).
- **Etapa “Extra”**: capturar tasks livres para a seção `# Extra` (o template atual nunca a preenche).
- **Alerta de tempo total planejado**: somar as durações e avisar quando passar de um limite (ex.: “4h30 de 6h disponíveis”).
- **`ctrl-d` como atalho de concluir** etapa no fzf, além da linha `✅` (útil em listas longas).

---

## ⏱️ Timer: lacunas

- `daylist timer status` — item atual, tempo decorrido, pausado ou não.
- `daylist timer status --prompt` — saída curta (`⏱ 25m`) para usar no `RPROMPT` do zsh. Esse recurso existia no script antigo e se perdeu na migração.
- `daylist timer` sem subcomando mostra o status (hoje só imprime a ajuda).
- Aviso/auto-stop quando o timer passar de N horas ou virar o dia.
- Estado do timer vive em `.daylist-timer-state.json` na raiz do vault; considerar `$XDG_STATE_HOME/daylist/` para não poluir/sincronizar o vault.

---

## 🧱 Engenharia

- `cmd/new.go` tem ~340 linhas; extrair o wizard para `internal/wizard` com **clock e picker injetados** torna `resolveTargetDate`, resolução de tempo/prioridade e totais testáveis sem TTY.
- Cobertura adicionada em 2026-10-29: `timer.Finish` (linha editada, CRLF, arredondamento, já registrado), `vault` (validação, semana ISO, caminhos) e fluxos de `timer stop` e weeklist alvo em `cmd`.
- Testes que ainda faltam:
  - `cmd`: escolha da data alvo e mapeamento de flags;
  - wizard de `new` sem TTY (depende da extração para `internal/wizard`).
- `Makefile`: `build`, `install`, `test`, `vet`, `fmt`, `lint`; `go test -race -cover ./...`.
- `daylist version` + `--version` (ldflags) — útil com binário em `~/go/bin`.
- README curto: instalação, comandos, teclas do wizard, variáveis de ambiente, estrutura do vault.
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
2. **Rápido**: `timer status` + `--prompt` e `--height` no fzf — plano em [`plans/timer-status-fzf/00-visao-geral.md`](plans/timer-status-fzf/00-visao-geral.md).
3. **Médio**: portar `week`/`feedback` para Go.
4. **Médio**: voltar etapa, confirmação no Esc, etapa Extra, alerta de tempo total.
5. **Contínuo**: README/Makefile/CI, testes restantes de `cmd` e extração do wizard.

---

## Apêndice — decisões já incorporadas no wizard por etapas

- `# Tasks` é a primeira etapa; depois cada categoria de hábito na ordem da weeklist.
- Enter marca/desmarca; `✅ Concluir <etapa>` finaliza; `Esc` cancela tudo sem gravar.
- Parser tolerante: `tempo (flag)`, `tempo`, `(flag)` e texto livre; `[x]` fica de fora.
- Duração é perguntada quando o tempo está vazio ou é `ST`/`xST`; prioridade, quando a flag está vazia.
- Tempo/prioridade são resolvidos por etapa, com resumo antes da próxima.
- `new` grava a daylist de amanhã por padrão; se a de hoje não existir, pergunta qual criar; `--today`, `--tomorrow`, `--date` e `--dry-run` disponíveis.
- `new` nunca abre o Obsidian; o `daylist` sem argumento abre ao final.
