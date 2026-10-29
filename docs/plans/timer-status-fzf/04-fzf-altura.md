# Task 4 — Altura e layout do fzf

**Depende de:** nada · **Bloqueia:** nada
**Arquivos:** `internal/ui/pick.go`, `internal/ui/pick_test.go`

## Problema

`pickFzf` e `toggleFzf` rodam o fzf sem `--height`, então cada etapa toma a tela inteira e apaga os resumos das etapas anteriores. O wizard fica mais difícil de revisar e comparar.

Além disso, hoje qualquer saída de erro do fzf é tratada como cancelamento. Ao introduzir novas flags, um fzf antigo que rejeite uma opção (exit 2) faria o wizard “cancelar” silenciosamente.

## Mudança

### 1. Layout comum

Em `internal/ui/pick.go`:

```go
// fzfLayoutArgs mantém a lista compacta e preserva o contexto da etapa anterior.
func fzfLayoutArgs() []string {
    return []string{"--height=70%", "--border", "--info=inline"}
}
```

Aplicar em `pickFzf`:

```go
args := append(fzfLayoutArgs(), "--prompt="+prompt)
cmd := exec.Command(path, args...)
```

E em `toggleFzf`, junto das flags atuais (`--no-multi`, `--no-sort`, `--delimiter`, `--with-nth`, `--prompt`, `--header`, `--bind`):

```go
args := append(fzfLayoutArgs(),
    "--no-multi",
    "--no-sort",
    "--delimiter=\t",
    "--with-nth=2..",
    "--prompt="+prompt,
    "--header=Enter marca/desmarca · Esc cancela",
    fmt.Sprintf("--bind=start:pos(%d)", pos),
)
cmd := exec.Command(path, args...)
```

O fallback `pickPlain`/`togglePlain` não muda.

### 2. Erro real do fzf não é cancelamento

Adicionar um discriminador:

```go
// fzfCanceled reconhece os códigos de cancelamento/sem match do fzf:
// 1 = sem match, 130 = interrompido. Outros erros (ex.: 2, opção inválida)
// devem ser propagados.
func fzfCanceled(err error) bool {
    var exitErr *exec.ExitError
    if !errors.As(err, &exitErr) {
        return false
    }
    switch exitErr.ExitCode() {
    case 1, 130:
        return true
    default:
        return false
    }
}
```

Usar em `pickFzf`:

```go
if err != nil {
    if fzfCanceled(err) {
        return "", nil
    }
    return "", err
}
```

E em `toggleFzf`:

```go
if err != nil {
    if fzfCanceled(err) {
        return nil, ErrCanceled
    }
    return nil, err
}
```

Assim, flags incompatíveis ou fzf quebrado aparecem como erro, em vez de fingir que o usuário cancelou.

### 3. Verificar versão mínima

As flags usadas existem há várias versões do fzf, mas `--info=inline` é a mais recente delas. Se o ambiente tiver uma versão muito antiga, o hardening acima transforma o problema em erro visível. Documentar no README (quando existir) que o fzf precisa suportar `--height`, `--border` e `--info=inline`.

## Testes (`internal/ui/pick_test.go`)

Criar um fzf falso em `t.TempDir()`:

```sh
#!/bin/sh
printf '%s\n' "$@" > "$FZF_FAKE_ARGS"
IFS= read -r line
printf '%s\n' "$line"
```

- `TestPickFzfLayoutFlags`: com o falso no `PATH`, chamar `Pick` e conferir no arquivo de args:
  - `--height=70%`
  - `--border`
  - `--info=inline`
  - `--prompt=...`
- `TestToggleFzfLayoutFlags`: chamar `Toggle` com o falso (a primeira linha é a de concluir) e conferir:
  - as três flags de layout;
  - as flags existentes (`--delimiter`, `--with-nth`, `--bind`) não foram perdidas.
- `TestPickFzfUsageError`: fzf falso que sai com código 2; `Pick` deve devolver erro, não `""`.
- `TestToggleFzfUsageError`: idem para `Toggle`; deve devolver erro, não `ErrCanceled`.

Detalhes: usar `t.Setenv("PATH", ...)` e `t.Setenv("FZF_FAKE_ARGS", ...)`; pular o teste em Windows (`runtime.GOOS == "windows"`), já que o projeto hoje é Linux-only.

## Critérios de aceite

- `daylist new` e `daylist timer start` mostram o fzf com ~70% de altura e borda.
- Resumos das etapas anteriores continuam visíveis durante a etapa atual.
- O fallback numerado (sem fzf) permanece igual.
- Erro de uso do fzf (exit 2) não é confundido com cancelamento.
- Esc/Ctrl+C e “sem match” continuam sendo cancelamento.

## Verificação

```bash
gofmt -l internal/ui
go vet ./internal/ui/...
go test ./internal/ui/... -race -cover
go test ./... -race
go build ./...
```

Manual:

```bash
daylist new
# conferir altura/borda e o cabeçalho visível
daylist timer start
# conferir o mesmo layout na escolha do item
```
