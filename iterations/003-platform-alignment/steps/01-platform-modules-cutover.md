# Etapa: Corte infra-gitops → platform-modules e nome do catálogo

Status: `done`

## Objetivo

Atualizar as referências do repositório para `platform-modules` (sucessor do
`infra-gitops` arquivado) e `platform-catalog` (antes
`infrastructure-platform`/`infra-platform`), atendendo a issue #24.

## Escopo

- `README.md`, `CLAUDE.md`, `CONTRIBUTING.md`, `iterations/BACKLOG.md`
- `docs/ecosystem.md`, `docs/index.md`, `docs/progress.md`,
  `docs/lab-environments.md`, `docs/vagrant-controller-integration.md`
- `vagrant/tests/test-vagrantfile-contract.sh`,
  `container/tests/test-compose-contract.sh`

## Fora de Escopo

- registros históricos da iteração 002 (mantidos como escritos)
- qualquer mudança de código Go

## Critérios de Aceite

- nenhuma referência ativa a `infra-gitops`/nomes antigos do catálogo, exceto
  menções a "sucessor do `infra-gitops`", registros históricos e o padrão de
  rejeição dos testes de contrato
- testes de contrato passando
- links novos verificados no GitHub

## Validação

```bash
bash vagrant/tests/test-vagrantfile-contract.sh
bash container/tests/test-compose-contract.sh
git diff --check
```

## Resultado

- `docs/ecosystem.md`: seção `platform-modules` reescrita, apontando para o
  programa e os ADR-0001..0003 no `platform-catalog`; tabela e diagrama
  Mermaid atualizados; data de verificação dos links atualizada.
- Padrão de rejeição dos testes de contrato passou a
  `'infra-gitops|platform-modules|provision-glpi|glpictl'`. A issue #24
  sugeria trocar o nome, mas manter os dois é uma proteção mais forte: os
  labs não podem depender de nenhum dos dois repositórios.

## Evidências

- `bash vagrant/tests/test-vagrantfile-contract.sh` →
  `[PASS] Vagrantfile contract tests completed.`
- `bash container/tests/test-compose-contract.sh` →
  `[PASS] Container lab contract tests completed.`
- nenhum arquivo em `vagrant/` ou `container/` (fora de `tests/`) cita
  `platform-modules`
- links verificados via `gh api repos/<repo>/contents/<path>?ref=<branch>`:
  `platform-modules` (`README.md` e dois guias em `docs/en/`, branch `lab`)
  e `platform-catalog` (programa e ADR-0001..0003, branch `main`)
- `git diff --check` limpo

## Pendências

- nenhuma nesta etapa
