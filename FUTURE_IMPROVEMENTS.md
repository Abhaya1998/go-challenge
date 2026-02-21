# Future Improvements

Suggestions for taking the Product Catalog app toward production readiness: UI parity, **test cases**, containerization, logging, error handling, design patterns, and structure. These are **future implementation** ideas, not requirements.

---

## 1. UI Parity with API (Missing from UI)

**Current:** Several API capabilities have no or limited UI; some JS exists but is not wired in templates.

**API features with no / partial UI:**

| API feature | UI status | Suggestion |
|-------------|----------|------------|
| **PUT /products/:id** (edit product) | No edit page | Add "Edit" on product detail; form pre-filled from GET product, submit via PUT. |
| **POST /products/:id/variants** (create variant) | No form | On product detail, add "Add variant" with SKU, name, price, quantity, attributes. |
| **PUT /products/:id/variants/:vid** (edit variant) | No form | In variants table, add "Edit" per row; modal or inline form. |
| **DELETE /products/:id/variants/:vid** | No button | Add "Remove" or "Delete" on each variant row (with confirm). |
| **GET /products/:id/reviews** + **POST** (create review) | No section on detail page | Add "Reviews" section: list reviews, form to submit (author, rating, comment). Wire existing `submitReview` in app.js. |
| **DELETE /products/:id/reviews/:rid** | JS exists, no list to act on | Show each review with "Delete" (and optionally "Approve"); wire `deleteReview`. |
| **POST .../reviews/:rid/approve** | No UI | Add "Approve" for unapproved reviews (e.g. for editor/admin role). |
| **GET /products/:id/inventory** | Not shown | Optionally show variant inventory summary (count, total stock) on product detail. |
| **GET /search?q=** | JS exists, no search UI in templates | Add search box (e.g. in nav or product list); `#search-input`, `#search-results`; wire `searchProducts`. |
| **GET /products/export** and **/products/export/json** | JS `exportProducts()` exists, no buttons | Add "Export CSV" / "Export JSON" (e.g. on product list or dashboard); wire `exportProducts(format)`. |
| **POST /products/import** | No UI | Add "Import" page or section: file input, upload CSV, show result (imported/skipped/errors). |
| **GET /categories** | No nav or filter | Add category filter on product list (dropdown or links); optionally nav link "Categories" that lists them. |
| **GET /sku/:sku** | No UI | Add "Lookup by SKU" (e.g. search box that calls `/sku/...` and shows variant or "Not found"). |
| **GET /health** | No UI | Optional: small "Status" or "Health" in nav/footer that calls `/health` and shows status/database/uptime. |
| **GET /audit** | No UI | Add "Audit log" page (e.g. admin): optional `?product_id=` and `?limit=`, table of entries. |

**Other UI improvements:**

- **Product list:** Category filter (dropdown or chips) using `?category=`.
- **Dashboard:** Ensure category table uses correct units (e.g. average price in dollars if API returns dollars).
- **Consistency:** Use the same error/success toasts and loading states for all API-backed actions (create/update/delete product, variant, review, import).

Implementing the rows above would bring the UI in line with the API and make the existing app.js helpers (search, export, review submit/delete) usable from the templates.

---

## 2. Test cases

**Current:** No automated tests; validation relies on manual checks and code review.

**Why it matters:** Tests protect against regressions, document expected behaviour, and make refactors and new features safer. They are important for production readiness.

**Suggestions:**

- **Unit tests (store):** Test store logic in isolation with an in-memory or test DB (e.g. SQLite `:memory:`). Cover: ListProducts (with/without category, deleted_at), GetProduct (cache, variant-sum), CreateProduct / UpdateProduct validation (empty name/category, negative price), DecrementQuantity / DecrementVariantQuantity (atomic, no negative), SearchProducts (variant-sum).
- **Handler / API tests:** Use `net/http/httptest` to call handlers with a test store (or mock). Cover: status codes and JSON for success and error (400, 404, 409), validation errors (category required, price &lt; 0), 404 when product is deleted (reviews, variants, inventory).
- **Integration tests (optional):** A small set of tests that start the app (or a test server) and hit real HTTP endpoints with a test DB, to validate routing and full request/response behaviour.
- **Table-driven tests:** In Go, use table-driven tests for multiple inputs (e.g. valid/invalid category, qty 0 vs &gt; 0) to keep tests readable and easy to extend.
- **CI:** Run tests in CI (e.g. GitHub Actions) on push/PR so regressions are caught early.

