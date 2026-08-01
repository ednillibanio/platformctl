# Etapa: Atualizar documentação

Status: `done`

## Objetivo

Documentar a nova forma `value` de credencial e o comportamento de redação
do `render`, mantendo a documentação consistente com o comportamento real.

## Escopo

- `docs/configuration.md` — seção `## Credentials`
- `docs/progress.md` — item de "Completed" sobre renderização
- `CLAUDE.md` — seções sobre `internal/validation` e `internal/render`

## Fora de Escopo

- Manual de uso (`docs/usage.md`) e panorama entre repositórios
  (`docs/ecosystem.md`) — ficam para a iteração `002`.

## Critérios de Aceite

- as três formas de credencial (`env`/`file`/`value`) estão documentadas
  com exemplo
- o comportamento de redação por padrão e `--reveal-secrets` está
  documentado
- nenhuma menção antiga a "exatamente um de env ou file" sobrevive

## Validação

```bash
grep -rn "exactly one of env" . --include="*.md" --include="*.go"
```

## Resultado

Concluído.

## Evidências

- `docs/configuration.md`: seção `## Credentials` reescrita com as três
  formas e o comportamento de redação.
- `docs/progress.md`: item "Completed" atualizado.
- `CLAUDE.md`: itens 2 e 4 da lista de componentes atualizados.
- `grep -rn "exactly one of env"` só retorna a nova mensagem em
  `internal/validation/validation.go` (com "env, file, or value").

## Pendências

Nenhuma.
