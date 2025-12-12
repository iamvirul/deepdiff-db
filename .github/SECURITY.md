# Security Policy

## Supported Versions

We release patches for security vulnerabilities. Which versions are eligible for receiving security updates depends on the CVSS v3.0 Rating:

| Version | Supported          |
| ------- | ------------------ |
| Latest  | :white_check_mark: |
| < Latest | :x:                |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them via one of the following methods:

- **Email**: [Your security email or create a security policy]
- **Private Security Advisory**: Use GitHub's private vulnerability reporting feature

Please include the following information in your report:

- Type of issue (e.g., buffer overflow, SQL injection, etc.)
- Full paths of source file(s) related to the manifestation of the issue
- The location of the affected source code (tag/branch/commit or direct URL)
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit the issue

This information will help us triage your report more quickly.

## Security Best Practices

When using DeepDiff DB:

- **Never commit** database credentials or config files with passwords to version control
- Use environment variables or secure secret management for sensitive data
- Regularly update to the latest version
- Review migration packs before applying them to production
- Use `--dry-run` flag to validate SQL before execution
- Always backup your database before applying migrations

## Disclosure Policy

When we receive a security bug report, we will:

1. Confirm the issue and determine affected versions
2. Audit code to find any potential similar problems
3. Prepare fixes for all releases still under support
4. Release a security update as soon as possible

We credit security researchers who responsibly disclose vulnerabilities.

