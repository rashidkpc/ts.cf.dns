# ts.cf.dns

A daemon written in Go, distributed as a Docker container, that syncs Tailscale hostnames to Cloudflare DNS.

## Prerequisites

- [Go 1.23+](https://go.dev/dl/)
- [Tailscale](https://tailscale.com/download) installed and connected to your tailnet

## Running locally

```sh
# Install dependencies
go mod tidy

# Run directly
go run .

# Or build and run
go build -o ts-cf-dns .
./ts-cf-dns
```

The program connects to the local `tailscaled` socket, so Tailscale must be running. On macOS, the Tailscale app satisfies this. On Linux, `tailscaled` must be running.

Set the required environment variables before running:

```sh
export CF_API_TOKEN=your-cloudflare-api-token
export CF_DOMAIN=example.com
```

## Running with Docker

Build the image:

```sh
docker build -t ts-cf-dns .
```

Run the container:

```sh
docker run \
  --cap-add NET_ADMIN \
  -e TS_AUTHKEY=tskey-auth-... \
  -v tailscale-state:/var/lib/tailscale \
  ts-cf-dns
```

| Environment variable | Description |
|---|---|
| `TS_AUTHKEY` | Tailscale auth key for connecting to your tailnet |
| `TS_HOSTNAME` | Hostname to register on the tailnet (default: `ts-cf-dns`) |
| `CF_API_TOKEN` | Cloudflare API token with DNS read/write permissions |
| `CF_DOMAIN` | Domain name of the Cloudflare zone to manage |

Auth keys can be generated at [login.tailscale.com/admin/settings/keys](https://login.tailscale.com/admin/settings/keys). Use a reusable, non-ephemeral key so the container reconnects after restarts with the same identity.
