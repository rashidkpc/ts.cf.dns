# ts.cf.dns

A daemon written in Go, distributed as a Docker container, that syncs Tailscale hostnames to Cloudflare DNS.

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
| `TS_AUTHKEY` | Docker only | Tailscale auth key for connecting to your tailnet |
| `TS_HOSTNAME` | No | Hostname to register on the tailnet (default: `ts-cf-dns`) |
| `TS_EXCLUDE_TAGS` | No | Comma-separated Tailscale ACL tags to exclude from sync (e.g. `tag:server,tag:exit-node`) |

### Cloudflare API token

1. Go to [dash.cloudflare.com/profile/api-tokens](https://dash.cloudflare.com/profile/api-tokens) and click **Create Token**.
2. Use the **Edit zone DNS** template, or create a custom token with:
   - **Permissions:** Zone / DNS / Edit
   - **Zone Resources:** Include / Specific zone / `<your domain>`
3. Copy the generated token into `CF_API_TOKEN`.

Scoping the token to a single zone limits exposure if the secret leaks.

### Tailscale auth key

1. Go to [login.tailscale.com/admin/settings/keys](https://login.tailscale.com/admin/settings/keys) and click **Generate auth key**.
2. Enable **Reusable** so the container can reconnect after restarts without generating a new key.
3. Leave **Ephemeral** unchecked so the node persists in your tailnet between restarts.
4. Copy the key (shown once) into `TS_AUTHKEY`.

## Running locally

**Prerequisites:** [Go 1.23+](https://go.dev/dl/) and [Tailscale](https://tailscale.com/download) running.

```sh
go mod tidy
source .env
go run .
```

`TS_AUTHKEY` is not needed locally — the program connects directly to the local `tailscaled` socket (the Tailscale app on macOS satisfies this).

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

The tests are unit tests with no external dependencies — no Tailscale daemon or Cloudflare credentials required.

## Running with Docker

```sh
docker compose up --build
```

The volume persists Tailscale state across container restarts so the node keeps its identity. The container restarts automatically unless explicitly stopped.

## How it works

1. Every 30 seconds, the daemon fetches the current Tailscale peer list (including the local node) via the local Tailscale socket.
2. It lists Cloudflare A records under `recordBase()` that carry the `managed-by:ts.cf.dns` comment.
3. It reconciles the two sets:
   - **Missing from Cloudflare** → create A record (TTL 60, not proxied)
   - **IP changed** → update the existing record
   - **No longer in Tailscale** → delete the record
4. Unmanaged records (no comment, or a different comment) are never modified.
