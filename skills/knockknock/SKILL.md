---
name: knockknock
description: Detect whether a website or HTTP response is protected by an antibot, WAF, or CAPTCHA (Cloudflare, DataDome, Akamai, Kasada, AWS WAF, etc.) using the knockknock CLI. Use when profiling scraping targets, diagnosing 403/429 blocks, or analyzing bot defenses.
---

# knockknock

Fast CLI to identify 20+ antibot, WAF, and CAPTCHA providers from HTTP responses.

## Key Rules for AI Agents

1. **Always use `--json`** for machine-readable output.
2. **Exit codes**:
   - `0` = Protection detected (`"protected": true`).
   - `1` = Clean / not protected (`"protected": false`). **Not a tool error.**
   - `2` = Command or network error.
   *If your environment fails on non-zero exit codes, append `|| true`:*
   ```bash
   knockknock --json https://example.com || true
   ```

## Workflows

### 1. Profile a Website
```bash
knockknock --json https://example.com || true
```
Output:
```json
{"protected": true, "vendors": ["cloudflare"], "mode": "presence", "source": "https://example.com"}
```

### 2. Check for Active Challenge vs Passive Presence
- Default (presence): detects passive signatures even on 200 OK.
- `-c` / `--challenge`: only triggers if an active block or challenge is served.
```bash
knockknock --json -c https://example.com || true
```

### 3. Diagnose TLS Fingerprinting
Compare browser TLS profile (default `chrome_133`) against standard Go/curl TLS:
```bash
knockknock --json https://example.com || true
knockknock --json --naive https://example.com || true
```
If `--naive` gets challenged while default succeeds, the site filters by TLS/JA3/JA4 fingerprint.

### 4. Inspect Raw HTTP Responses via Stdin
Pipe captured headers and body from curl or proxy logs:
```bash
curl -isS -L https://example.com | knockknock --json
```

## Options Reference

- `--json`: machine-readable output.
- `-c, --challenge`: report active challenge/block signals only.
- `-n, --naive`: use standard Go TLS fingerprint instead of browser-like TLS.
- `-p, --profile <P>`: TLS profile (default `chrome_133`).
- `--timeout <D>`: timeout (default `30s`).
- `--max-body <N>`: max body bytes to scan (default `64KiB`).

## Vendor Quick Reference

- **Cloudflare**: Passive presence (`cf-ray`) or active challenge (`cf-mitigated`, Turnstile).
- **DataDome**: Heavy behavioral JS profiling. Needs browser automation or residential IP.
- **Kasada**: Polyglot bytecode VM script. Standard HTTP clients fail.
- **PerimeterX / HUMAN**: Sensor script (`_px3`). Needs browser environment.
- **Akamai / AWS WAF**: IP reputation and token challenges. Verify headers / rotate IP.
