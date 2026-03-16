package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudflare/cloudflare-go"
)

// DNSRecord holds the type, name, and value of a Cloudflare DNS record.
type DNSRecord struct {
	Type    string
	Name    string
	Content string
}

// ListCloudflareDNSRecords returns all DNS records for the zone specified by
// the CF_DOMAIN and CF_API_TOKEN environment variables.
func ListCloudflareDNSRecords(ctx context.Context) ([]DNSRecord, error) {
	token := os.Getenv("CF_API_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("CF_API_TOKEN is not set")
	}
	domain := os.Getenv("CF_DOMAIN")
	if domain == "" {
		return nil, fmt.Errorf("CF_DOMAIN is not set")
	}

	api, err := cloudflare.NewWithAPIToken(token)
	if err != nil {
		return nil, fmt.Errorf("cloudflare client: %w", err)
	}

	zones, err := api.ListZones(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("list zones: %w", err)
	}
	if len(zones) == 0 {
		return nil, fmt.Errorf("no zone found for domain %q", domain)
	}

	zoneID := cloudflare.ZoneIdentifier(zones[0].ID)
	raw, _, err := api.ListDNSRecords(ctx, zoneID, cloudflare.ListDNSRecordsParams{})
	if err != nil {
		return nil, fmt.Errorf("list dns records: %w", err)
	}

	records := make([]DNSRecord, len(raw))
	for i, r := range raw {
		records[i] = DNSRecord{
			Type:    r.Type,
			Name:    r.Name,
			Content: r.Content,
		}
	}
	return records, nil
}
