# Iteração: Shared root dos `provision-*` (bootstrap-config, credential, doctor)

Status: `doing`

Issue: #34 (iteração); demanda: #33

## Objetivo

Implementar no `platformctl` o contrato do `platform-catalog`
`docs/standards/provision-shared-root-standard.md` (ADR-0005 seções 3, 6, 8
e 9, Accepted em 2026-11-11): os comandos `bootstrap-config`, `credential`
e `doctor`, que criam e verificam o shared root dos `provision-*`
(`${ASSEMBLEIARR_CONFIG_DIR:-~/.config/assembleiarr}/<env>/`) em todo
controller, e uma release instalável em controller Linux e macOS. O standard
é a fonte de verdade; em divergência, vale o standard.

## Escopo

Incluir:

- pacote novo `internal/sharedroot` (root, nomes, escrita segura, parsers,
  comandos) e os comandos em `internal/cli`
- dependências oficiais `golang.org/x/sys` (rename sem clobber de
  diretório) e `golang.org/x/crypto` (chaves ed25519 em formato OpenSSH)
- testes Go (unitários, golden files, idempotência, concorrência, par
  parcial, nomes inválidos) e um script e2e
- docs: `cli-reference.md`, `usage.md`, `installation.md`, `ecosystem.md`,
  `README.md`, `AGENTS.md` novo
- workflow de release por tag (binários linux/macos amd64/arm64 com
  checksums)

Fora de escopo:

- os comandos existentes (`init`, `environment init`, `validate`, `render`,
  `export`) e o modelo YAML pausado: não mudam
- `PLATFORMCTL_HOME` e `internal/config/scaffold.go`: não são reaproveitados
  (o scaffold tem `--force` e recria arquivos, o que o standard proíbe)
- adoção do pre-commit (#32); qualquer alteração nos `provision-*` (cada um
  tem sua issue de adaptação)
- Windows: os comandos novos retornam "not supported" e seus testes são
  pulados

## Decisões do operador (2026-11-11)

- A. Plataformas: Linux e macOS; Windows "not supported".
- B. Dependências: `golang.org/x/sys` e `golang.org/x/crypto`, sem chamar
  `ssh-keygen`.
- C. Validação: `go test`/`vet` num container simples com a imagem Go
  pinada (`docker run` via Podman), sem compose, enquanto
  al-rr/platform-modules#340 não entrega o compose no controller; o
  `container/lab.sh` continua sendo o lab do CI.
- D. Descrição do repositório no GitHub alinhada ao tipo Platform tooling.
- `--env` é obrigatório em todo comando novo (sem ambiente padrão).
- 2026-11-12: a tabela de identidades segue o standard do catálogo em
  `main` c2696ab (ADR-0007): só `vcenter-access-admin` é `active`;
  `provision-nexus3`, `provision-semaphoreui`, `platform-modules-live` e
  `platform-modules-packer` são `reserved`; `terraform-provisioner` é
  `deprecated` (recusado como declaração e como diretório). Correção num PR
  próprio para `lab`, antes da etapa 06.
- 2026-11-12: o platformctl é um produto genérico, sem acoplamento a nenhum
  `provision-*`. A única fonte do contrato é o
  `provision-shared-root-standard` do catálogo; exemplos, fixtures e testes
  usam o consumidor neutro `provision-example`. O cruzamento com o loader do
  `provision-semaphoreui` saiu da etapa 06 e da "Validação Esperada": provar
  que o loader de um consumidor aceita a árvore é teste do próprio
  consumidor (issue de adaptação dele). Só fatos espelhados do catálogo
  (linhas da tabela de identidades em `names.go`) nomeiam consumidores. O
  código já integrado na etapa 02 que ainda cita um consumidor é corrigido
  num PR próprio depois da etapa 06.

## Etapas

| Etapa | Status | Arquivo |
| --- | --- | --- |
| 01 | `done` | `steps/01-base.md` |
| 02 | `done` | `steps/02-parsers.md` |
| 03 | `done` | `steps/03-bootstrap-config.md` |
| 04 | `done` | `steps/04-credential.md` |
| 05 | `done` | `steps/05-doctor.md` |
| 06 | `done` | `steps/06-docs-e2e.md` |
| 07 | `doing` | `steps/07-release.md` |

## Critérios de Fechamento

- todas as etapas estão `done` ou `skipped`
- cada etapa possui evidência de validação
- pendências foram registradas em nova iteração ou issue
- PR aberto e integrado em `lab`; promoção `lab` → `main` e primeira tag
  só com confirmação do operador

## Validação Esperada

- `./scripts/go.sh vet ./...` e `./scripts/go.sh test ./...` no container
  Go pinado (decisão C)
- roteiro do plano com `ASSEMBLEIARR_CONFIG_DIR=<scratch>`:
  `bootstrap-config` duas vezes (a segunda não muda nada; nenhum `state/`
  ou `backups/`; declaração ausente = aviso, inválida = erro e nada
  escrito); arquivo divergente reportado e não sobrescrito; `credential`
  nunca sobrescreve; `doctor` falha em `$(...)`, `~` sem aspas, chave
  duplicada, chave declarada ausente e arquivo solto em
  `credentials/target-<t>/`; `--format json` parseia e não contém secrets
- audit `technical-documentation` e `codex-review` em cada ponto de revisão

## Resultado

Registrar o resumo final quando a iteração for fechada.
