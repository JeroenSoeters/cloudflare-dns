# Cloudflare DNS Plugin for Formae

A Formae plugin for managing Cloudflare DNS records. This plugin enables declarative management of DNS records within a Cloudflare zone.

## Installation

```bash
# Install the plugin
make install
```

## Supported Resources

| Resource Type | Description |
|---------------|-------------|
| `CLOUDFLARE::DNS::Record` | DNS record (A, AAAA, CNAME, MX, TXT, NS) |

## Configuration

### Target Configuration

Configure a target in your Forma file:

```pkl
new formae.Target {
    label = "cloudflare"
    namespace = "CLOUDFLARE"
    config = new Mapping {
        ["zone_id"] = read("env:CLOUDFLARE_ZONE_ID")
    }
}
```

### Credentials

Set the `CLOUDFLARE_API_TOKEN` environment variable with your Cloudflare API token:

```bash
export CLOUDFLARE_API_TOKEN="your-api-token"
```

The API token needs the following permissions:
- Zone.DNS: Edit

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `CLOUDFLARE_API_TOKEN` | Yes | Cloudflare API token for authentication |
| `CLOUDFLARE_ZONE_ID` | Yes | Zone ID for the target DNS zone |

## Resource: CLOUDFLARE::DNS::Record

### Properties

| Property | Type | Required | CreateOnly | Description |
|----------|------|----------|------------|-------------|
| `zone_id` | String | Yes | Yes | The zone ID where this record belongs |
| `name` | String | Yes | Yes | DNS record name (e.g., "example.com" or "sub.example.com") |
| `record_type` | String | Yes | Yes | Record type: A, AAAA, CNAME, MX, TXT, NS |
| `content` | String | Yes | No | Record content (IP address, hostname, or text) |
| `ttl` | Int | No | No | TTL in seconds (1 = automatic, 60-86400 for custom) |
| `proxied` | Boolean | No | No | Whether to proxy through Cloudflare (A, AAAA, CNAME only) |
| `comment` | String | No | No | Optional comment for the record |
| `priority` | Int | No | No | Priority for MX records |

### Example

```pkl
import "@cloudflare-dns/cloudflare-dns.pkl" as dns

new dns.Record {
    label = "my-record"
    zone_id = read("env:CLOUDFLARE_ZONE_ID")
    name = "myapp"
    record_type = "A"
    content = "203.0.113.50"
    ttl = 300
    proxied = true
    comment = "My application server"
}
```

## Examples

See the [examples/](examples/) directory for usage examples.

```bash
# Evaluate an example
formae eval examples/basic/main.pkl

# Apply resources
formae apply --mode reconcile --watch examples/basic/main.pkl
```

## Development

### Prerequisites

- Go 1.25+
- [Pkl CLI](https://pkl-lang.org/main/current/pkl-cli/index.html)
- Cloudflare account with API token

### Building

```bash
make build      # Build plugin binary
make test       # Run unit tests
make lint       # Run linter
make install    # Build + install locally
```

### Local Testing

```bash
# Install plugin locally
make install

# Set credentials
export CLOUDFLARE_API_TOKEN="your-api-token"
export CLOUDFLARE_ZONE_ID="your-zone-id"

# Start formae agent
formae agent start

# Apply example resources
formae apply --mode reconcile --watch examples/basic/main.pkl
```

### Conformance Testing

Conformance tests validate the plugin's CRUD lifecycle using test fixtures in `testdata/`:

| File | Purpose |
|------|---------|
| `resource.pkl` | Initial resource creation |
| `resource-update.pkl` | In-place update (mutable fields) |
| `resource-replace.pkl` | Replacement (createOnly fields) |

```bash
# Set credentials
export CLOUDFLARE_API_TOKEN="your-api-token"
export CLOUDFLARE_ZONE_ID="your-zone-id"

# Run conformance tests
make conformance-test                  # Latest formae version
make conformance-test VERSION=0.80.0   # Specific version
```

## Licensing

This plugin is licensed under the BSD-3-Clause license. See [LICENSE](LICENSE) for details.
