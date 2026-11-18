## Summary

<!-- What changed and why? Keep this to 2-5 concrete bullets. The title
follows Conventional Commits (it becomes the squash commit message). -->

-

## Scope

- Area(s) changed:
- Base branch and head branch:
- Workflow stage: topic → `lab`, or promotion `lab` → `main`
- Related issue: Closes #

## Behavior and compatibility

<!-- Describe user-visible behavior, schema/config compatibility, security impact,
or write "No user-visible behavior change". -->

## Validation

Run the checks relevant to the change and remove items that do not apply.

- [ ] `pre-commit run --all-files` (repository checks, same as the CI job `higiene`)
- [ ] `./scripts/go.sh vet ./...`
- [ ] `./scripts/go.sh test ./...`
- [ ] `./scripts/go.sh build -o bin/platformctl ./cmd/platformctl`
- [ ] `bash vagrant/tests/test-vagrantfile-contract.sh`
- [ ] `bash container/tests/test-compose-contract.sh`
- [ ] Example `validate`/`render` smoke test
- [ ] Repository paths named in the text exist (pre-commit checks only links)

## Security and operations

- [ ] No credentials, private keys, revealed renders, or private operator
      configuration were committed.
- [ ] Destructive or remote-mutating validation was not run, or is explicitly
      described below with approval/evidence.
- [ ] Documentation was updated for user-visible or operational changes.

## Risks and follow-ups

<!-- Known gaps, compatibility risks, external validation still required, or
"None". -->
