package main

import (
	"context"
	"fmt"

	"tailscale.com/client/tailscale"
)

// TailscaleHost holds the hostname and primary IP of a Tailscale peer.
type TailscaleHost struct {
	Hostname string
	IP       string
}

// ListTailscaleHosts returns all peers on the tailnet including the local node.
func ListTailscaleHosts(ctx context.Context) ([]TailscaleHost, error) {
	var lc tailscale.LocalClient

	status, err := lc.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("tailscale status: %w", err)
	}

	var hosts []TailscaleHost

	if len(status.Self.TailscaleIPs) > 0 {
		hosts = append(hosts, TailscaleHost{
			Hostname: status.Self.HostName,
			IP:       status.Self.TailscaleIPs[0].String(),
		})
	}

	for _, peer := range status.Peer {
		if len(peer.TailscaleIPs) > 0 {
			hosts = append(hosts, TailscaleHost{
				Hostname: peer.HostName,
				IP:       peer.TailscaleIPs[0].String(),
			})
		}
	}

	return hosts, nil
}
