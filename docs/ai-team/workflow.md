# AI Team Collaboration: Workflow & Task Lifecycle

## 1. Task State Machine

Tasks progress through a strict, deterministic state machine:

```
[BACKLOG]
   │  (PM creates task specification)
   ▼
[READY] ◄────────────── (Human Lead reviews & approves task)
   │
   │  (Developer claims task, creates worktree/branch)
   ▼
[IN_PROGRESS] ◄────────┐
   │                   │ (QA rejects or finds defects)
   │  (Dev finishes,   │
   │   runs tests,     │
   │   submits diff)   │
   ▼                   │
[REVIEW] ──────────────┘
   │
   │  (QA completes audit: all PASS, 0 security issues)
   ▼
[HUMAN_APPROVAL]
   │  (Human Lead verifies & approves merge)
   ▼
[DONE]
```

*Note*: If a blocker (missing dependency, external API failure, requirement ambiguity) arises at any stage, the task moves to `[BLOCKED]` until resolved.

---

## 2. State Definitions & Transition Gates

| State | Responsible Role | Entry Criteria | Exit Criteria |
|---|---|---|---|
| **BACKLOG** | PM Agent | Feature request or PRD requirement identified. | Task file generated with objective, scope whitelist, and acceptance criteria. |
| **READY** | Human Lead | Task specification reviewed by Human Lead. | Human Lead confirms task is approved for implementation. |
| **IN_PROGRESS** | Developer Agent | Task is in `READY` and dependencies are satisfied. | Code and automated tests written; CI passes; handoff summary prepared. |
| **REVIEW** | QA / Security Agent | Developer submits handoff summary and PR/diff. | Independent verification completed; all items marked `PASS`/`FAIL`/`NOT VERIFIED`. |
| **BLOCKED** | All | Unresolved external blocker, ambiguity, or critical defect. | Root cause addressed; unblocked by Human Lead or dependency task. |
| **DONE** | Human Lead | QA audit marked as `APPROVED` with zero open blockers. | Code merged into `main` and production release tagged. |

---

## 3. Git Branching & Worktree Isolation Rules

### 3.1 Worktree & Branch Isolation
- **No Concurrent Edits in Shared Workspace**: Multiple agents must **never** edit files simultaneously in the same working directory.
- **Dedicated Branching**: Every task must be developed in its own branch:
  ```bash
  # Branch naming convention
  task/<task-id>-<short-description>
  # Example:
  git checkout -b task/task-001-host-collector
  ```
- **Git Worktrees for Parallel Execution**: When multiple Developer Agents or QA Agents operate concurrently, each must be isolated inside a dedicated Git worktree:
  ```bash
  git worktree add ../worktrees/task-001 task/task-001-host-collector
  ```

### 3.2 Serialization Rules (No Parallel Editing)
The following tasks and files **must be executed serially** (one at a time):
1. **Entry Points & Shared Configurations**: `cmd/monitor/main.go`, `internal/config/config.go`.
2. **Database Schema & Migrations**: Changes touching SQLite table structures or migration scripts.
3. **Core Interface Definitions**: Public interfaces in `internal/model` or shared domain types.
4. **Interdependent Tasks**: Any task that depends on the unmerged output of another task.

### 3.3 Merge & Production Rules
- **No Autonomous Merging**: No agent is permitted to merge code into `main` without explicit Human Lead approval.
- **No Direct Push to Production**: Agents must never touch production VPS configurations (`158.179.29.27`) or deployment scripts without user authorization.

---

## 4. Structured Handoff Protocol

Chat conversations can be ephemeral and noisy. **All handoffs must be documented in structured files inside `docs/ai-team/tasks/`**.

### 4.1 PM -> Developer Handoff
- Task File: `docs/ai-team/tasks/TASK-XXX.md`
- Required Fields:
  - Task ID and Title.
  - Linked PRD Section and Objective.
  - Scope Whitelist (Exact list of files/directories permitted to create or modify).
  - Explicit Acceptance Checklist.
  - Testing Requirements.

### 4.2 Developer -> QA Handoff
- Appended to `docs/ai-team/tasks/TASK-XXX.md` (Implementation section):
  - Git Branch / Commit Hash.
  - Changed Files List (must match scope whitelist).
  - Test Execution Summary (automated test commands, execution results).
  - Known Caveats and Technical Debt.
  - Regression Risks.

### 4.3 QA -> Human Lead Handoff
- Filed as QA Report in `docs/ai-team/tasks/TASK-XXX-qa-report.md`:
  - Overall Verdict: `APPROVED` / `REJECTED` / `ACTION_REQUIRED`.
  - Itemized Acceptance Table (`PASS`, `FAIL`, `NOT VERIFIED`).
  - Security Audit Summary (Secrets, SSRF, permissions, memory budget).
  - Discovered Defects (with reproduction steps if rejected).
  - Merge Recommendation.
