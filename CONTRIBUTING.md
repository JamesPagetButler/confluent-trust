# Contributing to CTH

CTH follows the same conventions as the Helpful Engineering BMA project.
The two binding documents are:

- [`go-coding-guide.md`](../../BMA/doc/go-coding-guide.md) — Go style, package design, error handling, testing
- [`github-practices.md`](../../BMA/doc/github-practices.md) — branching, commits, issues, CI, signing

This file summarizes the parts that matter most for first-time contributors.

## Dev setup

Requirements:

- Go 1.24+
- `golangci-lint` (install: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`)
- `gh` CLI authenticated (`gh auth status`)

```bash
git clone git@github.com:JamesPagetButler/confluent-trust.git
cd confluent-trust
go build ./...
go test -race ./...
golangci-lint run
```

## Workflow

For each issue:

1. **Plan** — read the issue, design approach, post the plan as an issue comment
2. **Branch** — `feat/<short-slug>` (e.g., `feat/04-entropy`)
3. **Build** — implement, write tests, keep PRs under ~400 lines when possible
4. **PR** — `gh pr create` with the template; link the issue with `Closes #N`
5. **Review** — wait for CI green and self-review the diff in GitHub UI
6. **Squash-merge** — single clean commit on `main`; PR retains full history

## Commits — Conventional Commits

```
type(scope): subject

Body explains *why*, not *what*. Wrap at 72.

Co-authored-by: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
```

Types: `feat`, `fix`, `refactor`, `test`, `doc`, `chore`, `perf`.
Scopes: `model`, `compute`, `store`, `report`, `cmd`, `schema`, `ci`, `infra`.

Subject: imperative, lowercase after the colon, no period, ≤50 chars.

## Signed commits (SSH)

Branch protection on `main` will eventually require signed commits. Set
up locally:

```bash
git config --local commit.gpgsign true
git config --local gpg.format ssh
git config --local user.signingkey ~/.ssh/id_ed25519.pub
```

Upload `id_ed25519.pub` to GitHub as both an Authentication key **and**
a Signing key (Settings → SSH and GPG keys).

## Code style highlights

- Pure Go in `model/` and `compute/` — stdlib only, no external deps.
- Package names lowercase, single word, no underscores. No `utils`, `common`, `helpers`.
- Acronyms uppercase: `ID`, `MI`, `JSON`. So `ChainID`, `MutualInfoBits`.
- Errors lowercase, no period: `fmt.Errorf("model: anchor %s: %w", id, err)`.
- Don't create interfaces with one implementation. Wait for the second backend.
- `t.Helper()` in test helpers; `t.Context()` for cancellation.
- `iota` enums need explicit `MarshalJSON`/`UnmarshalJSON` — never serialize the integer.

Run `gofmt -w .`, `goimports -w .`, `go vet ./...`, `golangci-lint run`
before pushing.

## Adding a new fixture

Every fixture in `testdata/` must validate against
`schema/inventory.schema.json` — the schema test in `internal/validate/`
will fail CI if it doesn't.
