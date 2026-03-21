# Security Policy

## Supported Versions

Only the latest released version of DeepDiff DB receives security fixes. Older versions are not patched.

| Version | Supported |
|---------|-----------|
| Latest  | ✅ Yes    |
| Older   | ❌ No     |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub Issues.**

If you discover a security vulnerability, please report it privately so we can address it before public disclosure.

### How to Report

Use GitHub's private vulnerability reporting:

1. Go to the [Security Advisories](https://github.com/iamvirul/deepdiff-db/security/advisories/new) page
2. Click **"Report a vulnerability"**
3. Fill in the details — include steps to reproduce, impact, and suggested fix if known

Alternatively, email **virulnirmala@gmail.com** with the subject line `[SECURITY] DeepDiff DB — <brief description>`.

### What to Include

- A clear description of the vulnerability
- Steps to reproduce (config file, command, database setup)
- Potential impact (data exposure, privilege escalation, denial of service, etc.)
- Any suggested fix or mitigation

### Response Timeline

| Milestone | Target |
|-----------|--------|
| Acknowledgement | Within 48 hours |
| Severity assessment | Within 5 business days |
| Fix and release | Within 30 days for critical/high severity |
| Public disclosure | After fix is available (coordinated) |

If you do not receive a response within 48 hours, follow up via email.

---

## Security Considerations for Users

### Database Credentials

- Never commit your `deepdiffdb.yaml` config file if it contains plaintext credentials
- Use environment variable substitution: `password: ${PROD_DB_PASSWORD}`
- Add `deepdiffdb.yaml` to `.gitignore`

### Principle of Least Privilege

The production database user only needs **read access**. DeepDiff DB never writes to the production database during `diff` or `gen-pack` — only during `apply`.

Recommended grants for the prod read-only user:

```sql
-- MySQL
GRANT SELECT ON myapp.* TO 'deepdiff_ro'@'%';

-- PostgreSQL
GRANT CONNECT ON DATABASE myapp TO deepdiff_ro;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO deepdiff_ro;
```

### Migration Pack Review

Always review `migration_pack.sql` before running `apply`. The file contains DELETE and INSERT statements that modify production data. Use `--dry-run` first to validate syntax:

```bash
deepdiff-db apply --pack ./diff-output/migration_pack.sql --dry-run
```

### Network Security

Run DeepDiff DB from a host with direct, authenticated database access — not over the public internet. Use SSH tunnels or VPN for remote databases.

---

## Scope

Security issues in scope:

- Credential or secret exposure (e.g. credentials leaked to logs, error messages, or generated files)
- SQL injection via config values or column data affecting generated migration packs
- Path traversal in output directory or pack file path handling
- Denial of service via malformed config or database responses
- Privilege escalation via the `apply` command

Out of scope:

- Vulnerabilities in databases themselves (MySQL, PostgreSQL, etc.)
- Issues requiring local admin access to the machine running DeepDiff DB
- Social engineering attacks

---

## Disclosure Policy

We follow [coordinated vulnerability disclosure](https://en.wikipedia.org/wiki/Coordinated_vulnerability_disclosure). We will credit reporters in the release notes and security advisory unless they prefer to remain anonymous.
