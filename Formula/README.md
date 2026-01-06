# Homebrew Formula Directory

This directory contains the Homebrew formula for DeepDiff DB.

## Quick Start (For Users)

```bash
brew tap iamvirul/deepdiff-db
brew install deepdiff-db
```

## Formula File

- **deepdiff-db.rb** - The Homebrew formula that defines how to build and install deepdiff-db

## For Maintainers

See [HOMEBREW_TAP.md](../HOMEBREW_TAP.md) in the root directory for detailed instructions on:

- Updating the formula for new releases
- Testing the formula locally
- Calculating SHA256 checksums
- CI/CD integration options

## Testing the Formula Locally

Before pushing changes, always test the formula:

```bash
# Audit the formula
brew audit --strict Formula/deepdiff-db.rb

# Install from source
brew install --build-from-source Formula/deepdiff-db.rb

# Run tests
brew test deepdiff-db
```

## Resources

- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
- [How to Create and Maintain a Tap](https://docs.brew.sh/How-to-Create-and-Maintain-a-Tap)
