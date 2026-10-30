# Task 6 — Limpeza do QuickAdd e documentação

**Depende de:** tasks 3 e 5 · **Bloqueia:** nada
**Arquivos:** `internal/vault/vault.go`, `cmd/legacy.go`, `cmd/legacy_test.go`, `cmd/week.go`, `cmd/feedback.go`, `docs/roadmap.md`

## Objetivo

Remover o caminho antigo do QuickAdd, limpar arquivos de alvo legados e refletir a migração no roadmap.

## Mudança

### 1. Limpeza best-effort dos alvos legados

Criar `cmd/legacy.go`:

```go
const (
    legacyWeeklistTarget = ".daylist-weeklist-target.json"
    legacyFeedbackTarget = ".daylist-feedback-target.json"
)

// cleanupLegacyTargets remove os arquivos usados pelo fluxo antigo do
// QuickAdd. Erros são ignorados: são apenas resíduos de migração.
func cleanupLegacyTargets(v *vault.Vault) {
    for _, name := range []string{legacyWeeklistTarget, legacyFeedbackTarget} {
        _ = os.Remove(filepath.Join(v.Path, name))
    }
}
```

Chamar `cleanupLegacyTargets(v)` **somente** no caminho que grava (depois de `--print-path`/`--dry-run` e da checagem de arquivo existente), para preservar a promessa de que esses modos não têm efeitos colaterais.

### 2. Remover o código morto de QuickAdd/targets

Em `internal/vault/vault.go`, remover se não houver mais usos:

- `Vault.QuickAdd`
- `Vault.WeeklistTargetFile`
- `Vault.FeedbackTargetFile`
- `WriteTarget`
- `ClearTarget`
- imports que ficarem sem uso (provavelmente `encoding/json`)

Manter `OpenNote` e `openURL`; o Obsidian continua sendo usado para abrir notas.

### 3. Atualizar o roadmap

Em `docs/roadmap.md`:

- tabela de comandos:
  - `daylist week` → “Gerado em Go, sem QuickAdd”
  - `daylist feedback` → “Gerado em Go, sem QuickAdd”
- seção “🚀 Maior ganho: portar `week` e `feedback` para Go” → marcar como concluída e listar os comandos entregues; registrar que a weeklist agora copia hábitos **preservando tempo e prioridade** e aceita formatos parciais, resetando apenas o checkbox.
- ordem sugerida: item 3 ✅ com data.
- observar que `scripts/*.js` continuam no repositório apenas como referência/fallback.

## Testes (`cmd/legacy_test.go`)

- `TestCleanupLegacyTargets`: criar os dois arquivos no vault temporário; chamar o helper; conferir que sumiram.
- `TestCleanupLegacyTargetsIgnoresMissing`: chamar num vault sem os arquivos; não deve entrar em pânico nem retornar erro.
- `TestRunWeekCleansLegacyTarget`: criar os dois alvos e uma weeklist de origem; rodar `runWeek`; conferir que os alvos sumiram.
- `TestRunWeekDryRunKeepsLegacyTargets`: rodar `--dry-run`; conferir que os alvos **não** foram removidos (dry-run não tem efeitos colaterais).
- `TestRunFeedbackCleansLegacyTarget`: análogo para feedback.

## Verificação de código morto

```bash
rg 'QuickAdd|WriteTarget|ClearTarget|WeeklistTargetFile|FeedbackTargetFile' --glob '*.go'
```

Esperado: nenhuma ocorrência em código Go (só nos scripts/docs, se citados).

## Critérios de aceite

- `week` e `feedback` não dependem de QuickAdd nem criam arquivos de alvo.
- Alvos legados são removidos no primeiro uso real dos comandos.
- `--dry-run` e `--print-path` continuam sem efeitos colaterais.
- Roadmap reflete os comandos novos.
- Nenhum import ou função morta sobra em `internal/vault`.

## Verificação final

```bash
gofmt -l .
go vet ./...
go test ./... -race -cover
go build ./...

export DAYLIST_VAULT_PATH=$(mktemp -d)
mkdir -p "$DAYLIST_VAULT_PATH/Fleeting Notes/Daily/weeklists/2026-W44"
printf '# Tasks\n\n---\n# Hábitos\n## Saúde\n- [ ] 10 min (I) - correr\n' \
  > "$DAYLIST_VAULT_PATH/Fleeting Notes/Daily/weeklists/2026-W44/weeklist.md"

daylist week --current --dry-run
daylist week --next
daylist week --next            # já existe
daylist feedback --week 2026-W44 --dry-run
daylist feedback --week 2026-W44
```

Conferir também que nenhum `.daylist-weeklist-target.json`/`.daylist-feedback-target.json` foi criado e que os arquivos antigos sumiram.
