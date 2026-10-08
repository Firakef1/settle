# Frontend ↔ Backend API contract

Base URL: `/api/v1` — proxied by Next.js to `API_PROXY_TARGET` (default `http://localhost:8080`), see `next.config.ts`. The Go API sends no CORS headers, so call it through the proxy.

Auth: `Authorization: Bearer <JWT>`

The source of truth for every route, request body and response shape is
[`backend/docs/Api-contract.md`](../../backend/docs/Api-contract.md). Types in
`shared/types/index.ts` and the clients in `shared/services/` follow it.

Errors come back as `{ "error": "...", "code"?: "..." }`. Use `getApiError` /
`apiErrorMessage` from `shared/utils/apiError.ts` to read them.
