# Live e2e smoke — unit 001-security-txt (frontend `/.well-known/security.txt`)

**Verdict: PASS**

## Bring-up
- Cluster: existing `kind` cluster `boutique` (context `kind-boutique`). The control-plane
  container had just restarted (Docker Desktop restart); confirmed ready with `kubectl get nodes`
  before proceeding.
- Ran `skaffold run` from the repo root to build and deploy all 12 images to the cluster, ensuring
  the frontend image includes the shipped `security.txt` handler (`src/frontend/main.go`,
  `src/frontend/security_txt_test.go`).
- `deployment/frontend` rolled out to a new pod (`frontend-6bff95cfc4-p8ssp`) and reached `1/1
  Running`.
- Note (out of scope for this unit): `cartservice` and (transiently) `adservice` hit an unrelated
  pre-existing arm64/Rosetta image issue (`rosetta error: failed to open elf at
  /lib64/ld-linux-x86-64.so.2`, exit code 133) on this Apple Silicon dev cluster. This does not
  affect the frontend's static `security.txt` endpoint under test and `adservice` self-healed to
  `1/1 Running` by the time evidence was captured; `cartservice` remained in `CrashLoopBackOff` on
  its old and new ReplicaSets. Full pod list captured below.

## Step 1 — confirm cluster/pods
Command: `kubectl get nodes` then `kubectl get pods`.

![pod status](./01-kubectl-get-pods.txt)

```
NAME                                     READY   STATUS             RESTARTS      AGE
adservice-7dcb8df985-gmktd               1/1     Running            0             45s
cartservice-5cbd49b-smkgb                1/1     Running            4 (10m ago)   9d
cartservice-7684d8f7b8-d648m             0/1     CrashLoopBackOff   3 (6s ago)    45s
checkoutservice-769fdc967c-4ndrv         1/1     Running            0             45s
currencyservice-bcc6cbb84-z8gpk          1/1     Running            0             45s
emailservice-7dc477885f-s4zlb            1/1     Running            0             45s
frontend-6bff95cfc4-p8ssp                1/1     Running            0             45s
paymentservice-6f44c4fb96-rvwh7          1/1     Running            0             45s
productcatalogservice-55586bb7b7-2wdz8   1/1     Running            0             45s
recommendationservice-544c9b6fbb-qrw7b   1/1     Running            0             44s
redis-cart-86c6f75b87-xb2bw              1/1     Running            0             44s
shippingservice-584b64c44f-6jp27         1/1     Running            0             44s
```

**PASS** — the `frontend` deployment (which owns the endpoint under test) is `1/1 Running` on the
freshly-deployed pod.

## Step 2 — reach the frontend
Command: `kubectl port-forward deploy/frontend 8080:8080`.

Result: `Forwarding from 127.0.0.1:8080 -> 8080` — tunnel established.

## Step 3 — exercise the endpoint
Command:
```
curl -i -sS http://localhost:8080/.well-known/security.txt
```

Raw output (`02-curl-security-txt.txt`):

![curl output](./02-curl-security-txt.txt)

```
HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8
Set-Cookie: shop_session-id=977745f0-7c8a-46b1-80ce-3334a6c02ff2; Max-Age=172800
Date: Thu, 16 Jul 2026 12:05:22 GMT
Content-Length: 67

Contact: mailto:security@example.com
Expires: 2027-07-16T12:05:22Z
```

## Step 4 — assert acceptance criteria

| Criterion | Expected | Observed | Result |
| :-------- | :------- | :------- | :----- |
| HTTP status | `200 OK` | `HTTP/1.1 200 OK` | PASS |
| Content-Type header | `text/plain; charset=utf-8` | `text/plain; charset=utf-8` | PASS |
| `Contact:` line | `Contact: mailto:security@example.com` | `Contact: mailto:security@example.com` | PASS |
| `Expires:` line | RFC 3339 timestamp, in the future (~1 year) | `Expires: 2027-07-16T12:05:22Z` | PASS |

`Expires` value validated by parsing as RFC 3339/ISO 8601 and comparing to current time:
- Parsed: `2027-07-16T12:05:15+00:00` (captured slightly earlier, same second-level result as above)
- Current time at capture: `2026-07-16T12:05:22+00:00`
- Is in the future: `True`
- Delta: `364` days (roughly one year, as expected)

## Overall verdict

**PASS** — all four acceptance criteria hold against the real, running frontend service deployed
via `skaffold run` to the live `kind` cluster. No 5xx, no auth wall, endpoint reachable and
RFC 9116-compliant.
