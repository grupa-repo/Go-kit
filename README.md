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

## Using a private module

This repository is private, so Go needs to be told not to use the public proxy and needs credentials to fetch it:

```
go env -w GOPRIVATE=github.com/grupa-repo/*
```

CI and any hosted build that compiles a consumer need read access as well.

## Working on it next to a service

Create a `go.work` in the folder that holds the repositories, listing this one and the service you are changing. `go.work` is gitignored, so it stays local.

```
go work init ./Go-kit ./your-service
```

## Rules for what goes in

- More than one service needs it. Code only one service uses stays in that service.
- Standard library only, unless there is a strong reason. A dependency here becomes a dependency of every service.
- No service-specific names, codes or configuration.
- Released by tag (`vMAJOR.MINOR.PATCH`). Services pin a version in `go.mod`.

## Development

```
go vet ./...
go test -race ./...
```
