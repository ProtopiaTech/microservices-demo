# Plan: 001-security-txt

**Inputs:** [1-research.md](./1-research.md), [2-tests.md](./2-tests.md)

**Summary:** Add a `GET /.well-known/security.txt` endpoint to the `frontend` Go service, RFC 9116
compliant with the required-only body (`Contact` + dynamically-generated `Expires`), registered at
the literal root path so it stays conformant regardless of `BASE_URL`. Backed by the first
`src/frontend/*_test.go` file, covering T-001..T-008.

**Commit convention:** every task below is one atomic commit on branch `work/001-security-txt`
per [docs/commit-conventions.md](../../commit-conventions.md); the unit ends with a squash PR to
`main`, team-reviewed and merged manually.

## Locked decisions (from research/tests checkpoints — not re-opened)
- Body: REQUIRED fields only — exactly `Contact` and `Expires`.
- `Contact: mailto:security@example.com`.
- Path: literal root `/.well-known/security.txt`, registered independent of `baseUrl`/`BASE_URL`.
- `Expires`: generated at request time, `time.Now().UTC().AddDate(0, 6, 0)`, formatted
  `time.RFC3339`.
- Method: unrestricted — `r.HandleFunc(...)` with no `.Methods(...)`, matching the
  `robots.txt`/`_healthz` precedent (T-008 expects `200` on POST).

## Phases

### P3-01: Baseline green
- id: P3-01
- phase: baseline
- batch: B0
- mode: serial (orchestrator only, no subagent)
- files touched: none
- acceptance criterion: `cd src/frontend && go build ./... && go test ./...` passes before any
  change
- maps-to test IDs: n/a
- expected-red: no
- high-risk: no

### P3-02: Seam — securityTxtHandler stub + route registration
- id: P3-02
- phase: seams
- batch: B1
- mode: serial
- files touched: `src/frontend/main.go`
- acceptance criterion: TRUE scaffolding only, no behaviour. A named handler
  `securityTxtHandler(w http.ResponseWriter, _ *http.Request)` exists as a STUB — it writes only a
  placeholder/empty body and does **not** set the final `Content-Type: text/plain; charset=utf-8`,
  does **not** write the `Contact` line, and does **not** write the `Expires` line (no
  body-builder helper with real content is introduced here); it is registered via
  `r.HandleFunc("/.well-known/security.txt", securityTxtHandler)` in `main()` at the literal root
  path (NOT `baseUrl+...`), alongside the `robots.txt`/`_healthz` lines. Acceptance: `go build
  ./...` stays green and the route is reachable (returns a response at all) — no real
  security.txt behaviour ships in this task
- maps-to test IDs: n/a
- expected-red: no
- high-risk: no

### P3-03: Red tests — all 8 scenarios
- id: P3-03
- phase: red-tests
- batch: B2
- mode: serial
- files touched: `src/frontend/security_txt_test.go` (new)
- acceptance criterion: one test file with 8 sub-tests/cases (T-001..T-008), each named after its
  test ID, driving the router via `httptest`; the suite compiles and is committed in a genuinely
  red state: against P3-02's stub, T-002 (exact `Content-Type`), T-003 (`Contact` line), T-004
  (`Expires` line), and T-005 (required-only fields) fail, because the stub sets no
  `Content-Type` and writes none of the required field lines. The suite is expected-red plainly
  because the handler under test is still a stub, not the real body-builder — not because the
  tests are weak or trivial
- maps-to test IDs: T-001, T-002, T-003, T-004, T-005, T-006, T-007, T-008
- expected-red: yes
- high-risk: no

### P3-04: Implementation — RFC-conformant body + headers
- id: P3-04
- phase: implementation
- batch: B3
- mode: serial
- files touched: `src/frontend/main.go`
- acceptance criterion: replace the P3-02 stub's body with the real implementation:
  `securityTxtHandler` (backed by a body-builder helper) sets `Content-Type: text/plain;
  charset=utf-8`, writes exactly two field lines (`Contact: mailto:security@example.com` and a
  dynamically generated `Expires: <RFC3339 timestamp>` via `time.Now().UTC().AddDate(0, 6, 0)`
  formatted with `time.RFC3339`), no other field lines, method unrestricted (no `.Methods(...)`).
  This is the task that turns P3-03's red suite green: T-001..T-008 pass with `go test ./...`
  green
- maps-to test IDs: T-001, T-002, T-003, T-004, T-005, T-006, T-007, T-008
- expected-red: no
- high-risk: no

### P3-05: Consolidation — DRY + final green
- id: P3-05
- phase: consolidation
- batch: B4
- mode: serial
- files touched: `src/frontend/main.go`, `src/frontend/security_txt_test.go` (edge cleanup only,
  no new scenarios)
- acceptance criterion: body construction extracted to a small helper if not already (e.g.
  `securityTxtBody() string`) for readability/DRY; `gofmt`/`go vet` clean;
  `cd src/frontend && go build ./... && go test ./...` green as the final state of the feature
- maps-to test IDs: n/a
- expected-red: no
- high-risk: no

### P3-06: E2E live smoke
- id: P3-06
- phase: e2e
- batch: B5
- mode: serial (dispatched to `e2e-tester`, not `coder`)
- files touched: `docs/test/001-security-txt/` (new evidence + `summary.md`)
- acceptance criterion: this unit ships a user-observable new endpoint, so the e2e task is
  REQUIRED — there is no n/a fallback. Per `docs/test/README.md`'s `frontend` subsection bring-up
  (`skaffold run` + `kubectl port-forward deployment/frontend 8080:8080`), `curl -i
  http://localhost:8080/.well-known/security.txt` returns `200`, `Content-Type: text/plain;
  charset=utf-8`, and a body containing `Contact: mailto:security@example.com` and an `Expires:`
  line; PASS `summary.md` + ordered evidence committed to `docs/test/001-security-txt/`. If the
  live cluster proves infeasible in the execution environment, that is handled by the orchestrator
  at execute time as an explicit, justified exception — it is not pre-authorized here and is not a
  substitute for attempting the real smoke
- maps-to test IDs: n/a
- expected-red: no
- high-risk: no

## TODO (work-execute consumes this)

### Batch B0 — baseline (orchestrator only, no dispatch)
- [ ] P3-01 (baseline) verify `go build ./...` + `go test ./...` green in `src/frontend`

### Batch B1 — seams (1 coder dispatch, serial)
- [ ] P3-02 (seams) add `securityTxtHandler` STUB (placeholder body, no real Content-Type/Contact/
      Expires), register `/.well-known/security.txt` at literal root

### Batch B2 — red tests (1 coder dispatch, serial) [expected-red]
- [ ] P3-03 (red-tests) all 8 scenarios in `security_txt_test.go`, genuinely red against the P3-02
      stub -> maps T-001, T-002, T-003, T-004, T-005, T-006, T-007, T-008

### Batch B3 — implementation (1 coder dispatch, serial)
- [ ] P3-04 (implementation) replace stub with RFC-conformant body (Contact + dynamic Expires) +
      headers, required-only, literal-root, method-unrestricted, T-001..T-008 red -> green -> maps
      T-001, T-002, T-003, T-004, T-005, T-006, T-007, T-008

### Batch B4 — consolidation (1 coder dispatch, serial)
- [ ] P3-05 (consolidation) DRY body-builder helper, final `go build`/`go test` green

### Batch B5 — e2e live smoke (1 e2e-tester dispatch) [REQUIRED — no n/a fallback]
- [ ] P3-06 (e2e) live smoke per docs/test/README.md -> PASS summary.md + evidence in
      docs/test/001-security-txt/
