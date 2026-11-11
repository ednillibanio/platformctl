# Etapa: Documentação e e2e

Status: `todo`

## Objetivo

Documentar os comandos novos e provar o roteiro do plano de ponta a ponta.

## Escopo

- `docs/cli-reference.md`, `docs/usage.md`, `docs/installation.md`,
  `docs/ecosystem.md`, `README.md` (tipo Platform tooling, comandos novos,
  modelo YAML pausado), `AGENTS.md` novo conforme o `agents-standard` do
  catálogo
- script e2e (em `scripts/` ou `tests/`) com o roteiro da "Validação
  Esperada" da iteração, rodando no container Go pinado
- cruzamento com o loader do `provision-semaphoreui`: o skeleton preenchido
  passa em `hosts/vsphere/host.sh --target=<t> show` com o root de scratch

## Fora de Escopo

- instruções de release (etapa 07)

## Critérios de Aceite

- audit `technical-documentation` com registro; e2e e cruzamento verdes

## Validação

Mesma da etapa 01, mais o script e2e.

## Resultado

Preencher ao concluir a etapa.

## Evidências

## Pendências
