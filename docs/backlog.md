# Daylist — Backlog

Itens fora do escopo atual. Não são compromissos e não bloqueiam o roadmap.

## Próximo ciclo — TUI e metadados

### Decisões

- `daylist` sem argumentos abre o TUI quando houver terminal interativo.
- `daylist open` abre a daylist de hoje no Obsidian, preservando o comportamento
  atual de abertura.
- Itens “sempre” podem ser tasks ou hábitos da weeklist. Eles ficam
  pré-selecionados em toda daylist e são copiados para a weeklist seguinte.
- O marcador de “sempre” é visível: `#always` no fim da linha.
- Motivação existe somente na daylist, por item, em blockquote.
- Extra pode ser task ou hábito:
  - task Extra vai para `# Extra`;
  - hábito Extra é escolhido entre os hábitos da weeklist e entra na categoria
    original em `# Hábitos`, para contar no feedback normalmente.
- O TUI reutiliza o wizard atual para criar a daylist.

### 1. Metadados da weeklist

- [ ] W1. Reconhecer `#always` no fim de uma task ou hábito, sem deixar o
  marcador na descrição interpretada.
- [ ] W2. Fazer o wizard iniciar com os itens `#always` pré-selecionados,
  permitindo desmarcar.
- [ ] W3. Fazer `BuildWeeklist` copiar itens `#always` de `# Tasks` e de
  `# Hábitos` para a semana seguinte, preservando tempo, prioridade e o
  marcador; o checkbox volta para pendente.
- [ ] W4. Cobrir W1–W3 com testes, incluindo CRLF e itens sem `#always`.

### 2. Modelo da daylist

- [ ] D1. Criar parser de nota da daylist com seções, categorias, itens
  pendentes/concluídos, linha original, sufixo `⏱`, motivação e `# Extra`.
- [ ] D2. Criar operações atômicas para marcar/desmarcar item e editar/remover
  motivação, preservando CRLF e conteúdo não reconhecido.
- [ ] D3. Criar inserção de Extra:
  - task livre em `# Extra`;
  - hábito escolhido da weeklist na categoria correspondente de `# Hábitos`;
  - evitar duplicar um hábito que já está na daylist.
- [ ] D4. Garantir que o feedback conte hábitos extras naturalmente por estarem
  em `# Hábitos`, sem categoria artificial.
- [ ] D5. Cobrir D1–D4 com testes.

### 3. TUI

- [ ] T1. Adicionar Bubble Tea/Bubbles/Lipgloss e a estrutura `internal/tui`.
- [ ] T2. Fazer `daylist` abrir o TUI com TTY e criar `daylist open`.
- [ ] T3. Exibir a daylist de hoje com tasks, hábitos, Extra, concluídos,
  motivações e status do timer.
- [ ] T4. Criar a daylist de hoje reutilizando o wizard com suspensão do TUI.
- [ ] T5. Criar a weeklist atual quando estiver ausente.
- [ ] T6. Alternar item concluído/pendente pelo TUI.
- [ ] T7. Iniciar, pausar, retomar e parar o timer pelo TUI, com aviso de
  mínimo/máximo compartilhado com o CLI.
- [ ] T8. Adicionar Extra task/hábito:
  - task: formulário com descrição, tempo, prioridade e motivação;
  - hábito: escolher da weeklist e inserir na categoria original.
- [ ] T9. Editar motivação de item da daylist.
- [ ] T10. Ver a weeklist atual e alternar o marcador `#always`.
- [ ] T11. Tela de ajuda, mensagens de erro e confirmações.
- [ ] T12. Cobrir o modelo TUI com backend falso.

### 4. Documentação e validação

- [ ] DOC1. Atualizar o README com TUI, `open`, `#always`, motivação, Extra e
  feedback.
- [ ] DOC2. Atualizar o roadmap com o próximo ciclo.
- [ ] DOC3. Rodar `make ci` e testar o TUI manualmente em terminal real.

## Vault

- Autodetectar o vault do Obsidian, lendo os vaults registrados pelo aplicativo
  (Flatpak, Snap e instalações nativas).
- Adicionar `--vault` como override explícito.

## Timer

- Estado em `$XDG_STATE_HOME/daylist/`, fora do vault.
- `timer watch` para notificações contínuas sem depender de `timer status`.
- Auto-stop na virada do dia ou após um limite de horas; a decisão atual é
  apenas sinalizar mínimo e máximo.

## Produto

- `daylist new --carry`: trazer tasks não feitas de ontem para a daylist de hoje.
- Marcar o item correspondente na weeklist ao dar `timer stop`; ou
  `daylist week sync`.
- `daylist stats week`: tempo por categoria e taxa de conclusão.
- Streaks por hábito.
- `daylist today` / `daylist tomorrow --dry-run` para pré-visualizar.

## Fora de escopo por enquanto

- Suporte a macOS e WSL: Linux-only por decisão atual.
