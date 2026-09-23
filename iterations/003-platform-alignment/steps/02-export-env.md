# Etapa: Comando export com formato env

Status: `todo`

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
- segredos: `env` → `${VAR:?}`; `file` → atribuição e `export` separados,
  falha em arquivo ilegível, `~/` resolvido via `HOME`; `value` recusado sem
  `--reveal-secrets`
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

Preencher ao concluir a etapa.

## Evidências

Preencher ao concluir a etapa.

## Pendências

Preencher ao concluir a etapa.
