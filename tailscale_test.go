package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// setupTailscaleServer starts a test server mimicking the Tailscale API and
// redirects package-level HTTP calls to it via tsAPIBase.
func setupTailscaleServer(t *testing.T, devices []tsDevice) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/oauth/token":
			json.NewEncoder(w).Encode(tsOAuthResponse{AccessToken: "test-token"})
		case "/api/v2/tailnet/test-tailnet/devices":
			json.NewEncoder(w).Encode(tsDevicesResponse{Devices: devices})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	orig := tsAPIBase
	tsAPIBase = srv.URL
	t.Cleanup(func() { tsAPIBase = orig })

	t.Setenv("TS_OAUTH_CLIENT_ID", "test-id")
	t.Setenv("TS_OAUTH_CLIENT_SECRET", "test-secret")
	t.Setenv("TS_TAILNET", "test-tailnet")
}

func TestListTailscaleHosts_Basic(t *testing.T) {
	setupTailscaleServer(t, []tsDevice{
		{Name: "alice.example.ts.net.", Addresses: []string{"100.1.1.1", "fd7a::1"}},
		{Name: "bob.example.ts.net.", Addresses: []string{"100.2.2.2"}},
	})

	hosts, err := ListTailscaleHosts(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(hosts))
	}
	if hosts[0].Hostname != "alice" || hosts[0].IP != "100.1.1.1" {
		t.Errorf("unexpected host[0]: %+v", hosts[0])
	}
	if hosts[1].Hostname != "bob" || hosts[1].IP != "100.2.2.2" {
		t.Errorf("unexpected host[1]: %+v", hosts[1])
	}
}

func TestListTailscaleHosts_ExcludesByTag(t *testing.T) {
	setupTailscaleServer(t, []tsDevice{
		{Name: "alice.example.ts.net.", Addresses: []string{"100.1.1.1"}},
		{Name: "bob.example.ts.net.", Addresses: []string{"100.2.2.2"}, Tags: []string{"tag:server"}},
	})
	t.Setenv("TS_EXCLUDE_TAGS", "tag:server")

	hosts, err := ListTailscaleHosts(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 1 || hosts[0].Hostname != "alice" {
		t.Errorf("expected only alice, got %+v", hosts)
	}
}

func TestListTailscaleHosts_SkipsNoTailscaleIP(t *testing.T) {
	setupTailscaleServer(t, []tsDevice{
		{Name: "alice.example.ts.net.", Addresses: []string{"100.1.1.1"}},
		{Name: "noip.example.ts.net.", Addresses: []string{"fd7a::1"}}, // IPv6 only, no 100.x
	})

	hosts, err := ListTailscaleHosts(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 1 || hosts[0].Hostname != "alice" {
		t.Errorf("expected only alice, got %+v", hosts)
	}
}

func TestHasExcludedTag(t *testing.T) {
	excluded := map[string]struct{}{"tag:server": {}}
	if !hasExcludedTag([]string{"tag:server"}, excluded) {
		t.Error("expected true for matching tag")
	}
	if hasExcludedTag([]string{"tag:other"}, excluded) {
		t.Error("expected false for non-matching tag")
	}
	if hasExcludedTag(nil, excluded) {
		t.Error("expected false for nil tags")
	}
	if hasExcludedTag([]string{"tag:server"}, nil) {
		t.Error("expected false for nil excluded set")
	}
}
