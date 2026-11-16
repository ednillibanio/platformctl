# Etapa: Base do pacote `internal/sharedroot`

Status: `done`

## Objetivo

Fundação comum dos três comandos, conforme as seções "Root", "Names",
"Paths" e "Commands" do standard.

## Escopo

- resolução do root: `--config-dir`, senão `ASSEMBLEIARR_CONFIG_DIR`, senão
  `~/.config/assembleiarr`; caminho absoluto obrigatório
- validação de nomes (`<env>` contra `catalog/environments/` por padrão
  regex, `<t>`, `<consumer>`, `<identity>`, `NAME`) antes de qualquer I/O
- escrita segura: arquivo novo em nome temporário no mesmo diretório com o
  modo final, `fsync`, publicação por `os.Link` (falha se o destino existe)
  e remoção do temporário; diretório publicado por rename sem clobber
  (`renameat2(RENAME_NOREPLACE)` no Linux, `renamex_np(RENAME_EXCL)` no
  macOS, via `golang.org/x/sys`)
- criação de diretórios `0700` sem alterar diretórios existentes; recusa de
  symlink em qualquer componente dentro do root
- erros tipados e mapeamento para exit 0/1/2; "not supported" no Windows
  (build tags)

## Fora de Escopo

- comandos de CLI, parsers, chaves

## Critérios de Aceite

- testes cobrem: root override e default, nomes inválidos (incluindo `/`,
  `..`, maiúsculas), publicação concorrente (só um vence, nenhum resto),
  destino existente nunca alterado, symlink recusado, modos criados
  independentes do `umask`

## Validação

```bash
docker run --rm -v "$PWD":/src:Z -w /src -e HOME=/tmp docker.io/library/golang:1.26.5 ./scripts/go.sh vet ./...
docker run --rm -v "$PWD":/src:Z -w /src -e HOME=/tmp docker.io/library/golang:1.26.5 ./scripts/go.sh test ./...
```

## Resultado

Pacote `internal/sharedroot` criado e etapa fechada (2026-11-11, operador
autorizou commit e PR para `lab`).

- `errors.go`: `ErrExists`, `ErrSymlink`, `ErrNotDirectory`,
  `ErrNotSupported`, `UsageError`/`Usagef` e `ExitCode` (0 sucesso, 2 uso,
  1 demais).
- `root.go`: `ResolveRoot` (`--config-dir`, senão `ASSEMBLEIARR_CONFIG_DIR`,
  senão `~/.config/assembleiarr` a partir de `os.UserHomeDir`); override
  relativo é erro de uso.
- `names.go`: padrões do standard para env, target, consumer, identity e
  `NAME`; identidades `active` e `reserved` da tabela do standard. Decisão
  do operador (2026-11-11): o `platformctl` mantém a própria lista de
  ambientes (`lab`, `prod`, espelho de `catalog/environments/` em main
  561f0ca), atualizada no mesmo PR que acompanha um ambiente novo no
  catálogo.
- `tree_unix.go` (Linux/macOS): `Tree` aberto por descritor do root; cada
  componente é aberto com `O_DIRECTORY|O_NOFOLLOW` e toda operação usa
  syscalls `*at`, então um symlink trocado depois de uma checagem não
  redireciona a escrita. `EnsureDir` cria `0700` via `Fchmod` (independe
  do `umask`) e não altera diretórios existentes; root e pais ausentes são
  criados componente a componente do mesmo modo. `PublishFile`: temporário
  `O_EXCL|O_NOFOLLOW` com modo final, `fsync`, `Linkat` (destino existente =
  `ErrExists`), remoção do temporário e `fsync` do diretório. `PublishDir`:
  diretório temporário preenchido e publicado por `renameat2(RENAME_NOREPLACE)`
  / `renameatx_np(RENAME_EXCL)`; perdedor ou falha remove o temporário.
  Temporários usam o prefixo `TempPrefix` (`.platformctl-tmp-`) para o
  `doctor`.
- `tree_other.go`: demais sistemas (Windows) retornam `ErrNotSupported`.
- `go.mod`: `golang.org/x/sys v0.38.0` (a v0.50.0 elevaria a diretiva para
  `go 1.26.0`; a v0.38.0 mantém `go 1.24.0`).
- `.gitignore`: `/.agent-runs/` (pré-requisito do `codex-review`).

## Evidências

- Container `docker.io/library/golang:1.26.5` (Podman), com
  `GOPATH=/src/.cache/gopath`: `gofmt -l` vazio; `vet ./...` com
  `GOOS=linux`, `darwin` e `windows`; testes do pacote compilados nos três;
  `./scripts/go.sh test ./...` verde (também após a lista de ambientes); `test -race -count=3
  ./internal/sharedroot` verde.
- Testes cobrem: precedência e default do root, override relativo, nomes
  inválidos (`/`, `..`, maiúsculas, reservados), root/diretórios `0700` com
  `umask 0777`, diretório existente não alterado, symlink recusado (root,
  componente e destino), publicação concorrente de arquivo e diretório (16
  runs, só um vence, conteúdo inteiro do vencedor, nenhum temporário),
  destino existente nunca alterado, falha remove o temporário.
- `codex-review --topic 004-01`: rodada 001 CHANGES REQUESTED (High: `Chmod`
  após `Mkdir` seguia o caminho; Medium: `MkdirAll` criava pais com
  `umask`), corrigida; rodada 002 APPROVE sem achados. Registros em
  `.agent-runs/codex-reviews/20261111_19*-004-01-00{1,2}.*` (ignorados).
- CI do PR #35 (run #61) falhou em macOS e no container-lab: com `umask
  0777`, `Mkdirat(0700)` criava diretório `0000` que um usuário não root não
  abre (os testes locais rodavam como root). Correção: `mkdirOpenAt` cria,
  abre com `O_NOFOLLOW` e aplica `Fchmod 0700`; um `umask` que remove bits
  do dono vira `ErrUmask`, com o diretório vazio recém-criado removido. Não
  altera o `umask` do processo nem faz `chmod` por caminho. Testes passam a
  rodar também como não root (`--userns=keep-id --user 1000:1000`).
- `codex-review` rodada 003 (sobre uma primeira correção que estreitava o
  `umask` do processo): CHANGES REQUESTED (Medium: `umask` é global do
  processo); trocado pela abordagem acima. Limite de 3 rodadas atingido; a
  correção final não teve rodada extra.
- Docs de usuário não mudaram nesta etapa: audit `technical-documentation`
  não se aplica.

## Pendências

- Sistemas de arquivos sem hard link ou sem `renameat2` (por exemplo alguns
  synced folders do Vagrant) falham com erro explícito; não há fallback
  não atômico, de propósito.
- O ancestral existente do root pode ser alcançado por symlink (fora da
  árvore, como `~/.config` em dotfiles); só o root e o que está abaixo dele
  são recusados.
- `umask` que remove bits do dono (por exemplo `0777`) é recusado com
  `ErrUmask`; bits de grupo/outros do `umask` nunca importam.
- `ExitCode` ainda não está ligado ao `main` (comandos entram na etapa 03).
