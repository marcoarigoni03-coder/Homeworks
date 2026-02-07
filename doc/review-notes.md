# Repository review notes

## Commands executed

- `rg -n "TODO|FIXME|HACK" -S .`
- `go test ./...` (stopped due to long run; no output before stop)
- `go vet ./...` (stopped due to long run; no output before stop)

## Observations

- TODOs found only in vendored dependencies under `vendor/`, not in project-owned code.
