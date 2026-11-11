# Etapa: `platformctl bootstrap-config`

Status: `todo`

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

Preencher ao concluir a etapa.

## Evidências

## Pendências
