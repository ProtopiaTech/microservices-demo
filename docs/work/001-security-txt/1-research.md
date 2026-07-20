# 001 — security.txt — Research

**Unit:** Add a `GET /.well-known/security.txt` endpoint compliant with RFC 9116 to the Online
Boutique app, and determine which fields are required vs merely recommended.

## 1. Scope

**In scope**
- Serve a static/generated `security.txt` document over HTTP from the `frontend` Go service at the
  well-known path `/.well-known/security.txt` (RFC 8615 + RFC 9116).
- Emit the two RFC-required fields (`Contact`, `Expires`) plus a small, sensible set of RECOMMENDED
  fields appropriate for a public demo app.
- Set the correct `Content-Type: text/plain; charset=utf-8` response header.
- A few frontend tests pinning the endpoint's path, status, content type, and required-field
  presence.
- Answer the user's question: **which RFC 9116 fields are REQUIRED vs merely RECOMMENDED**
  (section 4 below).

**NOT in scope**
- **Digital signing** of the file. RFC 9116 §2.3 only RECOMMENDS an OpenPGP cleartext signature; a
  demo app has no signing key and no security-team key infrastructure. We serve plaintext.
- **Real security-team infrastructure** — no real inbox, PGP key, disclosure workflow, or
  vulnerability-handling process is created. `Contact`/`Policy` values point at existing repo
  resources (placeholder decision deferred to the checkpoint — see §5).
- **Changing the repo's root-vs-`baseUrl` routing convention** for other routes. The only open
  routing question is where *this one* endpoint is registered (see §2, §5c).
- Other services — this is a web/HTTP concern; `frontend` is the sole user-facing HTTP service.
- New dependencies, config, or build-tooling changes.

## 2. System slice — exact files & seams

- **`src/frontend/main.go`** — router setup (`gorilla/mux`, `mux.NewRouter()` at line 149). Routes
  are registered with a `baseUrl` prefix. The closest existing analogues are the inline literal-body
  handlers:
  - line 160: `r.HandleFunc(baseUrl+"/robots.txt", func(w, _){ fmt.Fprint(w, "...") })`
  - line 161: `r.HandleFunc(baseUrl+"/_healthz", func(w, _){ fmt.Fprint(w, "ok") })`

  The new route follows the same shape: `r.HandleFunc(<path>+"/.well-known/security.txt", <handler>)`.
- **`baseUrl`** (`main.go` line 58 `baseUrl = ""`; line 111 `baseUrl = os.Getenv("BASE_URL")`) —
  a global path prefix, empty by default and overridable via the `BASE_URL` env var. All app routes
  are registered as `baseUrl + "/..."`. **This is the routing open question** (see §5c): RFC 9116 §3
  requires the file at the site *root* `/.well-known/security.txt`; registering it as
  `baseUrl+"/.well-known/security.txt"` means that when `BASE_URL` is non-empty the file no longer
  sits at the domain root and is technically non-conformant. Flag for the checkpoint whether to
  register at the **literal root path** (`/.well-known/security.txt`, ignoring `baseUrl`) or **under
  `baseUrl`** for consistency with every other route. Note: with the default empty `baseUrl` both
  are identical, so this only matters for `BASE_URL`-prefixed deployments.
- **Optional small handler / helper** — the body can be an inline closure like `robots.txt`, or a
  tiny named handler/helper if we generate the `Expires` timestamp dynamically (recommended, §3).
  A helper that builds the document body is the DRY choice and is trivially unit-testable.
- **`src/frontend/handlers.go`** — handler + logging conventions (logrus `FieldLogger` pulled from
  request context via `r.Context().Value(ctxKeyLog{})`, `renderHTTPError` for failures). A static
  200 response needs no error path, so a `robots.txt`-style inline handler is a fair match; adopt the
  logging pattern only if a named handler is introduced.
- **Frontend tests** — there are currently **no `*_test.go` files under `src/frontend/`**, so this
  unit introduces the first frontend test file. Tests exercise the mux router / handler with
  `net/http/httptest` (no gRPC backends touched).
- **No outbound seam** — the response is pure static/generated text. No gRPC call, no backend
  dependency, nothing to mock.

## 3. Stack & seams

- **Language / libs:** Go, `github.com/gorilla/mux`, stdlib `net/http`. No new dependency.
- **Response construction:** set `w.Header().Set("Content-Type", "text/plain; charset=utf-8")`
  explicitly (RFC 9116 §3/§4 — MUST), then write the body. `fmt.Fprint` (as `robots.txt` does) does
  not set an RFC-conformant content type on its own, so the header must be set deliberately.
- **Dynamic `Expires`:** RFC 9116 §2.5.5 requires exactly one `Expires` and RECOMMENDS it be less
  than one year in the future. A hard-coded literal timestamp goes stale and eventually becomes
  invalid (expired). **Recommend generating `Expires` dynamically** at request time (or process
  start) as `now + N` (e.g. `time.Now().UTC().AddDate(1, 0, 0)` minus a margin, or ~6 months)
  formatted as RFC 3339 (`time.RFC3339`), so the file never expires. This is the main reason to
  prefer a small helper/handler over a pure literal string.
- **Line endings / encoding:** UTF-8 (§4); CRLF or LF both valid (§2.2) — plain `\n` is fine.
- **Testing seam:** `httptest.NewRecorder()` + the configured `*mux.Router` (or the handler
  directly) — assert status 200, `Content-Type`, and presence/shape of `Contact` and `Expires`.

## 4. RFC 9116 — required vs recommended fields

This section directly answers the user's question ("ustal, które pola są wymagane, a które tylko
zalecane").

