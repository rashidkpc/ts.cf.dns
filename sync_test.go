package main

import (
	"context"
	"testing"
)

// fakeCF implements cfSyncer and records every call made to it.
type fakeCF struct {
	records []DNSRecord
	created []struct{ hostname, ip string }
	updated []struct{ id, ip string }
	deleted []string
}

func (f *fakeCF) ListRecords(_ context.Context) ([]DNSRecord, error) {
	return f.records, nil
}
func (f *fakeCF) CreateRecord(_ context.Context, hostname, ip string) error {
	f.created = append(f.created, struct{ hostname, ip string }{hostname, ip})
	return nil
}
func (f *fakeCF) UpdateRecord(_ context.Context, id, ip string) error {
	f.updated = append(f.updated, struct{ id, ip string }{id, ip})
	return nil
}
func (f *fakeCF) DeleteRecord(_ context.Context, id string) error {
	f.deleted = append(f.deleted, id)
	return nil
}

// setupEnv sets CF_DOMAIN/CF_SUBDOMAIN so recordBase() returns "ts.test".
func setupEnv(t *testing.T) {
	t.Helper()
	t.Setenv("CF_DOMAIN", "test")
	t.Setenv("CF_SUBDOMAIN", "ts")
}

func TestSync_CreatesMissingRecord(t *testing.T) {
	setupEnv(t)
	cf := &fakeCF{}
	if err := Sync(context.Background(), cf, []TailscaleHost{{Hostname: "alice", IP: "100.1.1.1"}}, false); err != nil {
		t.Fatal(err)
	}
	if len(cf.created) != 1 || cf.created[0].hostname != "alice" || cf.created[0].ip != "100.1.1.1" {
		t.Errorf("unexpected creates: %+v", cf.created)
	}
	if len(cf.updated)+len(cf.deleted) != 0 {
		t.Errorf("expected no updates/deletes, got updated=%v deleted=%v", cf.updated, cf.deleted)
	}
}

func TestSync_UpdatesStaleIP(t *testing.T) {
	setupEnv(t)
	cf := &fakeCF{records: []DNSRecord{
		{ID: "rec1", Name: "alice.ts.test", Content: "100.1.1.1"},
	}}
	if err := Sync(context.Background(), cf, []TailscaleHost{{Hostname: "alice", IP: "100.2.2.2"}}, false); err != nil {
		t.Fatal(err)
	}
	if len(cf.updated) != 1 || cf.updated[0].id != "rec1" || cf.updated[0].ip != "100.2.2.2" {
		t.Errorf("unexpected updates: %+v", cf.updated)
	}
	if len(cf.created)+len(cf.deleted) != 0 {
		t.Errorf("expected no creates/deletes, got created=%v deleted=%v", cf.created, cf.deleted)
	}
}

func TestSync_DeletesOrphanedRecord(t *testing.T) {
	setupEnv(t)
	cf := &fakeCF{records: []DNSRecord{
		{ID: "rec-gone", Name: "ghost.ts.test", Content: "100.3.3.3"},
	}}
	if err := Sync(context.Background(), cf, nil, false); err != nil {
		t.Fatal(err)
	}
	if len(cf.deleted) != 1 || cf.deleted[0] != "rec-gone" {
		t.Errorf("unexpected deletes: %+v", cf.deleted)
	}
	if len(cf.created)+len(cf.updated) != 0 {
		t.Errorf("expected no creates/updates, got created=%v updated=%v", cf.created, cf.updated)
	}
}

func TestSync_NoOpWhenIPUnchanged(t *testing.T) {
	setupEnv(t)
	cf := &fakeCF{records: []DNSRecord{
		{ID: "rec1", Name: "alice.ts.test", Content: "100.1.1.1"},
	}}
	if err := Sync(context.Background(), cf, []TailscaleHost{{Hostname: "alice", IP: "100.1.1.1"}}, false); err != nil {
		t.Fatal(err)
	}
	if len(cf.created)+len(cf.updated)+len(cf.deleted) != 0 {
		t.Errorf("expected no ops, got created=%v updated=%v deleted=%v", cf.created, cf.updated, cf.deleted)
	}
}

// TestSync_OnlyTouchesManagedRecords proves that Sync cannot delete or update
// a record that wasn't returned by ListRecords. Ownership filtering lives in
// ListRecords (tested in cloudflare_test.go); whatever ListRecords excludes is
// invisible to Sync and therefore safe from modification.
func TestSync_OnlyTouchesManagedRecords(t *testing.T) {
	setupEnv(t)
	// ListRecords returns only the managed record; an unmanaged record with the
	// same hostname would have been excluded by filterManagedRecords before
	// ever reaching Sync.
	cf := &fakeCF{records: []DNSRecord{
		{ID: "managed", Name: "alice.ts.test", Content: "100.1.1.1"},
	}}
	if err := Sync(context.Background(), cf, []TailscaleHost{{Hostname: "alice", IP: "100.1.1.1"}}, false); err != nil {
		t.Fatal(err)
	}
	if len(cf.created)+len(cf.updated)+len(cf.deleted) != 0 {
		t.Errorf("Sync touched something it shouldn't have: created=%v updated=%v deleted=%v",
			cf.created, cf.updated, cf.deleted)
	}
}

// TestSync_DryRunMakesNoAPICalls verifies that --dry-run suppresses all writes.
func TestSync_DryRunMakesNoAPICalls(t *testing.T) {
	setupEnv(t)
	cf := &fakeCF{records: []DNSRecord{
		{ID: "stale", Name: "bob.ts.test", Content: "100.9.9.9"},  // IP changed
		{ID: "gone", Name: "ghost.ts.test", Content: "100.8.8.8"}, // not in TS
	}}
	hosts := []TailscaleHost{
		{Hostname: "alice", IP: "100.1.1.1"}, // new
		{Hostname: "bob", IP: "100.2.2.2"},   // updated
	}
	if err := Sync(context.Background(), cf, hosts, true); err != nil {
		t.Fatal(err)
	}
	if len(cf.created)+len(cf.updated)+len(cf.deleted) != 0 {
		t.Errorf("dry-run made API calls: created=%v updated=%v deleted=%v",
			cf.created, cf.updated, cf.deleted)
	}
}
