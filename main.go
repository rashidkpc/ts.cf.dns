package main

import (
	"context"
	"fmt"
	"log"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== Tailscale Hosts ===")
	tsHosts, err := ListTailscaleHosts(ctx)
	if err != nil {
		log.Fatalf("tailscale: %v", err)
	}
	for _, h := range tsHosts {
		fmt.Printf("%-40s %s\n", h.Hostname, h.IP)
	}

	fmt.Println("\n=== Cloudflare DNS Records ===")
	cfRecords, err := ListCloudflareDNSRecords(ctx)
	if err != nil {
		log.Fatalf("cloudflare: %v", err)
	}
	for _, r := range cfRecords {
		fmt.Printf("%-6s %-50s %s\n", r.Type, r.Name, r.Content)
	}
}
