# Etapa: Guias do operador

Status: `doing`

## Objetivo

Cobrir as lacunas de documentação vistas no fim da etapa 07, antes da
promoção para `main`, para que o operador entenda a ideia do shared root e
use a ferramenta só pelos guias.

## Escopo

- `docs/shared-root-concepts.md` (novo): por que o shared root existe, o
  que o platformctl faz e não faz, desenho da árvore com modos, quem
  escreve e quem lê cada arquivo, formato dos arquivos, consumidores e
  identidades, vários controllers, ciclo de vida
- `docs/shared-root-declaration.md` (novo): guia de quem mantém um
  `provision-*` para escrever e checar o `shared-root.yaml`
- `docs/usage.md`: links para os dois guias, resultado do esqueleto e
  saídas reais do `doctor` (texto e JSON)
- navegação: `docs/index.md` e `README.md` (e duas frases desatualizadas
  desde a etapa 07: "sem release publicada", "instala a partir do fonte")

## Fora de Escopo

- código; tradução dos docs (a regra do repositório é inglês fora de
  `iterations/`)

## Critérios de Aceite

- cada afirmação nova conferida contra o código ou o binário; exemplos de
  saída tirados de uma execução real
- links e âncoras sem quebra; audit `technical-documentation` registrado

## Validação

Binário de `lab` contra root de scratch (container `golang:1.26.5`) e
checagem de links/âncoras.

## Resultado

Implementado (2026-11-12), aguardando revisão e OK do operador.

- `docs/shared-root-concepts.md` e `docs/shared-root-declaration.md`
  novos, citando o standard do catálogo como fonte e só o consumidor neutro
  `provision-example`.
- `docs/usage.md`: saídas reais do `doctor` (esqueleto novo com 4
  problemas, árvore preenchida, linha `$(id)` recusada, JSON) e links.
- `docs/index.md`, `README.md`: entradas para os guias; frases de release
  atualizadas.

## Evidências

- Saídas de exemplo geradas pelo binário de `lab` `b617174` com a
  declaração `scripts/testdata/e2e/shared-root.yaml` num root de scratch.
- Afirmações do guia da declaração conferidas no binário: campo
  desconhecido, prefixo errado, mesma chave em `target` e `host`, `example`
  em `credentials`, identidade não ativa: exit 1 com "invalid declaration,
  nothing written"; declaração válida com `--dry-run`: `would create`,
  exit 0; `HOST_IP` em `vars.sh`: problema do `doctor`.
- No código: `credential` cria os diretórios que faltam e grava sem quebra
  de linha final (`internal/sharedroot/credential.go`); `bootstrap-config`
  cria um diretório por identidade declarada (`bootstrap.go`).
- Links e âncoras (regras de slug do GitHub) de `README.md`, `AGENTS.md`,
  `CONTRIBUTING.md` e `docs/*.md`: 0 quebrados.
- Audit: `.agent-runs/doc-audits/20261112-004-08-docs-audit.md`.
- `codex-review --topic 004-08`: rodada 001 CHANGES (exemplo usava a
  identidade do `provision-vsphere-platform` para `provision-example`),
  corrigida; rodada 002 CHANGES (nem todo `provision-*` já lê o shared root;
  nem todo problema do `doctor` tem linha), corrigida; rodada 003 CHANGES
  (identidade fora da tabela do catálogo é desconhecida, não `reserved`),
  corrigida nos três guias. Limite de 3 rodadas atingido; a última correção
  não teve rodada extra (só texto, conferido contra `names.go` e a tabela
  do standard).

## Pendências

- Nenhuma nova.
