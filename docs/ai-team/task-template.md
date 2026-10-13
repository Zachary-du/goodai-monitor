# Task Specification: TASK-XXX [Task Title]

## Metadata
- **Task ID**: `TASK-XXX`
- **Title**: [Concise Task Title]
- **Current Status**: `BACKLOG` <!-- Options: BACKLOG | READY | IN_PROGRESS | REVIEW | BLOCKED | DONE -->
- **Assigned Role**: Developer Agent
- **Reviewer**: QA / Security Agent
- **Created Date**: YYYY-MM-DD
- **Target Branch**: `task/TASK-XXX-[short-name]`
- **Dependencies**: None <!-- Or: TASK-YYY -->

---

## 1. Requirement & Objective
- **PRD / Architecture Reference**: [Link to PRD or docs/architecture.md section]
- **Business Goal**: [Explain why this task is needed and what problem it solves]
- **Detailed Description**: [Clear, unambiguous explanation of functionality]

---

## 2. Scope Whitelist (Allowed Files & Modules)
> [!IMPORTANT]
> The Developer Agent is strictly restricted to creating or editing the files listed below. Any modification outside this list will result in immediate QA rejection.

- [ ] `internal/package_name/target_file.go`
- [ ] `internal/package_name/target_file_test.go`
- [ ] `docs/...`

---

## 3. Acceptance Criteria
- [ ] **AC-1**: [Specific measurable criteria]
- [ ] **AC-2**: [Specific measurable criteria]
- [ ] **AC-3**: [No Chinese text: 100% English UI, logs, comments, and identifiers]
- [ ] **AC-4**: [Zero secrets or sensitive credentials in code or configs]
- [ ] **AC-5**: [Memory overhead remains within budget < 25 MiB]

---

## 4. Testing Requirements
- **Unit Tests**: [Describe required unit test coverage and mock scenarios]
- **Integration Tests**: [Describe required integration test scenarios]
- **Static Analysis**: `go vet ./...` / `golangci-lint` clean with 0 warnings.
- **Strict No-Build Rule**: Automated test execution must run within lightweight scope or GitHub Actions CI.

---

## 5. Developer Implementation Log (Filled by Developer Agent)
*This section is filled when transitioning the task to `REVIEW`.*

- **Implementation Commit**: `[commit-hash]`
- **Files Modified**:
  - `path/to/file1.go`
  - `path/to/file1_test.go`
- **Automated Test Execution**:
  ```
  [Paste go test output or CI workflow run link]
  ```
- **Known Limitations & Caveats**: [Any non-blocking caveats]
- **Potential Regression Risks**: [Any risks to existing modules]
- **Handoff Date**: YYYY-MM-DD

---

## 6. QA Verification Summary (Filled by QA / Security Agent)
*This section is filled during `REVIEW` phase.*

- **Audit Report Reference**: `docs/ai-team/tasks/TASK-XXX-qa-report.md`
- **Verdict**: `APPROVED` / `REJECTED` / `ACTION_REQUIRED`
- **Audit Date**: YYYY-MM-DD
- **Human Approval Request**: [Ready for Human Lead merge approval / Needs developer fixes]
