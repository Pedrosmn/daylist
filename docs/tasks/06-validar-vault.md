# Task 6 — Validar o caminho do vault (opcional)

**Roadmap:** correção 6 (citada na linha “timer + semana alvo + validação do vault” da ordem sugerida) · **Depende de:** nada

## Problema

`internal/vault/vault.go:28` (`Load`) aceita qualquer caminho. Um typo em `DAYLIST_VAULT_PATH` produz lá na frente um erro enganoso de “weeklist não encontrada”, sem indicar que o vault em si está errado.

## Mudança

Em `internal/vault/vault.go` (`Load`), depois de montar o caminho:

```go
info, err := os.Stat(path)
if errors.Is(err, os.ErrNotExist) || (err == nil && !info.IsDir()) {
    return nil, fmt.Errorf("vault não encontrado em %q (ajuste DAYLIST_VAULT_PATH)", path)
}
if err != nil {
    return nil, err
}
```

- Validar apenas em `Load`, sem mudar a assinatura.
- Mensagem acionável, citando a variável de ambiente.
- Não criar o diretório automaticamente.

## Testes (`internal/vault/vault_test.go`, novo)

- `TestLoadRejectsMissingVault`: `DAYLIST_VAULT_PATH` apontando para caminho inexistente → erro contendo o caminho e `DAYLIST_VAULT_PATH`.
- `TestLoadRejectsFile`: caminho que é arquivo, não diretório → erro.
- `TestLoadAcceptsDir`: `t.TempDir()` → sem erro.
- Aproveitar o arquivo para `TestISOWeek` (virada de ano: 2025-12-29 → `2026-W01`) e `TestWeeklistFile` (montagem do caminho).

## Critérios de aceite

- Vault inválido falha cedo, com mensagem que diz o que ajustar.
- Nenhum comando tenta abrir o Obsidian com caminho inválido.

## Verificação

```bash
DAYLIST_VAULT_PATH=/caminho/errado go run . week
go test ./internal/vault/...
```
