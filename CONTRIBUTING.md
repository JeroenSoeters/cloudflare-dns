# Contributing

This document covers local development for plugin authors. For user-facing
plugin docs (configuration, supported resources, examples), see
[README.md](README.md).

## Prerequisites

- Go 1.25+
- [Pkl CLI](https://pkl-lang.org/main/current/pkl-cli/index.html)
- Cloudflare API token and Zone ID for testing

## Local Installation

```bash
make install
```

## Building

```bash
make build      # Build plugin binary
make test       # Run all tests
make test-unit  # Run unit tests only
make lint       # Run linter
make install    # Build + install locally
```

## Local Testing

```bash
# Set credentials
export CLOUDFLARE_API_TOKEN="your-api-token"
export CLOUDFLARE_ZONE_ID="your-zone-id"

# Install plugin locally
make install

# Start formae agent
formae agent start

# Apply example resources
formae apply --mode reconcile --watch examples/basic/main.pkl
```

## Conformance Testing

Conformance tests validate the plugin's CRUD lifecycle:

| File | Purpose |
|------|---------|
| `testdata/resource.pkl` | Create A record |
| `testdata/resource-update.pkl` | Update content, TTL, add comment |
| `testdata/resource-replace.pkl` | Change name (triggers replacement) |

```bash
# Run conformance tests
make conformance-test

# Run with specific formae version
make conformance-test VERSION=0.80.0
```

The `scripts/ci/clean-environment.sh` script cleans up test DNS records with the `formae-` prefix.
