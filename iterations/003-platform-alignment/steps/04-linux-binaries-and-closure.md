# Etapa: Binários Linux e fechamento

Status: `done`

## Objetivo

Gerar `platformctl` para `linux/amd64` e `linux/arm64` dentro do container
lab, para levar à VM do Packer (fase 3, no `platform-modules`), e fechar a
iteração.

## Escopo

- build cruzado via `GOOS`/`GOARCH` no container lab, nunca no host
- checksums SHA-256 registrados como evidência
- fechamento da iteração, da issue #27 e atualização da tabela do programa

## Fora de Escopo

- empacotamento de release ou publicação de binários
- execução de qualquer coisa na VM do Packer (fase 3)

## Critérios de Aceite

- binários gerados fora do checkout, com checksums registrados
- `README.md` desta iteração com o resultado final

## Validação

```bash
cd container && ./lab.sh up && ./lab.sh status
```

## Resultado

- Binários Linux: `skipped` por decisão do operador em 2026-09-23. O teste
  na VM do Packer (fase 3) foi cancelado junto com o restante do programa,
  então não há consumidor para os binários.
- Fechamento: iteração 003 concluída; etapas 01–03 integradas em `lab`
  (PRs #28, #29, #30); programa marcado como pausado aqui
  (`iterations/BACKLOG.md`) e no `platform-catalog` (PR #22).

## Evidências

- `platform-catalog` #22 mergeado (`2436350`): ADR-0004, fases 3–5
  `cancelled` e ADR-0001..0004 `on hold`.
- `docs/ecosystem.md` e `docs/progress.md` não descrevem mais a adoção pelo
  `platform-modules` como planejada.

## Pendências

- Nenhuma nesta iteração. A promoção `lab` → `main` do platformctl fica a
  critério do operador.
