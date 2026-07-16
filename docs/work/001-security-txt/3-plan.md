# Plan: 001-security-txt

**Inputs:** [1-research.md](./1-research.md), [2-tests.md](./2-tests.md)

**Summary:** Add a `GET/HEAD {baseUrl}/.well-known/security.txt` endpoint to the frontend service
that serves a minimal RFC 9116-compliant body (`Contact` + a startup-computed `Expires` ~1 year
out) with an explicit `Content-Type: text/plain; charset=utf-8` header, backed by unit tests
(T-001..T-009) and a live e2e smoke against the deployed frontend.

**Commit convention:** every task below is one atomic commit on the unit's branch
(`work/001-security-txt`), per [docs/commit-conventions.md](../../commit-conventions.md); the unit
ends with a squash PR to `main`, team-reviewed and merged manually. The orchestrator never commits
to `main` and never merges its own PR.

---

## Phases

### Phase 1 — Baseline

#### P3-01: confirm frontend baseline is green
- id: P3-01
- phase: baseline
- batch: B0
- mode: serial
- files touched: none (verification only)
- acceptance criterion: `cd src/frontend && go build ./... && go test ./...` both succeed before
  any change lands
- maps-to test IDs: n/a
- expected-red: no
- high-risk: no

### Phase 2 — Seams

#### P3-02: scaffold security.txt handler, registration helper, and wiring
- id: P3-02
- phase: seams
- batch: B1
- mode: serial
- files touched: `src/frontend/security_txt.go` (new), `src/frontend/main.go`
- acceptance criterion: `src/frontend/security_txt.go` exists with (a) a package-level `Expires`
  value computed once at init as `time.Now().AddDate(1, 0, 0)` formatted with `time.RFC3339`, (b) a
  stub `securityTxtHandler(w http.ResponseWriter, r *http.Request)` that compiles and returns a
  placeholder/empty 200 response, and (c) `registerSecurityTxt(r *mux.Router, base string)` that
  registers `base+"/.well-known/security.txt"` on `securityTxtHandler` with
  `.Methods(http.MethodGet, http.MethodHead)`; `main.go`'s router block calls
  `registerSecurityTxt(r, baseUrl)` alongside the `/robots.txt` registration (line ~160). No
  behavioural body content yet. `cd src/frontend && go build ./...` passes.
- maps-to test IDs: n/a
- expected-red: no
- high-risk: no

### Phase 3 — Red tests (expected-red)

#### P3-03: author core happy-path tests (T-001, T-002, T-003, T-004, T-006)
- id: P3-03
- phase: red-tests
- batch: B2
- mode: serial
- files touched: `src/frontend/security_txt_test.go` (new)
- acceptance criterion: using `net/http/httptest` against a router built via
  `registerSecurityTxt(mux.NewRouter(), "")`, tests assert: GET returns 200 (T-001); the
  `Content-Type` response header equals exactly `text/plain; charset=utf-8` (T-002); the body
  contains the exact line `Contact: mailto:security@example.com` (T-003); exactly one `Expires:`
  line whose value parses via `time.Parse(time.RFC3339, ...)` and is strictly after
  `time.Now()` (T-004); and every non-blank body line's field name is one of `Contact`/`Expires`
  only (T-006). These fail against the P3-02 stub (empty/placeholder body, no explicit
  Content-Type) — run `go test ./...` and confirm the expected failures.
- maps-to test IDs: T-001, T-002, T-003, T-004, T-006
- expected-red: yes
- high-risk: no

#### P3-04: author edge/negative tests (T-005, T-007, T-008, T-009)
- id: P3-04
- phase: red-tests
- batch: B2
- mode: serial
- files touched: `src/frontend/security_txt_test.go`
- acceptance criterion: extends the same test file with: the parsed `Expires` falls within
  `[time.Now().AddDate(1,0,0).Add(-5*24*time.Hour), time.Now().AddDate(1,0,0).Add(+5*24*time.Hour)]`
  (T-005); `POST {baseUrl}/.well-known/security.txt` does not return 200 (T-007, asserted against
  the router from P3-02/registerSecurityTxt which already declares `.Methods(GET, HEAD)`, so this
  may already pass — assert it regardless as a regression guard); a router built with the
  package-level `baseUrl` var set to a non-empty prefix (e.g. `/some-base`) before calling
  `registerSecurityTxt`, in a subtest that resets `baseUrl = ""` afterwards, serves 200 at the
  prefixed path and does not serve 200 at the unprefixed path (T-008); the body is valid UTF-8
  (`utf8.ValidString`) and every non-blank line matches `name: value` shape (T-009). Run
  `go test ./...` and confirm the content-bearing assertions (T-005, T-009, and the body-matching
  part of T-008) fail against the P3-02 stub.
