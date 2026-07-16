# 001 — security.txt endpoint (frontend) — Research

## 1. Scope

**In scope:** Add a `GET /.well-known/security.txt` endpoint to the Online Boutique
**frontend** (Go, `src/frontend`). It serves a static, RFC 9116-compliant `security.txt`
document as `text/plain; charset=utf-8`, containing the minimal REQUIRED field set
(`Contact`, `Expires`) plus a small number of RECOMMENDED fields. Implemented as an inline
`gorilla/mux` handler, mirroring the existing `robots.txt` / `_healthz` precedents.

**Out of scope:**
- OpenPGP / digital signing of the file (RFC 9116 §2.3 — RECOMMENDED but explicitly out for a demo).
- Dynamic / per-tenant / per-request customization of field values beyond a possibly-computed
  `Expires` timestamp.
- A separate signed `security.txt.sig` file, `Canonical`/`Encryption` key hosting.
- Any backend/gRPC involvement — the response is static and self-contained.

## 2. System slice

Exact files/seams touched:
- `src/frontend/main.go` — the `gorilla/mux` router `r := mux.NewRouter()` (~line 149) where all
  routes are registered relative to `baseUrl` (env-configurable path prefix).
- **Inline-handler precedent** to mirror:
  - `src/frontend/main.go:160` — `robots.txt`:
    `r.HandleFunc(baseUrl+"/robots.txt", func(w, _){ fmt.Fprint(w, "User-agent: *\nDisallow: /") })`
  - `src/frontend/main.go:161` — `_healthz` returns `ok`.
  - `PathPrefix(baseUrl+"/static/")` serves static files via `http.FileServer`.

No new package, no gRPC client, no backend seam. The literal dot in
`/.well-known/security.txt` is fine as a plain `HandleFunc` string path in gorilla/mux.

## 3. Stack & seams

- **Language / router:** Go + `gorilla/mux`. Registration via `r.HandleFunc(...)`.
- **No external seam** — the handler writes a constant (or near-constant) body.
- **`baseUrl` prefix:** existing routes are registered as `baseUrl+"/..."`. The RFC canonical
  location is the domain-root `/.well-known/security.txt` (§3), so honoring `baseUrl` is an open
  question (see §5).
- **Content-Type concern:** RFC 9116 §3/§4 require `Content-Type: text/plain; charset=utf-8`.
  Go's `http.DetectContentType` would sniff plain text as `text/plain; charset=utf-8` only
  loosely; the handler MUST set the header explicitly rather than rely on sniffing (unlike the
  `robots.txt` precedent, which does not set it).
- **No unit-test infrastructure yet:** there are currently no `*_test.go` files in `src/frontend`.
  Frontend is a gated Go service (`cd src/frontend && go build ./... && go test ./...`), so
  1–2 `httptest`-based tests can be added to pin the behaviour.

## 4. Required vs recommended fields (RFC 9116)

| Field | Status (RFC 9116) | Meaning | Section |
| :--- | :--- | :--- | :--- |
| `Contact` | **REQUIRED** | How to report a vulnerability; one or more, in order of preference; `https://`, `mailto:`, or `tel:` URI | §2.5.3 |
| `Expires` | **REQUIRED** | Exactly one; date/time after which the data is stale; RFC 3339 timestamp; RECOMMENDED < 1 year out | §2.5.5 |
| `Encryption` | Optional/Recommended | URI to an encryption key for secure comms (`https://`, `openpgp4fpr:`, etc.) | §2.5.4 |
| `Acknowledgments` | Optional/Recommended | Link to a page thanking reporters (`https://`) | §2.5.1 |
| `Preferred-Languages` | Optional/Recommended | Comma-separated RFC 5646 language tags; no order/preference implied | §2.5.8 |
| `Canonical` | Optional/Recommended | Canonical URI(s) of this security.txt file (`https://`) | §2.5.2 |
| `Policy` | Optional/Recommended | Link to the vulnerability-disclosure policy (`https://`) | §2.5.7 |
| `Hiring` | Optional | Link to security-related job openings (`https://`) | §2.5.6 |

Notes:
- **Exactly two fields are REQUIRED:** `Contact` and `Expires` (§2.5.3, §2.5.5).
- Web-URI values MUST use `https://` (§2.5.x).
- `CSAF` is **not** part of RFC 9116 — it is a later IANA-registered extension.
- Unknown/unrecognized fields MUST be ignored by parsers (§2.4).

**What THIS endpoint will emit (recommendation):** the required minimum — `Contact` + `Expires`
— optionally augmented with `Policy` and `Preferred-Languages`. The exact contact address and the
expiry policy are open questions (§5).

## 5. Unknowns / open questions

1. **Contact value:** what address to publish. This is a demo with no real security team — likely
   a placeholder (`mailto:security@example.com` or a repo issues URL). Needs a decision.
2. **`Expires` value / freshness:** a hardcoded date goes stale and would eventually violate the
   `< 1 year` recommendation (§2.5.5). Options: (a) hardcode a date and accept staleness, or
   (b) compute `now + ~1 year` (e.g. `time.Now().AddDate(1,0,0).UTC().Format(time.RFC3339)`) at
   request time so it is always fresh and compliant. Recommend (b).
3. **`baseUrl` prefix:** RFC canonical location is domain-root `/.well-known/security.txt` (§3).
   Should the route be registered at `baseUrl+"/.well-known/security.txt"` (consistent with all
   other frontend routes) or at the absolute root? Recommend following existing `baseUrl` convention
   for consistency, since the demo may run under a path prefix.
4. **HEAD support:** other routes are registered with `HandleFunc` (which in gorilla/mux serves the
   given method(s)); decide whether HEAD should be explicitly supported like the rest.
5. **Line endings / encoding:** CRLF or LF both allowed (§2.2); UTF-8 Net-Unicode (§4). A Go raw
   string literal with `\n` is fine.

## 6. Findings & sources

- **RFC 9116** (https://www.rfc-editor.org/rfc/rfc9116.html):
  - Two REQUIRED fields: `Contact` (§2.5.3) and `Expires` (§2.5.5). `Expires` is exactly one,
    RFC 3339 timestamp, RECOMMENDED less than a year from publication.
  - Canonical location: `/.well-known/security.txt` (§3), served over `https` scheme, with
    response `Content-Type: text/plain; charset=utf-8` (§3, §4).
  - Comments begin with `#` (§2.1); line endings CRLF or LF (§2.2); UTF-8 Net-Unicode (§4);
    web-URI field values MUST be `https://`; digital signature allowed/RECOMMENDED (§2.3);
    unknown fields MUST be ignored (§2.4).
  - Optional fields: Encryption (§2.5.4), Acknowledgments (§2.5.1), Preferred-Languages (§2.5.8,
    RFC 5646 tags), Canonical (§2.5.2), Policy (§2.5.7), Hiring (§2.5.6).
- **securitytxt.org** (https://securitytxt.org/) — reference/generator confirming `Contact` +
  `Expires` as the required baseline.
- **IANA security.txt Fields registry**
  (https://www.iana.org/assignments/security-txt-fields/) — authoritative field list; CSAF present
  here as an extension but NOT in RFC 9116.
- **Local precedent:** `src/frontend/main.go:160-161` (robots.txt, `_healthz`) are the exact
  inline-handler pattern to follow; router at `main.go:~149`.

Task tier: lightweight — one file (`src/frontend/main.go`), a static inline handler with no new
dependency or backend seam, pinnable with 1–2 `httptest` tests.
