# Etapa: Implementar valor inline em credentials

Status: `done`

## Objetivo

Adicionar o campo `value` a `SecretRef`, validar exclusividade mútua entre
`env`/`file`/`value`, e mascarar `value` por padrão no `render`.

## Escopo

- `internal/config/types.go` — campo `Value string` em `SecretRef`
- `internal/validation/validation.go` — `checkSecrets` exige exatamente um
  dos três
- `internal/render/render.go` — função `Redact` + `RedactedValue`
- `internal/cli/cli.go` — flag `--reveal-secrets` no comando `render`
- `schemas/v1alpha1/config.schema.json` — `secretRef.oneOf` ganha `value`
- Testes em `internal/validation`, `internal/render`, `internal/cli`

## Fora de Escopo

- `examples/platform-home`
- `schemas/v1alpha1/rendered-environment.schema.json`

## Critérios de Aceite

- credencial com `value` sozinho é válida
- credencial com dois ou mais de `env`/`file`/`value` é inválida
- `render` sem `--reveal-secrets` nunca imprime o valor real de uma
  credencial `value`
- `render --reveal-secrets` imprime o valor real
- referências `env`/`file` nunca são mascaradas

## Validação

Comandos executados:

```bash
./scripts/go.sh vet ./...
./scripts/go.sh test ./...
./scripts/go.sh build -o bin/platformctl ./cmd/platformctl
cd container && ./lab.sh up
```

## Resultado

Concluído. Todos os pacotes (`cli`, `config`, `merge`, `platformhome`,
`render`, `schema`, `validation`) passaram, local e no `container/lab.sh`
(pipeline completo: bootstrap Go 1.26.5 `linux/arm64` via Colima, `go vet`,
`go test`, build, `validate`, `render` contra `examples/platform-home`,
`status=ready`).

## Evidências

- `git log -1 --stat` no commit `c60b5b7` (branch
  `feat/inline-credential-values`): 11 arquivos alterados
  (`internal/cli/cli.go`, `internal/cli/cli_test.go`,
  `internal/config/types.go`, `internal/render/render.go`,
  `internal/render/render_test.go`, `internal/validation/validation.go`,
  `internal/validation/validation_test.go`,
  `schemas/v1alpha1/config.schema.json`, mais docs — ver etapa 02).
- Testes novos: `TestValidationAcceptsInlineValueCredential`,
  `"credential with two references set"` (tabela de falhas),
  `TestRedactMasksInlineValuesOnly`, `TestRenderRedactsInlineSecretsByDefault`.
- `./scripts/go.sh test ./...`: todos os pacotes `ok`.
- `container/lab.sh up`: `status=ready`, `reason=validation_completed`.

## Pendências

Nenhuma.
