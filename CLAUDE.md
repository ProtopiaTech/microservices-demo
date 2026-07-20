# Online Boutique (microservices-demo)

> A cloud-native microservices demo: an 11-tier web e-commerce ("Online Boutique") app where
> users browse items, add them to a cart, and purchase them. Services span Go, C#/.NET, Java,
> Node.js, and Python, communicating over gRPC, and are deployed to Kubernetes.
>
> **Current state:** Full application exists and is deployed via Skaffold + Kubernetes manifests;
> per-service source lives under `src/`, with a shared gRPC contract in `protos/`. Kept honest by
> `work-docs` (pipeline Step 5) after every shipped unit.

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
  `main` directly. Open PRs/pushes on **`origin`** (the fork), never upstream. See
  [docs/commit-conventions.md](docs/commit-conventions.md).
- **Spec-driven for non-trivial work.** Follow the pipeline in
  [docs/work/README.md](docs/work/README.md): research → tests → plan → execute → docs. Artifacts
  live in `docs/work/NNN-<slug>/`.
- **Quality gate.** A Stop/SubagentStop hook runs the build + tests; nothing finishes red. See
  [.claude/hooks/quality-gate.sh](.claude/hooks/quality-gate.sh). This is a **multi-service** repo, so
  the gate is **routed**: [.claude/quality-gate.routes](.claude/quality-gate.routes) maps each
  service's path to its own build/test commands, so the gate runs only the service that changed —
  work in *the service you are editing* and check the green you get is that service's.
- **All docs and code are written in English.**

## Commands
Multi-service repo — no single build/test command. Per-service commands (also encoded in
[.claude/quality-gate.routes](.claude/quality-gate.routes)); build/test the service you're editing.

| Service / area | Build | Test |
| :------------- | :---- | :--- |
| `src/frontend/` (Go) | `cd src/frontend && go build ./...` | `cd src/frontend && go test ./...` |
| `src/checkoutservice/` (Go) | `cd src/checkoutservice && go build ./...` | `cd src/checkoutservice && go test ./...` |
| `src/productcatalogservice/` (Go) | `cd src/productcatalogservice && go build ./...` | `cd src/productcatalogservice && go test ./...` |
| `src/shippingservice/` (Go) | `cd src/shippingservice && go build ./...` | `cd src/shippingservice && go test ./...` |
| `src/cartservice/` (C#/.NET 10) | `dotnet build src/cartservice/cartservice.sln` | `dotnet test src/cartservice/` |
| `src/adservice/` (Java/Gradle) | `cd src/adservice && bash gradlew --no-daemon assemble` | — (no unit tests) |
| `src/currencyservice/` (Node) | `node --check` per `*.js` (npm ci fails on native deps) | — (no test suite) |
| `src/paymentservice/` (Node) | `node --check` per `*.js` (npm ci fails on native deps) | — (no test suite) |
| `src/emailservice/` (Python) | `python3 -m compileall -q src/emailservice` | — (no test suite) |
| `src/recommendationservice/` (Python) | `python3 -m compileall -q src/recommendationservice` | — |
| `src/loadgenerator/` (Python) | `python3 -m compileall -q src/loadgenerator` | — |
| `src/shoppingassistantservice/` (Python) | `python3 -m compileall -q src/shoppingassistantservice` | — |

**Run (whole app):** `skaffold run` (or `skaffold dev`) against a local/remote Kubernetes cluster
(minikube / kind / GKE); front end is exposed via the `frontend-external` service. There is no
lightweight local run — see [docs/test/README.md](docs/test/README.md).

> The `coder` subagent and the quality gate read these. They are the single source of truth for
> "is this task green?".

## Structure
- `src/<service>/` — the 12 microservices (Go, C#, Java, Node, Python), one dir each.
- `protos/` — shared gRPC contract (`demo.proto`) consumed by all services.
- `kubernetes-manifests/`, `kustomize/`, `helm-chart/`, `istio-manifests/` — deployment manifests.
- `terraform/` — infra provisioning.
- `skaffold.yaml`, `cloudbuild.yaml` — build/deploy orchestration.
- `docs/work/` — spec-driven pipeline artifacts (`NNN-<slug>/`).
- `docs/test/` — e2e runbook + live-smoke evidence.
- `.claude/` — committed agents, skills, hooks, settings, and `dev-kit.manifest` (the vendored dev-kit
  and its version stamp).

## Coding standards (the `coder` subagent loads these when the files match)
This repo ships no dedicated coding-standard skill; the standard for each service is its own
language toolchain and the repo's shared config:
- `.editorconfig` — whitespace/charset conventions across all services.
- Per-language toolchains — `go vet`/`gofmt` (Go), the `.csproj`/analyzers (C#), Gradle/Checkstyle
  config (Java), each service's `package.json` (Node), each service's `requirements.txt` (Python).
- Protopia Python conventions (`python-code-standard` skill) — apply when editing Python services.

## Tools & plugins
- `context7` MCP — up-to-date library/SDK docs, so APIs aren't guessed.
