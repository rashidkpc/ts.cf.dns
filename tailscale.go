package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"tailscale.com/client/tailscale"
	"tailscale.com/types/views"
)

// dnsLabel extracts the first label from a fully-qualified DNS name.
// e.g. "my-device.tailnet-name.ts.net." -> "my-device"
func dnsLabel(dnsName string) string {
	name := strings.TrimSuffix(dnsName, ".")
	if i := strings.IndexByte(name, '.'); i != -1 {
		return name[:i]
	}
	return name
}

// TailscaleHost holds the hostname and primary IP of a Tailscale peer.
type TailscaleHost struct {
	Hostname string
	IP       string
}

// excludeTags returns the set of tags from the TS_EXCLUDE_TAGS env var.
func excludeTags() map[string]struct{} {
	val := os.Getenv("TS_EXCLUDE_TAGS")
	if val == "" {
		return nil
	}
	tags := make(map[string]struct{})
	for _, t := range strings.Split(val, ",") {
		tags[strings.TrimSpace(t)] = struct{}{}
	}
	return tags
}

// hasExcludedTag reports whether any of the node's tags appear in the excluded set.
func hasExcludedTag(tags *views.Slice[string], excluded map[string]struct{}) bool {
	if len(excluded) == 0 || tags == nil {
		return false
	}
	for _, tag := range tags.AsSlice() {
		if _, ok := excluded[tag]; ok {
			return true
		}
	}
	return false
}

// ListTailscaleHosts returns all peers on the tailnet including the local node,
// excluding any nodes that carry a tag listed in TS_EXCLUDE_TAGS.
func ListTailscaleHosts(ctx context.Context) ([]TailscaleHost, error) {
	var lc tailscale.LocalClient

	status, err := lc.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("tailscale status: %w", err)
	}

	excluded := excludeTags()

	var hosts []TailscaleHost

	if len(status.Self.TailscaleIPs) > 0 && !hasExcludedTag(status.Self.Tags, excluded) {
		hosts = append(hosts, TailscaleHost{
			Hostname: dnsLabel(status.Self.DNSName),
			IP:       status.Self.TailscaleIPs[0].String(),
		})
	}

	for _, peer := range status.Peer {
		if len(peer.TailscaleIPs) > 0 && !hasExcludedTag(peer.Tags, excluded) {
			hosts = append(hosts, TailscaleHost{
				Hostname: dnsLabel(peer.DNSName),
				IP:       peer.TailscaleIPs[0].String(),
			})
		}
	}

	return hosts, nil
}
