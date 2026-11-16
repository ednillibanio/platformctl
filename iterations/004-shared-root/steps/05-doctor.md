# Etapa: `platformctl doctor`

Status: `done`

## Objetivo

Verificação somente leitura do shared root, conforme a seção
`platformctl doctor` do standard.

## Escopo

- `--env`, `--target`..., `--consumer`..., `--declaration`...,
  `--format text|json`, `--config-dir`
- todas as checagens do standard: modos e dono, symlinks, formato dos
  arquivos (incluindo `~`), prefixos, chave em dois arquivos, chaves e
  credenciais declaradas ausentes, vazias ou em placeholder, credencial
  declarada (nome ou valor) em arquivo `0644`, par de chaves divergente ou
  incompleto, restos de execução interrompida, credencial vazia, arquivo
  solto em `credentials/target-<t>/`, identidade desconhecida, declaração
  inválida; não declarados = avisos
- relata todos os problemas; exit 1 em qualquer falha
- JSON: achados, chaves e valores não secretos, nomes das credenciais
  presentes, fingerprint da chave pública; nunca um secret nem hash de
  secret

## Fora de Escopo

- corrigir qualquer problema (o `doctor` não escreve)

## Critérios de Aceite

- um teste por checagem; teste que garante que nenhum valor de credencial
  aparece em text nem em JSON

## Validação

Mesma da etapa 01.

## Resultado

`platformctl doctor` criado e etapa fechada (2026-11-12, operador
autorizou commit e PR para `lab`).

- `internal/sharedroot/tree.go`, `tree_unix.go`, `tree_other.go`:
  `Tree.Lstat` e `Tree.List` (fd-based, sem seguir symlink) e `EntryInfo`
  (tipo, modo, dono, tamanho); no Windows retornam "not supported".
- `internal/sharedroot/doctor.go`: verificação somente leitura de
  `<root>/<env>/`; nunca cria o root nem o ambiente (ausentes = problema,
  exit 1). Percorre a árvore inteira, inclusive diretórios desconhecidos e
  identidades inativas, e aplica as checagens gerais: dono, modo (`0700`
  diretórios, `0644` arquivos compartilhados, `0600` credenciais e chave
  privada), symlink e restos de execução interrompida (arquivo ou
  diretório temporário). Arquivos compartilhados (`vars.sh`, `hosts/<c>.sh`)
  passam pelo parser da etapa 02 (`$(...)`, `~` sem aspas, chave
  duplicada, prefixo errado, chave de host em `vars.sh` e de target em
  `hosts/`). Credencial vazia, arquivo direto em `credentials/target-<t>/`,
  identidade reservada ou desconhecida e par de chaves de automação
  divergente ou incompleto (`keys.go` `checkAutomationPair`) são problemas.
- Com declarações (`--declaration`, padrão `./shared-root.yaml` quando
  existe): chaves obrigatórias ausentes, vazias ou em placeholder,
  credenciais e credenciais de target ausentes, par de chaves declarado
  ausente, arquivo `hosts/` de consumidor declarado ausente, declaração
  inválida (exit 1). Nome ou valor de credencial declarada em qualquer
  arquivo legível por outros = problema, e todos os valores desse arquivo
  ficam fora do relatório. Chaves, credenciais e arquivos não declarados =
  avisos (exit 0).
- `--target` limita as checagens declaradas (padrão: todo target
  encontrado); `--consumer` acrescenta consumidores cujos arquivos devem
  existir.
- `doctor_output.go`: `--format text` (padrão) lista problemas e avisos com
  arquivo, linha e chave; `--format json` traz achados, chaves e valores
  não secretos, nomes das credenciais presentes e fingerprint SHA256 da
  chave pública; nunca um secret nem hash de secret.
