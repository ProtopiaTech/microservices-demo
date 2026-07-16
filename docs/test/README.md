# Live e2e smoke runbook — Online Boutique

The `e2e-tester` subagent follows THIS file to prove a change works against the **real, running**
system. It is the contract: prerequisites, entry point, exact steps, expected results,
troubleshooting. Evidence + `summary.md` go to `docs/test/NNN-<slug>/`.

Online Boutique only runs as a **full multi-service stack** — there is no standalone `run` for a
single service. The whole app is brought up once (below), then each service's smoke is exercised
through the running frontend / that service's gRPC endpoint.

## Prerequisites (shared bring-up)
- **Docker** running locally (Skaffold builds the images).
- **A Kubernetes cluster** reachable via `kubectl` — a local one is enough:
  `minikube start --cpus=4 --memory=8g` (or `kind create cluster`). Confirm with `kubectl get nodes`.
- **Skaffold** installed (`skaffold version`).
- Bring the whole app up from the repo root:
  ```bash
  skaffold run              # builds all 12 images and deploys to the current cluster
  kubectl get pods          # wait until every pod is Running / Ready
  ```
- Reach the frontend (choose one):
  ```bash
  kubectl port-forward deploy/frontend 8080:8080     # then open http://localhost:8080
  # or, if a LoadBalancer IP is provisioned:
  kubectl get service frontend-external -o jsonpath='{.status.loadBalancer.ingress[0].ip}'
  ```
- Tear down when done: `skaffold delete`.
- The e2e surface here is a **web UI** — the `e2e-tester` should drive it with the Playwright
  browser MCP (navigate, click, screenshot). gRPC-only services are smoked via the frontend flow
  that exercises them (see each subsection).

> Until a cluster is available in the working environment, `work-plan` may declare
> `e2e: n/a — no cluster available` for units that ship no user-observable change, and rely on the
> unit's own acceptance check + the routed quality gate.

## Expected verdict
PASS = the documented observable outcomes hold for the service(s) under test. A timeout, a 5xx, an
auth wall, a crashlooping pod, or fabricated output is a **FAIL** — captured, never a false pass.

## Services
The dispatch **names the service(s) under test**; the tester follows **that service's** subsection
only, against the shared stack brought up above.

### frontend (Go, web UI)
- **Entry point:** `http://localhost:8080` (port-forward above).
- **Steps & expected results:**
  1. Load the home page → product grid renders, currency selector present. Capture `01-home.png`.
  2. Open a product, click **Add to Cart** → cart badge increments, cart page lists the item.
     Capture `02-cart.png`.
  3. Complete checkout with the demo card → order-confirmation page with an order ID.
     Capture `03-checkout.png`.
- **Expected verdict:** PASS = home → cart → checkout all render without a 5xx or blank page.

### productcatalogservice (Go, gRPC)
- **Entry point:** exercised via the frontend product grid + product detail pages.
- **Steps & expected results:**
  1. Home page lists products with names/prices → catalog served. Capture `01-catalog.png`.
  2. Open a product detail page → correct name, price, image. Capture `02-product.png`.
- **Expected verdict:** PASS = products load and detail pages match the catalog data.

### cartservice (C#, gRPC + Redis)
- **Entry point:** exercised via the frontend cart flow.
- **Steps & expected results:**
  1. Add two different items → cart shows both with quantities. Capture `01-cart-items.png`.
  2. Empty the cart / start a new session → cart is empty. Capture `02-cart-empty.png`.
- **Expected verdict:** PASS = cart persists across page loads and empties correctly.

### shippingservice (Go, gRPC)
- **Entry point:** exercised via the checkout flow (quote + ship).
- **Steps & expected results:**
  1. Proceed to checkout with items in cart → a shipping cost is shown. Capture `01-shipping.png`.
  2. Complete the order → confirmation includes a shipping tracking ID. Capture `02-tracking.png`.
- **Expected verdict:** PASS = a shipping quote appears and the order confirms with a tracking ID.

### checkoutservice (Go, gRPC — orchestrates cart/payment/shipping/email)
- **Entry point:** exercised via the full checkout flow on the frontend.
- **Steps & expected results:**
  1. With items in cart, submit checkout with the demo card → order-confirmation page with order ID,
     shipping cost, and item list. Capture `01-order-confirmation.png`.
  2. Negative: submit checkout with an empty cart → handled gracefully (no 5xx). Capture `02-empty.png`.
- **Expected verdict:** PASS = a valid order confirms end-to-end; the empty-cart path degrades gracefully.

## Troubleshooting
- Pods stuck `Pending` / `ImagePullBackOff` → cluster low on resources or images not built; re-run
  `skaffold run`, give minikube more CPU/memory.
- `cartservice` failing → check the `redis-cart` deployment is Ready.
- Frontend 500s on checkout → a downstream gRPC service (payment/shipping/currency) is not Ready;
  `kubectl get pods` and check its logs with `kubectl logs deploy/<service>`.

## Evidence & retention
Screenshots are **committed to the repo** as a visual audit trail of what the smoke actually saw — a
deliberate feature, kept for the long run. The catch: a committed image is **permanent in git
history**, so the discipline here is about **security**, not avoidance:
- **Never capture a secret.** No screen showing real credentials, tokens, API keys, or PII — a leaked
  secret baked into an image is permanent and can't be `git rm`-ed out of history. This demo uses a
  fake payment card and synthetic data — keep it that way; redact anything sensitive before capture.
- **Capture the viewport, not the full page** — enough to prove the step, no more surface to leak.
- **`summary.md` stands alone.** It must read completely without the images; the screenshots back up
  what it already states, they don't replace it.