### REQUIRED (MUST) — only two fields

| Field | RFC ref | Rule | Value format |
| :---- | :------ | :--- | :----------- |
| **Contact** | §2.5.3 | MUST always be present. One or more allowed; may repeat, with the **first = most preferred**. | A URI: `mailto:`, `tel:`, or `https://`. Web URIs **MUST** be `https://`. |
| **Expires** | §2.5.5 | MUST always be present; **MUST NOT** appear more than once (exactly one). | RFC 3339 timestamp; RECOMMENDED to be **< 1 year** in the future. |

**Minimal valid file = `Contact` + `Expires` only.**

### RECOMMENDED / OPTIONAL — all optional per §2.5

| Field | RFC ref | Notes |
| :---- | :------ | :---- |
| Encryption | §2.5.4 | RECOMMENDED when `Contact` is an email address (points to a key). Value is a URI; MUST NOT point to a private key. |
| Acknowledgments | §2.5.1 | Link to a page recognizing reporters. |
| Preferred-Languages | §2.5.8 | Comma-separated language tags (RFC 5646). **MUST NOT** appear more than once. |
| Canonical | §2.5.2 | Canonical URI(s) of this file. RECOMMENDED when the file is signed. |
| Policy | §2.5.7 | Link to the vulnerability disclosure policy. Orgs SHOULD use it. |
| Hiring | §2.5.6 | Link to security-related job postings. |

### Format rules that shape the HTTP response

- **Path:** `/.well-known/security.txt` (RFC 8615), served over **https**, applies only to the exact
  domain it is served from (§3, §3.1).
- **Syntax:** `Field: value`, one field per line; field names are **case-insensitive** (§2).
  Comment lines start with `#` (§2.1).
- **Line endings:** CRLF or LF both valid (§2.2).
- **Encoding:** UTF-8 (§4).
- **Content-Type:** MUST be `text/plain; charset=utf-8` (§3/§4).
- **Signature:** OpenPGP cleartext signature is RECOMMENDED, not required (§2.3); `Canonical` is
  RECOMMENDED when the file is signed.

### Not an RFC 9116 field

- **CSAF** is **not** defined in RFC 9116. It is a later addition to the IANA
  "security.txt Fields" registry (an extension). Treat as an extension only; do not include it as a
  core requirement.

## 5. Unknowns / open questions (for the checkpoint)

- **(a) `Contact` value for a demo app.** There is no real security team or inbox. Options:
  a GitHub Security Advisories URL for the repo
  (`https://github.com/<org>/microservices-demo/security/advisories`), the repo Issues URL, or a
  placeholder `mailto:`. **Recommendation:** use an `https://` GitHub security-advisories / repo URL
  (avoids a fake mailbox and keeps the web-URI-must-be-https rule satisfied). Confirm the exact
  org/repo path at the checkpoint.
- **(b) Which RECOMMENDED fields to include.** **Suggested minimal-but-useful set:** `Contact`,
  `Expires` (required) + `Preferred-Languages: en` + `Policy` (and/or `Acknowledgments`) pointing at
  a repo resource. Skip `Encryption` (no PGP key), `Canonical` (only meaningful when signed),
  `Hiring`. Confirm the set and the target URLs at the checkpoint.
- **(c) Root path vs `baseUrl` prefix.** Register at the literal root `/.well-known/security.txt`
  (strict RFC conformance regardless of `BASE_URL`) or under `baseUrl` like every other route
  (repo consistency)? Both are identical with the default empty `baseUrl`. **Recommendation:** lean
  toward the literal root path for RFC conformance, but decide at the checkpoint.
- **(d) Static literal vs dynamically-generated `Expires`.** **Recommendation: dynamic** (§3) so the
  file never goes stale/expired. Confirm the rolling window (e.g. 6 months vs ~1 year) at the
  checkpoint.

## 6. Findings & sources

- **RFC 9116** — "A File Format to Aid in Security Vulnerability Disclosure."
  <https://www.rfc-editor.org/rfc/rfc9116.txt> · <https://datatracker.ietf.org/doc/html/rfc9116>
  - §2 syntax / case-insensitive field names; §2.1 comments; §2.2 line endings (CRLF or LF);
    §2.3 signing (RECOMMENDED, not required).
  - §2.5 all fields optional except as noted; §2.5.1 Acknowledgments; §2.5.2 Canonical;
    §2.5.3 **Contact (REQUIRED)**; §2.5.4 Encryption; §2.5.5 **Expires (REQUIRED, exactly one)**;
    §2.5.6 Hiring; §2.5.7 Policy; §2.5.8 Preferred-Languages (at most once).
  - §3 / §3.1 location `/.well-known/security.txt`, https, per-domain; MUST content type; §4 UTF-8.
- **securitytxt.org** — human-readable summary + generator. <https://securitytxt.org/>
- **IANA "security.txt Fields" registry** (source of truth for field names / extensions incl. CSAF).
  <https://www.iana.org/assignments/security-txt-fields/>
- **RFC 8615** — Well-Known URIs (the `/.well-known/` path convention).
- **Local code (verified by reading):** `src/frontend/main.go` lines 58, 111 (`baseUrl` default `""`
  / `BASE_URL`), 149-163 (router + `robots.txt`/`_healthz` inline handlers); `src/frontend/handlers.go`
  (logrus-from-context + `renderHTTPError` conventions); no `src/frontend/*_test.go` exists yet.
  Build/test per CLAUDE.md: `cd src/frontend && go build ./... && go test ./...`.

Task tier: lightweight — single Go service, 1-2 files, no new dependency, no outbound seam, pinnable with a handful of httptest-based tests, no config/build-tooling change.
