# Etapa: Documentação e e2e

Status: `done`

## Objetivo

Documentar os comandos novos e provar o roteiro do plano de ponta a ponta.

## Escopo

- `docs/cli-reference.md`, `docs/usage.md`, `docs/installation.md`,
  `docs/ecosystem.md`, `README.md` (tipo Platform tooling, comandos novos,
  modelo YAML pausado), `AGENTS.md` novo conforme o `agents-standard` do
  catálogo
- script e2e (em `scripts/` ou `tests/`) com o roteiro da "Validação
  Esperada" da iteração, rodando no container Go pinado, com a declaração
  neutra `provision-example` (forma do exemplo "Key declaration" do
  `provision-shared-root-standard` do catálogo)

## Fora de Escopo

- instruções de release (etapa 07)
- cruzamento com o loader de qualquer `provision-*`: é teste do consumidor
  (decisão do operador de 2026-11-12 no README da iteração)

## Critérios de Aceite

- audit `technical-documentation` com registro; e2e verde; nenhum exemplo,
  fixture ou teste da etapa nomeia um consumidor real

## Validação

Mesma da etapa 01, mais o script e2e.

## Resultado

Documentação e e2e implementados e aprovados pelo operador (2026-11-12).

- Antes da etapa, a correção das identidades da ADR-0007 entrou em PR
  próprio (#40, `c77deaf`), por decisão do operador; os docs descrevem o
  comportamento já corrigido.
- `docs/cli-reference.md`: os dois grupos de comandos (shared root atual,
  modelo YAML pausado), seleção do root, comportamento comum, uma seção por
  comando (flags, saída, exemplos, campos do JSON do `doctor`) e exit codes
  0/1/2; a afirmação antiga "todo erro sai com 1" foi corrigida.
- `docs/usage.md`: roteiro do operador num controller (declarações,
  esqueleto com `--dry-run`, outros controllers com
  `--automation-key-import -` via SSH, preencher valores, credenciais,
  `doctor`, comparar controllers por JSON, teste com root de scratch); o
  modelo YAML ficou como seção "paused".
- `docs/installation.md`: Linux/macOS para os comandos novos, checagem com
  root de scratch, link para o roteiro.
- `docs/ecosystem.md`: tipo Platform tooling, estado da adoção (issues de
  adaptação de cada repo; `provision-vsphere-platform` é o primeiro com
  `shared-root.yaml`), linha do `platform-modules` corrigida (export pausado,
  instala o platformctl via #337).
- `README.md`, `docs/index.md`, `docs/progress.md`, `docs/architecture.md`
  (seção do `internal/sharedroot` e mapa de componentes),
  `docs/troubleshooting.md` (seção Shared root com as mensagens reais),
  `docs/development.md` (seção do e2e), `CLAUDE.md`.
- `AGENTS.md` novo conforme o `agents-standard` do catálogo (papel
  `platform-tooling`, limites de escopo, documentos normativos, idioma,
  regras de edição).
- `scripts/e2e-shared-root.sh` novo: roda o binário contra um root de
  scratch e cobre o roteiro da "Validação Esperada" da iteração em 21
  checagens (`PASS`/`FAIL` por linha, exit 1 em falha). A declaração usada,
  `scripts/testdata/e2e/shared-root.yaml`, é o consumidor neutro
  `provision-example` na forma do exemplo "Key declaration" do standard do
  catálogo (sem `target_credentials`: a única identidade `active` é do
  `provision-vsphere-platform`).
- Desacoplamento (decisão do operador de 2026-11-12): saíram a opção
  `--semaphoreui` e o cruzamento com o loader do `provision-semaphoreui`;
  exemplos dos docs, o exemplo do help de `credential`
  (`internal/cli/sharedroot.go`) e a fixture do e2e passaram para
  `provision-example`.

## Evidências

- Depois do desacoplamento (2026-11-12), container `golang:1.26.5`, não
  root (`--userns=keep-id --user 1000:1000`) e root: `go test -race
  -count=1 ./...` verde; `vet ./...` com `GOOS=linux`, `darwin` e
  `windows`; `gofmt -l -e cmd internal schemas` vazio.
- `scripts/e2e-shared-root.sh` no mesmo container, não root e root: 21
  `PASS`, exit 0 nos dois. Execução negativa (`--bin /bin/false`): checagens
  `FAIL` e exit diferente de zero. O JSON do `doctor` é parseado (`python3`,
  ou `jq`) e checa `ok` e o formato do fingerprint. A saída capturada só é
  impressa em falha, com os secrets de teste mascarados. `bash -n`,
  `shellcheck` e `shfmt -d` limpos.
- Antes do desacoplamento: contratos estáticos de `vagrant/` e `container/`
  verdes; a mudança posterior não toca esses diretórios.
- Declaração real do `provision-vsphere-platform` (`lab` `b94f44f`):
  `bootstrap-config` exit 0 (cria `vcenter-access-admin/`); `doctor` exit 1
  com os 4 problemas esperados (placeholders e credenciais ausentes).
- Audit `technical-documentation`:
  `.agent-runs/doc-audits/20261112-004-06-docs-audit.md` (links e âncoras
  verificados, 0 quebrados; duas afirmações corrigidas pelo código: root
  symlink é recusado; já existe um consumidor com declaração).
- `codex-review --topic 004-06`: rodada 001 CHANGES REQUESTED (secret do
  stdin como argumento de `sh -c`; `sort -z` não portável para macOS;
  "nenhum arquivo parcial" exagerava a atomicidade, que é por arquivo)
  corrigida; rodada 002 CHANGES REQUESTED (JSON checado só por substring)
  corrigida com parse; rodada 003 CHANGES REQUESTED (saída impressa em falha
  podia expor um secret vazado) corrigida com o mascaramento em `show`.
  Limite de 3 rodadas atingido; a última correção não teve rodada extra
  (validada pelo e2e e por um teste do mascaramento). O desacoplamento
  também não teve rodada extra: só remove a opção `--semaphoreui` e troca
  nomes de exemplo, validado pelos testes e pelo e2e acima.

## Pendências

- O e2e não roda no CI: incluí-lo é mudança de workflow, fica como
  proposta.
- Com um binário que falha em tudo (`--bin /bin/false`) o e2e aborta no
  meio (um `cd`/`grep` sob `set -e`) e sai com 2, não com o exit 1
  documentado; continua falhando, mas não lista todas as checagens.
- `docs/vagrant-controller-integration.md` continua só sobre
  `PLATFORMCTL_HOME`; o shared root em controllers de teste é
  ednillibanio/platform-labs#128.
- Código da etapa 02 já em `lab` que ainda cita um consumidor
  (`internal/sharedroot/shellfile.go` cita o loader do
  `provision-semaphoreui` em vez da seção "File format" do catálogo;
  `testdata/shellfile/loader-check.sh`; fixtures e testes com nome
  `semaphoreui`): PR de correção próprio depois desta etapa.
- Comentar em al-rr/provision-semaphoreui#86 que o cruzamento do loader com
  uma árvore do platformctl é teste do próprio repositório.
