# Contributing

The shared issue, branch, commit, and pull request workflow across the
platform ecosystem (`platform-catalog`, `platform-modules`,
`provision-*`, `platformctl`, `platform-labs`) is documented at:

```text
platform-catalog/docs/standards/contribution-workflow.md
```

That document is normative; this file only records what is specific to
`platformctl`.

## Repository-local notes

- **Integration branch:** `lab`. `main` is the default branch and only
  receives reviewed `lab` → `main` promotion pull requests. Never commit or
  push directly to either — branch from `lab`, open a pull request targeting
  `lab`, and merge only after explicit operator confirmation.
- **Planning:** non-trivial work is an iteration under
  `iterations/NNN-topic/` (see [`iterations/README.md`](iterations/README.md)
  for the approval gates and evidence rules, and
  [`iterations/BACKLOG.md`](iterations/BACKLOG.md) for proposed work). One
  GitHub issue per iteration, never one per step.
- **Issue types:** `.github/ISSUE_TEMPLATE/` provides `iteracao.yml`,
  `melhoria.yml`, and `bug.yml`. Issues and pull requests are assigned to the
  operator.
- **Pull requests:** use
  [`.github/PULL_REQUEST_TEMPLATE.md`](.github/PULL_REQUEST_TEMPLATE.md).
  Reference the issue with `Refs #N`; `main` is the default branch, so
  `Closes #N` does not auto-close issues merged into `lab` — close them
  manually once acceptance is confirmed.
- **Validation:** Go runs only through `scripts/go.sh` / `scripts/go.ps1`,
  and is exercised in the disposable `container/` or `vagrant/` labs rather
  than on the operator host. See [Development](docs/development.md) for the
  commands, static lab contracts, and CI.
- **Language:** code, CLI help, errors, logs, and `docs/` are in English;
  files under `iterations/` are in Portuguese.
- **Cross-repository programs:** when work spans several repositories, the
  coordinating document lives in `platform-catalog`; each repository
  keeps only its own issue and iteration and links back to it.
