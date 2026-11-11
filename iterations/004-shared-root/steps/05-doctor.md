# Etapa: `platformctl doctor`

Status: `todo`

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

Preencher ao concluir a etapa.

## Evidências

## Pendências
