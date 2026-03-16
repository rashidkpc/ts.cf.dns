package main

import (
	"context"
	"flag"
	"log"
	"time"
)

const syncInterval = 30 * time.Second

func main() {
	dryRun := flag.Bool("dry-run", false, "log what would change without making API calls")
	flag.Parse()

	ctx := context.Background()

	cf, err := NewCloudflareClient(ctx)
	if err != nil {
		log.Fatalf("cloudflare: %v", err)
	}

	if *dryRun {
		log.Printf("dry-run mode enabled — no changes will be made")
	}
	log.Printf("starting sync loop (interval: %s, base: %s)", syncInterval, recordBase())

	sync := func() {
		hosts, err := ListTailscaleHosts(ctx)
		if err != nil {
			log.Printf("tailscale: %v", err)
			return
		}
		if err := Sync(ctx, cf, hosts, *dryRun); err != nil {
			log.Printf("sync error: %v", err)
		}
	}

	sync()
	ticker := time.NewTicker(syncInterval)
	defer ticker.Stop()
	for range ticker.C {
		sync()
	}
}
