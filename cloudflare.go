package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloudflare/cloudflare-go"
)

// recordBase returns the base domain under which DNS records are managed.
// If CF_SUBDOMAIN is set, it returns "subdomain.domain" (e.g. "ts.funkadelic.net").
// Otherwise it returns CF_DOMAIN (e.g. "funkadelic.net").
func recordBase() string {
	domain := os.Getenv("CF_DOMAIN")
	sub := os.Getenv("CF_SUBDOMAIN")
	if sub != "" {
		return sub + "." + domain
	}
	return domain
}

// DNSRecord holds the ID, type, name, and value of a Cloudflare DNS record.
type DNSRecord struct {
	ID      string
	Type    string
	Name    string
	Content string
}

// CloudflareClient wraps the Cloudflare API with a pre-resolved zone ID.
type CloudflareClient struct {
	api    *cloudflare.API
	zoneID *cloudflare.ResourceContainer
}

// NewCloudflareClient creates a CloudflareClient, resolving the zone ID once at startup.
func NewCloudflareClient(ctx context.Context) (*CloudflareClient, error) {
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

	return &CloudflareClient{
		api:    api,
		zoneID: cloudflare.ZoneIdentifier(zones[0].ID),
	}, nil
}

const managedComment = "managed-by:ts.cf.dns"

// filterManagedRecords returns only A records that are owned by this tool
// (correct comment) and fall under the given base domain.
func filterManagedRecords(raw []cloudflare.DNSRecord, base string) []DNSRecord {
	suffix := "." + base
	var records []DNSRecord
	for _, r := range raw {
		if r.Comment != managedComment {
			continue
		}
		if !strings.HasSuffix(r.Name, suffix) {
			continue
		}
		records = append(records, DNSRecord{
			ID:      r.ID,
			Type:    r.Type,
			Name:    r.Name,
			Content: r.Content,
		})
	}
	return records
}

// ListRecords returns all managed A records under recordBase().
func (c *CloudflareClient) ListRecords(ctx context.Context) ([]DNSRecord, error) {
	raw, _, err := c.api.ListDNSRecords(ctx, c.zoneID, cloudflare.ListDNSRecordsParams{
		Type: "A",
	})
	if err != nil {
		return nil, fmt.Errorf("list dns records: %w", err)
	}
	return filterManagedRecords(raw, recordBase()), nil
}

// CreateRecord creates a managed A record: {hostname}.{recordBase()} → ip.
func (c *CloudflareClient) CreateRecord(ctx context.Context, hostname, ip string) error {
	ttl := 60
	_, err := c.api.CreateDNSRecord(ctx, c.zoneID, cloudflare.CreateDNSRecordParams{
		Type:    "A",
		Name:    hostname + "." + recordBase(),
		Content: ip,
		TTL:     ttl,
		Proxied: cloudflare.BoolPtr(false),
		Comment: managedComment,
	})
	if err != nil {
		return fmt.Errorf("create record %s: %w", hostname, err)
	}
	return nil
}

// UpdateRecord updates the IP of an existing record by ID.
func (c *CloudflareClient) UpdateRecord(ctx context.Context, id, ip string) error {
	ttl := 60
	_, err := c.api.UpdateDNSRecord(ctx, c.zoneID, cloudflare.UpdateDNSRecordParams{
		ID:      id,
		Content: ip,
		TTL:     ttl,
		Proxied: cloudflare.BoolPtr(false),
		Comment: &[]string{managedComment}[0],
	})
	if err != nil {
		return fmt.Errorf("update record %s: %w", id, err)
	}
	return nil
}

// DeleteRecord deletes a DNS record by ID.
func (c *CloudflareClient) DeleteRecord(ctx context.Context, id string) error {
	if err := c.api.DeleteDNSRecord(ctx, c.zoneID, id); err != nil {
		return fmt.Errorf("delete record %s: %w", id, err)
	}
	return nil
}
