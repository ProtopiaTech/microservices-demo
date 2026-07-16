# 001 — security.txt endpoint (frontend) — Behaviour Spec

Given-When-Then scenarios for `GET {baseUrl}/.well-known/security.txt` on the frontend service, per
the unit fixed at the research checkpoint: an RFC 9116-compliant body with exactly two fields
(`Contact: mailto:security@example.com` and a startup-computed `Expires`, ~now + 1 year), response
header `Content-Type: text/plain; charset=utf-8`.

---

## 1. In scope / Out of scope to test

**In scope:**
- HTTP status code of the endpoint.
- Response `Content-Type` header, exact value.
- Body content: presence and format of the `Contact` field, presence/format/cardinality of the
  `Expires` field, absence of any other RFC 9116 field.
- `Expires` value is a syntactically valid RFC 3339 date-time that lies in the future and is
  approximately one year out (format + relative-time assertions, not an exact string).
- HTTP method handling on the route (GET happy path; a clearly-wrong method as a negative case).
- The route being reachable under a configured `baseUrl` prefix.

**Out of scope:**
- TLS/HTTPS enforcement — an ingress/load-balancer concern in this deployment, not the Go process.
- OpenPGP signature of the file (RECOMMENDED-only per RFC 9116 §2.3, not part of this unit).
- Any other microservice or cross-service interaction (this handler has no outbound seam).
- The exact wall-clock value of `Expires` — the test asserts format and future-ness/approximate
  offset, never a literal timestamp (the value is computed at process startup from "now").
- Optional RFC 9116 fields not shipped (`Canonical`, `Preferred-Languages`, `Encryption`, etc.) beyond
  asserting they are absent (T-006).

---

## 2. Seams

- **Seam under test:** the HTTP response produced by the `gorilla/mux` router/handler for the new
  route. Exercised with Go's standard `net/http/httptest` — `httptest.NewRecorder()` +
  `httptest.NewRequest(method, path, nil)` dispatched through the router (or `httptest.NewServer` if
  a full listener is more convenient for a given scenario).
- **No external seam, no mocks needed.** This handler is a pure static/computed text response — no
  gRPC call, no template render, no dependency on `frontendServer`'s downstream service connections.
  State this explicitly so the test author doesn't reach for mocks that aren't needed.
- **Time source note:** `Expires` is computed once from the current time when the server/router is
  constructed (startup), not per-request. Tests must NOT assert an exact literal value. Instead:
  parse the `Expires:` value with `time.Parse(time.RFC3339, ...)` and assert (a) parsing succeeds,
  (b) the parsed time is after `time.Now()` (in the future), and (c) the parsed time is within a
  tolerance window around `time.Now().AddDate(1, 0, 0)` (see T-005 for the exact window). No clock
  injection/mocking is required — real wall-clock time at test-run time is precise enough given the
  ~1-year tolerance window.

---

## 3. Scenarios

| ID | Type | Scenario |
| :--- | :--- | :--- |
| T-001 | [happy] | GET returns 200 |
| T-002 | [happy] | Content-Type is exactly `text/plain; charset=utf-8` |
| T-003 | [happy] | Body contains a `Contact:` line with value `mailto:security@example.com` |
| T-004 | [happy] | Body contains exactly one `Expires:` line whose value parses as RFC 3339 and is in the future |
| T-005 | [happy/edge] | `Expires` is approximately one year out (tolerance window stated) |
| T-006 | [edge] | Body contains ONLY `Contact` and `Expires` — no other RFC 9116 field |
| T-007 | [edge/negative] | POST to the path is not served as 200 |
| T-008 | [edge] | Route respects `baseUrl` — served at `{baseUrl}/.well-known/security.txt` |
| T-009 | [edge, optional] | Body is valid UTF-8 and every non-blank line matches `name: value` |

### T-001 — GET returns 200 [happy]
- **Given** the frontend router is constructed with default configuration (`baseUrl = ""`).
- **When** the test issues `GET /.well-known/security.txt` via `httptest.NewRequest` +
  `httptest.NewRecorder()` dispatched through the router.
- **Then** the recorded response status code is `200`.

### T-002 — Content-Type is exact [happy]
- **Given** the same setup as T-001.
- **When** `GET /.well-known/security.txt` is issued.
- **Then** the response header `Content-Type` equals exactly `text/plain; charset=utf-8` (not a
  sniffed value — contrast with `/robots.txt`, which sets no `Content-Type` and relies on Go's
  `http.DetectContentType`; this endpoint MUST set it explicitly before the first write).

### T-003 — Contact field present and correct [happy]
- **Given** the same setup as T-001.
- **When** `GET /.well-known/security.txt` is issued and the body is read as a string.
- **Then** the body contains a line equal to `Contact: mailto:security@example.com` (exact line
  match, not just substring, to also pin the `name: value` format with a single space after the
  colon).

### T-004 — Expires present once, valid RFC 3339, in the future [happy]
- **Given** the same setup as T-001.
- **When** the body is split into lines and filtered for lines starting with `Expires:`.
- **Then** there is exactly one such line; its value (trimmed, after `Expires: `) parses without
  error via `time.Parse(time.RFC3339, value)`; and the parsed time is strictly after the test's
  observed `time.Now()` (captured immediately before or after the request, allowing for the small
  clock skew between server startup and test execution).

