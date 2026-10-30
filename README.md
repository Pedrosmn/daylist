# daylist

[![CI](https://github.com/Pedrosmn/daylist/actions/workflows/ci.yml/badge.svg)](https://github.com/Pedrosmn/daylist/actions/workflows/ci.yml)

CLI em Go para gerenciar daylists, weeklists, feedback e timer em um vault do
Obsidian. O fluxo principal é: gerar a weeklist, escolher as tasks e hábitos no
terminal, criar a daylist do dia e registrar o tempo com o timer.

## Requisitos

- Go 1.23.2 ou superior (versão declarada no `go.mod`).
- Obsidian instalado para abrir as notas pelo comando `daylist` sem subcomando.
- `fzf` opcional. Sem ele, o CLI usa menus numerados e os mesmos comandos de
  etapa (`h`, `l`, `q`).
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

A estrutura esperada dentro do vault é:

```text
Fleeting Notes/Daily/Days/YYYY-MM-DD.md
Fleeting Notes/Daily/weeklists/YYYY-Www/weeklist.md
Fleeting Notes/Daily/weeklists/YYYY-Www/feedback.md
.daylist-timer-state.json
```

## Comandos

### `daylist`

Sem subcomando, abre a daylist de hoje no Obsidian. Se a nota ainda não existir,
inicia o wizard e abre a nota depois da criação.

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

### `daylist week` (`w`)

Cria a weeklist da próxima semana, copiando os hábitos da semana anterior. Tempo
e prioridade são preservados; apenas o checkbox volta para pendente.

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

`status --prompt` imprime uma saída curta, útil para prompt do shell:

```sh
daylist timer status --prompt
# ⏱ 25m
```

### `daylist completion`

Gera scripts de completion do Cobra para o shell atual.

## Wizard

As etapas usam navegação no estilo Vim. O fzf fica sem busca digitável nesta
versão: a lista é percorrida apenas com movimento e marcação.

| Tecla | Ação |
|---|---|
| `j` / `k` | desce / sobe |
| `g` / `G` | vai ao início / fim |
| `Ctrl-d` / `Ctrl-u` | meia página para baixo / cima |
| `Enter` / `Espaço` | marca / desmarca o item destacado |
| `h` | volta para a etapa anterior |
| `l` | conclui a etapa |
| `q` / `Esc` | cancela tudo, com confirmação se houver seleção |

Ao voltar uma etapa, seleções, tempos e prioridades já definidos são
preservados. A duração personalizada é digitada fora do fzf, quando você escolhe
`Personalizado`.

Ao final, o tempo total planejado é exibido. Tempos compostos aparecem como
faixa, por exemplo:

```text
Tempo planejado: 1h45–1h55
```

## Formato dos arquivos

A weeklist aceita o formato completo, só tempo, só prioridade ou texto livre:

```markdown
- [ ] 10 min (I) - Task A
- [ ] 20 min (U) - Task B
- [ ] 5/15 min (IU) - Ler
- [ ] 30 min - Revisar
- [ ] (I) - Responder mensagens
- [ ] Texto livre
```

- `5/15 min` representa uma faixa de 5 a 15 minutos.
- Prioridades: `I` (importante), `U` (urgente) ou `IU` (ambos).
- `ST` e `xST` pedem que a duração seja escolhida durante o wizard.
- Itens marcados como `[x]` não entram na seleção.
- O parser é tolerante a espaços e a variações de caixa em tempos e
  prioridades.

## Desenvolvimento

Alvos disponíveis no `Makefile`:

```sh
make help       # lista os alvos
make build      # gera ./daylist
make install    # instala com go install .
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

Estrutura resumida:

```text
cmd/                  comandos Cobra
internal/daylist/     parser, stages e renderização da daylist
internal/week/        semanas ISO, weeklist e feedback
internal/timer/       estado do timer e marcação ⏱
internal/wizard/      fluxo por etapas, voltar e tempo total
internal/ui/          Pick, Toggle, Input e Confirm
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
- A etapa `Extra` do wizard está adiada.
- CI e os testes restantes de `cmd` estão no roadmap.
