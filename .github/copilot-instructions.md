# GitHub Copilot Coding Agent Instructions

Read and strictly follow [AGENTS.md](../AGENTS.md) before starting any task.

Summary of mandatory workflow:
- Follow **Strict TDD (`Red -> Green -> Refactor`)** for `services/gateway/`, CMS backend/RBAC/PMS/billing logic in `apps/cms/`, and non-UI business logic in `apps/tv-receiver/`.
- Write failing tests for all `Mandatory TDD Scenarios` in the assigned GitHub Issue **before** writing implementation code, and verify the RED failure first.
- Keep `services/gateway/` compatible with **Go 1.18**.
- Before opening a PR, run `cd services/gateway && go vet ./... && go test -count=1 -v -race ./... && go build ./cmd/gateway` (plus CMS/Android tests once present) and fill out `.github/pull_request_template.md`.
