# Etapa: `platformctl credential`

Status: `todo`

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

Preencher ao concluir a etapa.

## Evidências

## Pendências
