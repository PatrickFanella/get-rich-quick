---
title: "Agent Execution Guide"
description: "Legacy execution guidance retained for historical context."
status: "archive"
updated: "2026-04-03"
tags: [agents, archive]
---

# Agent Execution Guide

This guide defines how autonomous coding agents should claim, execute, and complete issue backlog work.

## Workflow states

Agents move issues through this lifecycle:

1. **Ready** - Issue is clearly scoped and available to pick up.
2. **In Progress** - An agent has claimed and is actively working the issue.
3. **In Review** - Work is complete and a PR is open for review.
4. **Done** - PR is merged and acceptance criteria are satisfied.

## 1) Issue pickup conventions

Before starting work, an agent should confirm:

- The issue is in **Ready**.
- The issue has a valid `type:*` label.
- The issue has priority and phase labels when applicable.

When claiming an issue, the agent should:

- Assign itself (or add a comment indicating ownership if assignment is unavailable).
- Move the issue to **In Progress**.
- Add/confirm `workflow:agent-ready` was present at pickup time.

## 2) Signaling work in progress

To prevent duplicate effort, the agent must immediately signal WIP by:

- Setting issue status to **In Progress**.
- Posting a short claim comment that includes:
  - Start timestamp (UTC)
  - Planned scope summary
  - Expected output (for example, docs update or code + tests)

If no visible progress is made after a reasonable window, the issue should be unclaimed and returned to **Ready**.

## 3) Branch naming convention

Agents must create a dedicated branch for each issue:

`agent/{issue-number}-{slug}`

Examples:

- `agent/42-define-agent-execution-conventions`
- `agent/118-fix-order-retry-timeout`

Rules:

- Use lowercase kebab-case for `{slug}`.
- Keep slug short and descriptive.
- One issue per branch.

## 4) Pull request conventions

When implementation is complete, the agent should:

1. Open a PR from the issue branch to the default branch.
2. Reference the issue number in the PR body (for example, `Closes #42`).
3. Move the issue to **In Review**.
4. Summarize:
   - What changed
   - Why it changed
   - How it was validated
   - Any follow-up work

PR title format (recommended):

`[#<issue-number>] <short summary>`

## 5) Blocked issues

If blocked by dependency, missing decision, missing access, or failing external system:

1. Move issue to blocked state (for example, keep **In Progress** + add `workflow:blocked`).
2. Add a blocker comment with:
   - Blocker type
   - Exact unblock condition
   - Owner (if known)
   - Last successful step / evidence
3. If blocked longer than one business day, unassign and hand back to backlog triage.

## 6) Failure and partial completion reporting

When work cannot be completed as requested, the agent must still provide a clear handoff:

- What was attempted
- What failed (with logs/errors)
- What remains
- Suggested next action

If partial progress is valid:

- Open a PR marked as partial scope.
- Reference the issue and explicitly list completed vs remaining acceptance criteria.

## 7) Definition of Done (agent-completed work)

An issue is **Done** only when all apply:

- [ ] Acceptance criteria are met
- [ ] Required docs are updated
- [ ] Required tests are added/updated for changed behavior
- [ ] Relevant checks pass (or documented exception is approved)
- [ ] PR references issue number
- [ ] Reviewer feedback is addressed
- [ ] Issue moved to **Done**

