# 001 — security.txt endpoint (frontend) — Behaviour spec

## 1. In scope to test

Observable HTTP behaviour of `GET /.well-known/security.txt` served by `src/frontend`:

- Response status code (200 OK).
- Response `Content-Type` header is exactly `text/plain; charset=utf-8`.
- Response body contains a `Contact:` field with the exact value
  `mailto:security@example.com`.
- Response body contains an `Expires:` field whose value parses as an RFC 3339 timestamp,
  is strictly in the future relative to request time, and is no more than ~1 year out.
- Response body contains ONLY the two required fields (`Contact`, `Expires`) — no other
  `Name:` field lines (e.g. no `Policy:`, `Preferred-Languages:`, `Encryption:`, etc.).
- Each field appears on its own line, in `name: value` form.
- Routing is scoped to the exact path — an unrelated path does not resolve to this handler.

## 2. Out of scope to test

- OpenPGP / digital signature of the file (not implemented — RFC 9116 §2.3 optional).
- Any optional/recommended fields (`Policy`, `Preferred-Languages`, `Encryption`,
  `Acknowledgments`, `Canonical`, `Hiring`) — none are emitted by design.
- The exact literal value of `Expires` (it is computed per-request; only its parseability,
  future-ness, and ~1-year bound are asserted).
- Internals of the RFC 3339 parsing library/format (`time.Parse`/`time.RFC3339` correctness
  is Go stdlib's concern, not this handler's).
- Any other route on the router (`/robots.txt`, `/_healthz`, `/static/`, product/cart/checkout
  routes, etc.) beyond the one negative routing scenario below.
- HEAD-method behaviour: the research flagged this as an open question with no locked
  decision, so it is intentionally NOT asserted here. If the implementation ends up
  supporting HEAD via `HandleFunc`'s default behaviour, that's incidental, not a spec'd
  requirement.
- CRLF vs LF line-ending choice (RFC 9116 §2.2 permits both; tests key off field content, not
  exact byte-for-byte line endings).

## 3. Seams

No external seams — the handler is a static, self-contained response with no gRPC/backend
dependency. Tests exercise the real handler directly via `net/http/httptest`:

- Build a `httptest.NewRequest("GET", "/.well-known/security.txt", nil)` and a
  `httptest.NewRecorder()`, then invoke either:
  - the mux router built the same way `main.go` builds it (preferred — also proves the route
    is registered and reachable at the expected path under the default empty `baseUrl`), or
  - the handler function directly, if it's exposed as a package-level `func(w http.ResponseWriter, r *http.Request)` value.
- No mocks, no fakes — pure `net/http` + `net/http/httptest` + stdlib `time`/`strings` for
  assertions.

**Implementation note (not a test requirement):** for the router-based approach to be
testable, the router construction (or at least route registration) needs to be reachable from
a test — e.g. a small helper that builds the `*mux.Router` with routes registered, without
requiring the full `main()` (server startup, env vars, backend gRPC dials, etc.) to run. Flag
this to the implementer; the test spec does not mandate a specific refactor, only that the
route must be exercisable via `httptest` without live backend dependencies.

## 4. Scenarios

- **T-001** [happy] Given the frontend router with default (empty) `baseUrl`, when a client
  sends `GET /.well-known/security.txt`, then the response status is `200 OK`.

- **T-002** [happy] Given the same request as T-001, when the response is received, then the
  `Content-Type` header is exactly `text/plain; charset=utf-8`.

- **T-003** [happy] Given the same request as T-001, when the response body is inspected,
  then it contains a line `Contact: mailto:security@example.com`.

- **T-004** [happy] Given the same request as T-001, when the response body is inspected,
  then it contains a line matching `Expires: <value>` where `<value>` parses successfully as
  RFC 3339 (`time.Parse(time.RFC3339, value)`), the parsed time is strictly after the test's
  "now", and is not more than 366 days after "now" (allows for leap-year slack).

- **T-005** [happy] Given the same request as T-001, when the response body is split into
  non-comment, non-blank lines, then exactly two field lines are present and their field
  names are exactly `Contact` and `Expires` (no `Policy`, `Preferred-Languages`,
  `Encryption`, `Acknowledgments`, `Canonical`, or `Hiring` lines).

- **T-006** [happy] Given the same request as T-001, when each field line is inspected, then
  it matches the `name: value` shape (a field name, a colon, a single space, then the value)
  for both `Contact` and `Expires`.

- **T-007** [edge/negative] Given the same router, when a client sends `GET /some/unrelated/path`
  (e.g. `/foo`), then the response is NOT the security.txt body (status is not the security.txt
  handler's 200-with-Contact-line response — e.g. it 404s or is routed to a different handler,
  per gorilla/mux's normal unmatched-route behaviour). This guards against an over-broad route
  registration (e.g. an accidental `PathPrefix` instead of an exact path match).

All scenarios are directly executable as Go tests using `net/http/httptest` (`ResponseRecorder`
+ either the constructed router or the handler function), asserting on status code, headers,
and body content via `strings`/`time` from the standard library.
