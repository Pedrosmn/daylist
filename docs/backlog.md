# Daylist — Backlog

Itens fora do escopo atual. Não são compromissos e não bloqueiam o roadmap.

## Próximo ciclo — TUI e metadados

### Decisões

- `daylist` sem argumentos abre o TUI quando houver terminal interativo.
- `daylist open` abre a daylist de hoje no Obsidian, preservando o comportamento
  atual de abertura.
- O marcador `#always` é exclusivo de hábitos da weeklist: tasks são feitas uma
  vez e não usam esse marcador.
- Hábitos com `#always` são copiados para a próxima weeklist com o marcador
  preservado.
- Toda daylist daquela semana já inclui automaticamente os hábitos `#always`,
  com checkbox `[ ]` desmarcado. Eles não dependem de seleção manual e não podem
  ser removidos no wizard.
- Motivação existe somente na daylist, por item, em blockquote.
- Extra pode ser task ou hábito:
  - task Extra vai para `# Extra`;
  - hábito Extra é escolhido entre os hábitos da weeklist e entra na categoria
    original em `# Hábitos`, para contar no feedback normalmente.
- O TUI reutiliza o wizard atual para criar a daylist.

### 1. Metadados da weeklist

- [x] W1. Reconhecer `#always` apenas em hábitos, sem deixar o marcador na
  descrição interpretada; tasks tratam o texto normalmente.
- [x] W2. Fazer `BuildWeeklist` preservar `#always` nos hábitos copiados para a
  semana seguinte, mantendo tempo, prioridade e o checkbox pendente. Tasks
  continuam sem ser copiadas.
- [x] W3. Fazer o wizard incluir automaticamente todos os hábitos `#always` em
  toda daylist da semana, sem marcação manual e sem opção de remover.
- [x] W4. Cobrir W1–W3 com testes, incluindo CRLF, hábitos sem `#always` e
  garantia de que tasks não são copiadas.

### 2. Modelo da daylist

- [x] D1. Criar parser de nota da daylist com seções, categorias, itens
  pendentes/concluídos, linha original, sufixo `⏱`, motivação e `# Extra`.
- [x] D2. Criar operações atômicas para marcar/desmarcar item e editar/remover
  motivação, preservando CRLF e conteúdo não reconhecido.
- [x] D3. Criar inserção de Extra:
  - task livre em `# Extra`;
  - hábito escolhido da weeklist na categoria correspondente de `# Hábitos`;
  - evitar duplicar um hábito que já está na daylist.
- [x] D4. Garantir que o feedback conte hábitos extras naturalmente por estarem
  em `# Hábitos`, sem categoria artificial.
- [x] D5. Cobrir D1–D4 com testes.

### 3. TUI

- [x] T1. Adicionar Bubble Tea/Bubbles/Lipgloss e a estrutura `internal/tui`.
- [x] T2. Fazer `daylist` abrir o TUI com TTY e criar `daylist open`.
- [x] T3. Exibir a daylist de hoje com tasks, hábitos, Extra, concluídos,
  motivações e status do timer.
- [x] T4. Criar a daylist de hoje reutilizando o wizard com suspensão do TUI.
- [x] T5. Criar a weeklist atual quando estiver ausente.
- [x] T6. Alternar item concluído/pendente pelo TUI.
- [x] T7. Iniciar, pausar, retomar e parar o timer pelo TUI, com aviso de
  mínimo/máximo compartilhado com o CLI.
- [x] T8. Adicionar Extra task/hábito:
  - task: formulário com descrição, tempo, prioridade e motivação;
  - hábito: escolher da weeklist e inserir na categoria original.
- [x] T9. Editar motivação de item da daylist.
- [x] T10. Ver a weeklist atual e alternar o marcador `#always` apenas em
  hábitos.
- [x] T11. Tela de ajuda, mensagens de erro e confirmações.
- [x] T12. Cobrir o modelo TUI com backend falso.
- [x] T13. Mostrar uma prévia antes de criar a daylist:
  - hábitos `#always` que já entrarão automaticamente;
  - total de tempo planejado;
  - alerta quando o dia já nascer acima de um limite configurável.
- [x] T14. Adicionar undo no TUI para desfazer a última ação de edição.
- [x] T15. Adicionar ação “Fechar o dia” com resumo de planejado vs. real,
  concluídos e pendentes. O fechamento é somente leitura: não copia pendentes
  nem altera a nota.
- [x] T16. Mostrar a motivação do item ao destacá-lo e ao iniciar o timer.
- [x] T17. Criar a daylist de amanhã pelo TUI com `N`, reutilizando o wizard e
  a prévia antes de gravar.

### 4. Documentação e validação

- [x] DOC1. Atualizar o README com TUI, `open`, `#always`, motivação, Extra,
  prévia da daylist, undo e fechamento do dia.
- [x] DOC2. Atualizar o roadmap com o próximo ciclo.
- [x] DOC3. Rodar `make ci` e testar o TUI manualmente em terminal real.

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