Prioritise store and handler tests for the code paths that were fixed (list/get product, purchase, validation, variant-sum, 404 for deleted product).

---

## 3. Containerization

**Current:** App runs via `go run .` or binary; SQLite file on host.

**Suggestions:**

- **Dockerfile (multi-stage):** Build stage with `go build -o app`, final stage with minimal image (e.g. `scratch` or `alpine`) and the binary. Reduces image size and avoids shipping source/toolchain.
- **Runtime:** Run the app in the container; persist `catalog.db` with a volume so data survives restarts.
- **docker-compose:** Single service for the app plus a volume for SQLite. Optional second service for a shared DB (e.g. Postgres) if you migrate off SQLite.
- **Env:** Keep `PORT` and `DB_PATH` (or `DATABASE_URL`) as env; inject via Compose or orchestration.

**Example (minimal):**
```dockerfile
# Build
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/server .

# Run
FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8080
ENTRYPOINT ["./server"]
```

---

## 4. Logging

**Current:** Ad-hoc `log.Printf` and `log.Printf("ERROR: ...")`; no levels, structure, or request correlation.

**Suggestions:**

- **Structured logging:** Use a library (e.g. **zerolog**, **zap**, or **slog** in Go 1.21+) so logs are JSON and queryable (level, message, error, request_id, duration).
- **Log levels:** Use `Debug`, `Info`, `Warn`, `Error` and set level via env (e.g. `LOG_LEVEL=info` in prod).
- **Request context:** Generate a request ID in middleware (or use `X-Request-ID`), add it to `context.Context`, and log it on every line for that request. Helps trace a single request across handlers and store.
- **Avoid:** Reducing `log.Printf` for business/validation errors that are already returned to the client; reserve logs for unexpected errors, panics, and audit-sensitive events.
- **Sensitive data:** Never log request/response bodies or tokens; log only method, path, status, duration, and error type.

---

## 5. Global / Centralized Error Handling

**Current:** Each handler checks errors and calls `http.Error` or `json.NewEncoder(w).Encode(...)` with different formats and status codes.

**Suggestions:**

- **Error types:** Introduce small error types or sentinel errors (e.g. `ErrNotFound`, `ErrValidation`, `ErrConflict`) so middleware or a shared helper can map them to HTTP status and body.
- **API error shape:** Standardize JSON (e.g. `{"error": "message", "code": "NOT_FOUND"}` or RFC 7807) and a single function that writes status + JSON.
- **Recovery middleware:** Keep panic recovery in middleware; optionally log with stack and return a generic 500 and request ID (no stack to client).
- **Validation errors:** Return 400 with a list of field errors (e.g. `{"errors": [{"field": "price", "message": "must be non-negative"}]}`) instead of a single string when useful.
- **Handler signature:** Consider `func (h *Handler) Handle(ctx context.Context, w http.ResponseWriter, r *http.Request) error` and a single place that converts returned `error` to HTTP response (and logs).

---

## 6. Design Patterns

**Current:** Handlers call store directly; no interfaces; config and dependencies passed or global.

**Suggestions:**

- **Repository / Store interface:** Define interfaces per aggregate (e.g. `ProductStore`, `VariantStore`, `ReviewStore`) in the same package that uses them. Implementations live in a `store` or `repository` layer. Enables testing with mocks and swapping SQLite for another DB without touching handlers.
- **Dependency injection:** Construct server with store, logger, and config in `main` (or an `app` package) and pass them into handlers. Avoid global state and make tests and alternate configs easy.
- **Config struct:** Load `PORT`, `DB_PATH`, `LOG_LEVEL`, etc. into a `Config` struct (from env or file) and pass it through; validate once at startup.
- **Optionally:** Separate “transport” (HTTP parsing, status codes) from “application” (business rules). Handlers only parse request and call application services; services return domain errors; transport layer maps them to HTTP. Keeps HTTP details out of business logic.

