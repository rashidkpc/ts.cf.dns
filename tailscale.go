package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
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

// hasExcludedTag reports whether any of the tags appear in the excluded set.
func hasExcludedTag(tags []string, excluded map[string]struct{}) bool {
	if len(excluded) == 0 {
		return false
	}
	for _, tag := range tags {
		if _, ok := excluded[tag]; ok {
			return true
		}
	}
	return false
}

// tsAPIBase is the base URL for the Tailscale API. Overridden in tests.
var tsAPIBase = "https://api.tailscale.com"

type tsOAuthResponse struct {
	AccessToken string `json:"access_token"`
}

type tsDevice struct {
	Name      string   `json:"name"`
	Addresses []string `json:"addresses"`
	Tags      []string `json:"tags"`
}

type tsDevicesResponse struct {
	Devices []tsDevice `json:"devices"`
}

// tailscaleToken fetches a short-lived OAuth access token using client credentials.
func tailscaleToken(ctx context.Context) (string, error) {
	clientID := os.Getenv("TS_OAUTH_CLIENT_ID")
	clientSecret := os.Getenv("TS_OAUTH_CLIENT_SECRET")
	if clientID == "" {
		return "", fmt.Errorf("TS_OAUTH_CLIENT_ID is not set")
	}
	if clientSecret == "" {
		return "", fmt.Errorf("TS_OAUTH_CLIENT_SECRET is not set")
	}

	body := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"grant_type":    {"client_credentials"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		tsAPIBase+"/api/v2/oauth/token",
		strings.NewReader(body.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("oauth token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("oauth token: status %d: %s", resp.StatusCode, b)
	}

	var tok tsOAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", fmt.Errorf("oauth token decode: %w", err)
	}
	return tok.AccessToken, nil
}

// ListTailscaleHosts returns all devices on the tailnet, excluding any that
// carry a tag listed in TS_EXCLUDE_TAGS.
func ListTailscaleHosts(ctx context.Context) ([]TailscaleHost, error) {
	token, err := tailscaleToken(ctx)
	if err != nil {
		return nil, err
	}

	tailnet := os.Getenv("TS_TAILNET")
	if tailnet == "" {
		tailnet = "-"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		tsAPIBase+"/api/v2/tailnet/"+url.PathEscape(tailnet)+"/devices",
		nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list devices: status %d: %s", resp.StatusCode, b)
	}

	var result tsDevicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("list devices decode: %w", err)
	}

	excluded := excludeTags()
	var hosts []TailscaleHost
	for _, d := range result.Devices {
		if hasExcludedTag(d.Tags, excluded) {
			continue
		}
		var ip string
		for _, addr := range d.Addresses {
			if strings.HasPrefix(addr, "100.") {
				ip = addr
				break
			}
		}
		if ip == "" {
			continue
		}
		hosts = append(hosts, TailscaleHost{
			Hostname: dnsLabel(d.Name),
			IP:       ip,
		})
	}
	return hosts, nil
}
