# Iteração: Manual de uso e panorama entre repositórios

Status: `done`

## Objetivo

Registrar por escrito o entendimento que hoje só existe nesta conversa: como
usar o platformctl na prática (criar profile/service/host, validar,
renderizar, credenciais) e onde ele se encaixa no fluxo entre
`infrastructure-platform`, `infra-gitops`/`gitopsctl`, `platformctl` e os
repositórios `provision-*`.

## Escopo

Incluir:

- novo `docs/ecosystem.md` — panorama entre repositórios, em que fase cada
  um entra, quem consome o quê
- novo `docs/usage.md` — manual de uso do platformctl: fluxo completo,
  criação manual de cada tipo de documento, tabela de campos, credenciais
  (`env`/`file`/`value`), `validate`/`render`, exemplo guiado
- links cruzados a partir de `README.md`/`docs/architecture.md` onde fizer
  sentido

Fora de escopo:

- documentar o funcionamento interno de `infra-gitops`/`provision-*` em
  detalhe (pertence à documentação deles)
- qualquer mudança de código ou schema

## Etapas

| Etapa | Status | Arquivo |
| --- | --- | --- |
| 01 | `done` | `steps/01-ecosystem-doc.md` |
| 02 | `done` | `steps/02-usage-doc.md` |

## Critérios de Fechamento

- todas as etapas estão `done` ou `skipped`
- cada etapa possui evidência de validação
- PR aberto e integrado em `lab`

## Validação Esperada

- todos os links relativos novos resolvem
- exemplos de YAML no manual validam de fato (`platformctl validate`) via
  `container/lab.sh`

## Resultado

Concluído. `docs/ecosystem.md` e `docs/usage.md` criados e validados
(exemplo guiado testado de verdade no `container/lab.sh`). Pendência
registrada, fora do escopo: `README.md` ainda usa `docker compose` cru em
vez de `./lab.sh` na seção Docker. Falta abrir e integrar o PR pra `lab`.
