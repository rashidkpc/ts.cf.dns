# ts.cf.dns

Some vibe coded Go trash, distributed as a Docker container, that syncs Tailscale hostnames to Cloudflare DNS.

Every 30 seconds it reconciles a Tailscale peer list against an A-record set in Cloudflare — creating, updating, and deleting records as peers come and go. Cloudflare records it creates are commented with `managed-by:ts.cf.dns`. Only records with the `managed-by:ts.cf.dns` comment are ever touched; manually created records under the same subdomain are left alone.

This is intended to be run in Docker, and a docker compose configuration is included.

## Configuration

All configuration is via environment variables. Copy `.env.example` to `.env` and fill in your values:

```sh
cp .env.example .env
```

| Variable | Required | Description |
|---|---|---|
| `CF_API_TOKEN` | Yes | Cloudflare API token with **Zone / DNS / Edit** permission |
| `CF_DOMAIN` | Yes | Domain name of the Cloudflare zone to manage |
| `CF_SUBDOMAIN` | No | Subdomain prefix for DNS records (e.g. `ts` → `hostname.ts.example.com`; omit to place records at the zone apex) |
| `TS_OAUTH_CLIENT_ID` | Yes | Tailscale OAuth client ID |
| `TS_OAUTH_CLIENT_SECRET` | Yes | Tailscale OAuth client secret |
| `TS_TAILNET` | No | Tailnet name (default: `-`, meaning the tailnet of the authenticated client) |
| `TS_EXCLUDE_TAGS` | No | Comma-separated Tailscale ACL tags to exclude from sync (e.g. `tag:server,tag:exit-node`) |

### Cloudflare API token

1. Go to [dash.cloudflare.com/profile/api-tokens](https://dash.cloudflare.com/profile/api-tokens) and click **Create Token**.
2. Use the **Edit zone DNS** template, or create a custom token with:
   - **Permissions:** Zone / DNS / Edit
   - **Zone Resources:** Include / Specific zone / `<your domain>`
3. Copy the generated token into `CF_API_TOKEN`.

Scoping the token to a single zone limits exposure if the secret leaks.

### Tailscale OAuth client

1. Go to [login.tailscale.com/admin/settings/oauth](https://login.tailscale.com/admin/settings/oauth) and click **Generate OAuth client**.
2. Grant the **Devices: Read** scope — no other permissions are needed.
3. Copy the client ID and secret into `TS_OAUTH_CLIENT_ID` and `TS_OAUTH_CLIENT_SECRET`.

OAuth client secrets do not expire. A fresh access token is fetched automatically on every sync cycle.

## Running locally

**Prerequisites:** [Go 1.23+](https://go.dev/dl/)

```sh
go mod tidy
source .env
go run .
```

### Dry run

Preview what the daemon would create, update, or delete without making any changes:

```sh
source .env && go run . --dry-run
```

Output looks like:

```
2025/01/01 12:00:00 [dry-run] would create alice.ts.example.com → 100.64.0.1
2025/01/01 12:00:00 [dry-run] would update bob.ts.example.com: 100.64.0.2 → 100.64.0.3
2025/01/01 12:00:00 [dry-run] would delete ghost.ts.example.com
```

### Build and run

```sh
go build -o ts-cf-dns .
./ts-cf-dns
# or with dry-run
./ts-cf-dns --dry-run
```

## Running the tests

```sh
go test ./...
```

The tests are unit tests with no external dependencies — no Tailscale credentials or Cloudflare credentials required.

## Running with Docker

```sh
docker compose up --build
```

The container restarts automatically unless explicitly stopped.

## How it works

1. Every 30 seconds, the daemon fetches a fresh OAuth access token and calls the Tailscale API to list all devices on the tailnet.
2. It lists Cloudflare A records under the configured base domain that carry the `managed-by:ts.cf.dns` comment.
3. It reconciles the two sets:
   - **Missing from Cloudflare** → create A record (TTL 60, not proxied)
   - **IP changed** → update the existing record
   - **No longer in Tailscale** → delete the record
4. Unmanaged records (no comment, or a different comment) are never modified.
