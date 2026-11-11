# Etapa: Parsers (declaração e arquivos do shared root)

Status: `todo`

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

Preencher ao concluir a etapa.

## Evidências

## Pendências
