# Backlog

Este backlog agrupa o trabalho planejado para `platformctl`. Itens aqui não
autorizam execução. Cada grupo vira uma iteração (`iterations/NNN-topic/`)
somente após confirmação explícita do operador, um passo por vez.

## Consumo do `render` por `provision-platform-environments` e `provision-semaphoreui`

Status: `todo` — proposta registrada, preenchimento manual segue em uso por
enquanto.

Hoje `provision-platform-environments` (`manifests/<repo>/artifact-lab.yaml`,
`spec.variables`) e `provision-semaphoreui` (`manifests/java-projects.yaml`,
`spec.environments[].variables`) duplicam manualmente os mesmos fatos não
secretos do Nexus `artifact-lab` (`NEXUS_URL`, `NEXUS_REGISTRY`,
`NEXUS_RAW_REPOSITORY`, ...) e o endereço do host DEV real. Cada repositório
preenche esses valores à mão, sem fonte única — exatamente o problema que
`platformctl` foi desenhado para resolver (ver `docs/ecosystem.md`), mas essa
ponta ("runtime orchestration") está listada como *deferred* em
`docs/progress.md` e não existe ainda.

Desenho proposto (discutido em 2026-08-21):

- Um `Service` `nexus` por ambiente em `PLATFORMCTL_HOME`
  (`environments/<env>/services/nexus.yaml`) com os valores não secretos
  (`nexus.url`, `nexus.registry`, `nexus.raw_repository`,
  `nexus.maven_repository`, `nexus.maven_group_url`) e duas credenciais
  separadas: `nexus-ci` (conta `github-actions-ci`, consumida por
  `provision-platform-environments`) e `nexus-deploy` (conta
  `semaphore-deploy`, consumida por `provision-semaphoreui`) — mantendo a
  separação de contas já exigida em
  `provision-platform-environments/docs/nexus-account-contract.md`.
- Um `Host` por ambiente (ex.: `juventude-dev`) com o endereço real do host
  DEV, hoje `<fill-in>` duplicado nos dois manifests.
- Em cada repositório consumidor, um script fino (não existe ainda) que roda
  `platformctl render --env <env> --format json`, extrai os valores com
  `jq` e popula o `spec.variables`/inventário do manifest local; e que
  resolve a referência de credencial (`env`/`file`) para escrever o arquivo
  esperado pelo `--secrets-dir` de cada CLI (`platform-environments-ctl
  apply`, `semaphoreui-ctl apply`) só no momento do `apply`, nunca
  commitado.

Esse trabalho pertence aos repositórios consumidores (não a mudanças no
schema/CLI do `platformctl` em si), então normalmente vira iteração em
`provision-platform-environments` e `provision-semaphoreui`, não aqui.
Registrado neste backlog porque a decisão de layout (`Service`/`Host`/nomes
de credencial) nasceu do lado `platformctl`.

## Adoção real do `PLATFORMCTL_HOME`

Status: `todo` — bloqueado por decisão do operador

`PLATFORMCTL_HOME` ainda não foi inicializado para uso real; hoje só existe
o exemplo em `examples/platform-home`. Pré-requisito para o item acima.

## Programa: alinhamento da configuração da plataforma

Status: `doing` — fases 0a, 0 e 1 concluídas em 2026-09-23; fase 2 em
`iterations/003-platform-alignment/` (#27). Acompanhamento oficial em
`platform-catalog` `docs/architecture/platform-config-alignment.md`.

Objetivo: tornar `platformctl` a fonte única de valores e referências de
credenciais por ambiente da plataforma (catálogo, módulos, labs, serviços),
absorvendo o papel de armazenamento hoje feito por
`modulesctl bootstrap-config` (`~/.config/platform-modules`, `PM_*`). Os
consumidores passam a ler projeções do `render` (JSON/YAML e uma projeção
`env` com prefixo, ex.: `PM_*`), nunca a árvore interna de outro repositório.

Decisões já tomadas:

- `platformctl` absorve valores/credenciais do `modulesctl bootstrap-config`,
  com ponte de compatibilidade antes de qualquer depreciação;
- o repositório de catálogo (antes `infrastructure-platform`/`infra-platform`)
  foi renomeado para `platform-catalog` em 2026-09-23;
- o documento do programa e os ADRs ficam no catálogo; este backlog só
  aponta para eles.

Fases e repositório dono:

| Fase | Repositório | Conteúdo |
| --- | --- | --- |
| 0a | `platformctl` | `CONTRIBUTING.md` e fluxo documentado (este PR) |
| 0 | catálogo | rename para `platform-catalog` e referências |
| 1 | catálogo | documento do programa + ADRs (planos, contrato de configuração, migração do `bootstrap-config`) |
| 2 | `platformctl` | projeção `export --format env`, fixture espelhando `~/.config/platform-modules`, docs (reaproveita a issue #24 para o corte `infra-gitops` → `platform-modules`) |
| 3 | `platform-modules` | `pkr-vars.sh`/Ansible aceitam `PM_*` do export; teste de equivalência na VM de golden images |
| 4 | `platform-labs` | topologia dos cenários vinda do render |
| 5 | consumidores | Nexus (item acima), `provision-haproxy` |

Restrição: não alterar `platform-labs` enquanto a iteração 012 de lá
(revisão geral conduzida pelo Codex) não estiver fechada; conferir o status
e os worktrees em `.agent-runs/` antes da fase 4.
