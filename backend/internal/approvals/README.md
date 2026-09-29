# internal/approvals

Finance decisions on payout requests: approve, reject, mark paid, and payment failed.

Layers: handler → service → repository. `SetupHandler` injects those dependencies. Routes live in `internal/router/approval_router.go` and are not registered from `main` yet.
