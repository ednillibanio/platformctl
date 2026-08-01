# Etapa: docs/usage.md — manual de uso

Status: `todo`

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

Preencher ao concluir a etapa.

## Evidências

Registrar comandos executados, saídas relevantes e arquivos alterados.

## Pendências

Registrar riscos, bloqueios ou próximos passos.
