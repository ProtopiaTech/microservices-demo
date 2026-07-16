# 001 — security.txt endpoint (frontend) — Research

**Unit:** Add a `GET /.well-known/security.txt` endpoint to the **frontend** service (Go),
serving a body compliant with [RFC 9116](https://www.rfc-editor.org/rfc/rfc9116.txt) ("A File
Format to Aid in Security Vulnerability Disclosure", IETF Proposed Standard, April 2022).

---

## 1. Scope

**IS:**
- Register a new route `GET {baseUrl}/.well-known/security.txt` on the frontend's `gorilla/mux`
  router, alongside the existing `/robots.txt` and `/_healthz` inline handlers.
- Serve a static, RFC 9116-compliant text body containing (at minimum) the two REQUIRED fields
  `Contact` and `Expires`.
- Set the response `Content-Type: text/plain; charset=utf-8` explicitly (RFC 9116 §3 requires it).
- UTF-8 body, `name: value` lines (LF line endings are acceptable per §4).

**IS NOT:**
- No OpenPGP / digital signature of the file (RECOMMENDED only per §2.3 — out of scope).
- No admin UI, no config surface, no per-request dynamic generation beyond what is strictly needed
  to build the static body (e.g. a computed `Expires` is a candidate but not required — see §5).
- No changes to any other microservice.
- No ingress / TLS / load-balancer configuration changes. HTTPS is an environment concern handled by
  the ingress in this demo, not by the Go process (see §3).
- No IANA-extension fields (e.g. CSAF) — not part of RFC 9116 compliance.

---

## 2. System slice (files / seams)

Verified by reading the source:

- **`src/frontend/main.go`** — router block, lines **149–163**. New route registered here, near
  `/robots.txt` (line 160) and `/_healthz` (line 161). Both are inline `func(w, r)` closures — the
  established pattern for tiny static responses. The new route follows the same shape:
  `r.HandleFunc(baseUrl+"/.well-known/security.txt", ...)`.
- **`baseUrl`** (main.go line 58 default `""`, line 111 `baseUrl = os.Getenv("BASE_URL")`) prefixes
  every route. The new route MUST be prefixed too, so it is reachable under the configured base path.
  Note: RFC 9116 §3 specifies the well-known location as `/.well-known/security.txt` relative to the
  host root; a non-empty `BASE_URL` shifts it under a sub-path, which is an environment/deployment
  choice (in the default demo `BASE_URL` is empty, so the path is exactly `/.well-known/security.txt`).
- **Handler location** — small enough to be an inline closure in `main.go` (like `/robots.txt`), or a
  method on `*frontendServer` in `handlers.go` (style: `func (fe *frontendServer) xHandler(w, r)`).
  Either is consistent with the repo; the tests step will pin the choice.
- **No outbound seam** — this is a pure static text response. No gRPC call, no network I/O, no
  template render, no dependency on `fe`'s service connections. This is what makes the unit trivially
  testable with `net/http/httptest`.

---

## 3. Stack & seams

- **Go 1.25.0** (`src/frontend/go.mod`), router **`gorilla/mux`**.
- The **only seam is the HTTP response**. Contrast with the existing `/robots.txt` handler
  (main.go line 160):
  `r.HandleFunc(baseUrl+"/robots.txt", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "User-agent: *\nDisallow: /") })`
  — it does **not** set `Content-Type`, so Go's `http.DetectContentType` sniffs it. That is a gotcha:
  RFC 9116 §3 **REQUIRES** `Content-Type: text/plain; charset=utf-8`, so the security.txt handler
  **must** call `w.Header().Set("Content-Type", "text/plain; charset=utf-8")` **before** the first
  write. The repo already uses `w.Header().Set(...)` elsewhere (e.g. `handlers.go` Location headers),
  so this is idiomatic.
- **HTTPS (RFC 9116 §3):** the RFC mandates the file be served over `https` in production. In this
  demo TLS is terminated at the ingress / load-balancer, not in the Go process — the handler cannot
  and should not enforce the scheme. Flag as an environment concern, not handler logic.
- **No test files exist** under `src/frontend` today (`*_test.go` — none found). This unit introduces
  the first, using the standard library `net/http/httptest` (no new dependency).
- **Quality gate:** `.claude/quality-gate.routes` line 21 routes `src/frontend/` changes to
  `cd src/frontend && go build ./...` (build) and `cd src/frontend && go test ./...` (test). Green =
  those two commands pass.

---

## 4. RFC 9116 field decision

**Default rule (§2, §2.5):** every field is OPTIONAL unless the spec says otherwise; researchers MUST
ignore unknown fields (§2.4). Only two fields are REQUIRED.

| Field | RFC § | Required? | Cardinality | Value format | This endpoint serves? |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Contact** | §2.5.3 | **REQUIRED** ("MUST always be present") | 1+ (may repeat; order = preference) | URI (RFC 3986): `mailto:`, `tel:`, or `https://` page | **Yes** — required |
| **Expires** | §2.5.5 | **REQUIRED** ("MUST always be present and MUST NOT appear more than once") | exactly 1 | RFC 3339 date-time with TZ designator (`Z` / `±HH:MM`); RECOMMENDED < 1 yr out; a past value invalidates the file | **Yes** — required |
| Encryption | §2.5.4 | optional | 0+ | URI to a key — **never inline the key** | No (no PGP in scope) |
| Acknowledgments | §2.5.1 | optional | 0+ | URI | No |
| Canonical | §2.5.2 | optional | 0+ | URI where this file canonically lives | **Candidate** (see §5) |
| Policy | §2.5.7 | optional | 0+ | URI | No (open question) |
| Hiring | §2.5.6 | optional | 0+ | URI | No |
| Preferred-Languages | §2.5.8 | optional | 0 or 1 (MUST NOT appear >once) | comma-separated RFC 5646 tags on one line | **Candidate** (see §5) |

**Cardinality summary (§2.5.5, §2.5.8):** `Expires` and `Preferred-Languages` may appear **at most
once**; `Contact` and the URI fields may repeat.

**Serving requirements (§3, §4):**
- Path MUST be `/.well-known/security.txt` (the `/.well-known/` copy takes precedence over any legacy
  top-level copy).
- Response `Content-Type` MUST be `text/plain; charset=utf-8`.
- File MUST be UTF-8; line-based `name: value`; lines end CRLF **or** LF; `#` = comment lines.
- HTTPS scheme mandatory in production (§3) — environment concern here (see §3 above).
- OpenPGP signature is RECOMMENDED, NOT required (§2.3).

**Minimal valid file** (§2.5.3 + §2.5.5 — Contact + Expires only):
```
Contact: mailto:security@example.com
Expires: 2026-12-31T23:59:59Z
```

**Concrete field set for THIS endpoint (recommendation):** ship the **required minimum** —
`Contact` + `Expires` — as the compliant baseline, and OPTIONALLY add `Canonical` and
`Preferred-Languages` as sensible recommended additions. The exact **values** (contact URI, Expires
date, whether to include Canonical/Preferred-Languages) are open questions to pin in the tests step —
see §5. Sources: RFC 9116 §2.4, §2.5.1–§2.5.8, §3, §4; <https://securitytxt.org>.

---

## 5. Unknowns / open questions

1. **Contact URI** — what real value? A `mailto:` (e.g. `security@…`) vs an `https://` security page?
   Needs a real, monitored address/URL owned by whoever adopts this demo. (§2.5.3)
2. **Expires value & freshness** — what date, and how kept fresh? RFC RECOMMENDS < 1 year out and a
   past value **invalidates** the file (§2.5.5). Options: (a) hardcoded literal string (simplest,
   KISS, but rots and must be bumped manually); (b) computed at startup/build (e.g. now + N months) so
   it never goes stale. Trade-off: (b) means the handler builds the body dynamically and needs an
   injectable clock for deterministic tests. Recommend deciding in the tests step.
3. **Optional fields** — include `Canonical` (§2.5.2, the URL where the file lives — depends on the
   deployment host, and interacts with `BASE_URL`)? `Preferred-Languages` (§2.5.8, e.g. `en`)?
   `Policy` (§2.5.7)? All optional; default recommendation is to keep to the required minimum unless
   the user wants more.
4. **Body delivery** — inline string literal (like `/robots.txt`) vs `go:embed` of a static
   `.well-known/security.txt` file. Inline is simplest for a 2-field body; `go:embed` is cleaner if
   the body grows or if ops want to edit it without touching Go. Recommend inline for KISS unless a
   computed `Expires` is chosen.
5. **`baseUrl` / Canonical interaction** — if `Canonical` is included, should its URL reflect
   `BASE_URL`? Only relevant if we ship `Canonical`. The route itself must be `baseUrl`-prefixed
   regardless.
6. **HTTPS & signature** — recommend confirming both are **out of scope**: HTTPS is handled at the
   ingress (§3), and the OpenPGP signature is RECOMMENDED-only (§2.3).

---

## 6. Findings & sources

- **RFC 9116** — Foudil & Shafranovich, *A File Format to Aid in Security Vulnerability Disclosure*,
  IETF Proposed Standard, April 2022. <https://www.rfc-editor.org/rfc/rfc9116.txt>
  - §2 / §2.5: default-optional; §2.4 ignore unknown fields.
  - §2.3: digital (OpenPGP) signature RECOMMENDED, not required.
  - §2.5.1 Acknowledgments; §2.5.2 Canonical; §2.5.3 **Contact (REQUIRED)**; §2.5.4 Encryption;
    §2.5.5 **Expires (REQUIRED, exactly once)**; §2.5.6 Hiring; §2.5.7 Policy;
    §2.5.8 Preferred-Languages (at most once).
  - §3: location `/.well-known/security.txt`, `Content-Type: text/plain; charset=utf-8`, `https`.
  - §4: UTF-8, `name: value`, CRLF/LF line endings, `#` comments.
- **securitytxt.org** — companion / generator confirming the required-vs-optional split and the
  minimal Contact + Expires file. <https://securitytxt.org>
- **Local (verified by reading):** `src/frontend/main.go` router block lines 149–163 (`/robots.txt`
  at 160 sets no Content-Type — the gotcha), `baseUrl` from `BASE_URL` env (lines 58, 111);
  `src/frontend/handlers.go` handler style (`func (fe *frontendServer) …`) and idiomatic
  `w.Header().Set(...)`; no `*_test.go` under `src/frontend` yet; Go 1.25.0 (`go.mod`); quality gate
  route for `src/frontend/` in `.claude/quality-gate.routes` line 21.

---

Task tier: lightweight — one static text handler + route in a single service (frontend), no new dependency and no outbound seam, pinnable with a couple of `net/http/httptest` tests, and no config/build-tooling change.
