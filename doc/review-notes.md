# Repository review notes

## Commands executed

- `rg -n "TODO|FIXME|HACK" -S .`
- `go test ./...`
- `go vet ./...`

## Observations

- TODOs found only in vendored dependencies under `vendor/`, not in project-owned code.
- OpenAPI spec now includes the implemented `/liveness` and `/api/logout` endpoints for consistency with the router.
