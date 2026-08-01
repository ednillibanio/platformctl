# Etapa: docs/ecosystem.md — panorama entre repositórios

Status: `done`

## Objetivo

Criar `docs/ecosystem.md` explicando o papel de cada repositório do
ecossistema e em que fase/ordem cada um é acionado:

- `infrastructure-platform`: catálogo público, o que existe e quem é dono
  (sem valor real)
- `infra-gitops` / `gitopsctl`: biblioteca de automação reutilizável +
  config só do próprio `infra-gitops` (`bootstrap-config`: credencial
  vSphere, identidade git, chave SSH da lib)
- `platformctl`: árvore central de valores/credenciais por ambiente,
  consumida por outros repos — nunca provisiona nada
- `provision-*` (ex: `provision-talos-vsphere`, `provision-glpi`): consome
  o render do platformctl + a lib do infra-gitops pra executar o
  provisionamento de verdade (Terraform/Ansible/govc/talosctl/Packer)

Incluir um diagrama (texto ou mermaid) do fluxo de quem chama quem.

## Escopo

- `docs/ecosystem.md` (novo)
- link a partir de `docs/architecture.md` e `README.md`

## Fora de Escopo

- Detalhar o funcionamento interno de cada repo além do necessário pra
  entender o papel dele no fluxo
- Qualquer mudança de código

## Critérios de Aceite

- as quatro peças (catálogo, lib de automação, platformctl, provision-*)
  estão descritas com exemplo concreto de cada uma
- fica claro em que fase cada uma entra e o que ela NÃO faz
- links cruzados resolvem

## Validação

```bash
grep -rn '](.*\.md' docs/ecosystem.md README.md docs/architecture.md
```

## Resultado

Concluído. `docs/ecosystem.md` criado com a tabela das quatro peças
(catálogo, lib de automação, platformctl, provision-*), um diagrama
mermaid do fluxo de handoff, uma seção "Why platformctl exists as a
separate piece" (o problema do `vars.sh` duplicado por repo) e uma seção
"What platformctl is not". Links cruzados adicionados em
`docs/architecture.md` e `README.md`.

## Evidências

- Arquivos alterados: `docs/ecosystem.md` (novo), `docs/architecture.md`,
  `README.md`.
- Checagem de links relativos: todos resolvem, exceto o link pra
  `docs/usage.md` dentro de `docs/ecosystem.md` — esperado, é a etapa 02
  desta mesma iteração, ainda não criada.

## Pendências

- `docs/usage.md` ainda não existe — etapa 02 resolve o link pendente.
- README.md ainda mostra `docker compose ...` cru na seção "Docker" em vez
  de `./lab.sh ...` (ficou pra trás depois que criamos `container/lab.sh`
  na iteração anterior) — fora do escopo desta etapa, registrar como
  possível item futuro.
