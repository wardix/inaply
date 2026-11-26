# Claude Code Instructions

Read and strictly follow [AGENTS.md](./AGENTS.md) before modifying any file in this repository.

Key non-negotiable rules:
1. Follow **Strict TDD (`Red -> Green -> Refactor`)** for all logic in `services/gateway/`, CMS backend/RBAC/PMS/billing in `apps/cms/`, and non-UI business logic in `apps/tv-receiver/`.
2. Implement the issue's `Mandatory TDD Scenarios` in test files **first**, verify they fail (**RED**), then implement minimal production code until **GREEN**.
3. Keep `services/gateway` compatible with **Go 1.18** and verify with `go vet ./... && go test -count=1 -v -race ./...`.
4. Never push directly to `main`; open a PR filling out `.github/pull_request_template.md` with RED and GREEN test evidence.
