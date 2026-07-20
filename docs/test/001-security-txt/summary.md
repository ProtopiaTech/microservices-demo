# Live e2e smoke — unit 001-security-txt (service: frontend)

**Branch:** work/001-security-txt
**Verdict: PASS**

## What was proven
`GET /.well-known/security.txt` on the live, freshly-deployed `frontend` service returns HTTP 200,
`Content-Type: text/plain; charset=utf-8`, a `Contact: mailto:security@example.com` line, and
exactly one `Expires:` line with an RFC 3339 timestamp ~6 months in the future. The storefront
home page still serves 200 after the redeploy.

## Bring-up (why this is a real test, not the stale pod)
The running `frontend` pod predated the change (image built ~4 days ago, no new endpoint), so the
following was done before smoking the endpoint:

1. `docker build -t frontend:e2e-001-securitytxt src/frontend` — **PASS**, image built successfully
   from the current `work/001-security-txt` source (Go build completed, image exported as
   `frontend:e2e-001-securitytxt`).
2. `kind load docker-image frontend:e2e-001-securitytxt --name boutique` — **PASS**, image loaded
   onto node `boutique-control-plane`.
3. `kubectl set image deployment/frontend server=frontend:e2e-001-securitytxt` (container name
   confirmed via `kubectl get deployment frontend -o jsonpath='{.spec.template.spec.containers[*].name}'`
   → `server`) — **PASS**, deployment updated.
4. `kubectl rollout status deployment/frontend --timeout=180s` — **PASS**:
   ```
   deployment.apps/frontend image updated
   Waiting for deployment "frontend" rollout to finish: 1 old replicas are pending termination...
   deployment "frontend" successfully rolled out
   ```
   New pod `frontend-567f6d6bbb-kwjjq` came up `Running 1/1` on the new image (confirms the kind
   node pulled the locally-loaded image, not a stale cached one — otherwise the rollout would have
   stuck on `ImagePullBackOff`).
5. `kubectl port-forward deployment/frontend 8080:8080` (backgrounded) — **PASS**, forwarding
   established (`Forwarding from [::1]:8080 -> 8080`).

## Step 1 — `GET /.well-known/security.txt` — PASS

**Input sent:**
```
curl -isS http://localhost:8080/.well-known/security.txt
```

**Observed output** (full response, see [`01-curl-security-txt.txt`](./01-curl-security-txt.txt)):
```
HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8
Set-Cookie: shop_session-id=a58b718d-cbf3-4842-9d71-157cb61986c0; Max-Age=172800
Date: Mon, 20 Jul 2026 12:43:01 GMT
Content-Length: 67

Contact: mailto:security@example.com
Expires: 2027-01-20T12:43:01Z
```

Assertion checks:
- **Status 200** — PASS (`HTTP/1.1 200 OK`).
- **`Content-Type: text/plain; charset=utf-8`** — PASS, present verbatim in headers.
- **Body contains `Contact: mailto:security@example.com`** — PASS, present verbatim in body.
- **Exactly one `Expires:` line, RFC 3339, in the future (~6 months out)** — PASS.
  `grep -c '^Expires:'` → `1`. Value `2027-01-20T12:43:01Z` parses as RFC 3339 (`YYYY-MM-DDTHH:MM:SSZ`).
  Current time at capture was `2026-07-20T12:43:10Z` (via `date -u`), so the `Expires` value is
  ~6 months in the future, consistent with the expected ~6-month-out policy.

## Step 2 — Storefront sanity (`GET /`) — PASS

**Input sent:**
```
curl -isS http://localhost:8080/
```

**Observed output** (see [`02-home-sanity.txt`](./02-home-sanity.txt), truncated to headers +
top of body):
```
HTTP/1.1 200 OK
Set-Cookie: shop_session-id=71f1d4ac-aec0-45f6-ad4f-d9f13f48bb6b; Max-Age=172800
Date: Mon, 20 Jul 2026 12:43:04 GMT
Content-Type: text/html; charset=utf-8
Transfer-Encoding: chunked

<!DOCTYPE html>
<html lang="en">
...
<title>
    Online Boutique
```

- **Status 200, HTML storefront rendered** — PASS. Confirms the redeploy of the new frontend
  image did not break the app.

## Cleanup
The backgrounded `kubectl port-forward` process was terminated after capturing evidence
(`pkill -f "kubectl port-forward deployment/frontend"`); no forwarding left running.

## Conclusion
All 4 required assertions on `/.well-known/security.txt` hold, and the storefront home page still
serves correctly after the frontend was rebuilt/redeployed from the `work/001-security-txt`
branch. **Verdict: PASS.**
