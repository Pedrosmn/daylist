# Daylist — Backlog

Itens fora do escopo atual. Não são compromissos e não bloqueiam o roadmap.

## Ciclo entregue — Resumo e feedback da semana no TUI (2026-11-08)

Status: `RS1–RS9` concluídos em 2026-11-09, com `make ci` verde e validação
manual do TUI em pty.

Levar o feedback para o TUI e permitir acompanhar o resumo da semana durante a
semana, sem esperar o domingo. O `feedback.md` continua sendo a exportação; a
linha compacta e a tela do TUI são leituras do mesmo cálculo.

### Decisões

- Duas camadas: uma linha compacta na tela principal (acompanhamento contínuo)
  e a tela `S` com o resumo completo.
- A linha e a tela usam a mesma conta do `daylist feedback`: itens `[x]` sobre
  ocorrências de hábitos nas daylists. A semana atual conta até hoje; semanas
  passadas contam de segunda a domingo; daylists futuras são ignoradas.
- O percentual vem acompanhado de `dias com daylist` e de um recorte por dia,
  para não parecer 100% no começo da semana.
- Hábitos da weeklist sem ocorrências aparecem como `0/0 (—)` ou em uma seção
  de não usados; eles não alteram o total.
- A semana exibida é a da data visualizada no TUI (`m.day`); `y`/ontem usa a
  semana correspondente.
- O feedback da última semana encerrada é gerado automaticamente ao encerrar a
  semana: na virada domingo→segunda com o TUI aberto ou na primeira abertura a
  partir de segunda.
- O automático age apenas na última semana encerrada, somente se a weeklist
  existir e o `feedback.md` não existir. Nunca sobrescreve e não faz backfill.
- O resumo é somente leitura neste ciclo; não há atalho `g` para o feedback. A
  geração manual continua em `daylist feedback`.
- Tempo por categoria, comparação com a semana anterior, streaks e
  `feedback --progress` ficam para depois.

### 1. Cálculo compartilhado

- [x] RS1. Extrair `ComputeReport` em `internal/week` com categorias, hábitos,
  totais e estatísticas por dia; fazer `BuildFeedback` renderizar a partir
  dessa representação mantendo a saída atual idêntica. Critério: testes
  existentes verdes e novas coberturas para `ComputeReport`.
- [x] RS2. Adicionar `DayStat` e percentuais por hábito/categoria/semana; cobrir
  ordem de categorias e hábitos, CRLF, `#always`, sufixo `⏱`, hábitos da
  weeklist sem ocorrências e dias sem daylist. Critério: o cálculo do TUI e do
  CLI é o mesmo, verificado por testes.

### 2. Dados no TUI

- [x] RS3. Adicionar `WeekReportOn(day)` ao backend do TUI: lê a weeklist da
  semana da data e as daylists da semana, até hoje na semana atual e até domingo
  nas semanas passadas, ignorando ausentes e futuras. Critério: a data
  visualizada define a semana e a weeklist ausente é distinguível.
- [x] RS4. Adicionar `FeedbackExistsOn(day)` e
  `AutoGeneratePreviousFeedback(today)` no backend real: gera apenas a última
  semana encerrada, se a weeklist existir e o `feedback.md` não existir; nunca
  sobrescreve e não faz backfill. Critério: idempotente e coberto com vault
  temporário.

### 3. Interface

- [x] RS5. Exibir uma linha compacta na tela principal com barra, percentual,
  hábitos, dias com daylist e o recorte de hoje, adaptada à largura. Critério:
  atualiza ao marcar/desmarcar e na virada do dia; fica oculta se a weeklist não
  existir.
- [x] RS6. Criar o modo `S` de resumo da semana, somente leitura: barra geral,
  hábitos por categoria, recorte por dia, estado do `feedback.md`, destaque de
  hoje, `—` para ausente e `·` para futuro. Critério: `S`/`esc` navegam; semana
  atual mostra o parcial até hoje; semana passada mostra os sete dias; weeklist
  ausente mostra mensagem clara; terminal estreito não quebra.
- [x] RS7. Disparar o automático no `New` do TUI e na virada do dia
  (domingo→segunda), com mensagem discreta de sucesso. Critério: gera uma vez,
  não gera para a semana atual nem para semanas antigas e não bloqueia o uso.

### 4. Testes e documentação

- [x] RS8. Cobrir RS1–RS7 com testes de `internal/week`, backend real com
  `t.TempDir`, modelo do TUI (linha, `S`, rollover, ontem, weeklist ausente e
  automático) e view (barras, 0/0, sem dados e terminal estreito).
