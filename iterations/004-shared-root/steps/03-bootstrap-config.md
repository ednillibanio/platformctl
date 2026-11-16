# Etapa: `platformctl bootstrap-config`

Status: `done`

## Objetivo

Criar o esqueleto do shared root e o par de chaves de automação, conforme a
seção `platformctl bootstrap-config` do standard.

## Escopo

- `--env`, `--target`..., `--consumer`..., `--declaration`... (padrão
  `./shared-root.yaml` quando existe), `--config-dir`, `--dry-run`
- diretórios, `targets/<t>/vars.sh` e `targets/<t>/hosts/<c>.sh` com
  placeholders comentados (descrição como comentário), uma chave de target
  declarada por vários consumidores escrita uma vez,
  `credentials/target-<t>/<identity>/` por identidade declarada,
  `credentials/<c>/`; sem declaração: esqueleto genérico e aviso;
  declaração inválida: erro e nada escrito
- arquivo existente sem chaves declaradas: inalterado, imprime as linhas a
  acrescentar
- `--automation-key` (gera ed25519 OpenSSH sem passphrase, comentário com o
  ambiente) e `--automation-key-import -` (stdin, valida ed25519 sem
  passphrase, deriva a pública), mutuamente exclusivos; publicação do par
  como unidade (diretório temporário + rename sem clobber)
- nunca cria `state/` nem `backups/`; nunca escreve valores

## Fora de Escopo

- `credential`, `doctor`

## Critérios de Aceite

- golden files do esqueleto; segunda execução não muda nada (mtime e
  conteúdo); `--dry-run` não toca o disco; par parcial = erro; duas
  execuções concorrentes de `--automation-key` deixam um par inteiro

## Validação

Mesma da etapa 01, mais `platformctl bootstrap-config` em
`ASSEMBLEIARR_CONFIG_DIR=<scratch>` duas vezes.

## Resultado

`platformctl bootstrap-config` criado e etapa fechada (2026-11-12, operador
autorizou commit e PR para `lab`).

- `internal/sharedroot/bootstrap.go`: valida tudo antes de escrever (nomes,
  declarações, consumer repetido entre declarações, chave importada); plano
  com `<env>/targets`, `credentials`, por target `targets/<t>/hosts`,
  `credentials/target-<t>/<identity>/` por identidade declarada e
  `vars.sh` (união das chaves de target; a primeira descrição vale;
  `required` é OU), por consumer `credentials/<c>/` e
  `targets/<t>/hosts/<c>.sh`. Nunca cria `state/` nem `backups/`; nunca
  escreve valor. Placeholder: `# KEY (required|optional): descrição`,
  `# Example: ...` opcional e `# KEY=`.
- O runner lê antes de escrever (`Tree.Entry`/`Tree.ReadFile`, novos):
  segunda execução não faz nenhuma escrita (nem mtime de diretório).
  Arquivo existente fica inalterado; chaves declaradas ausentes são
  impressas como linhas a acrescentar. `--dry-run` não cria nem o root.
- `internal/sharedroot/keys.go`: `--automation-key` gera ed25519 OpenSSH
  sem passphrase, comentário `assembleiarr-<env>-automation`, só se
  `credentials/automation/` não existe; `--automation-key-import -` lê
  stdin (máx. 64 KiB, exatamente um bloco `OPENSSH PRIVATE KEY` sem texto
  antes ou depois, sem passphrase, ed25519), guarda a chave byte a byte e
  deriva a pública. Par publicado com `PublishDir`; diretório existente com
  arquivo ausente ou não regular = erro (exit 1), nada criado nele. Sem as
  flags, declaração com `automation_key: true` gera aviso.
- `internal/cli/sharedroot.go`: comando e flags do standard; `--env`
  obrigatório, `--automation-key-import` só aceita `-`, erro de flag ou
  argumento = exit 2; declaração padrão `./shared-root.yaml` quando existe.
  `cmd/platformctl/main.go` usa `sharedroot.ExitCode` (uso 2, demais 1,
  como antes para os comandos existentes).
- `go.mod`: `golang.org/x/crypto v0.45.0` (mantém `go 1.24.0` e
  `golang.org/x/sys v0.38.0`).

## Evidências

- Golden files em `internal/sharedroot/testdata/bootstrap/` (`vars.sh`, dois
  hosts files e a saída) com as declarações `semaphoreui.yaml` e
  `nexus3.yaml` (`GOVC_URL` declarado pelos dois, escrito uma vez).
- Testes: esqueleto lido como placeholders pelo `ParseShellFile`;
  segunda execução sem mudança (modo, mtime e conteúdo, árvore envelhecida
  antes); arquivo vazio existente mantido; `--dry-run` sem root e com
  árvore parcial; declaração inválida e erros de uso não escrevem nada;
  arquivo existente sem chaves; esqueleto genérico e avisos; 8 execuções
  concorrentes de `--automation-key` deixam um par inteiro e coerente;
  par parcial e diretório no lugar da chave = erro; importação igual ao
  primeiro controller (com e sem newline final); recusa de passphrase,
  não ed25519, lixo, duas chaves, texto antes, vazio e entrada grande.
  Testes de CLI: exit 2, `-` obrigatório, declaração padrão,
  `ASSEMBLEIARR_CONFIG_DIR`, "not supported" no Windows.
- Container `golang:1.26.5`, como não root (`--userns=keep-id --user
  1000:1000`) e como root: `gofmt -l` vazio; `vet ./...` com `GOOS=linux`,
  `darwin` e `windows`; `go test ./...` verde; `test -race -count=2
  ./internal/sharedroot` verde.
- Binário com `ASSEMBLEIARR_CONFIG_DIR=/tmp/scratch-root`, duas vezes: a
  primeira cria a árvore (modos `0700`/`0644`/`0600`), a segunda só relata
  `exists` e a árvore fica igual (modo, mtime, tamanho); sem `state/` nem
  `backups/`.
- `codex-review --topic 004-03`: rodada 001 CHANGES REQUESTED (entrada não
  regular contava como arquivo do par) corrigida; rodada 002 CHANGES
  REQUESTED (existência por conteúdo; chave importada normalizada)
  corrigida; rodada 003 CHANGES REQUESTED (texto antes do bloco PEM aceito)
  corrigida com teste. Limite de 3 rodadas atingido; a última correção não
  teve rodada extra.
- Docs de usuário ainda não mudaram (ficam para a etapa 06); o help do
  comando descreve o uso.

## Pendências

- `consumer` igual ao nome do repositório: o comando não conhece o nome do
  repositório (o diretório atual pode ser um worktree); fica para a
  documentação (etapa 06) e para o `doctor`, se for o caso.
- Documentação do comando (`cli-reference.md`, `usage.md`) na etapa 06.
