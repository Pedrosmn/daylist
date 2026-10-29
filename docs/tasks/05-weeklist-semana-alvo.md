# Task 5 — Wizard usa a weeklist da semana alvo

**Roadmap:** correção 5 · **Depende de:** nada

## Problema

`cmd/new.go:109` usa sempre `vault.ISOWeek(time.Now())`:

- `daylist new --date 2026-11-02` (outra semana) lê a weeklist desta semana;
- criar a daylist de amanhã no domingo (virada de semana ISO) usa a weeklist da semana anterior à correta.

## Mudança

### `cmd/new.go`

1. Criar helper puro e testável:

   ```go
   // weeklistForTarget devolve a weeklist da semana da data alvo; se não
   // existir, cai para a semana anterior.
   func weeklistForTarget(v *vault.Vault, target time.Time) (path, week string, fallback bool, err error)
   ```

2. Candidatas, nesta ordem:

   - semana de `target` → `vault.ISOWeek(target)`;
   - semana de `target.AddDate(0, 0, -7)` → `vault.ISOWeek(...)`.

3. Usar a primeira em que `os.Stat(v.WeeklistFile(week)) == nil`.

4. Se for fallback, avisar no stdout:
   `Weeklist de <semana-alvo> não encontrada; usando a de <semana-anterior>.`

5. Se nenhuma existir, erro com os caminhos tentados e sugestão:
   `rode "daylist week" para criar a weeklist da semana`.

6. `createDaylist` passa `target` (não `time.Now()`) para o helper.

7. Ajustar mensagens que hoje falam em “semana atual” (`cmd/new.go:113` e `:121`) para a semana efetivamente usada e atualizar o texto `Long` do comando.

### Observações

- Para `target` de outra semana, a regra é a mesma: semana alvo → anterior.
- A semana anterior é obtida por aritmética de data (`-7 dias`), não por decremento do número da semana — assim a virada de ano é tratada pelo `ISOWeek`.
- O fallback de **hábitos** entre semanas no script do QuickAdd é um pedido separado e **não** entra aqui.

## Testes (`cmd/new_test.go`, novo)

Estratégia: `vault.Vault{Path: t.TempDir()}` e criar `Fleeting Notes/Daily/weeklists/<semana>/weeklist.md`.

- `TestWeeklistForTargetUsesTargetWeek`
- `TestWeeklistForTargetFallsBackToPreviousWeek`
- `TestWeeklistForTargetNone` (erro cita os caminhos tentados)
- `TestWeeklistForTargetSundayToMonday` (virada de semana ISO)
- `TestWeeklistForTargetYearBoundary` (ex.: 2025-12-29 → `2026-W01`)

## Critérios de aceite

- `--date` de outra semana usa a weeklist daquela semana.
- Domingo → segunda-feira usa a weeklist da semana nova (ou da anterior, se ainda não existir).
- Sem nenhuma weeklist, mensagem clara e nenhum arquivo gravado.
- Fallback sempre visível no output.

## Verificação manual

```bash
daylist new --date 2026-11-02 --dry-run   # deve ler a weeklist de 2026-W45
daylist new --tomorrow --dry-run          # domingo: deve ler a weeklist da semana nova
```
