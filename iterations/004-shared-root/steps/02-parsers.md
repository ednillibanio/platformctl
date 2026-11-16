# Etapa: Parsers (declaração e arquivos do shared root)

Status: `done`

## Objetivo

Ler e validar as duas entradas do standard: a declaração
`shared-root.yaml` e os arquivos `vars.sh` / `hosts/<consumer>.sh`.

## Escopo

- `shared-root.yaml` versão 1, estrito (`KnownFields`, documento único):
  `version`, `consumer` (igual ao nome do repositório), `automation_key`,
  `target`, `host`, `credentials`, `target_credentials` com `identity`
  obrigatório e ativo; `name`, `required`, `description`, `example` só em
  `target`/`host`; duplicatas (em `target_credentials`, mesmo nome e mesma
  identidade), mesmo nome em `target` e `host`, prefixos (`GOVC_*`,
  `TARGET_*`, `DISPOSABLE_*` em `target`; `HOST_*` em `host`); política de
  versão (atual e anterior com aviso)
- tabela de identidades do standard embutida no binário (ativas e
  reservadas)
- parser de atribuição literal portado do loader do `provision-semaphoreui`
  (`hosts/lib/config.sh`, "Key rules") mais a regra do `~` sem aspas;
  placeholders comentados (`# KEY=`) reconhecidos como ausentes
- fixtures: os casos aceitos e recusados pelo loader, para os dois lados
  concordarem

## Fora de Escopo

- escrita de arquivos; comandos

## Critérios de Aceite

- cada regra do standard tem um teste que aceita e um que recusa
- os erros citam arquivo, linha e chave; nenhum erro imprime um valor que
  possa ser secret

## Validação

Mesma da etapa 01.

## Resultado

Parsers criados em `internal/sharedroot` e etapa fechada (2026-11-11,
operador autorizou commit e PR para `lab`).

- `problems.go`: `Problem` (arquivo, linha, chave, mensagem) e `Problems`
  (todos os achados, ordenados; exit 1). Nenhuma mensagem leva valor.
- `shellfile.go`: `ParseShellFile(file, kind, content)` para `vars.sh`
  (`GOVC_*`, `TARGET_*`, `DISPOSABLE_*`) e `hosts/<consumer>.sh`
  (`HOST_*`). Gramática de linha igual à do loader do
  `provision-semaphoreui` (`config_check_line_format`, origin/lab
  b8aaab2), incluindo comentário no fim da linha, mais a regra do `~` (valor
  sem aspas não aceita `~`). Recusa CR/NUL, nome de chave inválido, prefixo
  errado, chave repetida no mesmo arquivo e tipo de arquivo desconhecido.
  `# KEY=` vira placeholder (chave ausente).
- `declaration.go`: `ParseDeclaration` percorre o `yaml.Node` (não usa
  decode em struct) para ser estrito e ter linha: documento único, campos
  desconhecidos ou repetidos, tipos por tag (`!!int`, `!!bool`, `!!str`),
  campos obrigatórios, `description` de uma linha e não vazia, `example` só
  em `target`/`host`, `identity` obrigatória, só em `target_credentials` e
  ativa, duplicatas por seção (par identidade/nome em
  `target_credentials`), mesmo nome em `target` e `host`, prefixos e
  versão (1; caminho de versão anterior com aviso existe, vazio na v1).
  Aliases, merge key `<<`, `null` e tags como `!!binary` são recusados.
- A tabela de identidades da etapa 01 (`names.go`) é a embutida no binário.

## Evidências

- Fixtures `testdata/shellfile/{accepted,refused,refused-tilde}.txt`
  (inclui os casos do teste offline do loader: `declare -x`, `printf -v`,
  `;`, `$(...)`, crase, `${HOME}`, espaço) e
  `testdata/declaration/standard-example.yaml` (exemplo do standard).
- `TestShellFileAgreesWithLoader` roda a cópia literal do check do loader
  (`testdata/shellfile/loader-check.sh`) nas mesmas fixtures: o loader
  aceita `accepted.txt`, recusa `refused.txt` e ainda aceita
  `refused-tilde.txt` (regra do `~` pendente na issue de adaptação dele),
  que o Go recusa.
- Container `golang:1.26.5`, como não root (`--userns=keep-id --user
  1000:1000`) e como root: `gofmt -l` vazio; `vet ./...` com `GOOS=linux`,
  `darwin` e `windows`; `go test ./...` verde.
- `codex-review --topic 004-02`: rodada 001 CHANGES REQUESTED (identidade
  inválida/duplicada ecoava o valor) corrigida; rodada 002 CHANGES
  REQUESTED (`FileKind` desconhecido aceito; testes de alias/merge)
  corrigida; o High da 002 foi retirado pelo próprio Codex (recusa
  segura); Low mantido por decisão (nomes reservados são constantes;
  nomes de campo desconhecidos são ecoados de propósito, arquivo de Git sem
  secrets); rodada 003 APPROVE sem achados.
- Docs de usuário não mudaram: audit `technical-documentation` não se
  aplica.

## Pendências

- `consumer` igual ao nome do repositório: o parser só valida o padrão; a
  comparação fica para o comando (etapa 03), que conhece o consumer
  esperado.
- Checagens entre arquivos (chave em `vars.sh` e no hosts file, chave
  declarada ausente) ficam para o `doctor` (etapa 05).
- O loader do `provision-semaphoreui` ainda não aplica a regra do `~`
  (issue de adaptação provision-semaphoreui#86); a fixture
  `refused-tilde.txt` documenta a diferença.
