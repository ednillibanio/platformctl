# Etapa: Fixture platform-modules e documentação

Status: `done`

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

- `examples/platform-modules-home/`: `defaults.yaml` (camada `base`),
  `environments/lab/environment.yaml` (overlay `lab`) e os serviços
  `vsphere-esxi` (target) e `packer` (módulo). Não há segredos: as
  credenciais são referências `file: ~/...` apontando para a raiz de
  segredos do ADR-0002, mais `env: GITHUB_TOKEN`.
- `PM_DEFAULT_TARGET` ficou de fora de propósito. Com ele, o `packerctl`
  procuraria um `targets/<nome>.sh`, que não existe quando a configuração
  vem do `export`.
- `internal/render/examples_test.go`: saída golden da fixture
  (`testdata/platform-modules-export.golden`) e teste de contrato que exige
  que todo nome exportado esteja na lista de `PM_*` lida pelo
  `platform-modules` (`pkr-vars.sh` e `gitops-config/lib.sh` no `lab`,
  `eb8bc97`).
- Documentação:
  - `docs/cli-reference.md`: seção `platformctl export` e linha na sinopse;
  - `docs/configuration.md`: seção "Environment projection";
  - `docs/usage.md`: seção 6;
  - `docs/ecosystem.md`: status da integração;
  - `docs/progress.md`: implementado e pendente;
  - `README.md`: linha de status.

## Evidências

Tudo executado no container lab:

- `platformctl export --help` e `platformctl --help` conferidos contra a
  tabela de flags e a sinopse da documentação.
- `PLATFORMCTL_HOME=.../examples/platform-modules-home platformctl validate
  --env lab` → `Environment "lab" is valid`.
- Exemplo do `docs/usage.md`, de ponta a ponta, com segredos falsos em
  `$HOME/.config/assembleiarr/lab/credentials/platform-modules/`:
  - `eval` → `rc=0`, 24 variáveis `PM_*`, segredos lidos pelo shell;
  - removendo `PM_VSPHERE_PASSWORD` → mensagem clara e `rc=1`.
  A atomicidade (nada exportado quando há falha) é provada pelos testes
  unitários em shells limpos, não por esta demonstração.
- Golden gerado numa cópia gravável do workspace dentro do container e
  conferido: 24 exports, todos presentes na lista do `platform-modules`.

Tropeço registrado: na primeira revisão do golden parecia faltar
`PM_VSPHERE_NETWORK`. Era o meu filtro de saída (`grep -v Network`, usado
para esconder as mensagens do docker compose) que escondia a linha
`'VM Network'`. O arquivo foi trazido do container por base64, sem filtro,
e conferido.

## Pendências

- `platform-catalog` #21 / PR #22 (ADR-0004, formato de avaliação do
  `export`, substituindo só os exemplos da seção 3 do ADR-0002; tabela de
  fases com a fase 1 `done` e a fase 2 `in progress`), aberto junto com esta
  etapa. O merge é confirmado no fechamento da iteração (etapa 04).
