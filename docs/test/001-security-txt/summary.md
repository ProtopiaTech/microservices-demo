# Live e2e smoke — unit 001-security-txt (`GET/HEAD /.well-known/security.txt`, frontend service)

## Verdict: PASS

All required checks (GET 200 + headers + body content, POST 405, HEAD 200) passed against the
real, compiled frontend binary running standalone.

## Environment adaptation (deviation from `docs/test/README.md`)

`docs/test/README.md`'s shared bring-up requires a Kubernetes cluster + `skaffold run` (full
12-service stack). **No cluster is available in this environment** (`kubectl cluster-info` →
connection refused), so the full stack could not be brought up.

This endpoint is a **pure static handler with zero backend/gRPC dependencies** — it serves a fixed
`Contact`/`Expires` payload and does not call any downstream service. The frontend also builds its
gRPC clients lazily (`grpc.NewClient`), so it starts and serves HTTP fine even when given
unreachable dummy backend addresses. Given that, this smoke ran the **real, compiled
`src/frontend` binary standalone** (`go build`), started with dummy backend addresses (tracing and
profiling disabled), and exercised it over real HTTP with `curl`. The binary's actual router,
logging middleware, session middleware, and otelhttp instrumentation are all exercised exactly as
they would be in the full stack — only the 11 unrelated downstream services are absent, and this
endpoint never talks to them. This is a faithful live test for this specific endpoint, not a
UI/checkout flow that would require the full stack.

## Prerequisites / bring-up (adapted)

1. Built the frontend binary: `cd src/frontend && go build -o frontend-e2e .` — succeeded, no errors.
2. Started it on port 8080 with dummy backend addresses (`PRODUCT_CATALOG_SERVICE_ADDR=localhost:9991`,
   etc.), tracing/profiling disabled per its own startup log.
3. Polled `GET /_healthz` until `200` — returned `200` within 1 second. Startup log confirmed
   `"starting server on :8080"` with no fatal errors (the two `dial tcp 169.254.169.254:80:
   connect: host is down` lines are expected GCP-metadata-lookup failures in a non-GCP environment
   and are non-fatal — the server started and served traffic normally).

## Step-by-step results

### Step 3 — GET `/.well-known/security.txt`

Command: `curl -i -s http://localhost:8080/.well-known/security.txt`

Saved to: [`01-get-security-txt.txt`](./01-get-security-txt.txt)

```
HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8
Set-Cookie: shop_session-id=4b195a26-d63e-4991-8a9c-67ea68a08a9e; Max-Age=172800
Date: Thu, 16 Jul 2026 10:56:52 GMT
Content-Length: 72

Contact: mailto:security@example.com
Expires: 2027-07-16T12:56:42+02:00
```

Checks:
- Status `200` — **PASS**
- `Content-Type: text/plain; charset=utf-8` — **PASS**
- Body contains `Contact: mailto:security@example.com` — **PASS**
- Body contains an `Expires:` line with an RFC 3339 timestamp in the future — **PASS**.
  Observed value: `2027-07-16T12:56:42+02:00` (today is 2026-07-16, so this is ~1 year in the
  future, well-formed RFC 3339 with numeric UTC offset).
- Body contains ONLY the `Contact` and `Expires` fields (no other RFC 9116 fields such as
  `Encryption`, `Canonical`, `Preferred-Languages`, `Policy`, etc.) — **PASS**. The body is exactly
  two lines, matching `Content-Length: 72`.

### Step 4 — wrong method (POST)

Command: `curl -i -s -X POST http://localhost:8080/.well-known/security.txt`

Saved to: [`02-post-security-txt.txt`](./02-post-security-txt.txt)

```
HTTP/1.1 405 Method Not Allowed
Set-Cookie: shop_session-id=508ed9d7-657f-414c-9dcb-96bcdb363acc; Max-Age=172800
Date: Thu, 16 Jul 2026 10:56:57 GMT
Content-Length: 0
```

Check: status is not `200`, and is specifically `405 Method Not Allowed` — **PASS**.

### Step 5 — HEAD (optional)

Command: `curl -i -s -I http://localhost:8080/.well-known/security.txt`

Saved to: [`03-head-security-txt.txt`](./03-head-security-txt.txt)

```
HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8
Set-Cookie: shop_session-id=9dc4b13d-b0f2-48b7-81d5-79b731c7ba8b; Max-Age=172800
Date: Thu, 16 Jul 2026 10:56:57 GMT
Content-Length: 72
```

Check: `HEAD` is registered and returns `200` with the same `Content-Type` and `Content-Length` as
the `GET` — **PASS**.

### Step 6 — teardown

The background frontend process (PID `8845`) was killed after the smoke completed. Confirmed via
`ps -p 8845` (process not found) and a follow-up `curl` to `/_healthz` returning no response
(connection refused) — the server is not left running.

## Summary of all checks

| # | Check | Result |
|---|-------|--------|
| 1 | Binary builds cleanly | PASS |
| 2 | Server starts, `_healthz` returns 200 | PASS |
| 3a | GET returns `200` | PASS |
| 3b | `Content-Type: text/plain; charset=utf-8` | PASS |
| 3c | Body contains `Contact: mailto:security@example.com` | PASS |
| 3d | Body contains future RFC 3339 `Expires:` (`2027-07-16T12:56:42+02:00`) | PASS |
| 3e | Body contains ONLY `Contact` + `Expires` (no extra RFC 9116 fields) | PASS |
| 4 | POST → `405 Method Not Allowed` (not 200) | PASS |
| 5 | HEAD → `200`, same headers as GET | PASS |
| 6 | Background process cleanly stopped | PASS |

**Overall verdict: PASS** — the `/.well-known/security.txt` endpoint on the frontend service
behaves exactly as specified: serves the correct static RFC 9116 body only on `GET`/`HEAD`, with
the correct content type, contact address, and a future expiry timestamp, and rejects other
methods with `405`.