- [x] RS9. Atualizar README e ajuda do TUI (tecla `S`, linha da semana,
  automático ao encerrar a semana e ausência de `g`), roadmap e este backlog;
  rodar `make ci` e validar manualmente.

### Ordem de execução

`RS1 → RS2 → RS3 → RS4 → RS5 → RS6 → RS7 → RS8 → RS9`.

### Adiado

- Tempo planejado/real por categoria (`daylist stats week`).
- Comparação com a semana anterior e streaks por hábito.
- `daylist feedback --progress` para prompt/status bar.
- Backfill de feedbacks de semanas antigas.
- Atalho manual para gerar o feedback no TUI; se necessário, entra depois do
  automático. O `g`/`G` continua apenas como navegação da lista.

## Ciclo entregue — Ajustes de UX do TUI (2026-11-07)

Ajustes de uso diário no TUI, sem mudar o formato das notas nem o fluxo do
wizard. Status: `M1–M6`, `P1–P3`, `N1–N5`, `Y1–Y6` e `DOC1–DOC3` concluídos em
2026-11-08. A ordem sugerida está no fim da seção.

### Decisões

- A tecla `m` substitui `e` para editar ou remover a motivação. `e` deixa de
  existir, inclusive na ajuda e no README.
- O formulário de motivação mostra o nome do item no título e não repete o
  rótulo “Motivação”. O rodapé é contextual: só exibe teclas que funcionam
  naquele formulário.
- O campo de motivação continua sendo uma linha no modelo, mas com wrap visual:
  ao chegar na largura disponível o texto desce uma linha em vez de rolar
  lateralmente. `Enter` salva; `Esc` cancela.
- A motivação na tela principal também quebra pela largura do terminal, sem
  cortar o texto.
- Quando houver timer rodando, a tela principal mostra uma barra de progresso
  do tempo decorrido até o máximo planejado, com uma marcação no mínimo. Sem
  faixa de tempo conhecida, a barra não é exibida.
- A notificação de mínimo/máximo passa a funcionar com o TUI fechado por meio
  do daemon `daylist timer watch`, iniciado automaticamente e com instância
  única. Ele notifica uma vez por limite e encerra quando não houver timer.
- O TUI ganha a daylist de ontem: `y` alterna entre hoje e ontem; é possível
  ver o conteúdo, marcar/desmarcar itens e adicionar Extra. Timer e edição de
  motivação ficam desabilitados em ontem.
- Navegar por qualquer data e editar motivação em dias passados ficam adiados.

### 1. Motivação

- [x] M1. Trocar o atalho de motivação de `e` para `m` no TUI, na tela de
  ajuda, no README e nos testes. Critério: `m` abre o formulário; `e` não faz
  mais nada.
- [x] M2. Mostrar a descrição do item no título do formulário (por exemplo,
  `Motivação · 15 min (U) - AWS`) e remover o rótulo “Motivação” duplicado.
  Critério: o formulário identifica claramente qual item está sendo editado.
- [x] M3. Deixar o rodapé do formulário contextual: no formulário de motivação,
  exibir apenas `enter salva · esc cancela`; não mostrar `tab/↑↓ campo` nem
  `←/→ prioridade` quando esses controles não existem.
- [x] M4. Trocar o campo de linha única que rola lateralmente por um campo com
  wrap visual. Critério: ao digitar além da largura, o texto desce uma linha,
  o cursor acompanha, `Enter` salva e o valor gravado continua uma única linha.
  Aplicar o mesmo comportamento ao campo de motivação do Extra task.
- [x] M5. Quebrar a motivação exibida na tela principal pela largura
  disponível, descontando cursor e indentação. Critério: motivações longas e
  com múltiplas linhas aparecem completas, sem corte, inclusive em terminal
  estreito.
- [x] M6. Cobrir M1–M5 com testes de modelo e de renderização: atalho, título,
  rodapé contextual, wrap visual, `Enter`/`Esc` e valor persistido sem quebras.

### 2. Barra do timer

- [x] P1. Desenhar na tela principal, quando houver timer, uma barra do tempo
  decorrido até o máximo planejado, com marcação no mínimo e rótulos de mínimo,
  máximo e tempo atual. Critério: a barra acompanha o tick e usa a largura do
  terminal.
- [x] P2. Tratar os casos especiais: pausa (barra mostra o acumulado e indica o
  estado), mínimo igual ao máximo (marcador único no fim), ausência de faixa
  (barra oculta) e terminal estreito (barra mínima sem quebrar o layout).
- [x] P3. Cobrir o cálculo e a renderização da barra com testes: 0%, antes do
  mínimo, mínimo atingido, máximo atingido, pausa e larguras pequenas.

### 3. Notificação com o TUI fechado