### T-005 — Expires is approximately one year out [happy/edge]
- **Given** the same setup as T-001.
- **When** the `Expires` value is parsed as in T-004.
- **Then** the parsed time falls within the window
  `[time.Now().AddDate(1,0,0).Add(-5*24*time.Hour), time.Now().AddDate(1,0,0).Add(+5*24*time.Hour)]`
  — i.e. **within ±5 days of exactly one year from test-observed now**. Rationale for the tolerance:
  the value is computed once at server/router construction time (not per-request), so a few seconds
  to a few minutes of drift between construction and test assertion is expected; ±5 days is generous
  enough to absorb that drift and any leap-year/`AddDate` edge effects while still tightly pinning
  the "now + 1 year" decision (far tighter than the RFC's own "RECOMMENDED < 1 year out" guidance).

### T-006 — Only Contact and Expires present [edge]
- **Given** the same setup as T-001.
- **When** the body is split into non-blank lines.
- **Then** every non-blank line's field name (the token before the first `:`) is one of `Contact` or
  `Expires` — i.e. none of the other RFC 9116 fields (`Canonical`, `Preferred-Languages`,
  `Encryption`, `Acknowledgments`, `Policy`, `Hiring`) appear anywhere in the body. This pins the
  "minimum required fields only" decision from research §4/§5.

### T-007 — Non-GET method is not served as 200 [edge/negative]
- **Given** the same setup as T-001, with the route registered for `GET` only (see decision note
  below).
- **When** the test issues `POST /.well-known/security.txt`.
- **Then** the response status code is `405 Method Not Allowed` (gorilla/mux's default behavior when
  a route declares `.Methods(http.MethodGet)` and a request arrives with a different method), and is
  in particular **not** `200`.
- **Decision note (to pin at implementation):** unlike the existing `/robots.txt` and `/_healthz`
  inline handlers (main.go lines 160–161), which are registered via `r.HandleFunc(...)` with **no**
  `.Methods(...)` call and therefore match *any* HTTP method, the new route SHOULD be registered with
  `.Methods(http.MethodGet)` (optionally also `http.MethodHead`, matching the pattern used by e.g.
  `/`, `/product/{id}`, `/cart` at lines 150–152) so that this negative case is meaningful. If the
  implementation instead follows the unrestricted `/robots.txt` pattern, this scenario would need to
  change to a different negative case or be dropped — the plan step should make this call explicit
  rather than let it fall out implicitly. Recommendation: register `GET` (+ optionally `HEAD`); use
  `POST` (a clearly-wrong method) for this test regardless, to stay robust to the `HEAD` decision.

### T-008 — Route respects baseUrl [edge]
- **Given** the router is constructed with the package-level `baseUrl` variable set to a non-empty
  value (e.g. `/some-base`) before `mux.NewRouter()` and route registration run — mirroring how
  `main()` sets `baseUrl = os.Getenv("BASE_URL")` before building routes (main.go lines 111, 149–163).
- **When** the test issues `GET {baseUrl}/.well-known/security.txt` (e.g.
  `GET /some-base/.well-known/security.txt`).
- **Then** the response status is `200` and the body/headers match T-001–T-006 as usual; additionally,
  `GET /.well-known/security.txt` (without the base prefix) does **not** return 200 while `baseUrl` is
  set, confirming the route is only reachable under the prefix.
- **Executability note:** because `baseUrl` is a package-level `var` (main.go line 58) rather than a
  parameter threaded through router construction, exercising this cleanly in a unit test requires
  either (a) a small test-only helper that builds the router after setting the package var, run in a
  subtest that resets `baseUrl = ""` afterwards (Go tests in the same package can read/write package
  vars), or (b) refactoring route construction to accept `baseUrl` as a parameter. This spec does not
  mandate the refactor — it is executable as-is via (a) since the existing routes already depend on
  the same package var and any future test of them would face the same constraint. If the plan step
  finds (a) impractical for the chosen router-construction shape, downgrade this to a scenario
  verified only via route registration (asserting the registered path string), and note this
  explicitly as a reduced-fidelity check rather than silently dropping baseUrl coverage.

### T-009 — Body is valid UTF-8, name: value lines [edge, optional]
- **Given** the same setup as T-001.
- **When** the body is read.
- **Then** it is valid UTF-8 (`utf8.ValidString(body)` is `true`), and every non-blank line matches
  the `name: value` shape (contains a `:` separator, non-empty name token before it) — this test is
  optional / nice-to-have and may be folded into T-006's line-splitting logic rather than written
  standalone, to avoid duplicate parsing code.

---

## 4. Core acceptance set

**T-001, T-002, T-003, T-004, T-006, T-007** are the core acceptance set — they directly verify the
unit's fixed requirements (200 status, exact Content-Type, required Contact value, exactly-one valid
future Expires, minimum-fields-only, and correct method handling). **T-005** pins the "now + 1 year"
freshness decision and should be included but is slightly softer (tolerance-window based). **T-008**
(baseUrl) and **T-009** (UTF-8/format) are supplementary — valuable but lower-priority if time-boxed,
and T-008 carries an explicit executability caveat that the plan step must resolve one way or another.
