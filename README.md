# ts.cf.dns

A daemon written in Go, distributed as a Docker container, that syncs Tailscale hostnames to Cloudflare DNS.

## Configuration

All configuration is via environment variables. Copy `.env.example` to `.env` and fill in your values:

```sh
cp .env.example .env
```

| Variable | Required | Description |
|---|---|---|
| `CF_API_TOKEN` | Yes | Cloudflare API token with **Zone / DNS / Edit** permission |
| `CF_DOMAIN` | Yes | Domain name of the Cloudflare zone to manage |
| `TS_AUTHKEY` | Docker only | Tailscale auth key for connecting to your tailnet |
| `TS_HOSTNAME` | No | Hostname to register on the tailnet (default: `ts-cf-dns`) |
| `TS_EXCLUDE_TAGS` | No | Comma-separated Tailscale tags to exclude from DNS sync (e.g. `tag:server,tag:exit-node`) |

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

# Or build first
go build -o ts-cf-dns .
./ts-cf-dns
```

`TS_AUTHKEY` is not needed locally — the program connects directly to the local `tailscaled` socket (the Tailscale app on macOS satisfies this).

## Running with Docker

```sh
docker compose up --build
```

The volume persists Tailscale state across container restarts so the node keeps its identity. The container restarts automatically unless explicitly stopped.
