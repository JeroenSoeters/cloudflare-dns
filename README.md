# Cloudflare DNS Plugin for Formae

Formae plugin for managing Cloudflare DNS records. Supports all major record types including A, AAAA, CNAME, MX, TXT, NS, CAA, and SRV.

## Installation

```bash
# Install the plugin
make install
```

## Supported Resources

| Resource Type | Description |
|---------------|-------------|
| `CLOUDFLARE::DNS::Record` | Cloudflare DNS record (A, AAAA, CNAME, MX, TXT, NS, CAA, SRV) |

## Configuration

### Environment Variables

Set these environment variables before running Formae:

```bash
export CLOUDFLARE_API_TOKEN="your-api-token"
export CLOUDFLARE_ZONE_ID="your-zone-id"
```

To find your Zone ID:
1. Log in to the Cloudflare dashboard
2. Select your domain
3. The Zone ID is shown in the right sidebar under "API"

To create an API token:
1. Go to Cloudflare dashboard > My Profile > API Tokens
2. Create a token with "Zone > DNS > Edit" permissions

### Target Configuration

Configure a target in your Forma file:

```pkl
import "@cloudflare-dns/cloudflare-dns.pkl" as dns

new formae.Target {
    label = "cloudflare"
    namespace = "CLOUDFLARE"
    config = new dns.Config {
        api_token = read("env:CLOUDFLARE_API_TOKEN")
        zone_id = read("env:CLOUDFLARE_ZONE_ID")
    }
}
```

## Resource Fields

### DNSRecord

| Field | Type | Required | CreateOnly | Description |
|-------|------|----------|------------|-------------|
| `record_type` | String | Yes | Yes | Record type: A, AAAA, CNAME, MX, TXT, NS, CAA, SRV |
| `name` | String | Yes | Yes | DNS hostname (e.g., "www", "@" for root) |
| `content` | String | Yes | No | Record value (format varies by type) |
| `ttl` | Int | No | No | TTL in seconds (1 = automatic, default) |
| `proxied` | Boolean | No | No | Enable Cloudflare proxy (A/AAAA/CNAME only) |
| `priority` | Int | Conditional | No | Priority (required for MX and SRV) |
| `comment` | String | No | No | Optional note about the record |

### Content Format by Record Type

| Type | Content Format | Example | Priority |
|------|----------------|---------|----------|
| A | IPv4 address | `192.0.2.1` | No |
| AAAA | IPv6 address | `2001:db8::1` | No |
| CNAME | Target hostname | `target.example.com` | No |
| MX | Mail server hostname | `mail.example.com` | Yes (required) |
| TXT | Text value | `v=spf1 include:_spf.example.com ~all` | No |
| NS | Nameserver hostname | `ns1.example.com` | No |
| CAA | CAA record value | `0 issue "letsencrypt.org"` | No |
| SRV | weight port target | `5 5060 sipserver.example.com` | Yes (required) |

## Examples

### A Record (with Cloudflare proxy)

```pkl
new dns.DNSRecord {
    label = "web-server"
    record_type = "A"
    name = "www"
    content = "192.0.2.1"
    ttl = 300
    proxied = true
}
```

### MX Record

```pkl
new dns.DNSRecord {
    label = "mail-primary"
    record_type = "MX"
    name = "@"
    content = "mail.example.com"
    ttl = 3600
    priority = 10
}
```

### TXT Record (SPF)

```pkl
new dns.DNSRecord {
    label = "spf-record"
    record_type = "TXT"
    name = "@"
    content = "v=spf1 include:_spf.example.com ~all"
    ttl = 3600
}
```

### CAA Record

```pkl
new dns.DNSRecord {
    label = "caa-letsencrypt"
    record_type = "CAA"
    name = "@"
    content = "0 issue \"letsencrypt.org\""
    ttl = 3600
    comment = "Allow Let's Encrypt to issue certificates"
}
```

See the [examples/](examples/) directory for more complete examples.

## Development

### Prerequisites

- Go 1.25+
- [Pkl CLI](https://pkl-lang.org/main/current/pkl-cli/index.html)
- Cloudflare API token and Zone ID for testing

### Building

```bash
make build      # Build plugin binary
make test       # Run all tests
make test-unit  # Run unit tests only
make lint       # Run linter
make install    # Build + install locally
```

### Local Testing

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

### Conformance Testing

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

## Licensing

This plugin is licensed under BSD-3-Clause. See [LICENSE](LICENSE) for details.
