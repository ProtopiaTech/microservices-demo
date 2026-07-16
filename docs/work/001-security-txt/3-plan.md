# Plan: 001-security-txt

**Inputs:** [1-research.md](./1-research.md), [2-tests.md](./2-tests.md)

**Summary:** Add an RFC 9116-compliant `GET /.well-known/security.txt` endpoint to `src/frontend`,
serving `Contact` + `Expires` (computed at request time) as `text/plain; charset=utf-8`, registered
as an exact-path `gorilla/mux` route alongside the existing `robots.txt`/`_healthz` handlers.

**Commit convention:** Every task below is one atomic commit on this unit's branch
(`work/001-security-txt`) per `docs/commit-conventions.md` — one task, one commit, one acceptance
criterion, staged by path, green before it lands. The unit ends with a squash PR to `main`, opened
by `work-docs` and merged manually by the team; no step in this plan merges to `main` directly.

## Phases

### Phase: Baseline
Confirm the tree builds/tests green before any change (orchestrator only, no dispatch).

### P3-01: baseline green
- id: P3-01
- phase: baseline
- batch: B0
- mode: serial
- files touched: none
- acceptance criterion: `cd src/frontend && go build ./... && go test ./...` exits 0
- maps-to test IDs: n/a
- expected-red: no
- high-risk: no

### Phase: Seams
Extract the security.txt handler AND its registration into package-level, testable units so
`httptest` can exercise them without running `main()` (which dials gRPC backends). Minimal seam
chosen (per the risk/scope tradeoff for a lightweight unit): do **not** extract the whole
`mux.Router` construction out of `main()` (too broad for this unit) — instead extract two
package-level pieces that both `main()` and the test file call:
- `func securityTxtHandler(w http.ResponseWriter, r *http.Request)` — the handler itself.
- `func registerSecurityTxtRoute(r *mux.Router, baseUrl string)` — registers exactly
  `r.HandleFunc(baseUrl+"/.well-known/security.txt", securityTxtHandler)` on a router passed in.

`main()` calls `registerSecurityTxtRoute(r, baseUrl)` at the same place as the existing
`robots.txt`/`_healthz` lines (~160-161). The test file builds its OWN small `*mux.Router` (via
`mux.NewRouter()`, zero gRPC dials, zero backend setup) and calls the same
`registerSecurityTxtRoute` helper to register only this one route — which is what makes T-007
(routing exactness / unrelated path 404s) genuinely assertable against real `mux` routing
behaviour, not just a direct handler call. This is behaviour-neutral: no user-visible change in
this phase.

**Placeholder discipline (required for genuine redness in the next phase):** the placeholder body
registered by `securityTxtHandler` in this seam task MUST NOT set a `Content-Type` header and MUST
NOT write a body containing the substrings `Contact:` or `Expires:` — e.g. `fmt.Fprint(w, "TODO")`
is acceptable, `fmt.Fprint(w, "ok")` is acceptable; anything that already resembles the final body
is not. This guarantees the content-bearing tests (T-002–T-006) are genuinely red at B2.

### P3-02: extract testable security.txt handler + route-registration seam
- id: P3-02
- phase: seams
- batch: B1
- mode: serial
- files touched: `src/frontend/main.go`
- acceptance criterion: package-level `func securityTxtHandler(w http.ResponseWriter, r *http.Request)`
  and `func registerSecurityTxtRoute(r *mux.Router, baseUrl string)` exist in `main.go`;
  `main()` calls `registerSecurityTxtRoute(r, baseUrl)` in place of a literal `HandleFunc` call at
  the `robots.txt`/`_healthz` location; the placeholder body sets no `Content-Type` header and
  contains neither `Contact:` nor `Expires:` substrings; `go build ./...` stays green
- maps-to test IDs: n/a
- expected-red: no
- high-risk: no

## Phase: Red tests
All seven scenarios from `2-tests.md` written up front in one new test file. The test file builds
its own minimal `*mux.Router` via `mux.NewRouter()` and `registerSecurityTxtRoute(r, "")` (default
empty `baseUrl`, zero gRPC dials, zero `main()` execution), then drives it with
`net/http/httptest`.

**Expected redness is partial, not total:** T-001 (200 OK) and T-007 (unrelated path does not match)
depend only on routing, which the seam already wires correctly in P3-02 — those two pass from the
start. T-002 (`Content-Type` header), T-003 (`Contact:` line), T-004 (`Expires:` line/RFC3339),
T-005 (exactly two field lines), and T-006 (`name: value` shape) depend on the real body/headers
that only ship in P3-04 — those five fail (red) until then. The package-level `go test ./...` run
is still red overall at this checkpoint because of T-002–T-006, even though T-001/T-007 pass.

