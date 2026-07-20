# Live e2e smoke runbook — Online Boutique (microservices-demo)

The `e2e-tester` subagent follows THIS file to prove a change works against the **real, running**
system. It is the contract: prerequisites, entry point, exact steps, expected results,
troubleshooting. Evidence + `summary.md` go to `docs/test/NNN-<slug>/`.

> **Current status: e2e is `n/a` by default, but the `frontend` path is now proven.** The app only
> runs on a Kubernetes cluster via Skaffold — there is no lightweight local run — and this repo does
> not pin a reliable local cluster for every service. Unit 001-security-txt did exercise a real live
> smoke against a local `kind` cluster (`kind-boutique`): image rebuilt, `kind load`ed, rolled out,
> port-forwarded, curled — see `docs/test/001-security-txt/`. So units touching **`frontend`** can
> attempt a real live smoke against a local kind cluster per the subsection below, rather than
> defaulting to `e2e: n/a`. Other services still need their subsections filled and a cluster
> available; for those, `work-plan` should keep declaring `e2e: n/a — no running cluster wired yet`
> for units covered by build + unit tests. The routed quality gate (`.claude/quality-gate.routes`)
> is the mechanical floor in the meantime.

## Prerequisites (shared bring-up, once wired)
- A Kubernetes cluster: local (`minikube start` / `kind create cluster`) or remote (GKE).
- `kubectl` and `skaffold` on PATH; Docker for local image builds.
- Bring the whole app up from the repo root: `skaffold run` (build + deploy) or `skaffold dev`
  (rebuild-on-change). Wait for all deployments to become available
  (`kubectl wait --for=condition=available --timeout=600s deployment --all`).
- Reach the storefront: `kubectl port-forward deployment/frontend 8080:8080`, then open
  `http://localhost:8080` (or the `frontend-external` LoadBalancer IP on a cloud cluster).
- Never capture or commit secrets; use synthetic/test data only (see Evidence & retention).

## Entry point
See **Prerequisites** — the shared entry point is `skaffold run` + the frontend port-forward. Per
service, follow the matching subsection under **## Services** once filled.

## Expected verdict
PASS = the documented observable outcomes hold. A timeout, a 5xx, an auth wall, or fabricated
output is a **FAIL** — captured, never a false pass.

## Services (fill one subsection per service as the team works on it)
The dispatch names the service(s) under test; the tester follows **that service's** subsection only.
The storefront `frontend` is the natural smoke surface for most user-visible changes; back-end
services (checkout, cart, product catalog, shipping, currency, payment, etc.) are exercised through
the frontend flows or via direct gRPC calls (`grpcurl`) against a port-forwarded pod.

### frontend
- **Entry point:** `kubectl port-forward deployment/frontend 8080:8080` → `http://localhost:8080`.
- **Steps & expected results:**
  1. Load the home page → product grid renders with prices in the selected currency. Capture
     `01-home.png`.
  2. Open a product, "Add to Cart", then view cart → item appears with correct quantity/price.
     Capture `02-cart.png`.
  3. Complete checkout with test card data → order-confirmation page with an order ID. Capture
     `03-order-confirmation.png`.
  4. Curl the well-known security endpoint: `curl -isS http://localhost:8080/.well-known/security.txt`
     → `200`, `Content-Type: text/plain; charset=utf-8`, a `Contact:` line, and exactly one future
     RFC 3339 `Expires:` line. Capture the raw response to a text file (it's an HTTP resource, not a
     page, so a curl transcript stands in for the screenshot).
- **Expected verdict:** PASS = browse → cart → checkout completes with no 5xx and correct totals, and
  `/.well-known/security.txt` returns the expected headers and body.

## Troubleshooting
- Pods stuck `Pending` → cluster lacks resources; give minikube/kind more CPU/memory.
- `ImagePullBackOff` on a local cluster → build with Skaffold (`skaffold dev`) rather than pulling
  pre-built images, or point Skaffold at your local Docker daemon.
- Frontend 500s referencing a downstream service → that gRPC dependency isn't ready; re-check
  `kubectl get pods` and wait for all deployments to be available.

## Evidence & retention
Screenshots are **committed to the repo** as a visual audit trail of what the smoke actually saw — a
deliberate feature, kept for the long run. The catch: a committed image is **permanent in git
history**, so the discipline here is about **security**, not avoidance:
- **Never capture a secret.** No screen showing real credentials, tokens, API keys, or PII — a leaked
  secret baked into an image is permanent and can't be `git rm`-ed out of history. Use throwaway /
  test accounts and seeded or synthetic data; redact or crop anything sensitive before you capture.
- **Capture the viewport, not the full page** — enough to prove the step, no more surface to leak.
- **`summary.md` stands alone.** It must read completely without the images; the screenshots back up
  what it already states, they don't replace it.
