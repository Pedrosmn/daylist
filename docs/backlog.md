# Daylist — Backlog

Itens fora do escopo atual. Não são compromissos e não bloqueiam o roadmap.

## Ciclo entregue — TUI e metadados

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

## Ciclo entregue — Wizard em Bubble Tea (substitui o fzf)

O wizard (`daylist new`, `daylist open`, `n`/`N` do TUI e `timer start`) usava
fzf. O visual não acompanhava o TUI e o fzf era reaberto a cada marcação, o que
causava piscadas. Este ciclo moveu as interações para Bubble Tea/Lipgloss,
reaproveitando o tema do TUI, e aposentou o fzf. O fluxo por etapas, o
`Prompter` do wizard e o fallback sem TTY continuam iguais.

### Decisões

- O wizard usa a mesma linguagem visual do TUI: `j/k`, `g/G`,
  `Ctrl-d/Ctrl-u`, `Enter/Espaço`, `h/l/q`, cabeçalho com etapa e contador e
  ajuda no rodapé.
- O fzf deixa de ser usado e deixa de ser requisito. O menu numerado permanece
  como fallback quando `stdin`/`stderr` não forem TTY (testes, pipes e CI).
- A UI do wizard desenha em `stderr`; o `stdout` fica limpo para o `--dry-run`.
- Os prompts usam renderização inline (sem alt-screen): cada etapa deixa um
  resumo compacto e a próxima desenha abaixo, preservando o histórico do
  terminal.
- O tema (cores, estilos e marcadores) é compartilhado entre TUI e wizard.
- O TUI continua criando a daylist pelo wizard, via `tea.Exec`. O wizard é uma
  tela separada; unificar tudo em uma única tela do TUI fica para depois.
- Seleção no wizard não usa o estilo de item concluído (tachado): são conceitos
  diferentes.

### 1. Base compartilhada

- [x] B1. Extrair `internal/tui/styles.go` para `internal/theme`, com os estilos
  exportados, e deixar `internal/tui/styles.go` como alias para não mexer em
  `view.go`. Critério: nenhuma mudança visual e `make ci` verde.
- [x] B2. Criar em `internal/ui` um modelo de lista em Bubble Tea reutilizável
  por Toggle e Pick, com cursor, janela de itens visíveis, cabeçalho e rodapé de
  ajuda. Critério: navegação por `j/k`, `↑/↓`, `g/G`, `home/end` e
  `Ctrl-d/Ctrl-u`, sem quebrar listas longas.

### 2. Telas do wizard

- [x] B3. Implementar `Toggle` nativo com `▶`, `[x]/[ ]`, `(sempre)` e itens
  travados, contador de marcados, `AllowBack`, `Header` e as ações
  `ToggleFinish`, `ToggleBack` e `ToggleCancel`.
- [x] B4. Implementar `Pick` nativo de seleção única com prompt, `Enter`
  escolhe e `q/Esc` cancela; usar em data alvo, duração, prioridade e
  `timer start`.
- [x] B5. Implementar `Input` com `bubbles/textinput` (trim e validação
  continuam no wizard) e `Confirm` com `s`/`n`/`Esc`, preservando o padrão
  seguro (“Não, continuar”).
- [x] B6. Ao concluir ou cancelar cada prompt, deixar na tela apenas um resumo
  compacto (ex.: `✓ Etapa 1/3 · # Tasks — 2 selecionados`) em vez da lista
  inteira; listas longas não devem inflar o scrollback.

### 3. Integração e limpeza

- [x] B7. Selecionar a UI nativa só quando `stdin` e `stderr` forem TTY, com
  `tea.WithInput(os.Stdin)` e `tea.WithOutput(os.Stderr)`; caso contrário,
  manter `pickPlain`, `togglePlainOptions` e `Input` atuais. Critério: testes e
  pipes continuam funcionando sem TTY.
- [x] B8. Validar o wizard aninhado sob `tea.Exec` no TUI (`n`/`N`): raw mode,
  redimensionamento, `Ctrl+C`, retorno à tela do TUI e prévia da daylist.
- [x] B9. Remover `pickFzf`, `toggleFzf`, `fzfLayoutArgs`, `fzfNavArgs`,
  `fzfIgnoreArg`, `fzfCanceled` e os testes correspondentes; remover imports e
  dependências que ficarem sem uso.
- [x] B10. Cobrir os modelos novos com testes: navegação e limites, marcação e
  itens travados, seleção inicial, voltar/concluir/cancelar, pick, confirm,
  input e janela de itens.
- [x] B11. Atualizar `README.md`: remover o fzf dos requisitos, documentar a UI
  nativa do wizard, manter a tabela de teclas e explicar o fallback numerado
  sem TTY.
- [x] B12. Rodar `make ci` e validar manualmente: `daylist new`,
  `daylist timer start`, `n`/`N` no TUI, `--dry-run` com stdout redirecionado,
  lista longa, terminal estreito, `Ctrl+C` e execução sem TTY.

### Ordem de execução

`B1 → B2 → B3/B4/B5 → B6/B7 → B8 → B9 → B10 → B11/B12`.

### Adiado

- Wizard inteiro em uma única tela alt-screen do TUI, sem `tea.Exec`, com o
  resumo das etapas dentro do modelo. Só faz sentido depois que a migração
  básica estiver estável.
- Reintroduzir o fzf como opção via variável de ambiente, caso alguém prefira.

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
