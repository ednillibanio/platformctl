# Etapa: `platformctl credential`

Status: `done`

## Objetivo

Substituir o `gen-credential` do `provision-semaphoreui`, conforme a seção
`platformctl credential` do standard.

## Escopo

- `--env`, `--consumer <c>` ou `--target <t> --identity <i>`, argumentos
  `NAME`, `NAME=VALUE`, `NAME=-` (stdin), `--length` (padrão 32),
  `--config-dir`, `--dry-run`
- aleatório: `--length` bytes de `crypto/rand`, base64; arquivo `0600` sem
  newline final; existente = pulado e reportado
- nunca imprime um valor

## Fora de Escopo

- rotação ou remoção de credenciais

## Critérios de Aceite

- testes: nunca sobrescreve; stdin lido uma vez e só para `NAME=-`;
  identidade reservada ou desconhecida recusada; mesmo formato do
  `gen-credential` (sem newline final)

## Validação

Mesma da etapa 01.

## Resultado

`platformctl credential` criado e etapa fechada (2026-11-12, operador
autorizou commit e PR para `lab`).

- `internal/sharedroot/credential.go`: destino `credentials/<c>/`
  (`--consumer`) ou `credentials/target-<t>/<i>/` (`--target` com
  `--identity` ativa); combinações inválidas, identidade reservada ou
  desconhecida, nome inválido ou repetido, mais de um `NAME=-`, literal
  vazio ou com quebra de linha e `--length` fora de 1..1024 = erro de uso
  (exit 2), sem ler stdin e sem escrever nada.
- Ordem: abre o root só se já existe, decide os arquivos existentes
  (pulados e relatados, nunca sobrescritos; diretório ou symlink no nome =
  erro), resolve todos os valores (stdin lido uma vez e só se o arquivo do
  `NAME=-` não existe; aleatório gerado), e só então cria root e
  diretórios (`0700`, como o `mkdir -p` do `gen-credential`) e publica cada
  arquivo com `PublishFile` (`0600`, sem newline final). Valor inválido no
  stdin não deixa nada criado. Arquivo criado por execução concorrente
  (`ErrExists`) é relatado como `exists`.
- Aleatório: `--length` bytes (padrão 32) de `crypto/rand`, base64 padrão
  com padding (mesmo formato de `openssl rand -base64`, sem newline).
- Stdin: no máximo 64 KiB de valor; remove um único fim de linha (`\n` ou
  `\r\n`, nunca `\r` sozinho); vazio ou com outra quebra de linha = erro
  de uso.
- Saída: `created`/`exists`/`would create` com caminho e origem do valor
  (`random value, N characters`, `literal value`, `value from standard
  input`); nunca o valor. `--dry-run` não cria nada nem lê stdin.
- `internal/cli/sharedroot.go`: comando `credential` com `--env`
  obrigatório, `--consumer`, `--target`, `--identity`, `--length`,
  `--config-dir`, `--dry-run`; `--length 0` explícito = erro de uso.

## Evidências

- Testes `internal/sharedroot/credential_test.go`: formatos (literal,
  aleatório base64 de 32 bytes = 44 caracteres, stdin), modos `0600`/`0700`,
  saída sem valores; `--length 48`; segunda execução não muda a árvore
  (modo, mtime, conteúdo, árvore envelhecida) e não lê stdin; stdin não
  lido para arquivo existente; fins de linha aceitos e recusados (vazio,
  `\r` sozinho, dois fins, duas linhas, grande); limite de 64 KiB com e sem
  fim de linha; destino target/identidade; 20 erros de uso sem escrita e
  sem leitura de stdin; `--dry-run` sem root e com árvore existente;
  symlink e diretório no nome e symlink como diretório de destino
  recusados; 8 execuções concorrentes criam o arquivo uma vez, sem arquivo
  temporário; stdin inválido depois de argumentos válidos não cria nada
  (root ausente e árvore existente). CLI: exit 2, comando completo via
  `ASSEMBLEIARR_CONFIG_DIR`, segunda execução só `exists`, "not supported"
  no Windows.
- Container `golang:1.26.5`, não root (`--userns=keep-id --user
  1000:1000`) e root: `go test -race -count=1 ./...` verde; `vet ./...` com
  `GOOS=linux`, `darwin` e `windows`; `gofmt -l` vazio.
- Binário com `ASSEMBLEIARR_CONFIG_DIR=/tmp/r`: cria
  `credentials/provision-semaphoreui/{SEMAPHORE_DB_USER,SEMAPHORE_DB_PASS}`
  e `credentials/target-vsphere-lab/terraform-provisioner/GOVC_PASSWORD`
  (stdin `pw\n` gravado como `pw`, 2 bytes), modos `0600`/`0700`; segunda
  execução relata `exists`; identidade reservada = exit 2.
- `codex-review --topic 004-04`: rodada 001 CHANGES REQUESTED (`\r`
  sozinho removido do stdin) corrigida; rodada 002 CHANGES REQUESTED
  (valor inválido no stdin depois de arquivos já escritos) corrigida
  resolvendo todos os valores antes de criar qualquer coisa; rodada 003
  CHANGES REQUESTED (limite de 64 KiB contava o fim de linha) corrigida com
  testes de fronteira. Limite de 3 rodadas atingido; a última correção não
  teve rodada extra.

## Pendências

- Documentação do comando (`cli-reference.md`, `usage.md`) e a
  substituição do `gen-credential` nos docs dos consumidores ficam para a
  etapa 06 e para as issues de adaptação (provision-semaphoreui#86).
- Stdin vindo de terminal ecoa o valor digitado; o help mostra `read -rs`
  com pipe. Leitura sem eco (`x/term`) não está no escopo.
