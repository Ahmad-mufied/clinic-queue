# Rule: Git Workflow & Commit Guidelines
**Scope:** Entire Repository / All Agents  
**Status:** Active & Enforced

---

## 1. Strict Git Execution Rules

> [!CAUTION]
> **NEVER execute `git commit` or `git push` automatically under any circumstances.**

1. **User Ownership:** The USER retains full manual control over all Git commits and repository history.
2. **Agent Responsibility:** Agents must NEVER run `git commit`, `git merge`, or `git push`.
3. **Commit Proposals:** Upon completing a milestone, plan phase, or feature, the agent must propose formatted **Conventional Commit Messages** for the user to review, copy, and execute manually.

---

## 2. Commit Message Standard (Conventional Commits)

Format: `<type>(<scope>): <short description>`

### Commit Types:
- `feat`: A new feature or domain capability.
- `fix`: A bug fix or error correction.
- `docs`: Documentation updates, PRD, technical specs, or implementation plans.
- `test`: Adding or refactoring unit, integration, or E2E tests.
- `refactor`: Code restructuring without functional changes.
- `chore`: Tooling, configs, `.gitignore`, Docker setups, dependencies.
- `arch`: Architectural design updates, ports, adapters restructuring.
