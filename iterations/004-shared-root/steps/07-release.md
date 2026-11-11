# Etapa: Release

Status: `todo`

## Objetivo

Release instalável em controller Linux e macOS.

## Escopo

- workflow disparado por tag: binários `linux`/`darwin` × `amd64`/`arm64`,
  `SHA256SUMS`, versão embutida (`platformctl --version`)
- `docs/installation.md`: baixar, conferir checksum, instalar, verificar
- registrar a versão para o install do controller do `platform-modules`
  (al-rr/platform-modules#337)

## Fora de Escopo

- assinatura de binários; empacotamento (rpm/deb/brew)

## Critérios de Aceite

- workflow validado (actionlint) e testado por `workflow_dispatch`, que
  gera os artefatos sem publicar release
- a primeira tag real sai só depois da promoção `lab` → `main`, com
  confirmação do operador

## Validação

```bash
actionlint .github/workflows/*.yml
```

## Resultado

Preencher ao concluir a etapa.

## Evidências

## Pendências
