# Etapa: Release

Status: `done`

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

- workflow validado (actionlint) e testado sem publicar release: no PR
  (`pull_request`) e por `workflow_dispatch`, que só geram o artefato
- a primeira tag real sai só depois da promoção `lab` → `main`, com
  confirmação do operador

## Validação

```bash
actionlint .github/workflows/*.yml
```

## Resultado

Implementado e aprovado pelo operador (2026-11-12). O run de `release` do
PR #43 foi o teste sem publicar: `build` ok, `publish` pulado, artefato
`platformctl-dev-d3b8710c71f9`.

- `platformctl --version` (`platformctl version <v>`; `dev` em build de
  fonte): `Version` do root do cobra em `internal/cli/cli.go`, injetado com
  `-ldflags -X`; teste `TestVersion`.
- `scripts/release-build.sh <version> <out-dir>`: `platformctl-<os>-<arch>`
  para `linux`/`darwin` × `amd64`/`arm64` com `CGO_ENABLED=0` (estático no
  Linux; no macOS só `libSystem`), `-trimpath`, e `SHA256SUMS`; recusa versão
  fora de `vX.Y.Z[-pre]`/`dev-<commit>` (exit 2) e diretório com release
  (exit 1).
- `.github/workflows/release.yml`: testes, build pelo mesmo script, checagem
  (`sha256sum -c`, `--version`) e upload do artefato; o job `publish`
  (`contents: write`) só roda em tag `v*` e cria a release com
  `gh release create --verify-tag`.
- Decisão: o teste sem publicar é o run de `pull_request` (e
  `workflow_dispatch`, que só funciona depois que o workflow estiver em
  `main`, o branch padrão).
- Docs: `docs/installation.md` (baixar, conferir checksum, instalar,
  verificar), `docs/development.md` (Release build), `docs/cli-reference.md`
  (`--version`), `README.md`, `CLAUDE.md`.

## Evidências

- Container `golang:1.26.5`: `go test -race -count=1 ./...` verde, `gofmt`
  limpo; `release-build.sh` gerou os 4 binários, `sha256sum -c` OK,
  `--version` imprime a versão injetada; `file`: ELF estático (Linux),
  Mach-O (darwin). Versão inválida exit 2, diretório não vazio exit 1,
  `--help` exit 0.
- Trecho de instalação do guia executado sobre os arquivos locais (sem o
  `curl`): checksum, `install`, `--version` OK.
- actionlint 1.7.7 (container) limpo; `shellcheck` e `shfmt -d` limpos.
- Audit `technical-documentation`:
  `.agent-runs/doc-audits/20261112-004-07-docs-audit.md` (corrigiu
  "binário estático" para darwin e o exit do `--help`).
- `codex-review --topic 004-07` rodada 001 CHANGES REQUESTED, corrigida:
  (1) tag `v*` fora de `main` publicaria release: o build em tag agora exige
  `git merge-base --is-ancestor HEAD origin/main`; (2) o trecho de instalação
  podia "verificar" um `check` vazio: virou uma cadeia `&&` com
  `grep -x` da linha exata. Testado com arquivos locais: sucesso instala;
  sem `SHA256SUMS` (rc 22) e linha não encontrada (rc 1) não instalam nada.
- Rodada 002 CHANGES REQUESTED, corrigida: `publish` (e a versão de tag)
  só em `push` de tag, nunca num `workflow_dispatch` contra tag; checkout com
  `fetch-depth: 0` para a checagem de ancestralidade com `main` não falhar
  em histórico raso.

## Pendências

- Primeira tag real (`v0.1.0`) só depois da promoção `lab` → `main`, com
  confirmação do operador.
- Registrar a versão para o install do controller do `platform-modules`
  (al-rr/platform-modules#337) depois da tag.
- Download pelo macOS não executado (só o build cruzado).