- `internal/cli/sharedroot.go`: comando `doctor` com `--env` obrigatório,
  `--target`, `--consumer`, `--declaration`, `--format`, `--config-dir`;
  exit 0 sem problema, 1 com problema, 2 uso inválido; registrado em
  `cli.go`.

## Evidências

- Testes `internal/sharedroot/doctor_test.go`: árvore saudável sem
  achados; um caso por checagem em `TestDoctorChecks` (38 casos: modos de
  diretório, arquivo compartilhado, credencial, chave privada e `state/`;
  symlink e diretório symlink; `$(...)`; `~` sem aspas; chave duplicada;
  chave de host em `vars.sh` e de target em `hosts/`; chave obrigatória em
  placeholder, vazia e ausente; credencial e credencial de target
  ausentes; par declarado ausente; nome e valor de credencial em arquivo
  compartilhado, em arquivo desconhecido legível e em credencial com modo
  errado; par divergente e incompleto; arquivo e diretório temporários;
  credencial vazia; arquivo direto em `credentials/target-<t>/` (também
  legível por outros); identidade reservada, desconhecida e symlink dentro
  de identidade inativa; `vars.sh` ausente; `hosts/` de consumidor
  declarado ausente; diretório desconhecido com modo errado, com symlink e
  com entrada temporária); nenhum valor de credencial em text nem em JSON;
  valores retidos quando só o nome vaza; declaração inválida; avisos não
  falham; root e ambiente ausentes não são criados; escopo de `--target` e
  `--consumer`; erros de uso; dono (só como root). CLI: erros de uso
  (exit 2), skeleton recém-criado via `ASSEMBLEIARR_CONFIG_DIR` em text e
  JSON, "not supported" no Windows.
- Container `golang:1.26.5`, não root (`--userns=keep-id --user
  1000:1000`) e root: `go test -race ./...` verde (root com `-count=1`);
  `vet ./...` com `GOOS=linux`, `darwin` e `windows`; `gofmt -l` vazio.
- Binário com `ASSEMBLEIARR_CONFIG_DIR=<scratch>` e a declaração
  `testdata/bootstrap/semaphoreui.yaml`: depois do `bootstrap-config`,
  `doctor` = exit 1 (placeholders e `GOVC_PASSWORD` ausente); preenchidos
  `vars.sh`, `hosts/provision-semaphoreui.sh` e `GOVC_PASSWORD` via
  `credential`, exit 0; `--format json` parseia, `ok: true`, traz o
  fingerprint e não contém o valor da credencial; `X=$(id)` em `vars.sh` =
  exit 1 com o arquivo e a linha.
- `codex-review --topic 004-05`: rodada 001 CHANGES REQUESTED (diretórios
  desconhecidos não eram percorridos) corrigida; rodada 002 CHANGES
  REQUESTED (busca de vazamento só nos arquivos reconhecidos; nome
  encontrado não retinha os valores) corrigida varrendo todo arquivo
  legível por outros; rodada 003 CHANGES REQUESTED (identidade inativa e
  arquivo direto em `credentials/target-<t>/` escapavam das checagens
  gerais) corrigida separando `unknown` de `inspect`, com 2 casos novos.
  Limite de 3 rodadas atingido; a última correção não teve rodada extra.

## Pendências

- "Chave em dois arquivos" é coberta pela regra de prefixo (`GOVC_*`,
  `TARGET_*`, `DISPOSABLE_*` só em `vars.sh`; `HOST_*` só em `hosts/`), sem
  checagem separada.
- Valores de credencial com menos de 8 bytes não são buscados nos arquivos
  legíveis; uma credencial literal não secreta (por exemplo usuário de
  banco) com 8 ou mais bytes pode gerar falso positivo.
- O conteúdo de `backups/<c>/` não é inspecionado (só modos, dono e
  symlinks).
- Não se verifica se o nome do consumidor corresponde ao repositório
  `provision-*` real.
- Documentação do comando (`cli-reference.md`, `usage.md`) fica para a
  etapa 06.
