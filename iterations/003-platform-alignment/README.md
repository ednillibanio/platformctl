# Iteração: Alinhamento da configuração da plataforma (export env)

Status: `done`

Issue: #27

## Objetivo

Executar a fase 2 do programa de alinhamento da configuração da plataforma
(`platform-catalog` `docs/architecture/platform-config-alignment.md`):
implementar a projeção `platformctl export --format env` definida no
ADR-0002 e deixar o platformctl pronto para o teste de equivalência na VM do
Packer (ADR-0003, estágio A).

## Escopo

Incluir:

- corte das referências `infra-gitops` → `platform-modules` e do nome do
  catálogo (`platform-catalog`) em docs e testes de contrato (issue #24)
- comando `export` com formato `env`, prefixo, seleção por `--service` ou
  `--host`, referências de segredo conforme o ADR-0002, testes golden
- fixture sem segredos espelhando as camadas do `platform-modules`
- documentação de uso do `export`
- binários Linux gerados no container lab para a VM do Packer

Fora de escopo:

- qualquer alteração em `platform-modules` (fase 3) ou `platform-labs`
  (fase 4, congelado enquanto a iteração 012 de lá estiver aberta)
- `*_FILE`, materialização de `vars.sh` e validação de nomes de ambiente
  contra o catálogo (adiados pelo ADR-0002)

## Etapas

Cada etapa tem branch e PR próprios para `lab`.

| Etapa | Status | Arquivo |
| --- | --- | --- |
| 01 | `done` | `steps/01-platform-modules-cutover.md` |
| 02 | `done` | `steps/02-export-env.md` |
| 03 | `done` | `steps/03-fixture-and-docs.md` |
| 04 | `done` | `steps/04-linux-binaries-and-closure.md` |

## Critérios de Fechamento

- todas as etapas estão `done` ou `skipped`
- cada etapa possui evidência de validação
- PRs integrados em `lab`
- tabela de fases do programa atualizada no `platform-catalog`

## Validação Esperada

- `vet`, `test` e `build` somente no `container/lab.sh`
- contratos estáticos `vagrant/tests` e `container/tests`
- CI verde nos três sistemas operacionais e no container lab

## Resultado

Concluída em 2026-09-23:

- `platformctl export --format env` (ADR-0002/0004), com testes golden, de
  shell real (bash, dash e zsh) e de contrato;
- fixture `examples/platform-modules-home/` e documentação;
- corte de `infra-gitops` → `platform-modules` e do nome do catálogo.

Os binários Linux da etapa 04 foram `skipped` por decisão do operador. O
programa de alinhamento foi pausado depois desta iteração (ver
`iterations/BACKLOG.md`). O `export` continua no `lab`, sem consumidor
planejado.
