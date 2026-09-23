# Etapa: Fixture platform-modules e documentação

Status: `todo`

## Objetivo

Criar `examples/platform-modules-home/`, sem segredos, espelhando as camadas
do `platform-modules` (`base`, `overlays/lab`, target vSphere, módulo
`packer`), e documentar o `export`.

## Escopo

- fixture coberta por `go test`, como `examples/platform-home`
- `docs/cli-reference.md`, `docs/configuration.md`, `docs/usage.md` e
  `docs/ecosystem.md` com o `export`
- PR pequeno no `platform-catalog` marcando a fase 1 como `done` e a fase 2
  como `in progress` na tabela do programa

## Fora de Escopo

- valores reais de ambiente

## Critérios de Aceite

- fixture valida e exporta; saída golden dos nomes `PM_*` esperados pelo
  `pkr_apply_ig_aliases`
- links e exemplos da documentação conferidos contra `--help`

## Validação

```bash
cd container && ./lab.sh up && ./lab.sh status
```

## Resultado

Preencher ao concluir a etapa.

## Evidências

Preencher ao concluir a etapa.

## Pendências

Preencher ao concluir a etapa.
