# CLAUDE.md

**go-kit** — Go packages shared by Grupa's Go services. Module `github.com/grupa-repo/go-kit` (lowercase; the repository is `grupa-repo/Go-kit`). A library: nothing here runs or deploys on its own.

## Commands

- `go vet ./...` · `go test -race ./...` · `gofmt -l .` (must print nothing)
- CI (`.github/workflows/ci.yml`) runs those plus staticcheck on every push and pull request.

## Packages

- `apperr/` — errors whose message is safe for an API caller; `Message(err)` extracts it, `""` otherwise.
- `problem/` — the `application/problem+json` error body.
- `requestid/` — `X-Request-ID` middleware and `FromContext`. Rejects inbound values that are unsafe to log.
- `httpjson/` — `Write` and `Decode` (1 MiB cap, `ErrNoBody`, `ErrTooLarge`).
- `apidocs/` — `Register(router, enabled, title, spec)`; disabled means not registered, so the paths 404. `Router` is a one-method interface chi satisfies.
- `sqlpool/` — `Open(ctx, driver, dsn, Limits)`; `MaxOpen` is required. Tests use a fake `database/sql` driver, no database.
- `appenv/` — `IsDeployed` (deny-list of local names) and `DocsEnabled` (allow-list). The opposite shapes are deliberate; do not unify them.
- `secret/` — `Fingerprint`. Its output is compared across services and tools, so the algorithm must never change; the test pins known values.
- `httpserver/` — `Server.Run(ctx)` drains in-flight requests on cancel; `SignalContext` cancels on SIGINT/SIGTERM. Tests use `Serve` on a `127.0.0.1:0` listener.

## Rules

- **Only what more than one service needs.** Code one service uses belongs in that service.
- **Standard library only** unless asked. Every dependency added here lands in every consumer.
- **Nothing service-specific:** no hostnames, app names, error codes, configuration or credentials. Write every file as if the repository were public.
- **An exported name is contract.** Changing or removing one breaks consumers at their next version bump: add, do not rename, and call out any breaking change so the tag gets a major (or, before v1, a minor) bump.
- **JSON field names in `problem` are API contract** with clients. Never change them.
- **A change is not live until it is tagged and each consumer bumps its `go.mod`.** Say so when finishing work here.
- **Never move or reuse a tag.** Go's checksum database pins each version's contents.
- Every package has tests; a new exported function comes with one.
- Commit to `main`. Do not create a branch unless asked.
