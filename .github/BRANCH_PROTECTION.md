# Branch Protection Setup

This document explains how to configure branch protection rules to enforce PR checks before merging.

## Required Status Checks

The `pr-checks.yml` workflow provides a comprehensive set of checks that must pass before a PR can be merged.

### Setting Up Branch Protection

1. Go to your GitHub repository
2. Navigate to **Settings** → **Branches**
3. Click **Add rule** (or edit existing rule for `main`)
4. Configure the following:

#### Branch Name Pattern
```
main
```

#### Protection Rules to Enable

**✅ Require a pull request before merging**
- [x] Require approvals: 1
- [x] Dismiss stale pull request approvals when new commits are pushed
- [x] Require review from Code Owners (optional)

**✅ Require status checks to pass before merging**
- [x] Require branches to be up to date before merging

**Required Status Checks:**
- `Required Checks Summary` ← **Primary required check**
- `All Required Checks`
- `Validate PR Title`
- `Check for Merge Conflicts`
- `Unit Tests (1.23)`
- `Unit Tests (1.24)`
- `Integration Tests`
- `Lint`
- `Build`

**✅ Require conversation resolution before merging**
- [x] Enabled

**✅ Do not allow bypassing the above settings**
- [x] Enabled (recommended for production)
- [ ] Allow administrators to bypass (optional)

**✅ Restrict who can push to matching branches** (optional)
- Add users/teams who can push directly

## What Gets Checked

### 1. CI/CD Pipeline (`ci.yml`)
- **Unit Tests**: Runs on Go 1.23 and 1.24
- **Integration Tests**: Testcontainers-based integration tests
- **Coverage Merge**: Combines and uploads coverage to Codecov
- **Lint**: golangci-lint checks
- **Build**: Ensures binary builds successfully

### 2. PR Checks (`pr-checks.yml`)

#### All Required Checks
Waits for all CI jobs to complete successfully:
- Unit Tests (both Go versions)
- Integration Tests
- Lint
- Build

#### Validate PR Title
Ensures PR titles follow conventional commits format:
- Must start with type: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`
- Subject must start with uppercase letter
- Examples:
  - ✅ `feat: Add MODIFY COLUMN support`
  - ✅ `fix: Resolve varchar size issue`
  - ❌ `add modify column` (no type)
  - ❌ `feat: add modify column` (lowercase subject)

#### Check for Merge Conflicts
Automatically detects if the PR has conflicts with the base branch.

#### Ensure Tests Added (Warning Only)
Warns if Go files were modified but no tests were added.

#### Auto-Label
Automatically labels PRs based on changed files:
- `documentation` - Changes to .md files
- `tests` - Test file changes
- `core` - Internal package changes
- `samples` - Sample modifications
- `schema` - Schema-related changes
- etc.

### 3. Required Checks Summary
Final job that validates all required checks passed. This is the **main status check** to require in branch protection.

## Draft PRs

All checks are **skipped** for draft PRs. Convert to "Ready for review" to trigger checks.

## Workflow States

### ✅ All Checks Passed
```
✅ All required checks have passed!
🎉 This PR is ready to be merged!
```

### ❌ Checks Failed
```
❌ CI checks failed
- or -
❌ PR title validation failed
- or -
❌ Merge conflicts detected
```

## Bypassing Checks (Not Recommended)

If you need to bypass checks in emergency situations:

1. Ensure "Allow administrators to bypass" is enabled in branch protection
2. As an admin, you can merge despite failed checks
3. **Document the reason** in the merge commit message

## Testing the Workflow

1. Create a test branch:
   ```bash
   git checkout -b test/pr-checks
   ```

2. Make a change and push:
   ```bash
   echo "test" > test.txt
   git add test.txt
   git commit -m "test: Verify PR checks workflow"
   git push origin test/pr-checks
   ```

3. Open a PR and observe:
   - All checks should trigger
   - Auto-labels should be applied
   - Merge button should be blocked until checks pass

## Troubleshooting

### Checks Don't Run
- Ensure PR is not in draft mode
- Check workflow file syntax
- Verify GitHub Actions are enabled for the repository

### "Required Checks Summary" Not Showing
- Ensure the PR is targeting `main` or `development`
- Wait a few seconds after PR creation
- Check Actions tab for workflow runs

### False Positives on Merge Conflicts
- Rebase your branch on latest base branch:
  ```bash
  git fetch origin
  git rebase origin/main
  git push --force-with-lease
  ```

### PR Title Validation Fails
- Check conventional commits format
- Ensure subject starts with uppercase
- Valid types: feat, fix, docs, style, refactor, perf, test, build, ci, chore

## Best Practices

1. **Keep PRs Small**: Easier to review and test
2. **Write Tests**: Add tests for new features
3. **Update Documentation**: Keep README and docs in sync
4. **Follow Conventional Commits**: Makes changelog generation easier
5. **Resolve Conflicts Early**: Rebase frequently to avoid conflicts
6. **Run Checks Locally**: Use `go test ./...` and `golangci-lint run` before pushing

## Local Pre-commit Checks

Consider setting up a pre-commit hook:

```bash
#!/bin/bash
# .git/hooks/pre-commit

echo "Running pre-commit checks..."

# Run tests
if ! go test ./...; then
    echo "❌ Tests failed"
    exit 1
fi

# Run linter
if ! golangci-lint run --timeout=5m; then
    echo "❌ Linting failed"
    exit 1
fi

echo "✅ Pre-commit checks passed"
```

Make it executable:
```bash
chmod +x .git/hooks/pre-commit
```

## References

- [GitHub Branch Protection Rules](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/defining-the-mergeability-of-pull-requests/about-protected-branches)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [GitHub Actions Status Checks](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/collaborating-on-repositories-with-code-quality-features/about-status-checks)
