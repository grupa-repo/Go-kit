# go-kit

Small Go packages shared by Grupa's Go services. It holds only what more than one service would otherwise copy.

```
go get github.com/grupa-repo/go-kit
```

The module path is lowercase, although the repository is named `Go-kit`. GitHub ignores the case; Go does not, so import it exactly as written above.

## Packages

| Package | What it is |
|---|---|
| `apperr` | Marks the errors whose text is safe to return to an API caller. `apperr.Message(err)` is the only way an error's text reaches a response. |
| `problem` | The `application/problem+json` error body, with the `error_code` field clients switch on. |
| `requestid` | Middleware that takes `X-Request-ID` or generates one, plus `FromContext` to read it back for logging. |
| `httpserver` | Runs an HTTP server that drains in-flight requests when its context is cancelled, with `SignalContext` for SIGTERM. |
| `httpjson` | `Write` a JSON response; `Decode` a request body, capped at 1 MiB. |
| `apidocs` | Serves an OpenAPI document and a Scalar viewer at `/docs`, registered only when enabled. |
| `sqlpool` | Opens a `database/sql` pool with explicit connection limits and refuses an unlimited one. The caller imports the driver. |
| `appenv` | `IsDeployed` and `DocsEnabled`: the two decisions that depend on the `APP_ENV` name, each failing safe for an unrecognised one. |
| `secret` | `Fingerprint`: a short hash for comparing a secret across services without printing it. |

```go
var ErrTaskNotFound = apperr.New("task does not exist")

func (h *Handler) getTask(w http.ResponseWriter, r *http.Request) {
	task, err := h.tasks.Get(r.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Msg("get task")
		problem.Write(w, http.StatusNotFound, problem.New("tasks_get_not_found", "Not Found", apperr.Message(err)))
		return
	}
	// ...
}
```

```go
func main() {
	ctx, stop := httpserver.SignalContext(context.Background())
	defer stop()

	srv := httpserver.Server{Addr: ":8080", Handler: requestid.Middleware(router)}
	if err := srv.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
```

## Working on it next to a service

Create a `go.work` in the folder that holds the repositories, listing this one and the service you are changing. `go.work` is gitignored, so it stays local.

```
go work init ./Go-kit ./your-service
```

## Rules for what goes in

- More than one service needs it. Code only one service uses stays in that service.
- Standard library only, unless there is a strong reason. A dependency here becomes a dependency of every service.
- No service-specific names, codes or configuration.
- Released by tag. Services pin a version in `go.mod`.

## Releasing

A change here reaches a service in two steps: a tag, then a version bump in that service. Until both happen, nothing changes in production.

1. Merge to `main` and wait for CI to pass.
2. Tag the commit and push the tag:

   ```
   git tag -a v0.2.0 -m "v0.2.0"
   git push origin v0.2.0
   ```

3. In each service: `go get github.com/grupa-repo/go-kit@v0.2.0`, then build and test.

Versions follow `vMAJOR.MINOR.PATCH`. While the module is on `v0.x`, a breaking change bumps the minor version and everything else bumps the patch. A breaking change is anything that stops a service compiling or changes what it sends: a removed or renamed export, a changed signature, a changed JSON field.

Never move or delete a tag that has been pushed. Go's public proxy and checksum database keep the first contents they saw for a version, so a moved tag fails verification for everyone who fetches it. Fix a bad release with a new one.

## Development

```
go vet ./...
go test -race ./...
```
