# Etapa: docs/usage.md — manual de uso

Status: `done`

## Objetivo

Criar `docs/usage.md`, manual passo a passo (não referência, como
`docs/configuration.md` já é):

1. Fluxo do zero: `platformctl init` → `environment init <env>` → o que
   cada comando gera de fato
2. Como criar cada tipo de documento manualmente (profile, service, host) —
   não tem subcomando de CLI pra isso
3. Como eles se conectam (host → profile + services; service → ports;
   profile → extends)
4. `validate` e `render`, incluindo `--reveal-secrets`
5. Tabela de referência rápida por tipo de documento (campos,
   obrigatório/opcional, tipo)
6. Exemplo guiado próximo do caso real do operador (ex: um host que
   instala agente de um serviço, com credencial `value`)

## Escopo

- `docs/usage.md` (novo)
- link a partir de `README.md` e `docs/configuration.md`

## Fora de Escopo

- Duplicar o conteúdo já coberto por `docs/configuration.md` (merge
  resolvido, estrutura de diretórios) — linkar em vez de repetir
- Qualquer mudança de código

## Critérios de Aceite

- alguém que nunca usou o platformctl consegue, só com esse doc, criar um
  host novo do zero e entender o que `render` produz
- todos os exemplos de YAML validam de verdade
- links cruzados resolvem

## Validação

```bash
cd container && ./lab.sh ssh
# validar os exemplos do manual manualmente dentro do container
```

## Resultado

Concluído. `docs/usage.md` criado (init → autoria manual de cada tipo de
documento → como se conectam → credenciais → validate/render →
tabela de campos por tipo). Exemplo guiado (host `db1`, profile
`database extends base`, service `monitoring` com porta, credencial
`monitoring-agent-psk` como `value`) validado de verdade dentro do
container. Um erro real encontrado e corrigido: a saída de exemplo mostrava
`ports: []`, mas o service tem porta declarada — corrigido pra bater com o
render real.

## Evidências

Dependência descoberta e resolvida antes desta etapa: a branch tinha sido
criada a partir de `lab` antes do PR #11 (valor inline em credentials)
mergear, então o exemplo com `credentials.<nome>.value` não validava ainda.
PR #11 revisado e mergeado em `lab` (squash: merge commit), issue #9
fechado manualmente, branch rebaseada em `origin/lab` atualizado.

Comandos executados dentro do `container/lab.sh`:

```
platformctl init
platformctl environment init prod
platformctl validate --env prod        # "Environment "prod" is valid"
platformctl render --env prod          # credentials.monitoring-agent-psk.value: <redacted>
platformctl render --env prod --reveal-secrets   # mostra o valor real
```

Saída conferida linha a linha contra o que está documentado em
`docs/usage.md` — bate exatamente, após a correção do `ports`.

Checagem de links relativos em `docs/`+`README.md`: todos resolvem.

## Pendências

- `README.md` ainda mostra `docker compose` cru em vez de `./lab.sh` na
  seção Docker — mesma pendência já registrada na etapa 01, ainda não
  corrigida (fora do escopo desta iteração).
