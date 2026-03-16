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

- Cloudflare API tokens: [dash.cloudflare.com/profile/api-tokens](https://dash.cloudflare.com/profile/api-tokens) — scope to the specific zone, DNS edit only.
- Tailscale auth keys: [login.tailscale.com/admin/settings/keys](https://login.tailscale.com/admin/settings/keys) — use a reusable, non-ephemeral key so the container reconnects after restarts with the same identity.

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
docker build -t ts-cf-dns .

docker run \
  --cap-add NET_ADMIN \
  --env-file .env \
  -v tailscale-state:/var/lib/tailscale \
  ts-cf-dns
```

The volume persists Tailscale state across container restarts so the node keeps its identity.
