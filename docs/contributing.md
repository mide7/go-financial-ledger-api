# Contributing

## Documentation rule

**Every change updates the documentation it affects, in the same commit.** A
change isn't finished while any of the docs below describe the old behaviour.
Reviewers should reject a change that alters behaviour without the matching doc
update.

Find the docs a change affects with this map:

| If the change… | Update |
|---|---|
| adds, removes or changes a route, its request fields, validation rules, response shape or status codes | [api.md](api.md): the endpoints table **and** the route's section |
| adds or changes a domain error, or its HTTP mapping | the domain's **Errors** table in [domains/](domains/) and, for a new status code, [api.md → Errors](api.md#errors) |
| adds or changes a service or repository method, a domain type or a business rule | the domain doc in [domains/](domains/) |
| adds a domain or a package, or changes wiring or middleware | [architecture.md](architecture.md) (package layout, wiring, request flow) and the domain table in [README.md](README.md) |
| changes a migration, query, trigger or index | [database.md](database.md) |
| adds or changes configuration, Make targets, Docker or seeding | [getting-started.md](getting-started.md) |
| adds a test type, tag or command | [testing.md](testing.md) |
| fixes or introduces a bug or limitation | [known-issues.md](known-issues.md) (remove the fixed entry; add the new one) |
| completes or changes planned work | [todo.md](todo.md) (tick the item, remove its `TODO(<tag>)` markers) |
| changes what works overall | the **Status** section of [README.md](README.md) |

Also:

- **Godoc:** keep doc comments on exported identifiers true; a comment that
  describes old behaviour is a bug.
- **Examples:** a JSON example in the docs must match the current DTO and
  response field names.
- **Dates:** use absolute dates (`2026-11-24`), never "recently" or "now".
- **Pure refactors** that change no behaviour need no doc update unless they
  move or rename something a doc links to.

## Before committing

```sh
go build ./... && go vet ./... && go vet -tags integration ./...
go test -race -shuffle=on ./...
golangci-lint run ./... && golangci-lint run --build-tags integration ./...
```

Then check the doc map above against the diff.