- maps-to test IDs: T-005, T-007, T-008, T-009
- expected-red: yes
- high-risk: no

### Phase 4 — Implementation (red → green)

#### P3-05: implement securityTxtHandler body and confirm full green
- id: P3-05
- phase: implementation
- batch: B3
- mode: serial
- files touched: `src/frontend/security_txt.go`
- acceptance criterion: `securityTxtHandler` sets
  `w.Header().Set("Content-Type", "text/plain; charset=utf-8")` before the first write, then writes
  exactly two lines — `Contact: mailto:security@example.com` and `Expires: <the package-level
  RFC3339 value computed at init>` — and nothing else; `registerSecurityTxt`'s existing
  `.Methods(http.MethodGet, http.MethodHead)` registration and `base`-prefixed path continue to
  apply unchanged. `cd src/frontend && go build ./... && go test ./...` are green, with
  T-001..T-009 all passing.
- maps-to test IDs: T-001, T-002, T-003, T-004, T-005, T-006, T-007, T-008, T-009
- expected-red: no
- high-risk: no

### Phase 5 — Consolidation

#### P3-06: DRY/edge cleanup and final green verification
- id: P3-06
- phase: consolidation
- batch: B4
- mode: serial
- files touched: `src/frontend/security_txt.go`, `src/frontend/security_txt_test.go`
- acceptance criterion: `gofmt -l src/frontend` reports no files, `go vet ./...` is clean, no
  duplicate line-parsing logic between T-006/T-009 checks (fold into a shared helper in the test
  file if duplicated), and `cd src/frontend && go build ./... && go test ./...` is green with no
  regressions. If no cleanup is needed beyond re-running the checks, this task still exists to
  record the final green confirmation.
- maps-to test IDs: n/a
- expected-red: no
- high-risk: no

### Phase 6 — E2E live smoke (mandatory — user-observable change)

#### P3-07: live smoke of the deployed security.txt endpoint
- id: P3-07
- phase: e2e
- batch: B5
- mode: serial
- files touched: `docs/test/001-security-txt/` (new — evidence + `summary.md`)
- acceptance criterion: per [docs/test/README.md](../../test/README.md), against the live
  port-forwarded frontend (`kubectl port-forward deploy/frontend 8080:8080`), the `e2e-tester`
  issues `curl -i http://localhost:8080/.well-known/security.txt` (a plain-text endpoint — no
  browser interaction needed, unlike the README's screenshot-based UI flows) and confirms: HTTP
  status `200`; `Content-Type: text/plain; charset=utf-8` header present; body contains
  `Contact: mailto:security@example.com` and an `Expires:` line. Also issues
  `curl -i -X POST http://localhost:8080/.well-known/security.txt` and confirms it is not `200`
  (405). Ordered evidence (captured curl output/logs) and a PASS/fail `summary.md` land under
  `docs/test/001-security-txt/`.
- maps-to test IDs: n/a
- expected-red: no
- high-risk: no

---

## TODO (work-execute consumes this)

### Batch B0 — baseline (orchestrator only, no dispatch)
- [x] P3-01 (baseline) verify `src/frontend` build+test green

### Batch B1 — seams (1 coder dispatch, serial)
- [x] P3-02 (seams) scaffold `security_txt.go` (Expires var, stub handler, registerSecurityTxt) and wire into `main.go`

### Batch B2 — red tests (1 coder dispatch, serial) [expected-red]
- [x] P3-03 (red-tests) core happy-path specs -> maps T-001, T-002, T-003, T-004, T-006
- [x] P3-04 (red-tests) edge/negative specs -> maps T-005, T-007, T-008, T-009

### Batch B3 — implementation: security.txt body (1 coder dispatch, serial)
- [x] P3-05 (implementation) fill in Content-Type + Contact/Expires body -> maps T-001, T-002, T-003, T-004, T-005, T-006, T-007, T-008, T-009

### Batch B4 — consolidation (1 coder dispatch, serial)
- [ ] P3-06 (consolidation) DRY/edge cleanup, gofmt/go vet clean, final green

### Batch B5 — e2e live smoke (1 e2e-tester dispatch)
- [ ] P3-07 (e2e) live smoke per docs/test/README.md -> evidence + summary.md in docs/test/001-security-txt/
