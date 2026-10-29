# Task 1 — `Save` do estado do timer atômico

**Roadmap:** correção 3 · **Depende de:** nada · **Bloqueia:** task 3

## Problema

`internal/timer/timer.go:66` (`State.Save`) grava com `os.WriteFile` direto. Crash ou falta de energia no meio da escrita deixa o JSON corrompido; o próximo `timer.Load` falha com `estado do timer inválido` e o `timer start` fica bloqueado até alguém apagar o arquivo manualmente.

## Mudança

Em `internal/timer/timer.go`:

1. Criar helper local no pacote `timer` (evita depender de `internal/daylist`):

   ```go
   func writeAtomic(path string, data []byte) error {
       tmp := path + ".tmp"
       if err := os.WriteFile(tmp, data, 0o644); err != nil {
           return err
       }
       if err := os.Rename(tmp, path); err != nil {
           os.Remove(tmp)
           return err
       }
       return nil
   }
   ```

2. `Save` passa a usar `writeAtomic(path, data)`.

3. Considerar também os outros pontos que gravam estado/timer no mesmo arquivo, se houver, para manter o padrão.

## Testes (`internal/timer/timer_test.go`, novo)

- `TestSaveAtomic`: após `Save`, o arquivo é JSON válido e não sobra `*.tmp` no diretório.
- `TestSaveKeepsPreviousOnFailure`: forçar falha (ex.: diretório inexistente ou path somente-leitura) e garantir que o arquivo anterior permanece intacto.
- `TestSaveLoadRoundTrip`: `Save` → `Load` devolve os mesmos campos.

## Critérios de aceite

- `Save` nunca deixa JSON truncado no caminho final.
- Em caso de erro, o estado anterior continua legível.
- Nenhum `.tmp` sobra após sucesso.

## Verificação

```bash
go test ./internal/timer/...
```