- [x] N1. Criar `daylist timer watch`: lê o estado do timer, espera até o
  próximo limite e envia `notify-send` uma única vez por limite, usando os
  sinalizadores já persistidos em `MinNotified`/`MaxNotified`. Critério: sem
  TUI aberto, a notificação chega perto do mínimo e do máximo; o processo sai
  quando não há timer.
- [x] N2. Iniciar o watch automaticamente em `timer start` e ao iniciar o timer
  pelo TUI, em processo destacado. Critério: não abrir janela/terminal extra e
  não bloquear o comando.
- [x] N3. Garantir instância única (lock/arquivo de PID) e encerrar o watch no
  `timer stop`, ao limpar o estado ou ao detectar estado inexistente. Critério:
  não há notificações duplicadas nem processos órfãos.
- [x] N4. Respeitar pausa e retomada: timer pausado não notifica; ao retomar, o
  watch recalcula o próximo limite. Critério: pausar antes do mínimo não dispara
  notificação durante a pausa.
- [x] N5. Cobrir o watch com relógio injetável e estado falso: notificação
  única, mínimo, máximo, pausa, retomada, reinício e ausência de timer.
  Documentar que `notify-send` continua sendo o transporte no Linux.

### 4. Daylist de ontem

- [x] Y1. Separar a data visualizada da data de hoje no modelo do TUI. Critério:
  `reload` carrega a data selecionada e o cabeçalho indica claramente quando a
  tela não é a de hoje.
- [x] Y2. Adicionar o atalho `y` para alternar entre hoje e ontem, com `t`/`Esc`
  voltando explicitamente para hoje. Critério: se a daylist de ontem não
  existir, mostrar mensagem clara sem sair do TUI.
- [x] Y3. Permitir marcar/desmarcar itens e adicionar Extra na daylist de ontem,
  gravando sempre no arquivo daquela data. O Extra hábito usa a weeklist da
  semana da data visualizada; se ela não existir, avisar. Critério: o undo é
  isolado por dia e não grava conteúdo de ontem no arquivo de hoje.
- [x] Y4. Desabilitar timer e edição de motivação em ontem, com mensagem ou
  rodapé indicando a limitação. Critério: o banner e os controles do timer
  continuam se referindo apenas ao timer de hoje.
- [x] Y5. Tratar a virada do dia e o retorno para hoje sem perder cursor,
  seleção ou estado de edição. Critério: recarregar ontem depois da meia-noite
  não altera a data de hoje nem grava no arquivo errado.
- [x] Y6. Cobrir Y1–Y5 com testes de modelo: alternância de dia, arquivo de
  destino, toggle, Extra, undo por dia, timer desabilitado e virada do dia.

### 5. Documentação e validação

- [x] DOC1. Atualizar o README: tecla `m`, formulário de motivação com wrap,
  barra do timer, `timer watch`, atalho `y` e limitações da daylist de ontem.
- [x] DOC2. Atualizar o roadmap com o ciclo e o estado das tarefas.
- [x] DOC3. Rodar `make ci` e validar manualmente em terminal real: `m`,
  motivação longa, barra do timer, notificação com o TUI fechado, ontem com
  toggle/Extra e retorno para hoje. Validação feita em pty com vault
  temporário; o `notify-send` foi simulado para confirmar os avisos de mínimo
  e máximo com o TUI fechado.

### Ordem de execução

`M1 → M2/M3 → M4/M5 → M6 → P1 → P2/P3 → N1 → N2/N3 → N4/N5 → Y1 → Y2/Y3 → Y4/Y5 → Y6 → DOC1/DOC2/DOC3`.

### Adiado

- Navegar por qualquer data no TUI, em vez de apenas hoje/ontem.
- Editar motivação e remover itens em dias passados.
- Rodar o `timer watch` como serviço systemd de usuário; o daemon destacado é
  suficiente para o próximo ciclo.

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
- `timer watch` foi promovido ao ciclo planejado de UX do TUI; a notificação
  com o TUI fechado usa um daemon destacado e instância única.
- Auto-stop na virada do dia ou após um limite de horas; a decisão atual é
  apenas sinalizar mínimo e máximo.

## Produto

- `daylist new --carry`: trazer tasks não feitas de ontem para a daylist de hoje.
- Navegar por qualquer data no TUI foi adiado; o próximo ciclo cobre apenas
  hoje/ontem.
- Marcar o item correspondente na weeklist ao dar `timer stop`; ou
  `daylist week sync`.
- `daylist stats week`: tempo por categoria e taxa de conclusão.
- Streaks por hábito.
- `daylist today` / `daylist tomorrow --dry-run` para pré-visualizar.

## Fora de escopo por enquanto

- Suporte a macOS e WSL: Linux-only por decisão atual.
