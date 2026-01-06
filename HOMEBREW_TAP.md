# Homebrew Tap Guide

This document explains how to use and maintain the DeepDiff DB Homebrew tap.

## For Users

### Installing from the Homebrew Tap

Once the tap is published, users can install deepdiff-db using:

```bash
# Tap the repository
brew tap iamvirul/deepdiff-db

# Install deepdiff-db
brew install deepdiff-db
```

Or install in one command:

```bash
brew install iamvirul/deepdiff-db/deepdiff-db
```

### Installing from HEAD

To install the latest development version:

```bash
brew install --HEAD iamvirul/deepdiff-db/deepdiff-db
```

### Upgrading

To upgrade to the latest version:

```bash
brew upgrade deepdiff-db
```

### Uninstalling

```bash
brew uninstall deepdiff-db
brew untap iamvirul/deepdiff-db
```

## For Maintainers

### Setting Up the Tap Repository

The Homebrew tap is integrated directly into this repository. The Formula directory contains the formula file that Homebrew will use.

### Repository Structure

```
deepdiff-db/
├── Formula/
│   └── deepdiff-db.rb    # Homebrew formula
├── HOMEBREW_TAP.md        # This documentation
└── README.md              # Updated with Homebrew installation instructions
```

### Updating the Formula for New Releases

When releasing a new version, follow these steps:

1. **Create and push a new Git tag:**
   ```bash
   git tag v0.7
   git push origin v0.7
   ```

2. **Calculate the SHA256 for the release tarball:**
   ```bash
   # Download the release tarball
   wget https://github.com/iamvirul/deepdiff-db/archive/refs/tags/v0.7.tar.gz

   # Calculate SHA256
   shasum -a 256 v0.7.tar.gz
   ```

3. **Update the formula file (`Formula/deepdiff-db.rb`):**
   - Update the `url` field with the new version
   - Update the `sha256` field with the calculated hash
   - Example:
     ```ruby
     url "https://github.com/iamvirul/deepdiff-db/archive/refs/tags/v0.7.tar.gz"
     sha256 "abc123..." # Replace with actual hash
     ```

4. **Test the formula locally:**
   ```bash
   # Audit the formula for issues
   brew audit --strict Formula/deepdiff-db.rb

   # Test installation
   brew install --build-from-source Formula/deepdiff-db.rb

   # Run formula tests
   brew test deepdiff-db

   # Uninstall after testing
   brew uninstall deepdiff-db
   ```

5. **Commit and push the updated formula:**
   ```bash
   git add Formula/deepdiff-db.rb
   git commit -m "chore(homebrew): Update formula to v0.7"
   git push origin main
   ```

### Automated SHA256 Calculation

You can automate SHA256 calculation using this command:

```bash
VERSION=v0.7
curl -L "https://github.com/iamvirul/deepdiff-db/archive/refs/tags/${VERSION}.tar.gz" | shasum -a 256
```

### Formula Testing

Before pushing formula updates, always test:

```bash
# Validate syntax and style
brew audit --strict --online Formula/deepdiff-db.rb

# Test installation from source
brew install --build-from-source --verbose Formula/deepdiff-db.rb

# Test the installed binary
deepdiffdb --help

# Run automated tests
brew test deepdiff-db
```

### Common Issues and Solutions

**Issue: SHA256 mismatch**
- Solution: Recalculate the hash from the actual release tarball
- GitHub may regenerate tarballs, so always download and verify

**Issue: Build failures**
- Solution: Ensure Go version compatibility in the formula
- Test with `brew install --build-from-source --verbose`

**Issue: Formula audit warnings**
- Solution: Run `brew audit --strict` and fix reported issues
- Follow Homebrew's formula style guidelines

### Automated Formula Updates (GitHub Actions)

This repository includes a GitHub Action that **automatically updates the formula** when new releases are published.

**Workflow file:** `.github/workflows/update-homebrew-formula.yml`

**Triggers:**
- **Automatic:** Runs automatically when a new GitHub release is published
- **Manual:** Can be triggered manually via workflow_dispatch for specific versions

**What it does:**
1. Detects the new release version
2. Downloads the release tarball
3. Calculates the SHA256 checksum
4. Updates `Formula/deepdiff-db.rb` with new version and checksum
5. Commits and pushes changes to the main branch
6. Adds a summary comment to the release

**Manual trigger example:**
```bash
# Via GitHub CLI
gh workflow run update-homebrew-formula.yml -f version=v0.7

# Or use the GitHub Actions tab in the repository
```

**Note:** With this automation in place, you typically don't need to manually update the formula unless the automation fails or you need to make other changes (dependencies, build flags, etc.).

### Support and Documentation

- Users should refer to the main README.md for installation instructions
- Report formula issues to the GitHub Issues page
- Follow Homebrew's [Formula Cookbook](https://docs.brew.sh/Formula-Cookbook) for best practices

## Additional Resources

- [Homebrew Documentation](https://docs.brew.sh/)
- [Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
- [Acceptable Formulae](https://docs.brew.sh/Acceptable-Formulae)
- [How to Create and Maintain a Tap](https://docs.brew.sh/How-to-Create-and-Maintain-a-Tap)
