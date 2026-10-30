# Task 2 — Gerador puro de weeklist

**Depende de:** task 1 · **Bloqueia:** task 3
**Arquivos:** `internal/week/weeklist.go`, `internal/week/weeklist_test.go`

## Objetivo

Reproduzir `scripts/weeklist_script.js` sem Obsidian, mas **preservando tempo e prioridade** dos hábitos: a nova weeklist copia `- [ ] 5 min (U) - AWS` exatamente assim, mudando apenas o checkbox para pendente.

## Mudança

Criar a função pura:

```go
// BuildWeeklist monta a nova weeklist a partir do conteúdo da weeklist de
// origem. Fonte vazia gera uma weeklist só com as seções base.
func BuildWeeklist(source string) string
```

Regras:

- Percorrer o conteúdo linha a linha, removendo `\r` de CRLF.
- Entrar na seção em `# Hábitos` e sair no próximo `# `.
- Dentro da seção:
  - linhas `## Categoria` são preservadas como estão;
  - qualquer linha aceita por `week.ParseHabitLine` é copiada como `- [ ] ` + `HabitLine.Body`;
  - o checkbox é sempre resetado para `[ ]`, inclusive quando a origem estava `[x]`;
  - tempo e prioridade **não** são resetados: `5 min (U)` continua `5 min (U)`, e `xST ()` continua `xST ()`;
  - formatos parciais também são copiados: `5 min - AWS`, `(U) - AWS` e `AWS`;
  - linhas em branco, comentários e textos fora de checklist são ignorados.
- A descrição mantém o texto original; apenas o estado do checkbox é alterado.
- Ignorar `# Tasks` e `# Extra`.

Exemplo:

```markdown
# origem
- [x] 5 min (U) - AWS
- [ ] xST () - dormir
- [ ] 10 min - revisar
- [ ] (I) - planejar
- [ ] ler

# destino
- [ ] 5 min (U) - AWS
- [ ] xST () - dormir
- [ ] 10 min - revisar
- [ ] (I) - planejar
- [ ] ler
```

Saída:

```go
var b strings.Builder
b.WriteString("# Tasks\n\n---\n# Hábitos\n")
for _, line := range habits {
    b.WriteString(line)
    b.WriteString("\n")
}
```

Ou seja: sem fonte, o resultado termina em `# Hábitos\n`.

## Testes (`internal/week/weeklist_test.go`)

Fonte de exemplo cobrindo:

```markdown
# Tasks
- [ ] tarefa qualquer

---
# Hábitos
## Saúde
- [x] 10 min (I) - correr
- [ ] xST () - dormir

## Estudos
- [ ] 20 min (U) - ler
# Extra
- [ ] ignorar
```

- `TestBuildWeeklistCopiesHabits`: saída contém `- [ ] 10 min (I) - correr`, `- [ ] xST () - dormir` e `- [ ] 20 min (U) - ler`.
- `TestBuildWeeklistPreservesTimeAndFlag`: uma origem `- [ ] 5 min (U) - AWS` gera exatamente `- [ ] 5 min (U) - AWS`.
- `TestBuildWeeklistPreservesPartialFormats`: `5 min - revisar`, `(I) - planejar` e `ler` são copiados como estão.
- `TestBuildWeeklistResetsCheckbox`: origem `[x]` com tempo/prioridade vira `[ ]` mantendo `5 min (U)`.
- `TestBuildWeeklistPreservesCategories`: `## Saúde` antes de `## Estudos`, na ordem original.
- `TestBuildWeeklistIgnoresTasksAndExtra`: nenhuma linha de `# Tasks`/`# Extra` é copiada.
- `TestBuildWeeklistEmptySource`: saída exata `# Tasks\n\n---\n# Hábitos\n`.
- `TestBuildWeeklistCRLF`: mesmo resultado de LF.
- `TestBuildWeeklistIgnoresNonChecklistLines`: texto solto não entra.

## Critérios de aceite

- Saída no formato esperado, com a diferença intencional de preservar tempo/prioridade.
- Hábitos concluídos (`[x]`) são copiados como pendentes (`[ ]`).
- Tempo, prioridade e formato original são preservados; apenas o checkbox muda.
- Formatos parciais aceitos pelo parser também são copiados.
- Ordem de categorias e hábitos é preservada.
- Nenhum acesso a disco na função.

## Verificação

```bash
gofmt -l internal/week
go vet ./internal/week/...
go test ./internal/week/... -race -cover
```
