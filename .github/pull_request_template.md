## Summary
<!-- Describe what this PR implements and link the issue -->
Closes #

## TDD Evidence (Mandatory for Gateway, Backend, PMS, Billing, & Core TV Logic)
- **Test files created/updated BEFORE implementation:**
  - `path/to/file_test.go` (or `.test.ts` / `Test.kt`)

- **RED Phase (Failing test output observed before implementation):**
```text
Paste failing test command output here
```

- **GREEN Phase (Passing test output after final edit):**
```text
Paste passing `go test -count=1 -v -race ./...` (or CMS/Gradle test) output here
```

## Verification Checklist
- [ ] Followed `AGENTS.md` Strict TDD policy (`Red -> Green -> Refactor`)
- [ ] All existing and new tests pass without weakening or skipping assertions
- [ ] Cross-room isolation & multi-tenant isolation verified (where applicable)
- [ ] `contracts/api-v1.yaml` and `docs/` updated if API contracts changed
- [ ] Go code remains compatible with Go 1.18 (`services/gateway/`)
