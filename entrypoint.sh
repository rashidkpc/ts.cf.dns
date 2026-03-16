#!/bin/sh
set -e

mkdir -p /var/run/tailscale /var/lib/tailscale

# Start tailscaled in the background
tailscaled \
  --state=/var/lib/tailscale/tailscaled.state \
  --socket=/var/run/tailscale/tailscaled.sock \
  &
TAILSCALED_PID=$!

# Wait for tailscaled to be ready
echo "Waiting for tailscaled to start..."
until tailscale status > /dev/null 2>&1; do
  sleep 1
done
echo "tailscaled is ready."

# Connect to tailnet using the provided auth key
if [ -n "${TS_AUTHKEY}" ]; then
  tailscale up \
    --authkey="${TS_AUTHKEY}" \
    --hostname="${TS_HOSTNAME:-ts-cf-dns}" \
    --accept-routes
  echo "Connected to tailnet as ${TS_HOSTNAME:-ts-cf-dns}."
else
  echo "Warning: TS_AUTHKEY is not set. Tailscale will not connect to the tailnet."
fi

# Run the main daemon if provided, otherwise keep tailscaled alive
if [ $# -gt 0 ]; then
  exec "$@"
else
  wait $TAILSCALED_PID
fi
