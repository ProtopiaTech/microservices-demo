# 001 — security.txt — Behaviour spec (tests)

**Unit:** `GET /.well-known/security.txt` on the `frontend` Go service, RFC 9116 compliant.

Locked decisions this spec pins (from the research checkpoint — not re-opened here):
- Fields: REQUIRED ONLY — exactly `Contact` and `Expires`, nothing else.
- `Contact` value: placeholder `mailto:security@example.com`.
- Route registered at the **literal root** `/.well-known/security.txt` (not under `baseUrl`).
- `Expires` generated **dynamically** at request time as a rolling ~6 months in the future,
  formatted `time.RFC3339`; always in the future and less than 1 year out.

## 1. In scope / Out of scope

**In scope to test**
- HTTP status code of the response.
- `Content-Type` response header value.
- Presence and exact value of the `Contact` field line.
- Presence, uniqueness, and validity (RFC 3339, in-the-future, <1 year out) of the `Expires` field
  line.
- That no field lines other than `Contact` and `Expires` appear in the body (pins the
  "required-only" decision).
- That the route is reachable at the literal path `/.well-known/security.txt`, independent of the
  `baseUrl`/`BASE_URL` mechanism used by other routes.
- Allowed HTTP method(s) for the route (`GET`, mirroring the other routes' `.Methods(...)`
  declarations).

**Out of scope to test**
- Digital signing (OpenPGP cleartext signature) — not implemented, per research §"NOT in scope".
- Real security-team infrastructure (a real inbox, PGP key, disclosure workflow) — the `Contact`
  value is an acknowledged placeholder, not a live channel.
- TLS/HTTPS enforcement — a deployment/ingress concern, not something the Go handler controls.
- The `baseUrl` prefix behaviour of *other* routes (`/`, `/cart`, `/product/{id}`, etc.) — untouched
  by this unit.
- Any RECOMMENDED RFC 9116 field (`Policy`, `Preferred-Languages`, `Encryption`,
  `Acknowledgments`, `Canonical`, `Hiring`) — explicitly excluded by the "required only" decision.
- gRPC backends / `frontendServer` dependencies — this handler has no outbound seam.

## 2. Seams

- **Router/handler seam only.** The test drives the registered route through
  `net/http/httptest`: build an `httptest.NewRecorder()` + an `http.Request` (via
  `httptest.NewRequest`), dispatch it through the `*mux.Router` that owns the
  `/.well-known/security.txt` registration, and assert on the recorded response
  (`recorder.Code`, `recorder.Header()`, `recorder.Body.String()`).
- **No gRPC backends, no mocks.** The handler is pure — no call to `frontendServer` fields or any
  backend service — so the test needs no fakes/mocks and does not need to construct a
  `frontendServer{}` at all if the handler is a free function/closure (consistent with the
  `robots.txt`/`_healthz` pattern at `main.go` lines 160-161).
- **How the test obtains a router:** this repo currently has **no `*_test.go` files** under
  `src/frontend/`, so this unit introduces the first one (e.g. `src/frontend/main_test.go`).
  The test needs a `*mux.Router` (or the bare handler) that has the security.txt route registered.
  Two viable options, left to the test author/coder to pick at implementation time:
  1. A small test-only helper that builds a minimal `mux.Router` and registers just this one
     route (fast, fully isolated, but registers the route a second time outside `main()` — a
     minor duplication risk flagged as an open question below).
  2. Exercising the real route registration path from `main.go` (e.g. by extracting the
     `r.HandleFunc(".../.well-known/security.txt", ...)` line into a small exported/testable
     registration function called from both `main()` and the test), which avoids duplication
     but is an implementation-shape decision for the plan step, not this spec.
  Either way, the *observable behaviour* under test is identical: a GET to
  `/.well-known/security.txt` on the assembled router returns the RFC 9116-conformant response.

## 3. Scenarios

### T-001 — Happy path: 200 OK
- **Tag:** happy-path
- **Given** the frontend router has the `/.well-known/security.txt` route registered
- **When** the test sends `GET /.well-known/security.txt`
- **Then** the response status code is `200`

### T-002 — Content-Type is exactly `text/plain; charset=utf-8`
- **Tag:** happy-path
- **Given** the frontend router has the `/.well-known/security.txt` route registered
- **When** the test sends `GET /.well-known/security.txt`
- **Then** the response `Content-Type` header equals exactly `text/plain; charset=utf-8`
  (not merely containing "text/plain" — the full RFC 9116 §3/§4-mandated value)

