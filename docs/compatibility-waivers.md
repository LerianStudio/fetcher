# Compatibility Waivers

## lib-commons/v5 pinned at v5.2.0

| Field | Value |
|-------|-------|
| Owner | Platform Engineering / Fetcher maintainers |
| Dependency | `github.com/LerianStudio/lib-commons/v5` |
| Current version | `v5.2.0` |
| Target version | `v5.3.0` or newer |
| Reason | `v5.3.0` currently breaks Fetcher MongoDB manager API compatibility in the tenant-manager integration path. Fetcher remains pinned until that compatibility break is resolved upstream or Fetcher receives the matching adapter migration. |
| Expiry / removal condition | Remove this waiver and upgrade once Fetcher compiles and passes worker/manager bootstrap, multi-tenant MongoDB manager, and readyz tests against `lib-commons/v5 >= v5.3.0`. Review no later than 2026-06-30. |
| Validation evidence | Current remediation keeps `go.mod` pinned to `v5.2.0`; verification must include `go test ./...`, changed-package tests, and changed-package `golangci-lint run`. |