### P3-03: red tests for security.txt behaviour
- id: P3-03
- phase: red-tests
- batch: B2
- mode: serial
- files touched: `src/frontend/security_txt_test.go`
- acceptance criterion: new test file compiles and contains 7 test functions/cases covering
  T-001..T-007, built against a router assembled via `mux.NewRouter()` +
  `registerSecurityTxtRoute(r, "")` (no `main()`, no gRPC dials); running `go test ./...` shows
  T-001 and T-007 passing already (routing-only) while T-002–T-006 fail (red) against the
  placeholder body/headers from P3-02
- maps-to test IDs: T-001, T-002, T-003, T-004, T-005, T-006, T-007
- expected-red: yes
- high-risk: no

## Phase: Implementation
Implement `securityTxtHandler` to emit the RFC 9116-compliant body and headers, turning the
remaining red tests (T-002–T-006) green while T-001/T-007 stay green.

### P3-04: implement compliant security.txt response
- id: P3-04
- phase: implementation
- batch: B3
- mode: serial
- files touched: `src/frontend/main.go`
- acceptance criterion: `securityTxtHandler` sets `Content-Type: text/plain; charset=utf-8`, writes
  exactly two lines — `Contact: mailto:security@example.com` and
  `Expires: <time.Now().UTC().Add(365*24*time.Hour) formatted as time.RFC3339>` — and
  `go test ./...` (specifically `security_txt_test.go`) is fully green for T-001..T-007 (T-007
  remains satisfied by the exact `HandleFunc` path registered via `registerSecurityTxtRoute`, no
  `PathPrefix`)
- maps-to test IDs: T-001, T-002, T-003, T-004, T-005, T-006, T-007
- expected-red: no
- high-risk: no

## Phase: Consolidation
Final full-suite green check and formatting; fold in any DRY/edge cleanup surfaced by the previous
batches (none anticipated given the small surface).

### P3-05: consolidate and format
- id: P3-05
- phase: consolidation
- batch: B4
- mode: serial
- files touched: `src/frontend/main.go`, `src/frontend/security_txt_test.go`
- acceptance criterion: `gofmt -l src/frontend` reports no diffs and
  `cd src/frontend && go build ./... && go test ./...` is fully green
- maps-to test IDs: T-001, T-002, T-003, T-004, T-005, T-006, T-007
- expected-red: no
- high-risk: no

## Phase: E2E live smoke
User-observable change (a new public HTTP endpoint) — a real live smoke is required, not `n/a`.

### P3-06: live e2e smoke of security.txt endpoint
- id: P3-06
- phase: e2e
- batch: B5
- mode: serial
- files touched: `docs/test/001-security-txt/` (evidence + summary.md)
- acceptance criterion: per `docs/test/README.md`, bring up the stack, run
  `curl -i http://<frontend-host>/.well-known/security.txt`, assert `200`, `Content-Type: text/plain;
  charset=utf-8`, a `Contact:` line, and a parseable-future `Expires:` line; PASS `summary.md` +
  ordered evidence committed under `docs/test/001-security-txt/`; if no cluster is available at
  execute time, the orchestrator surfaces that instead of fabricating a pass
- maps-to test IDs: n/a
- expected-red: no
- high-risk: no

## TODO (work-execute consumes this)

### Batch B0 — baseline (orchestrator only, no dispatch)
- [x] P3-01 (baseline) verify `go build ./...` and `go test ./...` green in `src/frontend`

### Batch B1 — seams (1 coder dispatch, serial)
- [x] P3-02 (seams) extract package-level `securityTxtHandler` + `registerSecurityTxtRoute`,
      wire into `main()` at the exact-path `baseUrl+"/.well-known/security.txt"` route,
      Content-Type-free/Contact-Expires-free placeholder body, build stays green

### Batch B2 — red tests (1 coder dispatch, serial) [expected-red]
- [ ] P3-03 (red-tests) add `security_txt_test.go` covering T-001..T-007 via httptest against a
      router built with `registerSecurityTxtRoute` (no main(), no gRPC dials); T-001/T-007 pass,
      T-002-T-006 red -> maps T-001, T-002, T-003, T-004, T-005, T-006, T-007

### Batch B3 — implementation (1 coder dispatch, serial)
- [ ] P3-04 (implementation) emit compliant `Contact`/`Expires` body + `Content-Type` header,
      T-002-T-006 go green (T-001/T-007 stay green) -> maps T-001, T-002, T-003, T-004, T-005,
      T-006, T-007

### Batch B4 — consolidation (1 coder dispatch, serial)
- [ ] P3-05 (consolidation) `gofmt` clean, full `go build ./... && go test ./...` green -> maps
      T-001, T-002, T-003, T-004, T-005, T-006, T-007

### Batch B5 — e2e live smoke (1 e2e-tester dispatch)
- [ ] P3-06 (e2e) live smoke per docs/test/README.md -> evidence + PASS summary.md in
      docs/test/001-security-txt/
