# Etapa: docs/ecosystem.md — panorama entre repositórios

Status: `todo`

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

Preencher ao concluir a etapa.

## Evidências

Registrar comandos executados, saídas relevantes e arquivos alterados.

## Pendências

Registrar riscos, bloqueios ou próximos passos.
