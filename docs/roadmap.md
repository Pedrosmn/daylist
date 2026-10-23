# Daylist — Revisão e roadmap

> Revisão feita em 2026-10-23, sobre o código em `~/projects/daylist` (Go 1.23.2, Cobra 1.10.2).
> As referências `arquivo:linha` correspondem a este commit; podem mudar depois.

## Estado atual

O sistema está sólido: separação `cmd/` (comandos Cobra) e `internal/` (regras), escrita atômica das notas, testes nas partes de parsing/seleção e um wizard por etapas com busca fuzzy. O que falta está concentrado em **robustez do timer** e na **dependência residual do Obsidian** para `week`/`feedback`.

| Comando | Estado |
|---|---|
| `daylist` | Abre a daylist de hoje no Obsidian; se não existir, cria pelo wizard no terminal |
| `daylist new` (`n`) | Wizard por etapas: Tasks → categorias de hábitos, com tempo/prioridade |
| `daylist week` (`w`) | Ainda via QuickAdd (Obsidian) |
| `daylist feedback` (`f`) | Ainda via QuickAdd (Obsidian) |
| `daylist timer start/pause/stop` (`t`) | Funcional, mas com lacunas de robustez |
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

## 🐞 Correções prioritárias

### 1. Timer grava no arquivo errado quando vira o dia
`cmd/timer.go:115` usa `v.DaylistFile(time.Now())` no `stop`, mas o estado já guarda `file` (`timer.State.File`, `internal/timer/timer.go:27`).

Cenário: inicia 23:50, para 00:10 → procura a daylist de amanhã (que pode nem existir) e o tempo se perde.

**Fix:** resolver o arquivo a partir do estado:
```go
file := filepath.Join(v.Path, filepath.FromSlash(s.File))
mins, err := s.Finish(file)
```

### 2. Timer descarta o tempo se a linha for editada
`internal/timer/timer.go:123` compara a linha **exata** (`if l == s.Line`). Qualquer edição — inclusive marcar `[x]` à mão — gera `ErrLineNotFound`, e `cmd/timer.go:118` **apaga o estado**.

**Fix sugerido:**
1. casar pela descrição normalizada (reutilizando `descRe`);
2. depois por descrição parcial;
3. se ainda não achar, **manter** o estado e imprimir instruções, em vez de descartar silenciosamente.

### 3. `Save` do estado não é atômico
`internal/timer/timer.go:66` grava com `os.WriteFile` direto. Crash/corrupção → `Load` falha com "estado do timer inválido" e `timer start` fica bloqueado.

**Fix:** tmp + `rename`, como já é feito em `State.Finish` e `daylist.Write`.

### 4. Sessões curtas gravam `0min`
`internal/timer/timer.go:132` trunca (`TotalMs() / 60000`).

**Fix:** arredondar (`(ms + 30_000) / 60_000`) ou garantir mínimo de 1 min para sessões acima de ~30s.

### 5. Wizard lê a weeklist da semana errada
`cmd/new.go:109` usa sempre `vault.ISOWeek(time.Now())`. Problemas:

- `daylist n --date 2026-11-02` (outra semana) usa a weeklist desta semana;
- criar a daylist de amanhã no domingo (vira semana ISO) usa a weeklist da semana anterior.

**Fix:** usar a semana da **data alvo**; se ela não existir, cair para a semana anterior (é de onde o `week` copia os hábitos); se nenhuma existir, erro claro sugerindo `daylist week`.

### 6. Vault não é validado
`internal/vault/vault.go:28` (`Load`) aceita qualquer caminho. Um typo em `DAYLIST_VAULT_PATH` produz "weeklist não encontrada".

**Fix:** validar com `os.Stat` e retornar algo como `vault não encontrado em "<path>" (veja DAYLIST_VAULT_PATH)`.

### 7. Abertura de nota é Linux-only
`internal/vault/vault.go:109` usa `xdg-open`. Se um dia usar macOS/WSL, vale escolher entre `open`, `xdg-open` e `wslview`.

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

- `cmd/new.go` tem 311 linhas; extrair o wizard para `internal/wizard` com **clock e picker injetados** torna `resolveTargetDate`, resolução de tempo/prioridade e totais testáveis sem TTY.
- Testes que faltam:
  - `timer.Finish`: linha editada, sessão atravessando meia-noite, arredondamento;
  - `vault`: montagem de caminhos, semana ISO, validação do vault;
  - `cmd`: escolha da data alvo e mapeamento de flags.
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

1. **Crítico e rápido**: correções 1–5 (timer + semana alvo + validação do vault).
2. **Rápido**: `timer status` + `--prompt` e `--height` no fzf.
3. **Médio**: portar `week`/`feedback` para Go.
4. **Médio**: voltar etapa, confirmação no Esc, etapa Extra, alerta de tempo total.
5. **Contínuo**: README/Makefile/CI e testes de `cmd`, `vault` e `timer`.

---

## Apêndice — decisões já incorporadas no wizard por etapas

- `# Tasks` é a primeira etapa; depois cada categoria de hábito na ordem da weeklist.
- Enter marca/desmarca; `✅ Concluir <etapa>` finaliza; `Esc` cancela tudo sem gravar.
- Parser tolerante: `tempo (flag)`, `tempo`, `(flag)` e texto livre; `[x]` fica de fora.
- Duração é perguntada quando o tempo está vazio ou é `ST`/`xST`; prioridade, quando a flag está vazia.
- Tempo/prioridade são resolvidos por etapa, com resumo antes da próxima.
- `new` grava a daylist de amanhã por padrão; se a de hoje não existir, pergunta qual criar; `--today`, `--tomorrow`, `--date` e `--dry-run` disponíveis.
- `new` nunca abre o Obsidian; o `daylist` sem argumento abre ao final.
