package main

import (
	"context"
	"log"
	"strings"
)

// cfSyncer is the subset of CloudflareClient used by Sync, exposed as an
// interface so tests can substitute a fake without hitting the real API.
type cfSyncer interface {
	ListRecords(ctx context.Context) ([]DNSRecord, error)
	CreateRecord(ctx context.Context, hostname, ip string) error
	UpdateRecord(ctx context.Context, id, ip string) error
	DeleteRecord(ctx context.Context, id string) error
}

// Sync reconciles the provided Tailscale hosts with Cloudflare DNS records.
// When dryRun is true it logs what would happen but makes no API changes.
// It returns the first error encountered but continues processing all entries.
func Sync(ctx context.Context, cf cfSyncer, hosts []TailscaleHost, dryRun bool) error {
	cfRecords, err := cf.ListRecords(ctx)
	if err != nil {
		return err
	}

	base := recordBase()
	suffix := "." + base

	// Build map: hostname → DNSRecord (managed records only, by construction)
	cfMap := make(map[string]DNSRecord, len(cfRecords))
	for _, r := range cfRecords {
		host := strings.TrimSuffix(r.Name, suffix)
		cfMap[host] = r
	}

	// Build set of TS hostnames for the deletion pass
	tsMap := make(map[string]string, len(hosts))
	for _, h := range hosts {
		tsMap[h.Hostname] = h.IP
	}

	var firstErr error

	// Create or update
	for _, h := range hosts {
		existing, exists := cfMap[h.Hostname]
		switch {
		case !exists:
			if dryRun {
				log.Printf("[dry-run] would create %s%s → %s", h.Hostname, suffix, h.IP)
				continue
			}
			if err := cf.CreateRecord(ctx, h.Hostname, h.IP); err != nil {
				log.Printf("error: %v", err)
				if firstErr == nil {
					firstErr = err
				}
			} else {
				log.Printf("created %s%s → %s", h.Hostname, suffix, h.IP)
			}
		case existing.Content != h.IP:
			if dryRun {
				log.Printf("[dry-run] would update %s%s: %s → %s", h.Hostname, suffix, existing.Content, h.IP)
				continue
			}
			if err := cf.UpdateRecord(ctx, existing.ID, h.IP); err != nil {
				log.Printf("error: %v", err)
				if firstErr == nil {
					firstErr = err
				}
			} else {
				log.Printf("updated %s%s: %s → %s", h.Hostname, suffix, existing.Content, h.IP)
			}
		}
	}

	// Delete managed records for hosts no longer in Tailscale
	for host, r := range cfMap {
		if _, ok := tsMap[host]; !ok {
			if dryRun {
				log.Printf("[dry-run] would delete %s%s", host, suffix)
				continue
			}
			if err := cf.DeleteRecord(ctx, r.ID); err != nil {
				log.Printf("error: %v", err)
				if firstErr == nil {
					firstErr = err
				}
			} else {
				log.Printf("deleted %s%s", host, suffix)
			}
		}
	}

	return firstErr
}
