# Etapa: Comando export com formato env

Status: `done`

## Objetivo

Implementar `platformctl export --env <env> [--service <name>]... [--host
<name>] --format env --prefix <PREFIX>` conforme o ADR-0002 do
`platform-catalog`.

## Escopo

- nova projeção em `internal/render` (reutilizando `Resolve` e as regras de
  merge existentes) e comando em `internal/cli`
- seleção `defaults → environment → --service` na ordem dada, ou a ordem de
  resolução do `--host` (mutuamente exclusivos)
- nomes: segmentos unidos por `_`, `-` e `.` viram `_`, maiúsculas, prefixo
- tipos: string, número e booleano; lista, `null` e mapa vazio geram erro;
  colisão de nomes gera erro
- segredos: `env` exige variável definida e não vazia; `file` lido com
  `cat`, `~/` resolvido via `HOME`; `value` recusado sem `--reveal-secrets`
- saída ordenada, determinística, com aspas simples escapadas

## Fora de Escopo

- `*_FILE`, materialização de arquivos, validação de ambientes contra o
  catálogo

## Critérios de Aceite

- testes golden cobrindo nomes, tipos, colisões, cada forma de segredo e
  aspas
- teste que avalia a saída num `bash` real e confirma falha para arquivo
  ausente
- `vet`/`test`/`build` verdes no container lab

## Validação

```bash
cd container && ./lab.sh up && ./lab.sh status
```

## Resultado

- `internal/render/select.go`: `ResolveServices` (defaults → environment →
  serviços na ordem dada, erro para serviço inexistente ou repetido) e
  `HostConfig`.
- `internal/render/env.go`: `EncodeEnv` com nomes, tipos, colisões e
  aspas conforme o ADR-0002.
- `internal/cli/cli.go`: comando `export` (`--env`, `--service`
  repetível, `--host`, `--format env`, `--prefix`, `--reveal-secrets`).
- `.gitattributes`: `*.golden` sempre com LF, para o CI do Windows.

**Ajuste em relação ao texto do ADR-0002.** O ADR mostra `${VAR:?}` para
`env` e atribuição seguida de `export` para `file`. Na prática:

- `${VAR:?}` encerra um shell não interativo;
- um `|| false` solto não muda o status final do `eval`.

A saída passou a ser uma função temporária que:

1. lê todos os segredos em variáveis locais, com `return 1` na primeira falha;
2. só então exporta tudo;
3. remove a si mesma.

Resultado: ou tudo é exportado ou nada, o status fica diferente de zero e o
shell do chamador continua vivo. A correção do texto do ADR entra no PR do
catálogo da etapa 03.

## Evidências

Tudo executado no container lab (`container/`), nunca no host:

- `./lab.sh up` → bootstrap, `vet`, `test ./...` e build verdes;
  `./lab.sh status` → `status=ready`, `result_code=0`
- `gofmt -l ./internal ./cmd` → vazio; `go vet ./...` → ok
- testes novos, todos `PASS`:
  - `TestEncodeEnvGolden` (`internal/render/testdata/export-env.golden`)
  - `TestEncodeEnvRejectsUnexportableInput` (11 casos)
  - `TestEncodeEnvRevealSecrets`
  - `TestEncodeEnvEvaluatesInShells` (subtestes `bash` e `sh`)
  - `TestResolveServicesOrderAndErrors`
  - `TestExportCommand`
- `TestEncodeEnvEvaluatesInShells` avalia a saída em shells reais e
  confirma:
  - aspas e quebras de linha preservadas;
  - `~/` resolvido via `HOME`;
  - com arquivo ausente ou variável não definida, `status=1`, nenhuma
    variável exportada e o shell continua executando
- o golden foi gerado com `UPDATE_GOLDEN=1` numa cópia gravável do
  workspace dentro do container, porque a montagem do código é somente
  leitura, e revisado linha a linha antes de ser commitado

## Pendências

- `zsh` não existe no container lab, mas o runner macOS do CI tem. Na
  primeira execução do CI, o zsh derrubou o próprio teste, porque `status` é
  uma variável somente leitura no zsh; o problema era do script de teste, não
  da saída do `export`. O teste passou a usar `rc` e a verificar que a função
  temporária não fica definida depois do `eval`, em sucesso ou em falha.
- Documentação do `export` (`docs/cli-reference.md` e outros) na etapa 03.