---

## 7. Folder / Package Structure

**Current:** All Go files in repo root, single `package main`.

**Suggestions (pick one direction and stay consistent):**

**Option A – Layered by technical role**
```
cmd/
  server/
    main.go          # build entry; wires config, store, server, run
internal/
  config/
    config.go
  store/             # or repository
    store.go
    store_reviews.go
    store_variants.go
    store_stats.go
  server/
    server.go
    routes.go
  handler/
    product.go
    variant.go
    review.go
    ...
  middleware/
    middleware.go
  model/             # or domain
    product.go
    variant.go
    ...
pkg/                 # only if you have reusable, non-app-specific code
  errutil/
    errors.go
```

**Option B – Group by domain (DDD-style)**
```
cmd/server/main.go
internal/
  product/
    handler.go
    store.go
    model.go
  variant/
    ...
  review/
    ...
  shared/
    middleware.go
    errors.go
```

**Option C – Minimal split**
```
cmd/server/main.go
internal/
  api/               # HTTP handlers + routes
  store/             # all DB access
  model/             # structs and DTOs
```

**Recommendation:** Start with **Option A** or **Option C** so “handlers,” “store,” and “model” are clearly separated. Move to domain-based (Option B) only if the domain grows and you want ownership by feature.

---

## 8. RBAC (Role-Based Access Control)

**Current:** No authentication or authorization; all endpoints are effectively public.

**Suggestions:**

- **Roles:** Define roles (e.g. `viewer`, `editor`, `admin`). Viewer: read-only (list, get, search, export, stats, health). Editor: + create/update/delete products and variants, purchase, manage reviews (create, delete, approve). Admin: + import, audit log, user/role management if you add it.
- **Auth mechanism:** Add authentication first (e.g. JWT in `Authorization` header, or session cookies). Middleware extracts identity and attaches role to `context.Context`. Handlers or a second middleware check role before proceeding.
- **Per-route or per-resource:** Either protect entire routes by role (e.g. `POST /products` requires `editor`) or use a small policy layer (e.g. `requireRole(ctx, "editor")` at the start of handlers that mutate data). Optionally, soft-delete or audit might be admin-only.
- **UI:** Hide or disable actions the user's role doesn't allow (e.g. no Delete/Edit for viewers; no Import for non-admins). Keep API enforcing checks so that direct API calls still respect RBAC.
- **Audit:** Once RBAC exists, consider logging who (user/role) did what (action, resource id) for sensitive operations (delete, approve, import).

---

## Summary

| Area              | Direction |
|-------------------|-----------|
| UI parity         | Edit product/variant, reviews section (list/add/delete/approve), search/export/import, category filter, SKU lookup, health/audit pages. |
| Test cases        | Unit tests for store (variant-sum, validation, atomic decrement); handler/API tests (status, validation, 404); table-driven; CI. |
| Containerization  | Dockerfile (multi-stage), volume for DB, docker-compose; env for PORT/DB_PATH. |
| Logging           | Structured logs (zerolog/zap/slog), levels, request ID in context. |
| Error handling    | Typed/sentinel errors, single API error format, recovery in middleware. |
| Design patterns   | Store interfaces, DI in main, config struct, optional transport/service split. |
| Folder structure  | `cmd/server`, `internal/{store,handler,config,model}`, optional `pkg` for shared libs. |
| RBAC              | Roles (e.g. viewer/editor/admin), auth (JWT/session), per-route or per-handler checks, audit who did what. |

Implement these incrementally (e.g. add Docker first, then structured logging, then error types and folder layout) so the app stays stable while improving operability and maintainability.
