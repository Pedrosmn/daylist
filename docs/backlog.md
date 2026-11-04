# Daylist — Backlog

Itens fora do escopo atual. Não são compromissos e não bloqueiam o roadmap.

## Vault

- Autodetectar o vault do Obsidian, lendo os vaults registrados pelo aplicativo
  (Flatpak, Snap e instalações nativas).
- Adicionar `--vault` como override explícito.

## Wizard

- Etapa Extra no wizard.

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
