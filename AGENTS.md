# Repository Guidelines for AI Coding Agents (Inaply)

This repository implements **Inaply** (**Inaply Cast**), a hospitality-grade TV casting, welcome screen, TV ads, and guest messaging platform.

All AI coding agents (Antigravity, Claude Code, Codex, Copilot, Cursor, etc.) MUST follow the TDD and engineering rules below when working on any GitHub Issue or Pull Request.

---

## 1. Monorepo Layout & Tech Stack

- `services/gateway/` — **Go 1.18+** On-Premise Gateway (`github.com/inaply/gateway`): mDNS/SSDP proxy, session store, dynamic firewall (`nftables`), local edge cache, captive portal & HTTP API.
  - **Compatibility requirement:** Keep `services/gateway` compatible with **Go 1.18** (avoid stdlib packages added in Go 1.19+ such as `slices`, `maps`, `cmp`, `atomic.Bool`, `min`/`max` builtins, or Go 1.22 `http.ServeMux` method/path patterns).
- `apps/cms/` — **TypeScript / Next.js / PostgreSQL** Cloud Multi-Tenant CMS (`@inaply/cms`), schema in `apps/cms/db/schema.sql`.
- `apps/tv-receiver/` — **Kotlin Android TV** Kiosk Receiver (`com.inaply.receiver`), Device Owner MDM, HDMI-CEC, Backdrop/Ads/Overlay player.
- `contracts/api-v1.yaml` — Shared OpenAPI 3.0.3 contract across Gateway, TV Receiver, CMS, and PMS.
- `docs/` — [PRD.md](./docs/PRD.md) and [ARCHITECTURE.md](./docs/ARCHITECTURE.md).

---

## 2. Mandatory TDD Policy (Test-First)

### A. Strict TDD Required (`Red -> Green -> Refactor`)
You MUST write failing tests **before** writing implementation code for:
1. **All Go Gateway changes (`services/gateway/**`):** mDNS/SSDP packet handling, session pairing/eviction/checkout, RADIUS/DHCP/Wi-Fi controller adapters, `firewall` drivers & `conntrack`, edge persistence, local asset cache, HTTP APIs.
2. **All CMS backend & domain logic (`apps/cms/**`):** API routes, multi-tenant RBAC/authorization (preventing cross-property/cross-partner leaks), PMS connectors, billing/metering calculations, analytics tracking, Cloud-to-Edge sync.
3. **All non-UI business logic in Android TV (`apps/tv-receiver/**`):** `GatewaySyncClient`, `reset_epoch` session termination, ad playlist filtering/scheduling, LAN media cache & SHA-256 validation, OTA updater, `HdmiCecController` command state, and `DeviceOwnerManager` policy logic.

### B. Test-After / Preview Allowed (Visual UI Only)
Pure visual presentation layers (Jetpack Compose TV layout/animations, Next.js JSX/CSS styling) may be iterated with component/render tests and visual preview or device QA **after** their backing view-models/state logic have been built via Strict TDD.

---

## 3. Required TDD Execution Loop

For every issue requiring Strict TDD:
1. **Read the Issue & Contract:** Read the GitHub Issue's `Mandatory TDD Scenarios`, `contracts/api-v1.yaml`, and `docs/ARCHITECTURE.md`.
2. **RED Phase (Test First):**
   - Write unit/integration tests covering every scenario and edge case listed in the issue (plus cross-room / cross-tenant isolation checks).
   - Run the test suite and verify the new tests **FAIL** for the expected missing feature/behavior (not due to syntax/import mistakes).
   - If working with separate commits, commit the failing tests first (`test: add failing tests for #<issue>`).
3. **GREEN Phase (Minimal Implementation):**
   - Write clean, modular production code using dependency injection (`CmdRunner`, `FirewallDriver`, interfaces for network/OS/clock calls) so all tests run deterministically without root or real hardware.
   - Run tests and verify 100% pass.
4. **REFACTOR Phase:**
   - Remove duplication, keep functions focused and readable, preserve Go 1.18 compatibility, and re-run the full verification suite.

### Hard Prohibitions
- **NEVER** weaken, skip, or delete assertions to make a failing test pass.
- **NEVER** hardcode test-specific return values just to satisfy one assertion.
- **NEVER** break room isolation (Guest in Room A must NEVER discover, receive mDNS/SSDP for, or reach Room B's TV) or multi-tenant isolation (Property A must NEVER access Property B's data).
- **NEVER** push directly to `main`; always open a Pull Request and complete `.github/pull_request_template.md`.

---

## 4. Verification Commands

Run the relevant verification suite after your final edit and include the observed output in your PR description:

- **Go On-Premise Gateway:**
  ```bash
  cd services/gateway && go vet ./... && go test -count=1 -v -race ./... && go build ./cmd/gateway
  ```
- **Cloud CMS (once `tsconfig.json` / test runner are configured):**
  ```bash
  cd apps/cms && bun run typecheck && bun test
  ```
- **Android TV Receiver (once Gradle wrapper is added):**
  ```bash
  cd apps/tv-receiver && ./gradlew testDebugUnitTest
  ```
