# Iteração: Suportar valor inline em credentials

Status: `doing`

## Objetivo

Suportar um valor inline (`value`) em `credentials`, além das referências
já existentes (`env`, `file`), para segredos sem outro lugar natural pra
morar (ex: senha de build gerada) — mantendo tudo fora do git, dentro de
`PLATFORMCTL_HOME`. Como `value` guarda o segredo de verdade, `render`
passa a redigi-lo por padrão (`<redacted>`), só mostrando o valor real com
`--reveal-secrets`.

## Escopo

Incluir:

- `internal/config` (schema Go do `SecretRef`)
- `internal/validation` (exigir exatamente um entre `env`/`file`/`value`)
- `internal/render` (função `Redact`)
- `internal/cli` (flag `--reveal-secrets` em `render`)
- `schemas/v1alpha1/config.schema.json`
- `docs/configuration.md`, `docs/progress.md`, `CLAUDE.md`

Fora de escopo:

- `examples/platform-home` (não alterado, pra não invalidar o exemplo
  resolvido já documentado em `docs/configuration.md`)
- `schemas/v1alpha1/rendered-environment.schema.json` (não precisou mudar —
  `credentials` já era `type: object` genérico ali)
- `vagrant/`, `container/` (não relacionados)

## Etapas

| Etapa | Status | Arquivo |
| --- | --- | --- |
| 01 | `done` | `steps/01-implementar-valor-inline.md` |
| 02 | `done` | `steps/02-atualizar-documentacao.md` |

## Critérios de Fechamento

- todas as etapas estão `done` ou `skipped`
- cada etapa possui evidência de validação
- PR aberto e integrado em `lab`

## Validação Esperada

```bash
./scripts/go.sh vet ./...
./scripts/go.sh test ./...
cd container && ./lab.sh up
```

## Resultado

Pendente — fechar após integração do PR em `lab`.