### T-003 — Contact field present with the correct placeholder value
- **Tag:** happy-path
- **Given** the frontend router has the `/.well-known/security.txt` route registered
- **When** the test sends `GET /.well-known/security.txt`
- **Then** the response body contains a line exactly equal to
  `Contact: mailto:security@example.com`

### T-004 — Expires field present exactly once, RFC 3339-valid, and in the future
- **Tag:** happy-path
- **Given** the frontend router has the `/.well-known/security.txt` route registered
- **When** the test sends `GET /.well-known/security.txt`
- **Then** the response body contains **exactly one** line starting with `Expires:`;
  **and** the value after `Expires: ` parses successfully with `time.Parse(time.RFC3339, value)`;
  **and** the parsed time is after `time.Now()` (in the future);
  **and** the parsed time is less than `time.Now().AddDate(1, 0, 0)` (under ~1 year out — RFC 9116
  §2.5.5 RECOMMENDED bound)

### T-005 — Only required fields appear (no RECOMMENDED/extension fields)
- **Tag:** edge-negative
- **Given** the frontend router has the `/.well-known/security.txt` route registered
- **When** the test sends `GET /.well-known/security.txt`
- **Then** every non-comment, non-blank line in the body matches the field-line pattern
  `^(Contact|Expires): `; **and** no line begins with any other known RFC 9116 field name
  (`Policy:`, `Preferred-Languages:`, `Encryption:`, `Acknowledgments:`, `Canonical:`, `Hiring:`),
  pinning that the "required-only" decision holds

### T-006 — Route resolves at the literal root, independent of `BASE_URL`
- **Tag:** edge-negative
- **Given** the frontend router is constructed with a non-empty `baseUrl` (e.g. simulating
  `BASE_URL=/shop` the way `main.go` line 111 would set it), while the security.txt route itself
  is registered at the literal `/.well-known/security.txt` (per the locked routing decision)
- **When** the test sends `GET /.well-known/security.txt` (the literal path, with no `baseUrl`
  prefix)
- **Then** the response status is `200` (the route resolves at the domain root regardless of
  `baseUrl`)
- **Seam note:** expressing "independent of `BASE_URL`" as an observable behaviour depends on how
  the test constructs the router (see §2's two options). If the test only ever builds a router
  with the route registered directly at the literal path, this scenario reduces to "the route is
  registered outside/independent of any `baseUrl`+prefix call" — verified by inspecting that the
  registration under test does not concatenate `baseUrl`, rather than by varying an actual
  `BASE_URL` env var end-to-end. Treat as a seam limitation, not a scenario to weaken.

### T-007 — GET is accepted
- **Tag:** happy-path
- **Given** the frontend router has the `/.well-known/security.txt` route registered
- **When** the test sends `GET /.well-known/security.txt`
- **Then** the response status is `200` (restates T-001's method explicitly, to pin GET as the
  supported method before any method-restriction is added)

### T-008 — POST to the route (negative, open question — see §4)
- **Tag:** edge-negative
- **Given** the frontend router has the `/.well-known/security.txt` route registered
- **When** the test sends `POST /.well-known/security.txt`
- **Then** *(open — depends on whether `.Methods(http.MethodGet, http.MethodHead)` is added, as
  the other routes do at `main.go` lines 150-158)*:
  - if the route declares `.Methods(...)` restricting to GET/HEAD: expect `405 Method Not Allowed`
  - if the route is registered without `.Methods(...)` (like `robots.txt`/`_healthz` today, which
    accept any method): expect `200` (gorilla/mux's default `HandleFunc` matches all methods)
  This scenario is written to be resolved at plan time per the decision in §4; whichever behaviour
  is chosen, the test MUST assert that exact status, not skip the case.

## 4. Open questions

- **Method restriction (GET only vs GET+HEAD vs unrestricted):** `main.go`'s existing routes are
  inconsistent — most declare `.Methods(http.MethodGet, http.MethodHead)` (e.g. `/`, `/product/{id}`,
  `/cart`), but the two closest analogues (`robots.txt`, `_healthz`, lines 160-161) declare **no**
  `.Methods(...)` at all, so gorilla/mux accepts any method there. The research did not lock this
  down. **Recommendation carried into T-007/T-008:** match the `robots.txt`/`_healthz` precedent
  (no explicit `.Methods(...)`) unless the plan step decides HEAD support is worth adding for a
  static text resource — either choice is a one-line change and T-008 is written to assert
  whichever is chosen, not to leave it untested.
- **Router-construction seam (T-006):** whether the coder introduces a small extracted
  registration helper (testable against a synthetic router) or the test builds a minimal
  purpose-built router replicating just this one `HandleFunc` call. Left to the plan/implementation
  step; both satisfy every scenario above identically.
