# Online Boutique (microservices-demo)

> Cloud-first microservices demo: a web-based e-commerce app (browse, cart, checkout) built from
> 12 polyglot microservices communicating over gRPC, deployed to Kubernetes.
>
> **Current state:** 12 services exist and run as a full stack via Skaffold/Kubernetes — Go
> (frontend, productcatalog, shipping, checkout), C# (cart), Node (currency, payment), Python
> (email, recommendation, shoppingassistant, loadgenerator), Java (ad). Kept honest by `work-docs`
> (pipeline Step 5) after every shipped unit.

**Think Before Coding** — Don't assume. Don't hide confusion. Surface tradeoffs.
**Simplicity First** — Minimum code that solves the problem. Nothing speculative.
**Surgical Changes** — Touch only what you must. Clean up only your own mess.
**Goal-Driven Execution** — Define success criteria. Loop until verified.

## Coding principles
- **KISS** — the simplest thing that works.
- **YAGNI** — don't build what isn't asked for.
- **SRP** — one reason to change per unit.
- **DRY** — one source of truth; don't duplicate logic.

## Development workflow — dev-kit
This repo uses the **dev-kit** spec-driven pipeline. For ANY feature, bug fix, or refactor, the main
session acts ONLY as an **orchestrator**: it invokes the `dev-kit` skill and runs
research → tests → plan → execute → docs, **stopping after every step for the user's explicit
approval** before the next one starts. Production code and tests are written by the
[coder](.claude/agents/coder.md) subagent (and live smokes by `e2e-tester`) — **never inline in
the main thread**. Exempt: pure questions about how the code works, a one-line edit the user
explicitly asks to apply inline, and non-code chores.
The flow is the same for every unit; only the model tier differs — run the orchestrator (this main
session) on **Opus** for pipeline work (`/model opus`; there is deliberately no repo-wide `model` pin
in `.claude/settings.json`, so unrelated sessions aren't forced onto Opus), and the worker subagents
run on **Sonnet** for a **lightweight** unit (classified at the end of research) or **Opus** for a
**standard** one.

## How we work
- **Always track work in the TODO tool** (`TaskCreate`/`TaskUpdate`/`TaskList`) for any task with
  3+ steps or multiple delegations: write the plan first, one `in_progress` at a time.
- **Subagents do the coding.** The main session is an **orchestrator only** — it plans, delegates,
  verifies, and commits. It does not write production code directly. Delegate to the
  [coder](.claude/agents/coder.md) subagent, one small atomic task at a time.
- **Internet research is ALWAYS delegated to a clean subagent.** For library/SDK/cloud APIs prefer
  the docs MCPs (context7; Microsoft Learn for Azure/Foundry) over open-web guessing or memory.
- **Branch per unit; squash PR to `main`; manual merge.** Each unit of work runs on its own branch
  (`work/NNN-<slug>`) off `main`; every commit is **atomic** and lands on that branch; the unit
  ends with a **squash PR to `main`** that the team reviews and merges manually — never commit to
  `main` directly. See [docs/commit-conventions.md](docs/commit-conventions.md).
- **Spec-driven for non-trivial work.** Follow the pipeline in
  [docs/work/README.md](docs/work/README.md): research → tests → plan → execute → docs. Artifacts
  live in `docs/work/NNN-<slug>/`.
- **Quality gate.** A Stop/SubagentStop hook runs build + tests; nothing finishes red. See
  [.claude/hooks/quality-gate.sh](.claude/hooks/quality-gate.sh). This repo is polyglot, so the gate
  is **routed**: [.claude/quality-gate.routes](.claude/quality-gate.routes) maps each Go/C# service's
  path to its own build/test commands, so the gate runs only the service that changed — work in
  *the service you are editing* and check the green you get is that service's. Node/Python/Java
  (adservice) changes are not gated (no CI unit tests there); those units rely on their own executed
  acceptance check and the full skaffold e2e.
- **All docs and code are written in English.**

## Commands
Polyglot repo — no single build/test command. Build/test the service you're editing; the per-service
commands are the source of truth in [.claude/quality-gate.routes](.claude/quality-gate.routes).

| Service / area | Stack | Build | Test |
| :------------- | :---- | :---- | :--- |
| `src/shippingservice` | Go | `cd src/shippingservice && go build ./...` | `cd src/shippingservice && go test ./...` |
| `src/productcatalogservice` | Go | `cd src/productcatalogservice && go build ./...` | `cd src/productcatalogservice && go test ./...` |
| `src/frontend` | Go | `cd src/frontend && go build ./...` | `cd src/frontend && go test ./...` |
| `src/checkoutservice` | Go | `cd src/checkoutservice && go build ./...` | `cd src/checkoutservice && go test ./...` |
| `src/cartservice` | C# | `dotnet build src/cartservice/cartservice.sln` | `dotnet test src/cartservice/` |
| `src/adservice` | Java | `cd src/adservice && ./gradlew build` | `cd src/adservice && ./gradlew test` |
| `src/currencyservice`, `paymentservice` | Node | `npm --prefix src/<svc> ci` | — (no unit tests) |
| `src/emailservice`, `recommendationservice`, `shoppingassistantservice`, `loadgenerator` | Python | `pip install -r requirements.txt` | — (no unit tests) |

| Action | Command |
| :----- | :------ |
| Run (full stack) | `skaffold run` (or `skaffold dev`) against a k8s cluster — see [docs/test/README.md](docs/test/README.md) |
| Format (Go) | `gofmt -w .` within the service |

> The `coder` subagent and the quality gate read these. They are the single source of truth for
> "is this task green?". Only Go + C# are wired into the gate (they are what CI unit-tests and the
> only stacks that build/test fast without Docker).

## Structure
- `src/<service>/` — one microservice each (see the Commands table for stack).
- `protos/` — shared gRPC contracts; each service generates code via its own `genproto.sh`.
- `kubernetes-manifests/`, `kustomize/`, `helm-chart/`, `istio-manifests/` — deployment manifests.
- `skaffold.yaml` — builds all service images and deploys to a cluster (the local run path).
- `terraform/` — GKE provisioning.
- `docs/work/` — spec-driven pipeline artifacts (`NNN-<slug>/`).
- `docs/test/` — e2e runbook + live-smoke evidence.
- `.claude/` — committed agents, skills, hooks, settings, and `dev-kit.manifest` (the vendored dev-kit
  and its version stamp).

## Coding standards (the `coder` subagent loads these when the files match)
This repo enforces style through its per-language toolchains and `.editorconfig` (Go: tabs; C#/Java:
4-space; Python: 4-space; default 2-space). No repo-wide style-guide doc beyond that.
- `.editorconfig` — indentation/whitespace for all files.
- `gofmt` / `go vet` — Go services (`src/{frontend,productcatalogservice,shippingservice,checkoutservice}`).
- `dotnet format` — C# (`src/cartservice`).
- If a Protopia language coding-standard skill is available in the session, apply it when writing
  Python (`python-code-standard`) or frontend/TS code.

## Tools & plugins
- `context7` docs MCP — up-to-date library/SDK APIs (don't guess gRPC / framework APIs from memory).
