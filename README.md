# daylist

[![CI](https://github.com/Pedrosmn/daylist/actions/workflows/ci.yml/badge.svg)](https://github.com/Pedrosmn/daylist/actions/workflows/ci.yml)

CLI em Go para gerenciar daylists, weeklists, feedback e timer em um vault do
Obsidian. O fluxo principal é: gerar a weeklist, escolher as tasks e hábitos no
terminal, criar a daylist do dia e registrar o tempo com o timer.

## Requisitos

- Go 1.23.2 ou superior (versão declarada no `go.mod`).
- Obsidian instalado para abrir as notas pelo comando `daylist open`.
- Sem dependência de `fzf`: o wizard usa uma UI nativa em Bubble Tea quando
  `stdin` e `stderr` são um terminal. Sem TTY, cai em menus numerados com os
  mesmos comandos de etapa (`h`, `l`, `q`).
- `notify-send` (libnotify) opcional, para notificações do timer no Linux.
- Linux para abertura automática da nota, pois o comando usa `xdg-open`.

## Instalação

Clone o repositório e instale pelo Makefile:

```sh
git clone https://github.com/Pedrosmn/daylist.git
cd daylist
make install
```

Ou use o Go diretamente:

```sh
go install .
```

O `go install .` coloca o binário em `$GOBIN` ou `$GOPATH/bin`. Confirme que
esse diretório está no `PATH`.

Para gerar um binário local sem instalar:

```sh
make build
./daylist --help
```

O módulo ainda se chama `daylist`, então a instalação é feita a partir do clone
local; não use `go install github.com/Pedrosmn/daylist@latest` enquanto o
`go.mod` não for atualizado.

## Configuração

O CLI procura o vault no seguinte caminho por padrão:

```text
~/documents/Obsidian/Vault 33
```

O nome usado na URI do Obsidian é `Vault 33`. Para outro vault, defina:

```sh
export DAYLIST_VAULT_PATH="$HOME/caminho/para/Seu Vault"
export DAYLIST_VAULT_NAME="Seu Vault"
```

Adicione esses `export` ao `~/.zshrc`, `~/.bashrc` ou equivalente para manter a
configuração entre sessões.

O limite diário usado na prévia da daylist pode ser ajustado com:

```sh
export DAYLIST_DAY_LIMIT="8h"   # padrão; aceita "7h30", "480 min" etc.
```

O limite só gera um alerta visual na prévia; não bloqueia a criação da daylist.

A estrutura esperada dentro do vault é:

```text
Fleeting Notes/Daily/Days/YYYY-MM-DD.md
Fleeting Notes/Daily/weeklists/YYYY-Www/weeklist.md
Fleeting Notes/Daily/weeklists/YYYY-Www/feedback.md
.daylist-timer-state.json
```

## Comandos

### `daylist`

Sem subcomando, abre o TUI da daylist de hoje quando stdin e stdout são um
terminal interativo. Sem TTY, mantém o comportamento antigo de abrir a nota no
Obsidian (criando pelo wizard se necessário).

### `daylist open`

Abre a daylist de hoje no Obsidian. Se a nota ainda não existir, inicia o wizard
e abre a nota depois da criação.

### `daylist new` (`n`)

Cria a daylist escolhendo os itens pendentes da weeklist da semana alvo, em
etapas: primeiro `# Tasks`, depois cada categoria de hábitos.

```sh
daylist new
daylist new --today
daylist new --tomorrow
daylist new --date 2026-11-02
daylist new --dry-run
```

Por padrão cria a daylist de amanhã. Se a daylist de hoje ainda não existir,
pergunta se você quer criar a de hoje ou a de amanhã. As flags pulam essa
pergunta. Quando a weeklist da semana alvo não existe, usa a da semana anterior
e avisa.

Hábitos da weeklist marcados com `#always` entram automaticamente em toda
daylist daquela semana, com o checkbox pendente. Eles aparecem como `(sempre)`
e não podem ser desmarcados no wizard. O marcador vale somente para hábitos.

### `daylist week` (`w`)

Cria a weeklist da próxima semana, copiando os hábitos da semana anterior. Tempo,
prioridade e o marcador `#always` são preservados; apenas o checkbox volta para
pendente.

```sh
daylist week
daylist week --current
daylist week --next
daylist week --dry-run
daylist week --print-path
```

O padrão é `--next`.

### `daylist feedback` (`f`)

Gera o feedback da semana atual (ou da semana informada), contando os hábitos
concluídos nas daylists de segunda a domingo.

```sh
daylist feedback
daylist feedback --week 2026-W45
daylist feedback --dry-run
daylist feedback --print-path
```

### `daylist timer` (`t`)

Controla o timer de um item da daylist de hoje.

```sh
daylist timer start
daylist timer pause
daylist timer status
daylist timer status --prompt
daylist timer stop
```

O timer grava o tempo na daylist em que foi iniciado, mesmo que o dia vire. O
estado fica em `.daylist-timer-state.json` na raiz do vault. `timer` sem
subcomando é equivalente a `timer status`.

O tempo planejado do item define os limites mínimo e máximo. `5/15 min` usa 5 e
15; `30 min` usa 30 e 30; `ST`/`xST` não têm limites. O status avisa ao atingir
cada limite, mas não para o timer automaticamente. Se `notify-send` estiver
disponível, uma notificação desktop é disparada uma vez por limite.

`status --prompt` imprime uma saída curta, útil para prompt do shell:

```sh
daylist timer status --prompt
# ⏱ 25m
# ⏱ 5m✓    (mínimo atingido)
# ⏱ 15m⚠   (máximo atingido)
```

### `daylist version`

Mostra a versão, o commit e a data de build embutidos pelo `Makefile`.

```sh
daylist version
daylist --version
```

`make build` e `make install` preenchem esses dados via `-ldflags`. Um
`go install .` direto usa os valores padrão (`dev`).

### `daylist completion`

Gera scripts de completion do Cobra para o shell atual.

## Wizard

As etapas usam navegação no estilo Vim em uma UI Bubble Tea/Lipgloss nativa,
com o mesmo tema do TUI. A interface é inline (sem alt-screen), desenha em
`stderr` e deixa `stdout` limpo para `--dry-run` e redirecionamentos. Ao
concluir ou cancelar uma etapa, a lista é substituída por um resumo compacto,
como `✓ Etapa 1/3 · # Tasks — 2 selecionados`, para não inflar o scrollback.

| Tecla | Ação |
|---|---|
| `j` / `k`, `↑` / `↓` | desce / sobe |
| `g` / `G`, `home` / `end` | vai ao início / fim |
| `Ctrl-d` / `Ctrl-u` | meia página para baixo / cima |
| `Enter` / `Espaço` | marca / desmarca o item destacado |
| `h` | volta para a etapa anterior |
| `l` | conclui a etapa |
| `q` / `Esc` | cancela tudo, com confirmação se houver seleção |

Ao voltar uma etapa, seleções, tempos e prioridades já definidos são
preservados. A duração personalizada é digitada em um prompt de texto, quando
você escolhe `Personalizado`. Confirmações usam `s` para sim e `n`/`Esc` para
não, com a opção segura (“Não, continuar”) como padrão.

Sem TTY (testes, pipes e CI), o wizard mantém o menu numerado. Nele, digite o
número para marcar/desmarcar, `l` ou `Enter` para concluir, `h` para voltar e
`q` para cancelar.

Hábitos com `#always` aparecem com o sufixo `(sempre)`, já entram marcados em
toda daylist da semana e não podem ser desmarcados.

Ao final, o tempo total planejado é exibido. Tempos compostos aparecem como
faixa, por exemplo:

```text
Tempo planejado: 1h45–1h55
```

## TUI

`daylist` sem subcomando abre o TUI baseado em Bubble Tea/Lipgloss. A tela
mostra a daylist de hoje com tasks, hábitos por categoria, Extra, itens
concluídos, motivações e o status do timer.

| Tecla | Ação |
|---|---|
| `j` / `k`, `↑` / `↓` | mover o cursor |
| `g` / `G`, `home` / `end` | início / fim da lista |
| `Espaço` / `Enter` | marcar ou desmarcar o item |
| `s` | iniciar o timer no item selecionado |
| `p` | pausar ou retomar o timer |
| `x` | parar o timer e gravar o tempo na daylist de origem |
| `e` | editar ou remover a motivação do item |
| `a` | adicionar Extra task |
| `A` | adicionar hábito Extra escolhido da weeklist |
| `W` | ver a weeklist atual e alternar `#always` em hábitos |
| `u` | desfazer a última edição da daylist |
| `n` | criar a daylist de hoje pelo wizard, com prévia antes de gravar |
| `N` | criar a daylist de amanhã pelo wizard, com prévia antes de gravar |
| `c` | criar a weeklist atual, se estiver faltando |
| `F` | fechar o dia com resumo de planejado vs. real |
| `o` | abrir a daylist de hoje no Obsidian |
| `r` | recarregar a daylist do disco |
| `?` | tela de ajuda |
| `q` / `Ctrl+C` | sair |

A criação da daylist (de hoje com `n` ou de amanhã com `N`) suspende o TUI e
executa o wizard normal no terminal.
Antes de gravar, o TUI mostra uma prévia com o conteúdo completo que será
gravado — incluindo os hábitos `#always` —, o tempo planejado e um alerta
quando o dia passa do limite configurado em `DAYLIST_DAY_LIMIT` (padrão de 8h).
`Enter`/`s` grava e `Esc` cancela sem criar a nota.

O hábito Extra é escolhido entre os hábitos pendentes da weeklist e entra na
categoria original de `# Hábitos`, contando no feedback. A task Extra vai para
`# Extra`. A motivação aparece abaixo do item e também é mostrada ao iniciar o
timer. O aviso de tempo mínimo/máximo do timer usa as mesmas regras e mensagens
do CLI.

`F` abre o resumo de fechamento do dia com tempo planejado, tempo real já
registrado, concluídos e pendentes. Fechar o dia apenas mostra o resumo: nada é
alterado na nota e nenhum pendente é copiado para amanhã.

O `#always` é exclusivo de hábitos: na tela da weeklist, `Espaço` alterna o
marcador apenas quando o item está em `# Hábitos`.

## Formato dos arquivos

A weeklist aceita o formato completo, só tempo, só prioridade ou texto livre:

```markdown
- [ ] 10 min (I) - Task A
- [ ] 20 min (U) - Task B
- [ ] 5/15 min (IU) - Ler
- [ ] 30 min - Revisar
- [ ] (I) - Responder mensagens
- [ ] Texto livre
- [ ] 5 min () - Fio dental #always
```

- `5/15 min` representa uma faixa de 5 a 15 minutos.
- Prioridades: `I` (importante), `U` (urgente) ou `IU` (ambos).
- `ST` e `xST` pedem que a duração seja escolhida durante o wizard.
- `#always` no fim de um hábito faz o item entrar automaticamente em toda
  daylist da semana e ser copiado para a próxima weeklist. Use apenas em
  hábitos; tasks não recebem esse marcador.
- Itens marcados como `[x]` não entram na seleção.
- O parser é tolerante a espaços e a variações de caixa em tempos e
  prioridades.

## Desenvolvimento

Alvos disponíveis no `Makefile`:

```sh
make help       # lista os alvos
make build      # gera ./daylist com versão embutida
make install    # instala com go install . e versão embutida
make test       # go test ./...
make test-race  # go test -race -cover ./...
make vet        # go vet ./...
make fmt        # gofmt -w .
make fmt-check  # falha se houver arquivo fora do gofmt
make lint       # fmt-check + vet
make cover      # gera coverage.out e coverage.html
make ci         # fmt-check + vet + test-race + build
make clean      # remove binário e artefatos de cobertura
```

O alvo `lint` é intencionalmente autocontido: usa `gofmt` e `go vet`, sem exigir
`golangci-lint`.

Itens adiados e ideias futuras ficam em [docs/backlog.md](docs/backlog.md).

Estrutura resumida:

```text
cmd/                  comandos Cobra
internal/daylist/     parser, stages e renderização da daylist
internal/week/        semanas ISO, weeklist e feedback
internal/timer/       estado do timer e marcação ⏱
internal/wizard/      fluxo por etapas, voltar e tempo total
internal/planner/     monta a daylist em memória para prévia e grava sob confirmação
internal/ui/          UI nativa Bubble Tea (Pick, Toggle, Input, Confirm) + fallback numerado
internal/theme/       estilos compartilhados entre TUI e wizard
internal/tui/         modelo Bubble Tea, backend, formulários e telas
internal/vault/       caminhos, validação e abertura do Obsidian
docs/                 roadmap e documentação
scripts/              scripts JS históricos do QuickAdd
```

Os scripts `scripts/weeklist_script.js` e `scripts/feedback_script.js` são
referência histórica do fluxo antigo e não fazem parte da geração atual.

## Limitações atuais

- A abertura da nota usa `xdg-open`; macOS e WSL ainda não são suportados.
- O vault ainda não é autodetectado: use `DAYLIST_VAULT_PATH` e
  `DAYLIST_VAULT_NAME`.
- A etapa `Extra` continua fora do wizard; a inclusão de Extra é feita pelo TUI.
- A ação “Fechar o dia” é somente um resumo: ainda não copia pendentes nem
  reorganiza o dia seguinte.
